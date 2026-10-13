# -*- coding: utf-8 -*-
"""
build_word_doc.py
Скрипт формирования итогового документа Word (.docx) для Лабораторной работы №3
«Декомпозиция и проектирование микросервисной системы» (Информационная система цирка).
"""

import os
from docx import Document
from docx.shared import Inches, Pt, RGBColor
from docx.enum.text import WD_ALIGN_PARAGRAPH
from docx.enum.table import WD_TABLE_ALIGNMENT, WD_ALIGN_VERTICAL
from docx.oxml import OxmlElement
from docx.oxml.ns import qn

def set_cell_background(cell, hex_color):
    tcPr = cell._tc.get_or_add_tcPr()
    shd = OxmlElement('w:shd')
    shd.set(qn('w:val'), 'clear')
    shd.set(qn('w:color'), 'auto')
    shd.set(qn('w:fill'), hex_color)
    tcPr.append(shd)

def set_cell_margins(cell, top=100, bottom=100, left=150, right=150):
    tcPr = cell._tc.get_or_add_tcPr()
    tcMar = OxmlElement('w:tcMar')
    for m, val in [('top', top), ('bottom', bottom), ('left', left), ('right', right)]:
        node = OxmlElement(f'w:{m}')
        node.set(qn('w:w'), str(val))
        node.set(qn('w:type'), 'dxa')
        tcMar.append(node)
    tcPr.append(tcMar)

def create_report():
    doc = Document()

    # Поля страницы 2 см со всех сторон
    sections = doc.sections
    for section in sections:
        section.top_margin = Inches(0.8)
        section.bottom_margin = Inches(0.8)
        section.left_margin = Inches(0.8)
        section.right_margin = Inches(0.8)

    # Базовые стили
    style_normal = doc.styles['Normal']
    style_normal.font.name = 'Calibri'
    style_normal.font.size = Pt(11)
    style_normal.font.color.rgb = RGBColor(33, 33, 33)

    # Заголовок документа
    title = doc.add_paragraph()
    title.alignment = WD_ALIGN_PARAGRAPH.CENTER
    run_title = title.add_run("Лабораторная работа №3\n«Декомпозиция и проектирование микросервисной системы»")
    run_title.bold = True
    run_title.font.size = Pt(18)
    run_title.font.color.rgb = RGBColor(26, 35, 126)

    sub_title = doc.add_paragraph()
    sub_title.alignment = WD_ALIGN_PARAGRAPH.CENTER
    run_sub = sub_title.add_run("Предметная область: Информационная система цирка («Цирк на Манеже»)\n")
    run_sub.font.size = Pt(13)
    run_sub.font.italic = True
    run_sub.font.color.rgb = RGBColor(84, 110, 122)

    # -------------------------------------------------------------------------
    # ШАГ 1
    # -------------------------------------------------------------------------
    h1 = doc.add_heading("Шаг 1. Анализ предметной области", level=1)
    h1.style.font.color.rgb = RGBColor(21, 101, 192)

    doc.add_heading("1.1. Описание системы", level=2)
    p_desc = doc.add_paragraph(
        "Информационная система цирка предназначена для зрителей, администраторов и маркетологов циркового представления. "
        "Система предоставляет интерактивную онлайн-афишу с возможностью фильтрации и детального просмотра информации о шоу, "
        "позволяет зарегистрированным пользователям выбирать места вокруг манежа (партер, амфитеатр, ложи), бронировать их с удержанием "
        "на 15 минут, покупать билеты и участвовать в многоуровневой программе лояльности с накоплением баллов и уровней привилегий. "
        "Система обеспечивает приём оплаты через банковский эквайринг, генерирует электронные билеты с QR-кодами и доставляет их через Email и SMS. "
        "Администраторы централизованно управляют афишей и расписанием, а маркетологи настраивают правила начисления бонусов, создают промоакции "
        "и стимулируют спрос на представления."
    )

    doc.add_heading("1.2. Акторы системы", level=2)
    actors = [
        ("Зритель (зарегистрированный пользователь)", "выбирает сеансы и места, списывает баллы, оплачивает билеты, получает билеты с QR-кодами, запрашивает возврат при необходимости."),
        ("Гость (неавторизованный пользователь)", "просматривает афишу представлений и расписание сеансов, проходит регистрацию."),
        ("Администратор цирка", "создаёт и ведёт цирковую афишу, составляет расписание сеансов, настраивает схему зала и тарифные пояса."),
        ("Маркетолог цирка", "настраивает правила программы лояльности, повышенный кэшбэк на семейные или утренние сеансы, проводит рассылки."),
        ("Внешний платёжный шлюз (эквайринг)", "принимает оплату по банковским картам и СБП, обрабатывает фискальные чеки и проводит возвраты."),
        ("Внешний SMS/Email провайдер", "осуществляет доставку электронных билетов и сервисных напоминаний зрителям."),
        ("Keycloak (IAM провайдер)", "выполняет регистрацию, сквозную аутентификацию (OAuth 2.0 / OIDC) и выдачу JWT токенов.")
    ]
    for act, desc in actors:
        p = doc.add_paragraph(style='List Bullet')
        r_b = p.add_run(f"{act}: ")
        r_b.bold = True
        p.add_run(desc)

    doc.add_heading("1.3. Ключевые пользовательские сценарии (User Stories)", level=2)
    stories = [
        "Как зритель, я хочу фильтровать список представлений афиши по жанрам и датам, чтобы выбрать удобный сеанс для всей семьи.",
        "Как зритель, я хочу забронировать конкретные места на схеме манежа с временным удержанием на 15 минут, чтобы успеть оплатить заказ без риска потери мест.",
        "Как зритель, я хочу видеть свой баланс баллов и уровень привилегий («Новичок», «Знаток манежа», «Почётный зритель»), чтобы понимать доступную выгоду.",
        "Как зритель, я хочу частично оплатить заказ накопленными бонусами лояльности, чтобы получить скидку на посещение цирка.",
        "Как зритель, я хочу получить электронный билет с QR-кодом сразу после оплаты на Email и SMS, чтобы предъявить его на турникете манежа.",
        "Как зритель, я хочу иметь возможность вернуть билет при форс-мажоре, получая возврат до 50% стоимости в соответствии с правилами цирка.",
        "Как администратор цирка, я хочу добавлять новые цирковые программы и открывать сеансы в расписании, чтобы поддерживать актуальную афишу.",
        "Как маркетолог, я хочу настраивать повышенный процент бонусов на утренние показы, чтобы стимулировать заполняемость зала."
    ]
    for st in stories:
        p = doc.add_paragraph(style='List Bullet')
        p.add_run(st)

    doc.add_heading("1.4. Доменные события системы (Event Storming)", level=2)
    events_list = [
        ("ShowCreated", "Администратор создал новую цирковую программу в афише."),
        ("ShowUpdated", "Администратор обновил информацию или цены представления."),
        ("ShowArchived", "Шоу снято с активного репертуара."),
        ("BookingCreated", "Зритель выбрал места; бронь создана в статусе ожидания оплаты (удержание мест на 15 минут)."),
        ("BookingConfirmed", "Бронь успешно подтверждена после фиксации банковской оплаты, выпущены билеты."),
        ("BookingCancelled", "Бронь отменена зрителем либо банком из-за отказа в проведении платежа."),
        ("BookingExpired", "Бронь аннулирована по истечении 15-минутного таймаута без оплаты."),
        ("PaymentCompleted", "Банковский платёж успешно проведён."),
        ("PaymentFailed", "Платёж отклонён банком (недостаточно средств или отказ карты)."),
        ("RefundRequested", "Зритель подал заявку на возврат билета."),
        ("RefundCompleted", "Денежные средства (50% от суммы) возвращены на карту плательщика."),
        ("PointsReserved", "Бонусные баллы лояльности временно заморожены под оформляемый заказ."),
        ("PointsRedeemed", "Баллы окончательно списаны в счёт скидки после подтверждения оплаты."),
        ("PointsAccrued", "Начислен бонусный кэшбэк за покупку билетов на представление."),
        ("TierUpgraded", "Зритель достиг порога посещений и переведён на более высокий ранг лояльности."),
        ("PenaltyApplied", "Списаны баллы и применены санкции лояльности при возврате билета."),
        ("NotificationDispatched", "Электронный билет или уведомление успешно переданы шлюзу связи.")
    ]
    for ev, desc in events_list:
        p = doc.add_paragraph(style='List Bullet')
        r_ev = p.add_run(f"{ev}: ")
        r_ev.bold = True
        r_ev.font.color.rgb = RGBColor(46, 125, 50)
        p.add_run(desc)

    doc.add_heading("1.5. Ограниченные контексты (Bounded Contexts) и глоссарии", level=2)

    contexts = [
        ("1. Контекст: Каталог цирковых программ (catalog-service)",
         "Управление репертуаром цирка, афишей, сеансами и ценовыми секторами манежа.",
         [
             ("Шоу (Show)", "Творческая программа цирка (название, жанр, длительность, возрастной ценз, постер)."),
             ("Сеанс (Session)", "Показ шоу в зале в конкретные дату и время."),
             ("Сектор манежа (Sector)", "Категория мест вокруг манежа (Партер А, Партер Б, Амфитеатр, Ложа), определяющая базовую стоимость билета."),
             ("Афиша (Playbill)", "Сводное расписание актуальных сеансов для витрины цирка.")
         ]),
        ("2. Контекст: Бронирование и продажа билетов (booking-service)",
         "Контроль доступности мест, временное удержание на 15 минут, выпуск билетов и сага-оркестрация покупки.",
         [
             ("Бронь (Booking)", "Заказ на покупку мест в статусе ожидания оплаты (PENDING_PAYMENT) либо подтверждения (CONFIRMED)."),
             ("Удержание места (Seat Hold)", "Временная блокировка места в Redis с временем жизни 15 минут."),
             ("Билет (Ticket)", "Электронный документ с уникальным номером и защищённым QR-кодом для прохода в зал цирка."),
             ("Возврат билета (Ticket Refund)", "Операция отмены брони и возврата до 50% стоимости по правилам цирка.")
         ]),
        ("3. Контекст: Оплата и финансовые расчёты (payment-service)",
         "Интеграция с банковским эквайрингом, фискализация чеков и проведение возвратов средств.",
         [
             ("Платёж (Payment)", "Транзакция списания денег с карты зрителя через банк на сумму за вычетом бонусных баллов."),
             ("Платёжная сессия (Payment Session)", "Временный URL для безопасного перенаправления на платёжный шлюз."),
             ("Фискальный чек (Receipt)", "Официальный фискальный документ, подтверждающий оплату либо возврат."),
             ("Возврат денег (Refund)", "Обратное перечисление денежных средств на карту плательщика.")
         ]),
        ("4. Контекст: Программа лояльности (loyalty-service)",
         "Учёт бонусных баллов, списание скидок, уровни привилегий зрителей и маркетинговые формулы кэшбэка.",
         [
             ("Баланс (Loyalty Balance)", "Количество активных бонусов на счёте зрителя (1 балл = 1 рубль)."),
             ("Резерв баллов (Points Reservation)", "Заморозка баллов на время действия неоплаченной брони."),
             ("Уровень привилегий (Tier)", "Ранг зрителя («Новичок» — 5%, «Знаток манежа» — 10%, «Почётный зритель» — 15%)."),
             ("Маркетинговое правило (Rule)", "Формула повышенного начисления баллов (например, акция выходного дня).")
         ]),
        ("5. Контекст: Уведомления и информирование (notification-service)",
         "Формирование и отправка электронных билетов с QR-кодами, чеков и напоминаний через Email и SMS.",
         [
             ("Шаблон (Template)", "Параметризованный текст письма или SMS с подстановкой номера заказа и ссылок."),
             ("Канал доставки (Channel)", "Транспорт отправки: Email, SMS или Push."),
             ("Статус доставки (Delivery Status)", "Результат взаимодействия с внешним шлюзом связи (отправлено, доставлено, сбой).")
         ])
    ]

    for c_title, c_resp, terms in contexts:
        doc.add_heading(c_title, level=3)
        p_c = doc.add_paragraph()
        p_c.add_run("Отвечает за: ").bold = True
        p_c.add_run(c_resp)
        p_g = doc.add_paragraph()
        p_g.add_run("Глоссарий контекста:").italic = True
        for term, tdesc in terms:
            p_t = doc.add_paragraph(style='List Bullet')
            p_t.add_run(f"{term} — ").bold = True
            p_t.add_run(tdesc)

    # -------------------------------------------------------------------------
    # ШАГ 2
    # -------------------------------------------------------------------------
    doc.add_page_break()
    h2 = doc.add_heading("Шаг 2. Список и описание сервисов", level=1)
    h2.style.font.color.rgb = RGBColor(21, 101, 192)

    doc.add_paragraph(
        "В соответствии с требованиями методички спроектировано 5 бизнес-сервисов (API Gateway и Keycloak входят в инфраструктурный контур). "
        "Каждый сервис автономен, обладает собственной базой данных и реализует чётко очерченную бизнес-возможность."
    )

    cards = [
        ("catalog-service", [
            ("Ответственность", "Управление афишей цирка, расписанием сеансов, карточками шоу и базовыми тарифами по секторам зала."),
            ("Владеет данными", "Show (Шоу), Session (Сеанс), HallSector (Сектор зала), Genre (Жанр)."),
            ("Предоставляет", "REST API: GET /api/v1/shows, GET /api/v1/shows/{id}, GET /api/v1/sessions, GET /api/v1/sessions/{id}, GET /internal/v1/sessions/{id}, POST /internal/v1/shows, PUT /internal/v1/shows/{id}, DELETE /internal/v1/shows/{id}.\nСобытия: catalog.show.created, catalog.show.updated, catalog.show.archived."),
            ("Зависит от", "Полностью автономен; является первичным источником истины по репертуару цирка."),
            ("Нефункциональные требования", "Экстремально высокая нагрузка на чтение (в 50-100 раз чаще записи), задержка ответа p99 < 50 мс, обязательное кэширование в Redis."),
            ("Обоснование границы", "(1) Разные НФТ: чтение во много раз превышает запись — масштабируется независимо.\n(2) Чёткое владение данными афиши.\n(3) Автономность: доступен даже при сбое оплаты или бронирования.")
        ]),
        ("booking-service", [
            ("Ответственность", "Выбор и временное удержание мест на 15 минут, координация саги оформления заказа, выпуск электронных билетов с QR-кодами, обработка возвратов."),
            ("Владеет данными", "Booking (Бронь), SeatHold (Удержание места), Ticket (Билет), TicketRefund (Заявка на возврат)."),
            ("Предоставляет", "REST API: POST /api/v1/bookings (с Idempotency-Key), GET /api/v1/bookings/{id}, GET /api/v1/bookings/my, POST /api/v1/bookings/{id}/cancel, POST /api/v1/bookings/{id}/refund, GET /internal/v1/sessions/{id}/seats/status, POST /internal/v1/bookings/{id}/expire.\nСобытия: booking.created, booking.confirmed, booking.cancelled, booking.expired, refund.requested."),
            ("Зависит от", "Синхронно: catalog-service (сеанс и цена), loyalty-service (заморозка баллов), payment-service (создание платежа).\nАсинхронно: подписан на payment.completed, payment.failed, refund.completed."),
            ("Нефункциональные требования", "Высокая конкуренция за места у манежа в пиковые часы, строгая транзакционность блокировок, p99 < 150 мс, RPO=0."),
            ("Обоснование границы", "(1) Ядро бизнеса цирка — управление продажей и удержанием посадочных мест.\n(2) Сложная конкурентная логика распределённых блокировок мест в Redis на 15 минут.\n(3) Оркестрация распределённой транзакции (Саги) между билетами, баллами и оплатой.")
        ]),
        ("payment-service", [
            ("Ответственность", "Инициация платежей в банковском эквайринге, обработка банковских вебхуков, фискализация чеков, проведение возвратов средств."),
            ("Владеет данными", "Payment (Платёж), PaymentRefund (Возврат средств), Receipt (Фискальный чек)."),
            ("Предоставляет", "REST API: POST /internal/v1/payments (с Idempotency-Key), GET /internal/v1/payments/{id}, POST /internal/v1/payments/{id}/refund, POST /api/v1/payments/webhooks/acquiring.\nСобытия: payment.completed, payment.failed, refund.completed."),
            ("Зависит от", "Внешняя система: банковский эквайринг (REST API банка, Webhook).\nАсинхронно: подписан на refund.requested."),
            ("Нефункциональные требования", "Требования безопасности финансовых транзакций (PCI-DSS), идемпотентность платежей и возвратов, строгий аудит, задержка вебхука < 100 мс."),
            ("Обоснование границы", "(1) Изоляция финансовых рисков и банковских протоколов от бизнес-логики цирка.\n(2) Слабое зацепление: замена платёжного шлюза банка не требует изменений в booking-service.\n(3) Идемпотентность финансовых транзакций.")
        ]),
        ("loyalty-service", [
            ("Ответственность", "Начисление бонусных баллов (кэшбэк), заморозка и списание баллов в счёт скидки на билеты, расчёт рангов зрителей и настройка маркетинговых правил."),
            ("Владеет данными", "LoyaltyBalance (Баланс), LoyaltyTier (Ранг привилегий), PointsReservation (Заморозка баллов), MarketingRule (Маркетинговые правила)."),
            ("Предоставляет", "REST API: GET /api/v1/loyalty/me, GET /api/v1/loyalty/tiers, POST /internal/v1/loyalty/reservations, DELETE /internal/v1/loyalty/reservations/{id}, POST /internal/v1/loyalty/reservations/{id}/confirm, POST /internal/v1/loyalty/accrual, POST /internal/v1/loyalty/penalty, GET /api/v1/loyalty/rules, POST /api/v1/loyalty/rules.\nСобытия: loyalty.points.accrued, loyalty.points.redeemed, loyalty.tier.upgraded, loyalty.penalty.applied."),
            ("Зависит от", "Асинхронно: подписан на события booking.confirmed, refund.requested."),
            ("Нефункциональные требования", "Защита от double-spending (двойного списания баллов), строгая консистентность баланса клиента, гибкость конфигурации правил."),
            ("Обоснование границы", "(1) Высокая связность внутри: маркетинговые формулы, ранги и кэшбэк меняются независимо от покупки билетов.\n(2) Владение данными бонусов.\n(3) Отказоустойчивость: временный сбой программы лояльности не блокирует покупку билетов за рубли.")
        ]),
        ("notification-service", [
            ("Ответственность", "Формирование и отправка электронных билетов с QR-кодами, чеков оплаты, напоминаний о сеансах и сервисных оповещений на Email и SMS."),
            ("Владеет данными", "NotificationRecord (История отправок), NotificationTemplate (Шаблоны сообщений), NotificationPreferences (Настройки каналов)."),
            ("Предоставляет", "REST API: GET /api/v1/notifications/my, PUT /api/v1/notifications/preferences, POST /internal/v1/notifications/send, GET /internal/v1/notifications/templates, POST /internal/v1/notifications/marketing/broadcast.\nСобытия: notification.dispatched, notification.failed."),
            ("Зависит от", "Внешние системы: SMS-провайдер, Email SMTP/API.\nАсинхронно: подписан на booking.confirmed, booking.expired, refund.completed, loyalty.tier.upgraded."),
            ("Нефункциональные требования", "Фоновая асинхронная обработка, устойчивость к сбоям внешних провайдеров, повторные попытки (Retry + DLQ)."),
            ("Обоснование границы", "(1) Изоляция отказов: сбой почтового или SMS-шлюза не ломает покупку билета.\n(2) Асинхронность: клиент не ждёт отправку почты в рамках запроса создания брони.\n(3) Полиморфные структуры сообщений и логов.")
        ])
    ]

    for s_name, rows in cards:
        doc.add_heading(f"Карточка сервиса: {s_name}", level=2)
        tbl = doc.add_table(rows=len(rows)+1, cols=2)
        tbl.alignment = WD_TABLE_ALIGNMENT.CENTER
        tbl.style = 'Table Grid'
        
        # Header
        hdr = tbl.rows[0].cells
        hdr[0].text = "Поле"
        hdr[1].text = "Значение"
        hdr[0].paragraphs[0].runs[0].bold = True
        hdr[1].paragraphs[0].runs[0].bold = True
        set_cell_background(hdr[0], "E3F2FD")
        set_cell_background(hdr[1], "E3F2FD")
        set_cell_margins(hdr[0], 80, 80, 120, 120)
        set_cell_margins(hdr[1], 80, 80, 120, 120)

        for i, (k, v) in enumerate(rows, start=1):
            r_cells = tbl.rows[i].cells
            r_cells[0].text = k
            r_cells[1].text = v
            r_cells[0].paragraphs[0].runs[0].bold = True
            set_cell_background(r_cells[0], "F5F5F5")
            set_cell_margins(r_cells[0], 80, 80, 120, 120)
            set_cell_margins(r_cells[1], 80, 80, 120, 120)

        doc.add_paragraph()

    doc.add_heading("Обоснование отказа от альтернативных вариантов разбиения", level=2)
    p_alt = doc.add_paragraph()
    p_alt.add_run("1. Отказ от выделения отдельного user-service:\n").bold = True
    p_alt.add_run(
        "Рассматривался вариант создания сервиса пользователей для хранения профилей зрителей. "
        "Однако в соответствии с рекомендациями DDD и требованиями избежать антипаттерна «сервис на таблицу» (распределённая СУБД без бизнес-логики), "
        "от отдельного user-service отказались: аутентификацию, регистрацию, управление сессиями и базовые атрибуты пользователей (email, телефон, ФИО) "
        "полностью берёт на себя готовый промышленный IAM-компонент Keycloak. Данные о бонусах и рангах перенесены в loyalty-service, а история сообщений — в notification-service. "
        "Это исключило лишний сервис-сателлит и цепочки синхронных вызовов.\n\n"
    )
    p_alt.add_run("2. Отказ от включения синхронного recommendation-service в транзакционный контур:\n").bold = True
    p_alt.add_run(
        "Рассматривалось выделение recommendation-service (рекомендации представлений на основе просмотров и оценок). "
        "Было решено не включать его в ядро транзакционного контура на данном этапе: рекомендации строятся асинхронно на основе аналитики событий в Kafka "
        "и не должны влиять на критический путь покупки билетов. Сбор событий транслируется в Kafka, а рекомендательный движок вынесен в асинхронный аналитический контур.\n\n"
    )
    p_alt.add_run("3. Отказ от объединения booking-service и payment-service:\n").bold = True
    p_alt.add_run(
        "Объединение бронирования и оплаты привело бы к размытию ответственности и компрометации требований безопасности: payment-service интегрируется с банковским эквайрингом "
        "и требует сертификации PCI-DSS и строгого финансового аудита, тогда как booking-service сфокусирован на конкуренции за места и удержании в Redis."
    )

    # -------------------------------------------------------------------------
    # ШАГ 3
    # -------------------------------------------------------------------------
    doc.add_page_break()
    h3 = doc.add_heading("Шаг 3. Выбор готового ПО", level=1)
    h3.style.font.color.rgb = RGBColor(21, 101, 192)

    doc.add_paragraph(
        "Каждый компонент инфраструктурного стека выбран исходя из специфических требований конкретных сервисов системы."
    )

    infra_table = [
        ("PostgreSQL 16\n(отдельная БД на сервис)", "catalog, booking, payment, loyalty", 
         "Строгие ACID-транзакции, реляционные связи, строгая типизация схем, изоляция данных между сервисами (catalog_db, booking_db, payment_db, loyalty_db). Предотвращение double-spending бонусов и финансовых расхождений."),
        ("Redis 7\n(отдельные инстансы)", "catalog, booking", 
         "1. catalog-service: кэш горячих сеансов афиши (ускорение чтения в 50+ раз).\n2. booking-service: атомарные команды SETNX с TTL 15 минут для распределённого удержания мест у манежа. Автоматическое освобождение мест при падении или таймауте."),
        ("MongoDB 7", "notification", 
         "Документоориентированное хранилище для полиморфных шаблонов писем/SMS и журнала отправок с произвольной полезной нагрузкой. Данные читаются целиком как один документ."),
        ("Apache Kafka 3.7", "booking, payment, loyalty, notification", 
         "Распределённый лог событий с гарантией строгой очередности по ключу партиции (bookingId), независимыми группами консьюмеров, поддержкой повторного чтения (event replay) и высокой пропускной способностью."),
        ("Spring Cloud Gateway", "Вход для клиентов", 
         "Единая точка входа /api/v1/*, маршрутизация, rate limiting по пользователю/IP, CORS, валидация JWT токенов Keycloak, сокрытие внутренних эндпоинтов /internal/v1/*."),
        ("Keycloak 24", "API Gateway, клиенты", 
         "Промышленный IAM-провайдер: OAuth 2.0 / OpenID Connect, регистрация пользователей, управление ролями (VIEWER, ADMIN, MARKETER) и выпуск криптографически подписанных JWT токенов."),
        ("MinIO (S3)", "catalog, booking", 
         "Объектное хранилище медиафайлов: постеров представлений, схем манежа и сгенерированных PDF-билетов.")
    ]

    tbl_infra = doc.add_table(rows=len(infra_table)+1, cols=3)
    tbl_infra.alignment = WD_TABLE_ALIGNMENT.CENTER
    tbl_infra.style = 'Table Grid'

    hdr_i = tbl_infra.rows[0].cells
    hdr_i[0].text = "Компонент"
    hdr_i[1].text = "Кто использует"
    hdr_i[2].text = "Техническое обоснование"
    for c in hdr_i:
        c.paragraphs[0].runs[0].bold = True
        set_cell_background(c, "E3F2FD")
        set_cell_margins(c, 80, 80, 100, 100)

    for i, (comp, who, why) in enumerate(infra_table, start=1):
        r_c = tbl_infra.rows[i].cells
        r_c[0].text = comp
        r_c[1].text = who
        r_c[2].text = why
        r_c[0].paragraphs[0].runs[0].bold = True
        set_cell_margins(r_c[0], 80, 80, 100, 100)
        set_cell_margins(r_c[1], 80, 80, 100, 100)
        set_cell_margins(r_c[2], 80, 80, 100, 100)

    # -------------------------------------------------------------------------
    # ШАГ 4
    # -------------------------------------------------------------------------
    doc.add_page_break()
    h4 = doc.add_heading("Шаг 4. Схема взаимодействий и диаграммы последовательности", level=1)
    h4.style.font.color.rgb = RGBColor(21, 101, 192)

    doc.add_heading("4.1. Архитектурная схема системы (контейнерный уровень)", level=2)
    doc.add_paragraph("Сплошные стрелки обозначают синхронные вызовы (REST/HTTPS), пунктирные — асинхронные события в Apache Kafka.")

    if os.path.exists("docs/architecture.png"):
        doc.add_picture("docs/architecture.png", width=Inches(6.8))
        p_cap = doc.add_paragraph("Рис. 1. Контейнерная архитектурная схема микросервисной системы цирка")
        p_cap.alignment = WD_ALIGN_PARAGRAPH.CENTER
        p_cap.runs[0].italic = True

    doc.add_heading("4.2. Таблица межсервисных взаимодействий", level=2)
    interactions = [
        ("1", "Клиент → Gateway → catalog", "sync", "GET /api/v1/shows, GET /api/v1/sessions", "Просмотр афиши цирка и выбор сеанса"),
        ("2", "Клиент → Gateway → booking", "sync", "POST /api/v1/bookings (Idempotency-Key)", "Создание брони, удержание мест на 15 мин, расчет скидки"),
        ("3", "booking → catalog", "sync", "GET /internal/v1/sessions/{id}", "Проверка доступности сеанса и базовой стоимости места"),
        ("4", "booking → loyalty", "sync", "POST /internal/v1/loyalty/reservations", "Заморозка баллов лояльности под оформляемый заказ"),
        ("5", "booking → payment", "sync", "POST /internal/v1/payments (Idempotency-Key)", "Создание платежа на сумму за вычетом баллов, получение ссылки на оплату"),
        ("6", "Банк → payment", "sync", "POST /api/v1/payments/webhooks/acquiring", "Вебхук эквайринга о результате оплаты картой (HMAC подпись)"),
        ("7", "payment → Kafka", "async", "payment.completed / payment.failed", "Публикация статуса платежа в топик payments (ключ: bookingId)"),
        ("8", "Kafka → booking", "async", "payment.completed / payment.failed", "Подтверждение заказа (выпуск билетов) либо отмена брони"),
        ("9", "booking → Kafka", "async", "booking.confirmed / booking.expired", "Публикация финального статуса бронирования (топик bookings)"),
        ("10", "Kafka → loyalty", "async", "booking.confirmed", "Окончательное списание баллов и начисление кэшбэка"),
        ("11", "Kafka → notification", "async", "booking.confirmed / booking.expired", "Генерация билетов с QR и отправка зрителю на Email/SMS"),
        ("12", "loyalty → Kafka", "async", "loyalty.tier.upgraded", "Публикация события повышения ранга зрителя"),
        ("13", "Kafka → notification", "async", "loyalty.tier.upgraded", "Поздравительное сообщение зрителю с новым рангом"),
        ("14", "Клиент → Gateway → booking", "sync", "POST /api/v1/bookings/{id}/refund", "Запрос на возврат билета зрителем (возврат до 50% суммы)"),
        ("15", "booking → Kafka", "async", "refund.requested", "Инициация саги возврата средств и штрафа лояльности"),
        ("16", "Kafka → payment", "async", "refund.requested", "Вызов эквайринга на перечисление 50% суммы на карту"),
        ("17", "payment → Kafka", "async", "refund.completed", "Подтверждение успешного финансового возврата"),
        ("18", "Kafka → notification", "async", "refund.completed", "Отправка зрителю чека возврата денежных средств")
    ]

    tbl_inter = doc.add_table(rows=len(interactions)+1, cols=5)
    tbl_inter.alignment = WD_TABLE_ALIGNMENT.CENTER
    tbl_inter.style = 'Table Grid'
    hdr_in = tbl_inter.rows[0].cells
    hdr_in[0].text = "№"
    hdr_in[1].text = "Инициатор → Получатель"
    hdr_in[2].text = "Тип"
    hdr_in[3].text = "Эндпоинт или Событие"
    hdr_in[4].text = "Назначение"
    for c in hdr_in:
        c.paragraphs[0].runs[0].bold = True
        set_cell_background(c, "E3F2FD")
        set_cell_margins(c, 60, 60, 80, 80)

    for i, (num, sender_rec, t, ep, purp) in enumerate(interactions, start=1):
        r_c = tbl_inter.rows[i].cells
        r_c[0].text = num
        r_c[1].text = sender_rec
        r_c[2].text = t
        r_c[3].text = ep
        r_c[4].text = purp
        set_cell_margins(r_c[0], 60, 60, 80, 80)
        set_cell_margins(r_c[1], 60, 60, 80, 80)
        set_cell_margins(r_c[2], 60, 60, 80, 80)
        set_cell_margins(r_c[3], 60, 60, 80, 80)
        set_cell_margins(r_c[4], 60, 60, 80, 80)

    doc.add_page_break()
    doc.add_heading("4.3. Диаграмма последовательности: Успешный сценарий (Happy Path)", level=2)
    doc.add_paragraph("Сквозной бизнес-сценарий покупки билета с частичной оплатой бонусами лояльности и оплатой остатка банковской картой:")

    if os.path.exists("docs/sequence-success.png"):
        doc.add_picture("docs/sequence-success.png", width=Inches(6.8))
        p_cap = doc.add_paragraph("Рис. 2. Диаграмма последовательности успешной покупки билета")
        p_cap.alignment = WD_ALIGN_PARAGRAPH.CENTER
        p_cap.runs[0].italic = True

    doc.add_heading("4.4. Диаграмма последовательности: Сбои и компенсирующие действия (Сага)", level=2)
    doc.add_paragraph("Сценарии обработки сбоев: отклонение оплаты банком, таймаут 15 минут, запоздалая оплата:")

    if os.path.exists("docs/sequence-failure.png"):
        doc.add_picture("docs/sequence-failure.png", width=Inches(6.8))
        p_cap = doc.add_paragraph("Рис. 3. Диаграмма последовательности сценариев сбоя и компенсирующих действий")
        p_cap.alignment = WD_ALIGN_PARAGRAPH.CENTER
        p_cap.runs[0].italic = True

    # -------------------------------------------------------------------------
    # ШАГ 5
    # -------------------------------------------------------------------------
    doc.add_page_break()
    h5 = doc.add_heading("Шаг 5. Контракты API и события брокера", level=1)
    h5.style.font.color.rgb = RGBColor(21, 101, 192)

    doc.add_paragraph(
        "Все контракты API спроектированы в стандарте OpenAPI 3.1.0 с единой моделью ошибок RFC 7807 (ProblemDetails), "
        "заголовками идемпотентности Idempotency-Key и пагинацией списков. Спецификации сохранены в каталоге api/ репозитория:\n"
        "1. api/catalog-service.yaml — каталог цирковых представлений, сеансы, цены секторов;\n"
        "2. api/booking-service.yaml — создание броней, удержание мест на 15 мин, выпуск билетов с QR, возвраты;\n"
        "3. api/payment-service.yaml — платежи через эквайринг, чеки, вебхуки, возвраты денежных средств;\n"
        "4. api/loyalty-service.yaml — начисление/списание бонусов, уровни зрителей, правила маркетолога;\n"
        "5. api/notification-service.yaml — доставка электронных билетов с QR, email/SMS шаблоны и настройки."
    )

    doc.add_heading("Перечень событий в брокере сообщений Apache Kafka", level=2)
    events_kafka = [
        ("circus.payments.events", "payment.completed", "payment-service", "booking-service, loyalty-service", "bookingId"),
        ("circus.payments.events", "payment.failed", "payment-service", "booking-service", "bookingId"),
        ("circus.bookings.events", "booking.confirmed", "booking-service", "loyalty-service, notification-service", "bookingId"),
        ("circus.bookings.events", "booking.expired", "booking-service", "loyalty-service, notification-service", "bookingId"),
        ("circus.bookings.events", "booking.cancelled", "booking-service", "loyalty-service, notification-service", "bookingId"),
        ("circus.refunds.events", "refund.requested", "booking-service", "payment-service, loyalty-service", "bookingId"),
        ("circus.refunds.events", "refund.completed", "payment-service", "booking-service, notification-service", "bookingId"),
        ("circus.loyalty.events", "loyalty.points.accrued", "loyalty-service", "notification-service", "userId"),
        ("circus.loyalty.events", "loyalty.tier.upgraded", "loyalty-service", "notification-service", "userId")
    ]

    tbl_k = doc.add_table(rows=len(events_kafka)+1, cols=5)
    tbl_k.alignment = WD_TABLE_ALIGNMENT.CENTER
    tbl_k.style = 'Table Grid'
    hdr_k = tbl_k.rows[0].cells
    hdr_k[0].text = "Топик Kafka"
    hdr_k[1].text = "Событие (eventType)"
    hdr_k[2].text = "Издатель"
    hdr_k[3].text = "Подписчики"
    hdr_k[4].text = "Ключ"
    for c in hdr_k:
        c.paragraphs[0].runs[0].bold = True
        set_cell_background(c, "E3F2FD")
        set_cell_margins(c, 60, 60, 80, 80)

    for i, (top, ev, pub, sub, k) in enumerate(events_kafka, start=1):
        r_c = tbl_k.rows[i].cells
        r_c[0].text = top
        r_c[1].text = ev
        r_c[2].text = pub
        r_c[3].text = sub
        r_c[4].text = k
        set_cell_margins(r_c[0], 60, 60, 80, 80)
        set_cell_margins(r_c[1], 60, 60, 80, 80)
        set_cell_margins(r_c[2], 60, 60, 80, 80)
        set_cell_margins(r_c[3], 60, 60, 80, 80)
        set_cell_margins(r_c[4], 60, 60, 80, 80)

    # Save to original user file and a clean name file
    doc_path1 = "Информационная система цирка предназначена для зрителей.docx"
    doc_path2 = "Отчет_Лабораторная_3_Микросервисы_Цирк.docx"
    doc.save(doc_path1)
    doc.save(doc_path2)
    print(f"Saved complete Word documents to '{doc_path1}' and '{doc_path2}'!")

if __name__ == '__main__':
    create_report()
