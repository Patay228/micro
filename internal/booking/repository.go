package booking

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	CreateBooking(b BookingResponse, idempotencyKey string) (*BookingResponse, bool, error)
	GetBookingByID(id string) (*BookingResponse, error)
	GetBookingByIdempotencyKey(key string) (*BookingResponse, error)
	GetBookingsByUserID(userID string, status string) ([]BookingResponse, error)
	UpdateBookingStatus(id string, status string) error
	UpdateBookingPaymentDetails(id string, paymentURL string, pointsDiscount, payableAmount float64) error
	DeleteBooking(id string) error
	HoldSeats(sessionID int, bookingID string, seats []SeatIdentifier, expiresAt time.Time) error
	ReleaseSeats(bookingID string) error
	ConfirmSeatsAndCreateTickets(b *BookingResponse) ([]Ticket, error)
	CreateRefund(bookingID, userID string, amount float64, reason string) (*RefundResponse, error)
	GetSeatsStatus(sessionID int) (*SessionSeatsStatusResponse, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateBooking(b BookingResponse, idempotencyKey string) (*BookingResponse, bool, error) {
	if idempotencyKey != "" {
		var existingID string
		err := r.db.QueryRow("SELECT id FROM bookings WHERE idempotency_key = $1", idempotencyKey).Scan(&existingID)
		if err == nil {
			ex, err := r.GetBookingByID(existingID)
			return ex, true, err
		}
	}

	query := `INSERT INTO bookings (id, session_id, user_id, status, total_price, points_discount, payable_amount, payment_url, idempotency_key, created_at, expires_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.db.Exec(query, b.ID, b.SessionID, b.UserID, b.Status, b.TotalPrice, b.PointsDiscount, b.PayableAmount, b.PaymentURL, idempotencyKey, b.CreatedAt, b.ExpiresAt)
	if err != nil {
		return nil, false, err
	}

	return &b, false, nil
}

func (r *PostgresRepository) GetBookingByIdempotencyKey(key string) (*BookingResponse, error) {
	if key == "" {
		return nil, nil
	}
	var existingID string
	err := r.db.QueryRow("SELECT id FROM bookings WHERE idempotency_key = $1", key).Scan(&existingID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.GetBookingByID(existingID)
}

func (r *PostgresRepository) GetBookingByID(id string) (*BookingResponse, error) {
	var b BookingResponse
	var paymentURL sql.NullString
	query := `SELECT id, session_id, user_id, status, total_price, points_discount, payable_amount, payment_url, created_at, expires_at
	          FROM bookings WHERE id = $1`
	err := r.db.QueryRow(query, id).Scan(
		&b.ID, &b.SessionID, &b.UserID, &b.Status, &b.TotalPrice, &b.PointsDiscount, &b.PayableAmount, &paymentURL, &b.CreatedAt, &b.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if paymentURL.Valid {
		b.PaymentURL = paymentURL.String
	}

	// Seats
	seatRows, err := r.db.Query("SELECT sector_code, row_num, seat_num FROM seat_holds WHERE booking_id = $1", id)
	if err == nil {
		defer seatRows.Close()
		for seatRows.Next() {
			var s SeatIdentifier
			if err := seatRows.Scan(&s.SectorCode, &s.Row, &s.SeatNumber); err == nil {
				b.Seats = append(b.Seats, s)
			}
		}
	}

	// Tickets if confirmed
	ticketRows, err := r.db.Query("SELECT ticket_number, sector_code, row_num, seat_num, price, qr_code_url FROM tickets WHERE booking_id = $1", id)
	if err == nil {
		defer ticketRows.Close()
		for ticketRows.Next() {
			var t Ticket
			if err := ticketRows.Scan(&t.TicketNumber, &t.SectorCode, &t.Row, &t.SeatNumber, &t.Price, &t.QRCodeURL); err == nil {
				b.Tickets = append(b.Tickets, t)
				if len(b.Seats) == 0 {
					b.Seats = append(b.Seats, SeatIdentifier{SectorCode: t.SectorCode, Row: t.Row, SeatNumber: t.SeatNumber})
				}
			}
		}
	}

	return &b, nil
}

func (r *PostgresRepository) GetBookingsByUserID(userID string, status string) ([]BookingResponse, error) {
	query := `SELECT id, session_id, user_id, status, total_price, points_discount, payable_amount, created_at, expires_at
	          FROM bookings WHERE user_id = $1`
	var args []any
	args = append(args, userID)
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []BookingResponse
	for rows.Next() {
		var b BookingResponse
		if err := rows.Scan(&b.ID, &b.SessionID, &b.UserID, &b.Status, &b.TotalPrice, &b.PointsDiscount, &b.PayableAmount, &b.CreatedAt, &b.ExpiresAt); err == nil {
			list = append(list, b)
		}
	}
	return list, nil
}

func (r *PostgresRepository) UpdateBookingStatus(id string, status string) error {
	_, err := r.db.Exec("UPDATE bookings SET status = $1 WHERE id = $2", status, id)
	return err
}

func (r *PostgresRepository) UpdateBookingPaymentDetails(id string, paymentURL string, pointsDiscount, payableAmount float64) error {
	_, err := r.db.Exec(`UPDATE bookings SET payment_url = $1, points_discount = $2, payable_amount = $3 WHERE id = $4`,
		paymentURL, pointsDiscount, payableAmount, id)
	return err
}

func (r *PostgresRepository) DeleteBooking(id string) error {
	_, err := r.db.Exec("DELETE FROM bookings WHERE id = $1", id)
	return err
}

func (r *PostgresRepository) HoldSeats(sessionID int, bookingID string, seats []SeatIdentifier, expiresAt time.Time) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Удаляем устаревшие блокировки
	_, _ = tx.Exec("DELETE FROM seat_holds WHERE expires_at < NOW()")

	for _, s := range seats {
		// Проверяем, не продано ли место
		var soldCount int
		_ = tx.QueryRow(`SELECT COUNT(*) FROM tickets WHERE session_id = $1 AND sector_code = $2 AND row_num = $3 AND seat_num = $4`,
			sessionID, s.SectorCode, s.Row, s.SeatNumber).Scan(&soldCount)
		if soldCount > 0 {
			return fmt.Errorf("seat already sold: sector %s, row %d, seat %d", s.SectorCode, s.Row, s.SeatNumber)
		}

		// Проверяем, не удерживается ли место
		var holdCount int
		_ = tx.QueryRow(`SELECT COUNT(*) FROM seat_holds WHERE session_id = $1 AND sector_code = $2 AND row_num = $3 AND seat_num = $4 AND expires_at >= NOW()`,
			sessionID, s.SectorCode, s.Row, s.SeatNumber).Scan(&holdCount)
		if holdCount > 0 {
			return fmt.Errorf("seat currently held by another user: sector %s, row %d, seat %d", s.SectorCode, s.Row, s.SeatNumber)
		}

		// Вставляем удержание
		_, err = tx.Exec(`INSERT INTO seat_holds (session_id, sector_code, row_num, seat_num, booking_id, expires_at)
		                  VALUES ($1, $2, $3, $4, $5, $6)`, sessionID, s.SectorCode, s.Row, s.SeatNumber, bookingID, expiresAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *PostgresRepository) ReleaseSeats(bookingID string) error {
	_, err := r.db.Exec("DELETE FROM seat_holds WHERE booking_id = $1", bookingID)
	return err
}

func (r *PostgresRepository) ConfirmSeatsAndCreateTickets(b *BookingResponse) ([]Ticket, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var tickets []Ticket
	singlePrice := b.TotalPrice / float64(len(b.Seats))
	if singlePrice <= 0 {
		singlePrice = 1000
	}

	for _, s := range b.Seats {
		ticketNum := fmt.Sprintf("CRK-%d-%02d%02d-%s", b.SessionID, s.Row, s.SeatNumber, uuid.NewString()[:6])
		qrURL := fmt.Sprintf("https://cdn.circus-theatre.ru/qrcodes/%s.png", ticketNum)

		query := `INSERT INTO tickets (ticket_number, booking_id, session_id, sector_code, row_num, seat_num, price, qr_code_url)
		          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
		_, err = tx.Exec(query, ticketNum, b.ID, b.SessionID, s.SectorCode, s.Row, s.SeatNumber, singlePrice, qrURL)
		if err != nil {
			return nil, err
		}

		tickets = append(tickets, Ticket{
			TicketNumber: ticketNum,
			SectorCode:   s.SectorCode,
			Row:          s.Row,
			SeatNumber:   s.SeatNumber,
			Price:        singlePrice,
			QRCodeURL:    qrURL,
		})
	}

	// Удаляем удержание
	_, _ = tx.Exec("DELETE FROM seat_holds WHERE booking_id = $1", b.ID)
	// Обновляем статус брони
	_, _ = tx.Exec("UPDATE bookings SET status = 'CONFIRMED' WHERE id = $1", b.ID)

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return tickets, nil
}

func (r *PostgresRepository) CreateRefund(bookingID, userID string, amount float64, reason string) (*RefundResponse, error) {
	refundID := uuid.NewString()
	query := `INSERT INTO ticket_refunds (id, booking_id, user_id, refund_amount, penalty_percent, reason, status)
	          VALUES ($1, $2, $3, $4, 50, $5, 'REQUESTED')`
	_, err := r.db.Exec(query, refundID, bookingID, userID, amount, reason)
	if err != nil {
		return nil, err
	}

	_, _ = r.db.Exec("UPDATE bookings SET status = 'REFUNDED' WHERE id = $1", bookingID)

	return &RefundResponse{
		BookingID:      bookingID,
		RefundStatus:   "REQUESTED",
		RefundAmount:   amount,
		PenaltyPercent: 50,
	}, nil
}

func (r *PostgresRepository) GetSeatsStatus(sessionID int) (*SessionSeatsStatusResponse, error) {
	// Occupied
	occupiedRows, _ := r.db.Query("SELECT sector_code, row_num, seat_num FROM tickets WHERE session_id = $1", sessionID)
	var occupied []SeatIdentifier
	if occupiedRows != nil {
		defer occupiedRows.Close()
		for occupiedRows.Next() {
			var s SeatIdentifier
			if err := occupiedRows.Scan(&s.SectorCode, &s.Row, &s.SeatNumber); err == nil {
				occupied = append(occupied, s)
			}
		}
	}

	// Held
	heldRows, _ := r.db.Query("SELECT sector_code, row_num, seat_num FROM seat_holds WHERE session_id = $1 AND expires_at >= NOW()", sessionID)
	var held []SeatIdentifier
	if heldRows != nil {
		defer heldRows.Close()
		for heldRows.Next() {
			var s SeatIdentifier
			if err := heldRows.Scan(&s.SectorCode, &s.Row, &s.SeatNumber); err == nil {
				held = append(held, s)
			}
		}
	}

	return &SessionSeatsStatusResponse{
		SessionID:     sessionID,
		OccupiedSeats: occupied,
		HeldSeats:     held,
	}, nil
}

// InMemoryRepository - легковесный fallback для автономной работы и тестирования
type InMemoryRepository struct {
	mu          sync.RWMutex
	bookings    map[string]BookingResponse
	heldSeats   map[string]time.Time // key: "session:sector:row:seat" -> expiresAt
	heldBooking map[string]string    // key: "session:sector:row:seat" -> bookingID
	soldSeats   map[string]Ticket    // key: "session:sector:row:seat" -> ticket
	idempotency map[string]string
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		bookings:    make(map[string]BookingResponse),
		heldSeats:   make(map[string]time.Time),
		heldBooking: make(map[string]string),
		soldSeats:   make(map[string]Ticket),
		idempotency: make(map[string]string),
	}
}

func (m *InMemoryRepository) CreateBooking(b BookingResponse, idempotencyKey string) (*BookingResponse, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if idempotencyKey != "" {
		if bID, exists := m.idempotency[idempotencyKey]; exists {
			ex := m.bookings[bID]
			return &ex, true, nil
		}
		m.idempotency[idempotencyKey] = b.ID
	}

	m.bookings[b.ID] = b
	return &b, false, nil
}

func (m *InMemoryRepository) GetBookingByIdempotencyKey(key string) (*BookingResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if key == "" {
		return nil, nil
	}
	bID, ok := m.idempotency[key]
	if !ok {
		return nil, nil
	}
	b, ok := m.bookings[bID]
	if !ok {
		return nil, nil
	}
	return &b, nil
}

func (m *InMemoryRepository) GetBookingByID(id string) (*BookingResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.bookings[id]
	if !ok {
		return nil, nil
	}
	return &b, nil
}

func (m *InMemoryRepository) GetBookingsByUserID(userID string, status string) ([]BookingResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []BookingResponse
	for _, b := range m.bookings {
		if b.UserID == userID {
			if status == "" || b.Status == status {
				list = append(list, b)
			}
		}
	}
	return list, nil
}

func (m *InMemoryRepository) UpdateBookingStatus(id string, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.bookings[id]
	if !ok {
		return fmt.Errorf("booking not found")
	}
	b.Status = status
	m.bookings[id] = b
	return nil
}

func (m *InMemoryRepository) UpdateBookingPaymentDetails(id string, paymentURL string, pointsDiscount, payableAmount float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.bookings[id]
	if !ok {
		return fmt.Errorf("booking not found")
	}
	b.PaymentURL = paymentURL
	b.PointsDiscount = pointsDiscount
	b.PayableAmount = payableAmount
	m.bookings[id] = b
	return nil
}

func (m *InMemoryRepository) DeleteBooking(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.bookings, id)
	for k, v := range m.idempotency {
		if v == id {
			delete(m.idempotency, k)
		}
	}
	return nil
}

func (m *InMemoryRepository) HoldSeats(sessionID int, bookingID string, seats []SeatIdentifier, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	// Проверка
	for _, s := range seats {
		key := fmt.Sprintf("%d:%s:%d:%d", sessionID, s.SectorCode, s.Row, s.SeatNumber)
		if _, sold := m.soldSeats[key]; sold {
			return fmt.Errorf("seat already sold: sector %s, row %d, seat %d", s.SectorCode, s.Row, s.SeatNumber)
		}
		if exp, held := m.heldSeats[key]; held && exp.After(now) {
			return fmt.Errorf("seat currently held by another user: sector %s, row %d, seat %d", s.SectorCode, s.Row, s.SeatNumber)
		}
	}

	// Удержание
	for _, s := range seats {
		key := fmt.Sprintf("%d:%s:%d:%d", sessionID, s.SectorCode, s.Row, s.SeatNumber)
		m.heldSeats[key] = expiresAt
		m.heldBooking[key] = bookingID
	}
	return nil
}

func (m *InMemoryRepository) ReleaseSeats(bookingID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, bID := range m.heldBooking {
		if bID == bookingID {
			delete(m.heldBooking, key)
			delete(m.heldSeats, key)
		}
	}
	return nil
}

func (m *InMemoryRepository) ConfirmSeatsAndCreateTickets(b *BookingResponse) ([]Ticket, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var tickets []Ticket
	singlePrice := b.TotalPrice / float64(len(b.Seats))
	if singlePrice <= 0 {
		singlePrice = 1000
	}

	for _, s := range b.Seats {
		key := fmt.Sprintf("%d:%s:%d:%d", b.SessionID, s.SectorCode, s.Row, s.SeatNumber)
		ticketNum := fmt.Sprintf("CRK-%d-%02d%02d-%s", b.SessionID, s.Row, s.SeatNumber, uuid.NewString()[:6])
		t := Ticket{
			TicketNumber: ticketNum,
			SectorCode:   s.SectorCode,
			Row:          s.Row,
			SeatNumber:   s.SeatNumber,
			Price:        singlePrice,
			QRCodeURL:    fmt.Sprintf("https://cdn.circus-theatre.ru/qrcodes/%s.png", ticketNum),
		}
		m.soldSeats[key] = t
		delete(m.heldSeats, key)
		delete(m.heldBooking, key)
		tickets = append(tickets, t)
	}

	b.Status = "CONFIRMED"
	b.Tickets = tickets
	m.bookings[b.ID] = *b

	return tickets, nil
}

func (m *InMemoryRepository) CreateRefund(bookingID, userID string, amount float64, reason string) (*RefundResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if b, ok := m.bookings[bookingID]; ok {
		b.Status = "REFUNDED"
		m.bookings[bookingID] = b
	}

	return &RefundResponse{
		BookingID:      bookingID,
		RefundStatus:   "REQUESTED",
		RefundAmount:   amount,
		PenaltyPercent: 50,
	}, nil
}

func (m *InMemoryRepository) GetSeatsStatus(sessionID int) (*SessionSeatsStatusResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var occupied []SeatIdentifier
	var held []SeatIdentifier
	now := time.Now()

	for key, t := range m.soldSeats {
		var sID int
		_, _ = fmt.Sscanf(key, "%d:", &sID)
		if sID == sessionID {
			occupied = append(occupied, SeatIdentifier{SectorCode: t.SectorCode, Row: t.Row, SeatNumber: t.SeatNumber})
		}
	}

	for key, exp := range m.heldSeats {
		var sID int
		var sec string
		var r, num int
		_, _ = fmt.Sscanf(key, "%d:%[^:]:%d:%d", &sID, &sec, &r, &num)
		if sID == sessionID && exp.After(now) {
			held = append(held, SeatIdentifier{SectorCode: sec, Row: r, SeatNumber: num})
		}
	}

	return &SessionSeatsStatusResponse{
		SessionID:     sessionID,
		OccupiedSeats: occupied,
		HeldSeats:     held,
	}, nil
}
