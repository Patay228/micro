package payment

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	CreatePayment(req CreatePaymentRequest, idempotencyKey string) (*Payment, bool, error)
	GetPaymentByID(id string) (*Payment, error)
	GetPaymentByBookingID(bookingID string) (*Payment, error)
	UpdatePaymentStatus(id string, status, receiptURL string) (*Payment, error)
	CreateRefund(paymentID string, amount float64, reason string) (*RefundResultResponse, error)
}

// PostgresRepository использует чистые SQL-запросы к PostgreSQL без ORM.
type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreatePayment(req CreatePaymentRequest, idempotencyKey string) (*Payment, bool, error) {
	// Проверка идемпотентности
	if idempotencyKey != "" {
		var existing Payment
		err := r.db.QueryRow(`SELECT id, booking_id, user_id, amount, currency, status, payment_url, receipt_url, idempotency_key, created_at, updated_at 
		                      FROM payments WHERE idempotency_key = $1`, idempotencyKey).
			Scan(&existing.ID, &existing.BookingID, &existing.UserID, &existing.Amount, &existing.Currency,
				&existing.Status, &existing.PaymentURL, &existing.ReceiptURL, &existing.IdempotencyKey,
				&existing.CreatedAt, &existing.UpdatedAt)
		if err == nil {
			return &existing, true, nil // true = найден по idempotencyKey
		}
	}

	id := uuid.NewString()
	paymentURL := fmt.Sprintf("https://pay.bank-acquiring.ru/process?tx=%s", id)
	currency := req.Currency
	if currency == "" {
		currency = "RUB"
	}
	now := time.Now().UTC().Format(time.RFC3339)

	query := `INSERT INTO payments (id, booking_id, user_id, amount, currency, status, payment_url, idempotency_key, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, 'PENDING', $6, $7, $8, $9)`
	_, err := r.db.Exec(query, id, req.BookingID, req.UserID, req.Amount, currency, paymentURL, idempotencyKey, now, now)
	if err != nil {
		return nil, false, err
	}

	p := &Payment{
		ID:             id,
		BookingID:      req.BookingID,
		UserID:         req.UserID,
		Amount:         req.Amount,
		Currency:       currency,
		Status:         "PENDING",
		PaymentURL:     paymentURL,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	return p, false, nil
}

func (r *PostgresRepository) GetPaymentByID(id string) (*Payment, error) {
	var p Payment
	var receipt sql.NullString
	query := `SELECT id, booking_id, user_id, amount, currency, status, payment_url, receipt_url, idempotency_key, created_at, updated_at 
	          FROM payments WHERE id = $1`
	err := r.db.QueryRow(query, id).
		Scan(&p.ID, &p.BookingID, &p.UserID, &p.Amount, &p.Currency, &p.Status, &p.PaymentURL, &receipt, &p.IdempotencyKey, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if receipt.Valid {
		p.ReceiptURL = receipt.String
	}
	return &p, nil
}

func (r *PostgresRepository) GetPaymentByBookingID(bookingID string) (*Payment, error) {
	var p Payment
	var receipt sql.NullString
	query := `SELECT id, booking_id, user_id, amount, currency, status, payment_url, receipt_url, idempotency_key, created_at, updated_at 
	          FROM payments WHERE booking_id = $1 ORDER BY created_at DESC LIMIT 1`
	err := r.db.QueryRow(query, bookingID).
		Scan(&p.ID, &p.BookingID, &p.UserID, &p.Amount, &p.Currency, &p.Status, &p.PaymentURL, &receipt, &p.IdempotencyKey, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if receipt.Valid {
		p.ReceiptURL = receipt.String
	}
	return &p, nil
}

func (r *PostgresRepository) UpdatePaymentStatus(id string, status, receiptURL string) (*Payment, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	query := `UPDATE payments SET status = $1, receipt_url = $2, updated_at = $3 WHERE id = $4`
	_, err := r.db.Exec(query, status, receiptURL, now, id)
	if err != nil {
		return nil, err
	}
	return r.GetPaymentByID(id)
}

func (r *PostgresRepository) CreateRefund(paymentID string, amount float64, reason string) (*RefundResultResponse, error) {
	p, err := r.GetPaymentByID(paymentID)
	if err != nil || p == nil {
		return nil, fmt.Errorf("payment not found")
	}

	refundID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	query := `INSERT INTO payment_refunds (id, payment_id, booking_id, amount, status, reason, created_at)
	          VALUES ($1, $2, $3, $4, 'COMPLETED', $5, $6)`
	_, err = r.db.Exec(query, refundID, paymentID, p.BookingID, amount, reason, now)
	if err != nil {
		return nil, err
	}

	_, _ = r.UpdatePaymentStatus(paymentID, "REFUNDED", p.ReceiptURL)

	return &RefundResultResponse{
		RefundID:       refundID,
		PaymentID:      paymentID,
		RefundedAmount: amount,
		Status:         "COMPLETED",
		ProcessedAt:    now,
	}, nil
}

// InMemoryRepository - легковесный fallback при автономной работе без внешней БД.
type InMemoryRepository struct {
	mu           sync.RWMutex
	payments     map[string]Payment
	refunds      map[string]RefundResultResponse
	idempotency  map[string]string // key -> paymentID
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		payments:    make(map[string]Payment),
		refunds:     make(map[string]RefundResultResponse),
		idempotency: make(map[string]string),
	}
}

func (m *InMemoryRepository) CreatePayment(req CreatePaymentRequest, idempotencyKey string) (*Payment, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if idempotencyKey != "" {
		if pID, exists := m.idempotency[idempotencyKey]; exists {
			p := m.payments[pID]
			return &p, true, nil
		}
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	currency := req.Currency
	if currency == "" {
		currency = "RUB"
	}

	p := Payment{
		ID:             id,
		BookingID:      req.BookingID,
		UserID:         req.UserID,
		Amount:         req.Amount,
		Currency:       currency,
		Status:         "PENDING",
		PaymentURL:     fmt.Sprintf("https://pay.bank-acquiring.ru/process?tx=%s", id),
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	m.payments[id] = p
	if idempotencyKey != "" {
		m.idempotency[idempotencyKey] = id
	}

	return &p, false, nil
}

func (m *InMemoryRepository) GetPaymentByID(id string) (*Payment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.payments[id]
	if !ok {
		return nil, nil
	}
	return &p, nil
}

func (m *InMemoryRepository) GetPaymentByBookingID(bookingID string) (*Payment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.payments {
		if p.BookingID == bookingID {
			return &p, nil
		}
	}
	return nil, nil
}

func (m *InMemoryRepository) UpdatePaymentStatus(id string, status, receiptURL string) (*Payment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.payments[id]
	if !ok {
		return nil, fmt.Errorf("payment not found")
	}
	p.Status = status
	if receiptURL != "" {
		p.ReceiptURL = receiptURL
	}
	p.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	m.payments[id] = p
	return &p, nil
}

func (m *InMemoryRepository) CreateRefund(paymentID string, amount float64, reason string) (*RefundResultResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.payments[paymentID]
	if !ok {
		return nil, fmt.Errorf("payment not found")
	}

	p.Status = "REFUNDED"
	m.payments[paymentID] = p

	res := RefundResultResponse{
		RefundID:       uuid.NewString(),
		PaymentID:      paymentID,
		RefundedAmount: amount,
		Status:         "COMPLETED",
		ProcessedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	m.refunds[res.RefundID] = res
	return &res, nil
}
