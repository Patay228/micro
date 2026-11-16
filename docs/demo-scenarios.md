# Описание демонстрационных сценариев (Лабораторная работа №4)

В данном документе подробно описаны проверенные сценарии работы микросервисной системы цирка, спроектированной в Лабораторной работе №3 и реализованной на языке **Go (Gin Framework)**.

---

## 1. Сводная таблица реализации взаимодействий

Согласно требованиям методички, в системе четко разграничены реальные сервисы, протоколы взаимодействия и моки:

| Взаимодействие | Тип | Реализация | Описание |
| :--- | :---: | :--- | :--- |
| **Клиент → API Gateway** | `sync` | Реальный сервис (`:8000`) | Единая точка входа, проксирование `/api/v1/*`, защита внутренних эндпоинтов `/internal/*` |
| **Gateway → Микросервисы** | `sync` | Реальный HTTP ReverseProxy | Маршрутизация на 5 независимых сервисов |
| **booking → catalog** | `sync` | Реальный REST (`/internal/v1/sessions/:id`) | Проверка сеанса и базовой стоимости секторов зала |
| **booking → loyalty** | `sync` | Реальный REST (`/internal/v1/loyalty/reservations`) | Заморозка баллов лояльности при бронировании |
| **booking → payment** | `sync` | Реальный REST (`/internal/v1/payments`) | Создание платежа на сумму за вычетом бонусов |
| **Банк → payment (Webhook)** | `sync` | Эмулируется моком банка (`mocks/bank_mock.go`) | Вебхук об успехе (`SUCCESS`) или отказе (`FAIL: Insufficient funds`) |
| **payment → брокер сообщений** | `async` | RabbitMQ / In-Memory EventBus | Публикация событий `payment.completed` и `payment.failed` |
| **брокер → booking-service** | `async` | RabbitMQ / In-Memory EventBus | Выпуск электронных билетов с QR-кодом или запуск компенсирующих действий |
| **booking → брокер сообщений** | `async` | RabbitMQ / In-Memory EventBus | Публикация `booking.confirmed`, `booking.expired`, `refund.requested` |
| **брокер → loyalty-service** | `async` | RabbitMQ / In-Memory EventBus | Окончательное списание баллов и начисление 10% кэшбэка |
| **брокер → notification-service** | `async` | RabbitMQ / In-Memory EventBus | Формирование билетов и логов доставки |
| **notification → SMS/Email** | `async` | Эмулируется моком транспорта (`mocks/sms_mock.go`) | Доставка электронных билетов и чеков |

---

## 2. Сценарий 1: Успешная покупка билета с бонусами (Happy Path)

### Описание
Зритель оформляет билет на цирковое шоу «Магия цирка: Полет над манежем» (сеанс #105, партер, ряд 3, место 12) стоимостью 2500 руб. Зритель списывает 300 накопленных бонусов, оплачивает оставшиеся 2200 руб. картой, получает электронный билет с QR-кодом, а на его счет начисляется 10% кэшбэк (220 баллов).

### Пошаговое выполнение:
1. **Запрос баланса зрителя:**
   ```http
   GET /api/v1/loyalty/me
   X-User-Id: 45a8b9e1-12c4-4b56-8e12-32049e210a55
   ```
   *Ответ:* `200 OK`, `activeBalance: 500`, `currentTier: CONNOISSEUR`.

2. **Создание брони через API Gateway:**
   ```http
   POST /api/v1/bookings
   X-User-Id: 45a8b9e1-12c4-4b56-8e12-32049e210a55
   Idempotency-Key: a1b2c3d4-e5f6-7a8b-9c0d-e1f2a3b4c5d6
   Content-Type: application/json

   {
     "sessionId": 105,
     "seats": [
       { "sectorCode": "PARTERRE_A", "row": 3, "seatNumber": 12 }
     ],
     "loyaltyPointsToRedeem": 300
   }
   ```
   *Шаги Саги внутри системы:*
   - `booking-service` синхронно вызывает `catalog-service` (`GET /internal/v1/sessions/105`), валидирует цену сектора (2500 руб.).
   - `booking-service` блокирует место в хранилище на 15 минут (`expiresAt = now + 15m`).
   - `booking-service` синхронно вызывает `loyalty-service` (`POST /internal/v1/loyalty/reservations`), где 300 баллов переходят в статус `RESERVED`.
   - `booking-service` синхронно вызывает `payment-service` (`POST /internal/v1/payments`) на сумму $2500 - 300 = 2200$ руб.
   
   *Ответ клиенту:*
   ```json
   {
     "id": "7ec2e476-d372-4a25-948e-e08bfff1b2d8",
     "sessionId": 105,
     "userId": "45a8b9e1-12c4-4b56-8e12-32049e210a55",
     "status": "PENDING_PAYMENT",
     "totalPrice": 2500.00,
     "pointsDiscount": 300.00,
     "payableAmount": 2200.00,
     "paymentUrl": "https://pay.bank-acquiring.ru/process?tx=6bb894fa-f30b-4b79-8862-be0a85f3b7d5",
     "expiresAt": "2026-11-16T15:15:00Z",
     "seats": [{ "sectorCode": "PARTERRE_A", "row": 3, "seatNumber": 12 }]
   }
   ```

3. **Эмуляция оплаты в банке (Webhook эквайринга):**
   ```http
   POST /api/v1/payments/webhooks/acquiring
   Content-Type: application/json

   {
     "externalTransactionId": "BANK_ACQ_998811",
     "internalPaymentId": "6bb894fa-f30b-4b79-8862-be0a85f3b7d5",
     "status": "SUCCESS",
     "amount": 2200.00
   }
   ```

4. **Асинхронные события в брокере (RabbitMQ):**
   - `payment-service` публикует событие `payment.completed`.
   - `booking-service` консьюмит событие: выпускает электронный билет `CRK-105-0312-acc8eb` с QR-кодом, переводит бронь в `CONFIRMED` и публикует `booking.confirmed`.
   - `loyalty-service` консьюмит `booking.confirmed`: финализирует списание 300 баллов и начисляет 220 бонусов кэшбэка (10% от 2200 руб.). Баланс становится 420 баллов.
   - `notification-service` консьюмит `booking.confirmed`: отправляет письмо с билетом и QR-кодом зрителю.

---

## 3. Сценарий 2: Сбой оплаты и компенсирующие действия Саги

### Описание
Зритель пытается оплатить бронь, но банк отклоняет транзакцию (например, лимит карты или недостаточно средств). Система должна вернуть все ресурсы в исходное состояние (Eventual Consistency): аннулировать бронь, освободить заблокированные места у манежа и вернуть замороженные баллы на баланс зрителя.

### Пошаговое выполнение:
1. Зритель забронировал место (Ряд 5, Место 1) со списанием 100 баллов. Место заблокировано, 100 баллов заморожено.
2. Банковский эквайринг присылает вебхук отказа:
   ```http
   POST /api/v1/payments/webhooks/acquiring
   Content-Type: application/json

   {
     "externalTransactionId": "BANK_FAIL_007",
     "internalPaymentId": "pay_fail_id",
     "status": "FAIL",
     "failureReason": "Insufficient funds on card"
   }
   ```
3. `payment-service` обновляет статус платежа на `FAILED` и публикует событие `payment.failed` в топик `circus.payments.events`.
4. **Компенсирующие действия `booking-service`:**
   - Переводит бронь в статус `CANCELLED`.
   - Удаляет удержание места из базы/Redis — место мгновенно возвращается в свободную продажу.
   - Вызывает компенсирующий метод `DELETE /internal/v1/loyalty/reservations/:id` — 100 баллов размораживаются и возвращаются зрителю.
   - Публикует событие `booking.cancelled`.
5. **Проверка инварианта:** Другой зритель отправляет запрос на то же самое место (Ряд 5, Место 1) и успешно получает ответ `201 Created` — место не «зависло»!

---

## 4. Сценарий 3: Проверка идемпотентности (`Idempotency-Key`)

1. Клиент отправляет запрос `POST /api/v1/bookings` с заголовком:
   ```http
   Idempotency-Key: 6fbaf671-5604-4d77-9fdc-3fdaab9ceeb8
   ```
2. Сервис создает бронь с ID `c744c72c-3d37-419c-92ab-4f97f93dd486`.
3. Из-за сетевого сбоя клиент повторно отправляет точно такой же запрос с тем же `Idempotency-Key`.
4. Сервис не создает повторный заказ и не списывает баллы второй раз, а возвращает уже созданный заказ с тем же ID и HTTP-статусом `200/201`.

---

## 5. Сценарий 4: Коллизия мест и формат ошибки RFC 7807 (Problem Details)

1. Зритель 1 удерживает место Ряд 2, Место 10.
2. Зритель 2 пытается забронировать то же место.
3. Сервис возвращает ошибку `409 Conflict` со строгим форматом RFC 7807:
   ```json
   {
     "type": "https://circus-theatre.ru/errors/409",
     "title": "Seat Conflict",
     "status": 409,
     "detail": "seat currently held by another user: sector PARTERRE_A, row 2, seat 10",
     "instance": "/api/v1/bookings"
   }
   ```
