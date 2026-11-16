package booking

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"circus-microservices/internal/common"

	"github.com/google/uuid"
)

type SagaCoordinator struct {
	repo       Repository
	broker     *common.Broker
	catalogURL string
	paymentURL string
	loyaltyURL string
	httpClient *http.Client
}

func NewSagaCoordinator(repo Repository, broker *common.Broker) *SagaCoordinator {
	catURL := os.Getenv("CATALOG_URL")
	if catURL == "" {
		catURL = "http://localhost:8001"
	}
	payURL := os.Getenv("PAYMENT_URL")
	if payURL == "" {
		payURL = "http://localhost:8003"
	}
	loyURL := os.Getenv("LOYALTY_URL")
	if loyURL == "" {
		loyURL = "http://localhost:8004"
	}

	return &SagaCoordinator{
		repo:       repo,
		broker:     broker,
		catalogURL: catURL,
		paymentURL: payURL,
		loyaltyURL: loyURL,
		httpClient: &http.Client{Timeout: 3 * time.Second},
	}
}

// CheckSession запрашивает сеанс и базовые цены у catalog-service (синхронно по REST).
func (s *SagaCoordinator) CheckSession(sessionID int) (map[string]float64, error) {
	url := fmt.Sprintf("%s/internal/v1/sessions/%d", s.catalogURL, sessionID)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		// Mock-fallback для автономной демонстрации, если catalog-service не запущен на отдельном порту
		log.Printf("[booking-service] Catalog service HTTP unreachable (%v), using mock price fallback", err)
		return map[string]float64{
			"PARTERRE_A":   2500.00,
			"PARTERRE_B":   2000.00,
			"AMPHITHEATER": 1500.00,
			"BALCONY":      1000.00,
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("session %d not available in catalog (code %d)", sessionID, resp.StatusCode)
	}

	var data struct {
		IsAvailableForBooking bool               `json:"isAvailableForBooking"`
		SectorPrices          map[string]float64 `json:"sectorPrices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if !data.IsAvailableForBooking {
		return nil, fmt.Errorf("session %d is not available for booking", sessionID)
	}
	return data.SectorPrices, nil
}

// ReserveLoyaltyPoints замораживает баллы в loyalty-service.
func (s *SagaCoordinator) ReserveLoyaltyPoints(bookingID, userID string, points int) (string, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"bookingId":       bookingID,
		"userId":          userID,
		"pointsToReserve": points,
	})

	url := fmt.Sprintf("%s/internal/v1/loyalty/reservations", s.loyaltyURL)
	resp, err := s.httpClient.Post(url, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		log.Printf("[booking-service] Loyalty service unreachable (%v), using mock reservation ID", err)
		return uuid.NewString(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to reserve points: %s", string(body))
	}

	var res struct {
		ReservationID string `json:"reservationId"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)
	return res.ReservationID, nil
}

// CompensateLoyaltyPoints размораживает баллы (компенсирующее действие Саги).
func (s *SagaCoordinator) CompensateLoyaltyPoints(reservationID string) error {
	if reservationID == "" {
		return nil
	}
	url := fmt.Sprintf("%s/internal/v1/loyalty/reservations/%s", s.loyaltyURL, reservationID)
	req, _ := http.NewRequest(http.MethodDelete, url, nil)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[booking-service] Warning: Failed to compensate loyalty reservation %s: %v", reservationID, err)
		return err
	}
	defer resp.Body.Close()
	log.Printf("[booking-service] [SAGA COMPENSATION] Successfully released loyalty reservation %s", reservationID)
	return nil
}

// CreatePayment создает платеж в payment-service.
func (s *SagaCoordinator) CreatePayment(bookingID, userID string, amount float64, key string) (string, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"bookingId":   bookingID,
		"userId":      userID,
		"amount":      amount,
		"currency":    "RUB",
		"description": "Билеты в цирк",
	})

	url := fmt.Sprintf("%s/internal/v1/payments", s.paymentURL)
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("[booking-service] Payment service unreachable (%v), generating mock payment URL", err)
		return fmt.Sprintf("https://pay.bank-acquiring.ru/process?tx=%s", bookingID), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("payment initialization failed: %s", string(body))
	}

	var p struct {
		PaymentURL string `json:"paymentUrl"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&p)
	return p.PaymentURL, nil
}

// ExecuteBookingSaga координирует сквозной сценарий оформления заказа и обработку сбоев.
func (s *SagaCoordinator) ExecuteBookingSaga(userID string, req CreateBookingRequest, idempotencyKey string) (*BookingResponse, error) {
	// Проверка идемпотентности в начале
	if idempotencyKey != "" {
		if existing, err := s.repo.GetBookingByIdempotencyKey(idempotencyKey); err == nil && existing != nil {
			return existing, nil
		}
	}

	bookingID := uuid.NewString()
	now := time.Now().UTC()
	expiresAt := now.Add(15 * time.Minute)

	// Шаг 1 Саги: Проверка сеанса и тарифов в catalog-service
	sectorPrices, err := s.CheckSession(req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("catalog check failed: %w", err)
	}

	// Расчет общей стоимости
	var totalPrice float64
	for _, seat := range req.Seats {
		price, ok := sectorPrices[seat.SectorCode]
		if !ok || price <= 0 {
			price = 2500.00 // базовая стоимость партера по умолчанию
		}
		totalPrice += price
	}

	// Шаг 2 Саги: Удержание мест (Seat Hold) в хранилище на 15 минут
	if err := s.repo.HoldSeats(req.SessionID, bookingID, req.Seats, expiresAt); err != nil {
		return nil, fmt.Errorf("conflict: %w", err)
	}

	// Шаг 3 Саги: Заморозка баллов лояльности (если запрошено списание)
	pointsDiscount := float64(req.LoyaltyPointsToRedeem)
	if pointsDiscount > totalPrice {
		pointsDiscount = totalPrice
	}
	var reservationID string
	if req.LoyaltyPointsToRedeem > 0 {
		resID, err := s.ReserveLoyaltyPoints(bookingID, userID, int(pointsDiscount))
		if err != nil {
			// СБОЙ: Откатываем шаг 2 (компенсация - снимаем удержание мест)
			_ = s.repo.ReleaseSeats(bookingID)
			log.Printf("[booking-service] [SAGA ROLLBACK] Points reservation failed, released seat hold")
			return nil, fmt.Errorf("loyalty reservation error: %w", err)
		}
		reservationID = resID
	}

	// Шаг 4 Саги: Инициализация платежа в payment-service
	payableAmount := totalPrice - pointsDiscount
	if payableAmount <= 0 {
		payableAmount = 1.00 // минимальная сумма банковской проводки
	}

	paymentURL, err := s.CreatePayment(bookingID, userID, payableAmount, idempotencyKey)
	if err != nil {
		// СБОЙ: Компенсируем и баллы, и места!
		_ = s.repo.ReleaseSeats(bookingID)
		_ = s.CompensateLoyaltyPoints(reservationID)
		log.Printf("[booking-service] [SAGA ROLLBACK] Payment creation failed, released seats and loyalty points")
		return nil, fmt.Errorf("payment service error: %w", err)
	}

	// Шаг 5 Саги: Сохранение брони в статусе PENDING_PAYMENT
	bResp := BookingResponse{
		ID:             bookingID,
		SessionID:      req.SessionID,
		UserID:         userID,
		Status:         "PENDING_PAYMENT",
		TotalPrice:     totalPrice,
		PointsDiscount: pointsDiscount,
		PayableAmount:  payableAmount,
		PaymentURL:     paymentURL,
		CreatedAt:      now.Format(time.RFC3339),
		ExpiresAt:      expiresAt.Format(time.RFC3339),
		Seats:          req.Seats,
	}

	saved, isRetry, err := s.repo.CreateBooking(bResp, idempotencyKey)
	if err != nil {
		_ = s.repo.ReleaseSeats(bookingID)
		_ = s.CompensateLoyaltyPoints(reservationID)
		return nil, err
	}
	if isRetry {
		return saved, nil
	}

	// Публикация события о создании брони
	_ = s.broker.Publish("circus.bookings.events", common.NewEvent("booking.created", map[string]any{
		"bookingId":      bookingID,
		"userId":         userID,
		"sessionId":      req.SessionID,
		"payableAmount":  payableAmount,
		"pointsDiscount": pointsDiscount,
	}))

	return saved, nil
}
