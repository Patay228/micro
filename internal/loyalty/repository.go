package loyalty

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type Repository interface {
	GetProfile(userID string) (*LoyaltyProfileResponse, error)
	GetTiers() ([]LoyaltyTier, error)
	ReservePoints(req ReservePointsRequest) (*PointsReservationResponse, error)
	ReleaseReservation(reservationID string) (*PointsReservationResponse, error)
	ConfirmReservation(reservationID string) (*PointsReservationResponse, error)
	AccruePoints(req AccruePointsRequest) (*AccrualResultResponse, error)
	ApplyPenalty(req PenaltyRequest) (*PenaltyResponse, error)
	GetMarketingRules() ([]MarketingRule, error)
	FindReservationByBookingID(bookingID string) (*PointsReservationResponse, error)
}

// PostgresRepository использует чистые SQL-запросы без ORM.
type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetProfile(userID string) (*LoyaltyProfileResponse, error) {
	var prof LoyaltyProfileResponse
	query := `SELECT user_id, active_balance, reserved_balance, current_tier, shows_attended 
	          FROM loyalty_accounts WHERE user_id = $1`
	err := r.db.QueryRow(query, userID).
		Scan(&prof.UserID, &prof.ActiveBalance, &prof.ReservedBalance, &prof.CurrentTier, &prof.ShowsAttended)

	if err == sql.ErrNoRows {
		// Создаем нового зрителя с 500 стартовыми баллами
		prof = LoyaltyProfileResponse{
			UserID:            userID,
			ActiveBalance:     500,
			ReservedBalance:   0,
			CurrentTier:       "CONNOISSEUR",
			CashbackPercent:   10,
			ShowsAttended:     5,
			NextTierThreshold: 10,
		}
		_, _ = r.db.Exec(`INSERT INTO loyalty_accounts (user_id, active_balance, reserved_balance, current_tier, shows_attended)
		                  VALUES ($1, 500, 0, 'CONNOISSEUR', 5) ON CONFLICT DO NOTHING`, userID)
		return &prof, nil
	}
	if err != nil {
		return nil, err
	}

	// Рассчитываем кэшбэк
	prof.CashbackPercent = 5
	prof.NextTierThreshold = 3
	if prof.ShowsAttended >= 10 {
		prof.CurrentTier = "HONORARY"
		prof.CashbackPercent = 15
		prof.NextTierThreshold = 10
	} else if prof.ShowsAttended >= 3 {
		prof.CurrentTier = "CONNOISSEUR"
		prof.CashbackPercent = 10
		prof.NextTierThreshold = 10
	}

	return &prof, nil
}

func (r *PostgresRepository) GetTiers() ([]LoyaltyTier, error) {
	rows, err := r.db.Query("SELECT tier_code, tier_name, min_shows, cashback_percent, description FROM loyalty_tiers ORDER BY min_shows")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tiers []LoyaltyTier
	for rows.Next() {
		var t LoyaltyTier
		if err := rows.Scan(&t.TierCode, &t.TierName, &t.MinShows, &t.CashbackPercent, &t.Description); err == nil {
			tiers = append(tiers, t)
		}
	}
	return tiers, nil
}

func (r *PostgresRepository) ReservePoints(req ReservePointsRequest) (*PointsReservationResponse, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Проверяем баланс
	var activeBalance int
	err = tx.QueryRow("SELECT active_balance FROM loyalty_accounts WHERE user_id = $1 FOR UPDATE", req.UserID).Scan(&activeBalance)
	if err != nil {
		return nil, fmt.Errorf("user account not found")
	}

	if activeBalance < req.PointsToReserve {
		return nil, fmt.Errorf("insufficient active points: requested %d, available %d", req.PointsToReserve, activeBalance)
	}

	resID := uuid.NewString()
	_, err = tx.Exec(`UPDATE loyalty_accounts SET active_balance = active_balance - $1, reserved_balance = reserved_balance + $1 WHERE user_id = $2`,
		req.PointsToReserve, req.UserID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`INSERT INTO loyalty_reservations (id, booking_id, user_id, reserved_points, status)
	                  VALUES ($1, $2, $3, $4, 'RESERVED')`, resID, req.BookingID, req.UserID, req.PointsToReserve)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &PointsReservationResponse{
		ReservationID:  resID,
		BookingID:      req.BookingID,
		UserID:         req.UserID,
		ReservedPoints: req.PointsToReserve,
		Status:         "RESERVED",
	}, nil
}

func (r *PostgresRepository) ReleaseReservation(reservationID string) (*PointsReservationResponse, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var res PointsReservationResponse
	err = tx.QueryRow(`SELECT id, booking_id, user_id, reserved_points, status 
	                   FROM loyalty_reservations WHERE id = $1 FOR UPDATE`, reservationID).
		Scan(&res.ReservationID, &res.BookingID, &res.UserID, &res.ReservedPoints, &res.Status)
	if err != nil {
		return nil, fmt.Errorf("reservation not found")
	}

	if res.Status != "RESERVED" {
		return &res, nil // уже обработано
	}

	_, err = tx.Exec(`UPDATE loyalty_accounts SET active_balance = active_balance + $1, reserved_balance = reserved_balance - $1 WHERE user_id = $2`,
		res.ReservedPoints, res.UserID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec("UPDATE loyalty_reservations SET status = 'RELEASED' WHERE id = $1", reservationID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	res.Status = "RELEASED"
	return &res, nil
}

func (r *PostgresRepository) ConfirmReservation(reservationID string) (*PointsReservationResponse, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var res PointsReservationResponse
	err = tx.QueryRow(`SELECT id, booking_id, user_id, reserved_points, status 
	                   FROM loyalty_reservations WHERE id = $1 FOR UPDATE`, reservationID).
		Scan(&res.ReservationID, &res.BookingID, &res.UserID, &res.ReservedPoints, &res.Status)
	if err != nil {
		return nil, fmt.Errorf("reservation not found")
	}

	if res.Status != "RESERVED" {
		return &res, nil
	}

	// Списываем из reserved_balance
	_, err = tx.Exec(`UPDATE loyalty_accounts SET reserved_balance = reserved_balance - $1 WHERE user_id = $2`,
		res.ReservedPoints, res.UserID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec("UPDATE loyalty_reservations SET status = 'CONFIRMED' WHERE id = $1", reservationID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	res.Status = "CONFIRMED"
	return &res, nil
}

func (r *PostgresRepository) FindReservationByBookingID(bookingID string) (*PointsReservationResponse, error) {
	var res PointsReservationResponse
	err := r.db.QueryRow(`SELECT id, booking_id, user_id, reserved_points, status 
	                      FROM loyalty_reservations WHERE booking_id = $1 ORDER BY created_at DESC LIMIT 1`, bookingID).
		Scan(&res.ReservationID, &res.BookingID, &res.UserID, &res.ReservedPoints, &res.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *PostgresRepository) AccruePoints(req AccruePointsRequest) (*AccrualResultResponse, error) {
	prof, err := r.GetProfile(req.UserID)
	if err != nil {
		return nil, err
	}

	// Расчет 10% кэшбэка
	cashbackPercent := prof.CashbackPercent
	if cashbackPercent <= 0 {
		cashbackPercent = 10
	}
	pointsToAdd := int(req.PaidMoneyAmount * float64(cashbackPercent) / 100.0)
	if pointsToAdd < 1 {
		pointsToAdd = 10
	}

	newAttended := prof.ShowsAttended + 1
	newTier := prof.CurrentTier
	tierChanged := false
	if newAttended >= 10 && prof.CurrentTier != "HONORARY" {
		newTier = "HONORARY"
		tierChanged = true
	} else if newAttended >= 3 && prof.CurrentTier == "ROOKIE" {
		newTier = "CONNOISSEUR"
		tierChanged = true
	}

	_, err = r.db.Exec(`UPDATE loyalty_accounts SET active_balance = active_balance + $1, shows_attended = $2, current_tier = $3, updated_at = NOW() WHERE user_id = $4`,
		pointsToAdd, newAttended, newTier, req.UserID)
	if err != nil {
		return nil, err
	}

	return &AccrualResultResponse{
		AccruedPoints: pointsToAdd,
		NewBalance:    prof.ActiveBalance + pointsToAdd,
		TierChanged:   tierChanged,
		CurrentTier:   newTier,
	}, nil
}

func (r *PostgresRepository) ApplyPenalty(req PenaltyRequest) (*PenaltyResponse, error) {
	prof, err := r.GetProfile(req.UserID)
	if err != nil {
		return nil, err
	}

	deduct := 220
	if prof.ActiveBalance < deduct {
		deduct = prof.ActiveBalance
	}

	_, err = r.db.Exec(`UPDATE loyalty_accounts SET active_balance = active_balance - $1 WHERE user_id = $2`, deduct, req.UserID)
	if err != nil {
		return nil, err
	}

	return &PenaltyResponse{
		DeductedPoints: deduct,
		UpdatedBalance: prof.ActiveBalance - deduct,
		PenaltyApplied: true,
	}, nil
}

func (r *PostgresRepository) GetMarketingRules() ([]MarketingRule, error) {
	rows, err := r.db.Query("SELECT id, name, target_genre, extra_cashback_percent, is_active FROM marketing_rules WHERE is_active = TRUE")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []MarketingRule
	for rows.Next() {
		var mr MarketingRule
		var tg sql.NullString
		if err := rows.Scan(&mr.ID, &mr.Name, &tg, &mr.ExtraCashbackPercent, &mr.IsActive); err == nil {
			if tg.Valid {
				mr.TargetShowGenre = tg.String
			}
			rules = append(rules, mr)
		}
	}
	return rules, nil
}

// InMemoryRepository - легковесный fallback.
type InMemoryRepository struct {
	mu           sync.RWMutex
	accounts     map[string]LoyaltyProfileResponse
	reservations map[string]PointsReservationResponse
	tiers        []LoyaltyTier
}

func NewInMemoryRepository() *InMemoryRepository {
	repo := &InMemoryRepository{
		accounts:     make(map[string]LoyaltyProfileResponse),
		reservations: make(map[string]PointsReservationResponse),
		tiers: []LoyaltyTier{
			{TierCode: "ROOKIE", TierName: "Новичок", MinShows: 0, CashbackPercent: 5, Description: "5% кэшбэк"},
			{TierCode: "CONNOISSEUR", TierName: "Знаток манежа", MinShows: 3, CashbackPercent: 10, Description: "10% кэшбэк"},
			{TierCode: "HONORARY", TierName: "Почётный зритель", MinShows: 10, CashbackPercent: 15, Description: "15% кэшбэк"},
		},
	}

	// Seed user
	repo.accounts["45a8b9e1-12c4-4b56-8e12-32049e210a55"] = LoyaltyProfileResponse{
		UserID:            "45a8b9e1-12c4-4b56-8e12-32049e210a55",
		ActiveBalance:     500,
		ReservedBalance:   0,
		CurrentTier:       "CONNOISSEUR",
		CashbackPercent:   10,
		ShowsAttended:     5,
		NextTierThreshold: 10,
	}

	return repo
}

func (m *InMemoryRepository) GetProfile(userID string) (*LoyaltyProfileResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.accounts[userID]
	if !ok {
		p = LoyaltyProfileResponse{
			UserID:            userID,
			ActiveBalance:     500,
			ReservedBalance:   0,
			CurrentTier:       "CONNOISSEUR",
			CashbackPercent:   10,
			ShowsAttended:     5,
			NextTierThreshold: 10,
		}
		m.accounts[userID] = p
	}
	return &p, nil
}

func (m *InMemoryRepository) GetTiers() ([]LoyaltyTier, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tiers, nil
}

func (m *InMemoryRepository) ReservePoints(req ReservePointsRequest) (*PointsReservationResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.accounts[req.UserID]
	if !ok {
		p = LoyaltyProfileResponse{
			UserID:          req.UserID,
			ActiveBalance:   500,
			CurrentTier:     "CONNOISSEUR",
			CashbackPercent: 10,
		}
	}

	if p.ActiveBalance < req.PointsToReserve {
		return nil, fmt.Errorf("insufficient active points: requested %d, available %d", req.PointsToReserve, p.ActiveBalance)
	}

	p.ActiveBalance -= req.PointsToReserve
	p.ReservedBalance += req.PointsToReserve
	m.accounts[req.UserID] = p

	resID := uuid.NewString()
	res := PointsReservationResponse{
		ReservationID:  resID,
		BookingID:      req.BookingID,
		UserID:         req.UserID,
		ReservedPoints: req.PointsToReserve,
		Status:         "RESERVED",
	}
	m.reservations[resID] = res
	return &res, nil
}

func (m *InMemoryRepository) ReleaseReservation(reservationID string) (*PointsReservationResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	res, ok := m.reservations[reservationID]
	if !ok {
		return nil, fmt.Errorf("reservation not found")
	}

	if res.Status != "RESERVED" {
		return &res, nil
	}

	p := m.accounts[res.UserID]
	p.ActiveBalance += res.ReservedPoints
	p.ReservedBalance -= res.ReservedPoints
	m.accounts[res.UserID] = p

	res.Status = "RELEASED"
	m.reservations[reservationID] = res
	return &res, nil
}

func (m *InMemoryRepository) ConfirmReservation(reservationID string) (*PointsReservationResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	res, ok := m.reservations[reservationID]
	if !ok {
		return nil, fmt.Errorf("reservation not found")
	}

	if res.Status != "RESERVED" {
		return &res, nil
	}

	p := m.accounts[res.UserID]
	p.ReservedBalance -= res.ReservedPoints
	m.accounts[res.UserID] = p

	res.Status = "CONFIRMED"
	m.reservations[reservationID] = res
	return &res, nil
}

func (m *InMemoryRepository) FindReservationByBookingID(bookingID string) (*PointsReservationResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, res := range m.reservations {
		if res.BookingID == bookingID {
			return &res, nil
		}
	}
	return nil, nil
}

func (m *InMemoryRepository) AccruePoints(req AccruePointsRequest) (*AccrualResultResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p := m.accounts[req.UserID]
	cashbackPercent := p.CashbackPercent
	if cashbackPercent <= 0 {
		cashbackPercent = 10
	}
	points := int(req.PaidMoneyAmount * float64(cashbackPercent) / 100.0)
	if points < 1 {
		points = 10
	}

	p.ActiveBalance += points
	p.ShowsAttended += 1
	m.accounts[req.UserID] = p

	return &AccrualResultResponse{
		AccruedPoints: points,
		NewBalance:    p.ActiveBalance,
		TierChanged:   false,
		CurrentTier:   p.CurrentTier,
	}, nil
}

func (m *InMemoryRepository) ApplyPenalty(req PenaltyRequest) (*PenaltyResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	p := m.accounts[req.UserID]
	deduct := 220
	if p.ActiveBalance < deduct {
		deduct = p.ActiveBalance
	}
	p.ActiveBalance -= deduct
	m.accounts[req.UserID] = p

	return &PenaltyResponse{
		DeductedPoints: deduct,
		UpdatedBalance: p.ActiveBalance,
		PenaltyApplied: true,
	}, nil
}

func (m *InMemoryRepository) GetMarketingRules() ([]MarketingRule, error) {
	return []MarketingRule{
		{ID: 101, Name: "Двойные баллы на утренние семейные представления", TargetShowGenre: "Клоунада", ExtraCashbackPercent: 10, IsActive: true},
	}, nil
}
