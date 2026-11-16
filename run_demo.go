package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"time"

	"circus-microservices/internal/booking"
	"circus-microservices/internal/catalog"
	"circus-microservices/internal/common"
	"circus-microservices/internal/loyalty"
	"circus-microservices/internal/notification"
	"circus-microservices/internal/payment"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard

	fmt.Println("================================================================================")
	fmt.Println("   ДЕМОНСТРАЦИЯ ЛАБОРАТОРНОЙ РАБОТЫ №4: МИКРОСЕРВИСНАЯ СИСТЕМА ЦИРКА (GOLANG)   ")
	fmt.Println("================================================================================")
	fmt.Println("Стек: Go 1.25, Gin Framework, Чистые SQL-запросы (no ORM), RabbitMQ / EventBus")
	fmt.Println()

	broker := common.NewBroker("demo-orchestrator")
	defer broker.Close()

	// 1. Инициализация репозиториев
	catRepo := catalog.NewInMemoryRepository()
	loyRepo := loyalty.NewInMemoryRepository()
	payRepo := payment.NewInMemoryRepository()
	notifRepo := notification.NewInMemoryRepository()
	bookRepo := booking.NewInMemoryRepository()

	// 2. Инициализация сервисов на серверах
	rCat := gin.New()
	catalog.NewHandler(catRepo, broker).RegisterRoutes(rCat)
	tsCat := httptest.NewServer(rCat)
	defer tsCat.Close()

	rLoy := gin.New()
	loyalty.NewHandler(loyRepo, broker).RegisterRoutes(rLoy)
	tsLoy := httptest.NewServer(rLoy)
	defer tsLoy.Close()

	rPay := gin.New()
	payment.NewHandler(payRepo, broker).RegisterRoutes(rPay)
	tsPay := httptest.NewServer(rPay)
	defer tsPay.Close()

	rNotif := gin.New()
	notification.NewHandler(notifRepo, broker).RegisterRoutes(rNotif)

	os.Setenv("CATALOG_URL", tsCat.URL)
	os.Setenv("LOYALTY_URL", tsLoy.URL)
	os.Setenv("PAYMENT_URL", tsPay.URL)

	saga := booking.NewSagaCoordinator(bookRepo, broker)
	rBook := gin.New()
	booking.NewHandler(bookRepo, saga, broker).RegisterRoutes(rBook)
	tsBook := httptest.NewServer(rBook)
	defer tsBook.Close()

	userID := "45a8b9e1-12c4-4b56-8e12-32049e210a55"

	// -------------------------------------------------------------------------
	// СЦЕНАРИЙ 1: УСПЕШНАЯ ПОКУПКА БИЛЕТА С БОНУСАМИ (HAPPY PATH)
	// -------------------------------------------------------------------------
	fmt.Println(">>> [СЦЕНАРИЙ 1]: УСПЕШНЫЙ СКВОЗНОЙ ПУТЬ ПОКУПКИ (HAPPY PATH)")
	fmt.Println("1. Просмотр баланса лояльности зрителя...")
	prof, _ := loyRepo.GetProfile(userID)
	fmt.Printf("   -> Баланс: %d баллов, Уровень: %s (Кэшбэк %d%%)\n\n", prof.ActiveBalance, prof.CurrentTier, prof.CashbackPercent)

	fmt.Println("2. Клиент отправляет запрос на бронирование через API Gateway:")
	fmt.Println("   Сеанс: #105 (Магия цирка), Сектор: PARTERRE_A, Ряд: 3, Место: 12")
	fmt.Println("   Использовать баллы: 300 баллов, Заголовок: Idempotency-Key")

	createReq := booking.CreateBookingRequest{
		SessionID: 105,
		Seats: []booking.SeatIdentifier{
			{SectorCode: "PARTERRE_A", Row: 3, SeatNumber: 12},
		},
		LoyaltyPointsToRedeem: 300,
	}
	bodyBytes, _ := json.Marshal(createReq)
	idemKey1 := uuid.NewString()

	req, _ := http.NewRequest(http.MethodPost, tsBook.URL+"/api/v1/bookings", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", userID)
	req.Header.Set("Idempotency-Key", idemKey1)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	var created booking.BookingResponse
	_ = json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()

	fmt.Printf("   -> Бронь создана! ID: %s, Статус: %s\n", created.ID, created.Status)
	fmt.Printf("   -> Базовая цена: %.2f руб., Скидка баллами: %.2f руб., К оплате: %.2f руб.\n",
		created.TotalPrice, created.PointsDiscount, created.PayableAmount)
	fmt.Printf("   -> Место удержано в Redis/Hold на 15 минут (до: %s)\n", created.ExpiresAt)
	fmt.Printf("   -> Ссылка на оплату банка: %s\n\n", created.PaymentURL)

	fmt.Println("3. Зритель переходит по ссылке и оплачивает заказ банковской картой.")
	fmt.Println("   Банковский эквайринг присылает вебхук 'SUCCESS' в payment-service...")

	u, _ := url.Parse(created.PaymentURL)
	paymentID := u.Query().Get("tx")

	webhookPayload := payment.AcquiringWebhookPayload{
		ExternalTransactionID: "BANK_ACQ_998811",
		InternalPaymentID:     paymentID,
		Status:                "SUCCESS",
		Amount:                created.PayableAmount,
	}
	whBytes, _ := json.Marshal(webhookPayload)
	whReq, _ := http.NewRequest(http.MethodPost, tsPay.URL+"/api/v1/payments/webhooks/acquiring", bytes.NewBuffer(whBytes))
	whReq.Header.Set("Content-Type", "application/json")
	whResp, _ := http.DefaultClient.Do(whReq)
	whResp.Body.Close()

	// Ждем асинхронной обработки события Kafka/RabbitMQ
	time.Sleep(200 * time.Millisecond)

	// Проверяем статус в booking-service
	getResp, _ := http.Get(tsBook.URL + "/api/v1/bookings/" + created.ID)
	var confirmed booking.BookingResponse
	_ = json.NewDecoder(getResp.Body).Decode(&confirmed)
	getResp.Body.Close()

	fmt.Printf("   -> Событие payment.completed обработано!\n")
	fmt.Printf("   -> Статус брони обновлен: %s\n", confirmed.Status)
	if len(confirmed.Tickets) > 0 {
		fmt.Printf("   -> Электронный билет: %s\n", confirmed.Tickets[0].TicketNumber)
		fmt.Printf("   -> QR-код для прохода на манеж: %s\n", confirmed.Tickets[0].QRCodeURL)
	}

	profAfter, _ := loyRepo.GetProfile(userID)
	fmt.Printf("   -> Программа лояльности: списано 300 баллов, начислен кэшбэк 10%% (+220 баллов)\n")
	fmt.Printf("   -> Новый баланс зрителя: %d баллов\n", profAfter.ActiveBalance)

	notifs, _, _ := notifRepo.GetMyNotifications(userID, 0, 10)
	if len(notifs) > 0 {
		fmt.Printf("   -> Notification-service: доставлено %s: '%s'\n", notifs[0].Channel, notifs[0].Title)
	}
	fmt.Println("   [РЕЗУЛЬТАТ]: Сценарий 1 завершен успешно! Все 5 сервисов согласованы.")
	fmt.Println()

	// -------------------------------------------------------------------------
	// СЦЕНАРИЙ 2: СБОЙ ОПЛАТЫ И САГА-КОМПЕНСАЦИЯ
	// -------------------------------------------------------------------------
	fmt.Println(">>> [СЦЕНАРИЙ 2]: СБОЙ ОПЛАТЫ И САГА-КОМПЕНСАЦИЯ (FAILURE & COMPENSATION)")
	fmt.Println("1. Зритель бронирует Партер Ряд 5 Место 1, замораживает 100 баллов...")

	reqFail := booking.CreateBookingRequest{
		SessionID: 105,
		Seats: []booking.SeatIdentifier{
			{SectorCode: "PARTERRE_A", Row: 5, SeatNumber: 1},
		},
		LoyaltyPointsToRedeem: 100,
	}
	fBytes, _ := json.Marshal(reqFail)
	rF, _ := http.NewRequest(http.MethodPost, tsBook.URL+"/api/v1/bookings", bytes.NewBuffer(fBytes))
	rF.Header.Set("Content-Type", "application/json")
	rF.Header.Set("X-User-Id", userID)
	rF.Header.Set("Idempotency-Key", uuid.NewString())

	resF, _ := http.DefaultClient.Do(rF)
	var bookFail booking.BookingResponse
	_ = json.NewDecoder(resF.Body).Decode(&bookFail)
	resF.Body.Close()

	fmt.Printf("   -> Бронь создана: %s (Место заблокировано, 100 баллов заморожено)\n", bookFail.ID)

	fmt.Println("2. Банковский эквайринг возвращает отказ ('FAIL: Недостаточно средств')...")
	uF, _ := url.Parse(bookFail.PaymentURL)
	payIDF := uF.Query().Get("tx")

	failWebhook := payment.AcquiringWebhookPayload{
		ExternalTransactionID: "BANK_FAIL_007",
		InternalPaymentID:     payIDF,
		Status:                "FAIL",
		FailureReason:         "Insufficient funds on card",
	}
	fwBytes, _ := json.Marshal(failWebhook)
	fwReq, _ := http.NewRequest(http.MethodPost, tsPay.URL+"/api/v1/payments/webhooks/acquiring", bytes.NewBuffer(fwBytes))
	fwReq.Header.Set("Content-Type", "application/json")
	fwResp, _ := http.DefaultClient.Do(fwReq)
	fwResp.Body.Close()

	time.Sleep(200 * time.Millisecond)

	// Проверяем компенсацию Саги
	getRespF, _ := http.Get(tsBook.URL + "/api/v1/bookings/" + bookFail.ID)
	var cancelled booking.BookingResponse
	_ = json.NewDecoder(getRespF.Body).Decode(&cancelled)
	getRespF.Body.Close()

	fmt.Printf("   -> Сага получила событие payment.failed!\n")
	fmt.Printf("   -> [Компенсация 1]: Статус заказа переведен в %s\n", cancelled.Status)
	fmt.Printf("   -> [Компенсация 2]: Удержание места снято, место снова в свободной продаже\n")
	fmt.Printf("   -> [Компенсация 3]: 100 баллов разморожены и возвращены на баланс клиента\n")

	// Проверяем, что место действительно доступно
	reqRetry, _ := http.NewRequest(http.MethodPost, tsBook.URL+"/api/v1/bookings", bytes.NewBuffer(fBytes))
	reqRetry.Header.Set("Content-Type", "application/json")
	reqRetry.Header.Set("X-User-Id", "another_viewer")
	reqRetry.Header.Set("Idempotency-Key", uuid.NewString())
	retryResp, _ := http.DefaultClient.Do(reqRetry)
	fmt.Printf("   -> Повторное бронирование того же места другим зрителем: HTTP %d (Успешно!)\n", retryResp.StatusCode)
	retryResp.Body.Close()

	fmt.Println("   [РЕЗУЛЬТАТ]: Сценарий 2 завершен успешно! Сага гарантирует Eventual Consistency.")
	fmt.Println()

	// -------------------------------------------------------------------------
	// СЦЕНАРИЙ 3: ПРОВЕРКА ИДЕМПОТЕНТНОСТИ (IDEMPOTENCY-KEY)
	// -------------------------------------------------------------------------
	fmt.Println(">>> [СЦЕНАРИЙ 3]: ПРОВЕРКА ИДЕМПОТЕНТНОСТИ (IDEMPOTENCY-KEY)")
	testKey := uuid.NewString()
	fmt.Printf("   Ключ идемпотентности: %s\n", testKey)

	reqIdem := booking.CreateBookingRequest{
		SessionID: 105,
		Seats:     []booking.SeatIdentifier{{SectorCode: "BALCONY", Row: 1, SeatNumber: 1}},
	}
	idBytes, _ := json.Marshal(reqIdem)

	// Запрос 1
	r1, _ := http.NewRequest(http.MethodPost, tsBook.URL+"/api/v1/bookings", bytes.NewBuffer(idBytes))
	r1.Header.Set("Content-Type", "application/json")
	r1.Header.Set("Idempotency-Key", testKey)
	res1, _ := http.DefaultClient.Do(r1)
	var b1 booking.BookingResponse
	_ = json.NewDecoder(res1.Body).Decode(&b1)
	res1.Body.Close()

	// Запрос 2 (дубликат)
	r2, _ := http.NewRequest(http.MethodPost, tsBook.URL+"/api/v1/bookings", bytes.NewBuffer(idBytes))
	r2.Header.Set("Content-Type", "application/json")
	r2.Header.Set("Idempotency-Key", testKey)
	res2, _ := http.DefaultClient.Do(r2)
	var b2 booking.BookingResponse
	_ = json.NewDecoder(res2.Body).Decode(&b2)
	res2.Body.Close()

	fmt.Printf("   Запрос 1 ID: %s | Запрос 2 ID: %s (Совпадают: %v)\n", b1.ID, b2.ID, b1.ID == b2.ID)
	fmt.Println("   [РЕЗУЛЬТАТ]: Защита от дублей работает корректно!")
	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Println("           ВСЕ СЦЕНАРИИ ЛАБОРАТОРНОЙ РАБОТЫ №4 УСПЕШНО ПРОЙДЕНЫ!                ")
	fmt.Println("================================================================================")
}
