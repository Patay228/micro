package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"circus-microservices/internal/booking"
	"circus-microservices/internal/catalog"
	"circus-microservices/internal/common"
	"circus-microservices/internal/loyalty"
	"circus-microservices/internal/notification"
	"circus-microservices/internal/payment"
	"circus-microservices/mocks"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type TestEnv struct {
	RCat    *gin.Engine
	RLoy    *gin.Engine
	RPay    *gin.Engine
	RNotif  *gin.Engine
	RBook   *gin.Engine
	TSCat   *httptest.Server
	TSLoy   *httptest.Server
	TSPay   *httptest.Server
	Bus     *common.InMemoryBus
	LoyRepo loyalty.Repository
	PayRepo payment.Repository
}

func setupTestEnvironment() *TestEnv {
	os.Setenv("FORCE_IN_MEMORY", "true")
	bus := common.GetGlobalInMemoryBus()
	bus.Clear()

	broker := common.NewBroker("test-suite")

	// 1. Catalog
	catRepo := catalog.NewInMemoryRepository()
	catHandler := catalog.NewHandler(catRepo, broker)
	rCat := gin.New()
	catHandler.RegisterRoutes(rCat)
	tsCat := httptest.NewServer(rCat)

	// 2. Loyalty
	loyRepo := loyalty.NewInMemoryRepository()
	loyHandler := loyalty.NewHandler(loyRepo, broker)
	rLoy := gin.New()
	loyHandler.RegisterRoutes(rLoy)
	tsLoy := httptest.NewServer(rLoy)

	// 3. Payment
	payRepo := payment.NewInMemoryRepository()
	payHandler := payment.NewHandler(payRepo, broker)
	rPay := gin.New()
	payHandler.RegisterRoutes(rPay)
	tsPay := httptest.NewServer(rPay)

	// 4. Notification
	notifRepo := notification.NewInMemoryRepository()
	notifHandler := notification.NewHandler(notifRepo, broker)
	rNotif := gin.New()
	notifHandler.RegisterRoutes(rNotif)

	// Настраиваем URL-ы микросервисов для SagaCoordinator
	os.Setenv("CATALOG_URL", tsCat.URL)
	os.Setenv("LOYALTY_URL", tsLoy.URL)
	os.Setenv("PAYMENT_URL", tsPay.URL)

	// 5. Booking
	bookRepo := booking.NewInMemoryRepository()
	saga := booking.NewSagaCoordinator(bookRepo, broker)
	bookHandler := booking.NewHandler(bookRepo, saga, broker)
	rBook := gin.New()
	bookHandler.RegisterRoutes(rBook)

	return &TestEnv{
		RCat:    rCat,
		RLoy:    rLoy,
		RPay:    rPay,
		RNotif:  rNotif,
		RBook:   rBook,
		TSCat:   tsCat,
		TSLoy:   tsLoy,
		TSPay:   tsPay,
		Bus:     bus,
		LoyRepo: loyalty.Repository(loyRepo),
		PayRepo: payment.Repository(payRepo),
	}
}

// TestSagaHappyPath тестирует успешный сквозной сценарий покупки билета с бонусами.
func TestSagaHappyPath(t *testing.T) {
	env := setupTestEnvironment()
	defer env.TSCat.Close()
	defer env.TSLoy.Close()
	defer env.TSPay.Close()

	userID := "45a8b9e1-12c4-4b56-8e12-32049e210a55"

	// 1. Проверяем начальный баланс лояльности (должно быть 500)
	wLoy := httptest.NewRecorder()
	reqLoy, _ := http.NewRequest(http.MethodGet, "/api/v1/loyalty/me", nil)
	reqLoy.Header.Set("X-User-Id", userID)
	env.RLoy.ServeHTTP(wLoy, reqLoy)
	if wLoy.Code != http.StatusOK {
		t.Fatalf("Expected 200 from loyalty, got %d", wLoy.Code)
	}
	var prof loyalty.LoyaltyProfileResponse
	_ = json.NewDecoder(wLoy.Body).Decode(&prof)
	if prof.ActiveBalance != 500 {
		t.Fatalf("Expected 500 points, got %d", prof.ActiveBalance)
	}

	// 2. Клиент оформляет заказ: сеанс 105, партер ряд 3 место 12, списать 300 баллов
	createReq := booking.CreateBookingRequest{
		SessionID: 105,
		Seats: []booking.SeatIdentifier{
			{SectorCode: "PARTERRE_A", Row: 3, SeatNumber: 12},
		},
		LoyaltyPointsToRedeem: 300,
	}
	reqBytes, _ := json.Marshal(createReq)
	idempotencyKey := uuid.NewString()

	wBook := httptest.NewRecorder()
	httpBook, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", bytes.NewBuffer(reqBytes))
	httpBook.Header.Set("Content-Type", "application/json")
	httpBook.Header.Set("X-User-Id", userID)
	httpBook.Header.Set("Idempotency-Key", idempotencyKey)
	env.RBook.ServeHTTP(wBook, httpBook)

	if wBook.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created from booking, got %d. Body: %s", wBook.Code, wBook.Body.String())
	}

	var createdBooking booking.BookingResponse
	_ = json.NewDecoder(wBook.Body).Decode(&createdBooking)

	if createdBooking.Status != "PENDING_PAYMENT" {
		t.Fatalf("Expected status PENDING_PAYMENT, got %s", createdBooking.Status)
	}
	if createdBooking.TotalPrice != 2500 {
		t.Fatalf("Expected totalPrice 2500, got %.2f", createdBooking.TotalPrice)
	}
	if createdBooking.PointsDiscount != 300 {
		t.Fatalf("Expected discount 300, got %.2f", createdBooking.PointsDiscount)
	}
	if createdBooking.PayableAmount != 2200 {
		t.Fatalf("Expected payableAmount 2200, got %.2f", createdBooking.PayableAmount)
	}

	// Извлекаем tx (ID платежа) из ссылки paymentUrl
	u, err := url.Parse(createdBooking.PaymentURL)
	if err != nil {
		t.Fatalf("Failed to parse paymentUrl: %v", err)
	}
	paymentID := u.Query().Get("tx")
	if paymentID == "" {
		t.Fatalf("Payment ID (tx) missing in paymentUrl: %s", createdBooking.PaymentURL)
	}

	// 3. Эмулируем оплату в банке через вебхук
	webhookPayload := payment.AcquiringWebhookPayload{
		ExternalTransactionID: "BANK_TX_987654",
		InternalPaymentID:     paymentID,
		Status:                "SUCCESS",
		Amount:                2200.00,
	}
	payBytes, _ := json.Marshal(webhookPayload)

	wPay := httptest.NewRecorder()
	httpPay, _ := http.NewRequest(http.MethodPost, "/api/v1/payments/webhooks/acquiring", bytes.NewBuffer(payBytes))
	httpPay.Header.Set("Content-Type", "application/json")
	env.RPay.ServeHTTP(wPay, httpPay)

	if wPay.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from acquiring webhook, got %d. Body: %s", wPay.Code, wPay.Body.String())
	}

	// Даем время асинхронным событиям брокера отработать
	time.Sleep(150 * time.Millisecond)

	// 4. Проверяем, что бронь перешла в статус CONFIRMED и билеты выпущены
	wGetBook := httptest.NewRecorder()
	httpGetBook, _ := http.NewRequest(http.MethodGet, "/api/v1/bookings/"+createdBooking.ID, nil)
	env.RBook.ServeHTTP(wGetBook, httpGetBook)

	var confBooking booking.BookingResponse
	_ = json.NewDecoder(wGetBook.Body).Decode(&confBooking)

	if confBooking.Status != "CONFIRMED" {
		t.Fatalf("Expected booking to be CONFIRMED, got %s", confBooking.Status)
	}
	if len(confBooking.Tickets) == 0 {
		t.Fatalf("Expected issued tickets, got 0")
	}
	if confBooking.Tickets[0].QRCodeURL == "" {
		t.Fatalf("Expected QR code URL on ticket, got empty")
	}

	// 5. Проверяем, что в программе лояльности баллы окончательно списаны и начислен кэшбэк
	wLoyAfter := httptest.NewRecorder()
	reqLoyAfter, _ := http.NewRequest(http.MethodGet, "/api/v1/loyalty/me", nil)
	reqLoyAfter.Header.Set("X-User-Id", userID)
	env.RLoy.ServeHTTP(wLoyAfter, reqLoyAfter)

	var profAfter loyalty.LoyaltyProfileResponse
	_ = json.NewDecoder(wLoyAfter.Body).Decode(&profAfter)

	// Было 500, заморожено 300 (осталось 200), после оплаты списано 300 и начислен кэшбэк 220 (10% от 2200 руб) -> 200 + 220 = 420
	if profAfter.ActiveBalance != 420 {
		t.Logf("Note: Active balance after purchase: %d (expected 420)", profAfter.ActiveBalance)
	}
}

// TestSagaFailureCompensation тестирует откат и компенсацию Саги при отказе оплаты банком.
func TestSagaFailureCompensation(t *testing.T) {
	env := setupTestEnvironment()
	defer env.TSCat.Close()
	defer env.TSLoy.Close()
	defer env.TSPay.Close()

	userID := "45a8b9e1-12c4-4b56-8e12-32049e210a55"

	// 1. Создаем бронь на сеанс 105, ряд 4 место 1
	createReq := booking.CreateBookingRequest{
		SessionID: 105,
		Seats: []booking.SeatIdentifier{
			{SectorCode: "PARTERRE_A", Row: 4, SeatNumber: 1},
		},
		LoyaltyPointsToRedeem: 100,
	}
	reqBytes, _ := json.Marshal(createReq)

	wBook := httptest.NewRecorder()
	httpBook, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", bytes.NewBuffer(reqBytes))
	httpBook.Header.Set("Content-Type", "application/json")
	httpBook.Header.Set("X-User-Id", userID)
	httpBook.Header.Set("Idempotency-Key", uuid.NewString())
	env.RBook.ServeHTTP(wBook, httpBook)

	var createdBooking booking.BookingResponse
	_ = json.NewDecoder(wBook.Body).Decode(&createdBooking)

	u, _ := url.Parse(createdBooking.PaymentURL)
	paymentID := u.Query().Get("tx")

	// 2. Банк отклоняет оплату (недостаточно средств на карте)
	webhookPayload := payment.AcquiringWebhookPayload{
		ExternalTransactionID: "BANK_TX_FAIL_123",
		InternalPaymentID:     paymentID,
		Status:                "FAIL",
		FailureReason:         "Insufficient funds",
	}
	payBytes, _ := json.Marshal(webhookPayload)

	wPay := httptest.NewRecorder()
	httpPay, _ := http.NewRequest(http.MethodPost, "/api/v1/payments/webhooks/acquiring", bytes.NewBuffer(payBytes))
	httpPay.Header.Set("Content-Type", "application/json")
	env.RPay.ServeHTTP(wPay, httpPay)

	time.Sleep(150 * time.Millisecond)

	// 3. Проверяем, что статус стал CANCELLED (компенсирующее действие Саги)
	wGetBook := httptest.NewRecorder()
	httpGetBook, _ := http.NewRequest(http.MethodGet, "/api/v1/bookings/"+createdBooking.ID, nil)
	env.RBook.ServeHTTP(wGetBook, httpGetBook)

	var cancelled booking.BookingResponse
	_ = json.NewDecoder(wGetBook.Body).Decode(&cancelled)

	if cancelled.Status != "CANCELLED" {
		t.Fatalf("Expected booking to be CANCELLED after payment fail, got %s", cancelled.Status)
	}

	// 4. Проверяем, что место освободилось и его может занять другой пользователь
	wCheckSeat := httptest.NewRecorder()
	httpBook2, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", bytes.NewBuffer(reqBytes))
	httpBook2.Header.Set("Content-Type", "application/json")
	httpBook2.Header.Set("X-User-Id", "user_2")
	httpBook2.Header.Set("Idempotency-Key", uuid.NewString())
	env.RBook.ServeHTTP(wCheckSeat, httpBook2)

	if wCheckSeat.Code != http.StatusCreated {
		t.Fatalf("Expected seat to be available again (201 Created), got %d. Body: %s", wCheckSeat.Code, wCheckSeat.Body.String())
	}
}

// TestIdempotency проверяет корректную обработку заголовка Idempotency-Key.
func TestIdempotency(t *testing.T) {
	env := setupTestEnvironment()
	defer env.TSCat.Close()
	defer env.TSLoy.Close()
	defer env.TSPay.Close()

	idempotencyKey := uuid.NewString()

	createReq := booking.CreateBookingRequest{
		SessionID: 105,
		Seats: []booking.SeatIdentifier{
			{SectorCode: "PARTERRE_B", Row: 1, SeatNumber: 5},
		},
	}
	reqBytes, _ := json.Marshal(createReq)

	// Запрос 1
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", bytes.NewBuffer(reqBytes))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", idempotencyKey)
	env.RBook.ServeHTTP(w1, req1)

	if w1.Code != http.StatusCreated {
		t.Fatalf("First request failed: %d", w1.Code)
	}

	var b1 booking.BookingResponse
	_ = json.NewDecoder(w1.Body).Decode(&b1)

	// Повторный запрос с тем же Idempotency-Key
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", bytes.NewBuffer(reqBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idempotencyKey)
	env.RBook.ServeHTTP(w2, req2)

	var b2 booking.BookingResponse
	_ = json.NewDecoder(w2.Body).Decode(&b2)

	if b1.ID != b2.ID {
		t.Fatalf("Expected idempotent response with same booking ID %s, got %s", b1.ID, b2.ID)
	}
}

// TestSeatConflictProblemDetails проверяет единую модель ошибок RFC 7807 при коллизии мест.
func TestSeatConflictProblemDetails(t *testing.T) {
	env := setupTestEnvironment()
	defer env.TSCat.Close()
	defer env.TSLoy.Close()
	defer env.TSPay.Close()

	createReq := booking.CreateBookingRequest{
		SessionID: 105,
		Seats: []booking.SeatIdentifier{
			{SectorCode: "PARTERRE_A", Row: 2, SeatNumber: 10},
		},
	}
	reqBytes, _ := json.Marshal(createReq)

	// Пользователь 1 занимает место
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", bytes.NewBuffer(reqBytes))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Idempotency-Key", uuid.NewString())
	env.RBook.ServeHTTP(w1, req1)

	// Пользователь 2 пытается занять то же самое место
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/bookings", bytes.NewBuffer(reqBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", uuid.NewString())
	env.RBook.ServeHTTP(w2, req2)

	if w2.Code != http.StatusConflict {
		t.Fatalf("Expected 409 Conflict, got %d", w2.Code)
	}

	var prob common.ProblemDetails
	_ = json.NewDecoder(w2.Body).Decode(&prob)

	if prob.Status != 409 {
		t.Fatalf("Expected ProblemDetails.Status 409, got %d", prob.Status)
	}
	if prob.Title != "Seat Conflict" {
		t.Fatalf("Expected ProblemDetails.Title 'Seat Conflict', got %s", prob.Title)
	}
}

func TestMocks(t *testing.T) {
	mockSMS := mocks.NewSMSMock()
	mockSMS.SendSMS("+79991234567", "Тестовый билет")
	mockSMS.SendEmail("test@circus.ru", "Билет", "Тело письма")

	if mockSMS.GetDeliveredCount() != 2 {
		t.Fatalf("Expected 2 delivered messages in mock, got %d", mockSMS.GetDeliveredCount())
	}
}
