package catalog

import (
	"database/sql"
	"fmt"
	"sync"
)

type Repository interface {
	GetShows(genre string, ageRestriction int, page, size int) ([]Show, int, error)
	GetShowByID(id int) (*Show, error)
	GetSessions(showID int, dateFrom, dateTo string, page, size int) ([]Session, int, error)
	GetSessionDetail(id int) (*SessionDetail, error)
	GetInternalSessionVerification(id int) (*InternalSessionVerificationResponse, error)
	CreateShow(req CreateShowRequest) (*Show, error)
	ArchiveShow(id int) error
}

// PostgresRepository выполняет чистые SQL-запросы без ORM.
type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) GetShows(genre string, ageRestriction int, page, size int) ([]Show, int, error) {
	query := "SELECT id, title, description, genre, duration_minutes, age_restriction, base_price, poster_url, status FROM shows WHERE status = 'ACTIVE'"
	var args []any
	argIdx := 1

	if genre != "" {
		query += fmt.Sprintf(" AND genre = $%d", argIdx)
		args = append(args, genre)
		argIdx++
	}
	if ageRestriction > 0 {
		query += fmt.Sprintf(" AND age_restriction <= $%d", argIdx)
		args = append(args, ageRestriction)
		argIdx++
	}

	offset := page * size
	query += fmt.Sprintf(" ORDER BY id LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, size, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var shows []Show
	for rows.Next() {
		var s Show
		if err := rows.Scan(&s.ID, &s.Title, &s.Description, &s.Genre, &s.DurationMinutes, &s.AgeRestriction, &s.BasePrice, &s.PosterURL, &s.Status); err != nil {
			return nil, 0, err
		}
		shows = append(shows, s)
	}

	// Count
	var total int
	_ = r.db.QueryRow("SELECT COUNT(*) FROM shows WHERE status = 'ACTIVE'").Scan(&total)

	return shows, total, nil
}

func (r *PostgresRepository) GetShowByID(id int) (*Show, error) {
	query := "SELECT id, title, description, genre, duration_minutes, age_restriction, base_price, poster_url, status FROM shows WHERE id = $1"
	var s Show
	err := r.db.QueryRow(query, id).Scan(&s.ID, &s.Title, &s.Description, &s.Genre, &s.DurationMinutes, &s.AgeRestriction, &s.BasePrice, &s.PosterURL, &s.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *PostgresRepository) GetSessions(showID int, dateFrom, dateTo string, page, size int) ([]Session, int, error) {
	query := `SELECT s.id, s.show_id, sh.title, s.start_time, s.end_time, s.hall_name, s.status 
	          FROM sessions s JOIN shows sh ON s.show_id = sh.id WHERE s.status = 'SCHEDULED'`
	var args []any
	argIdx := 1

	if showID > 0 {
		query += fmt.Sprintf(" AND s.show_id = $%d", argIdx)
		args = append(args, showID)
		argIdx++
	}

	offset := page * size
	query += fmt.Sprintf(" ORDER BY s.start_time LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, size, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.ShowID, &s.ShowTitle, &s.StartTime, &s.EndTime, &s.HallName, &s.Status); err != nil {
			return nil, 0, err
		}
		sessions = append(sessions, s)
	}

	var total int
	_ = r.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE status = 'SCHEDULED'").Scan(&total)

	return sessions, total, nil
}

func (r *PostgresRepository) GetSessionDetail(id int) (*SessionDetail, error) {
	query := `SELECT s.id, s.show_id, sh.title, s.start_time, s.end_time, s.hall_name, s.status 
	          FROM sessions s JOIN shows sh ON s.show_id = sh.id WHERE s.id = $1`
	var s Session
	err := r.db.QueryRow(query, id).Scan(&s.ID, &s.ShowID, &s.ShowTitle, &s.StartTime, &s.EndTime, &s.HallName, &s.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Sectors
	rows, err := r.db.Query("SELECT sector_code, sector_name, price FROM pricing_sectors WHERE session_id = $1", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sectors []PricingSector
	for rows.Next() {
		var ps PricingSector
		if err := rows.Scan(&ps.SectorCode, &ps.SectorName, &ps.Price); err == nil {
			sectors = append(sectors, ps)
		}
	}

	return &SessionDetail{
		Session:        s,
		PricingSectors: sectors,
	}, nil
}

func (r *PostgresRepository) GetInternalSessionVerification(id int) (*InternalSessionVerificationResponse, error) {
	detail, err := r.GetSessionDetail(id)
	if err != nil || detail == nil {
		return nil, err
	}

	prices := make(map[string]float64)
	for _, sec := range detail.PricingSectors {
		prices[sec.SectorCode] = sec.Price
	}

	return &InternalSessionVerificationResponse{
		SessionID:             detail.Session.ID,
		ShowID:                detail.Session.ShowID,
		IsAvailableForBooking: detail.Session.Status == "SCHEDULED",
		StartTime:             detail.Session.StartTime,
		SectorPrices:          prices,
	}, nil
}

func (r *PostgresRepository) CreateShow(req CreateShowRequest) (*Show, error) {
	query := `INSERT INTO shows (title, description, genre, duration_minutes, age_restriction, base_price, poster_url, status)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, 'ACTIVE') RETURNING id`
	var id int
	err := r.db.QueryRow(query, req.Title, req.Description, req.Genre, req.DurationMinutes, req.AgeRestriction, req.BasePrice, req.PosterURL).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.GetShowByID(id)
}

func (r *PostgresRepository) ArchiveShow(id int) error {
	_, err := r.db.Exec("UPDATE shows SET status = 'ARCHIVED' WHERE id = $1", id)
	return err
}

// InMemoryRepository - легковесный fallback при отсутствии запущенной PostgreSQL.
type InMemoryRepository struct {
	mu       sync.RWMutex
	shows    map[int]Show
	sessions map[int]Session
	sectors  map[int][]PricingSector
	nextID   int
}

func NewInMemoryRepository() *InMemoryRepository {
	repo := &InMemoryRepository{
		shows:    make(map[int]Show),
		sessions: make(map[int]Session),
		sectors:  make(map[int][]PricingSector),
		nextID:   100,
	}

	// Seed data
	repo.shows[42] = Show{
		ID:              42,
		Title:           "Магия цирка: Полет над манежем",
		Description:     "Захватывающее цирковое представление с воздушными гимнастами, акробатами и иллюзионистами.",
		Genre:           "Акробатика и иллюзия",
		DurationMinutes: 130,
		AgeRestriction:  6,
		BasePrice:       1200.00,
		PosterURL:       "https://cdn.circus-theatre.ru/posters/show_42.jpg",
		Status:          "ACTIVE",
	}
	repo.shows[43] = Show{
		ID:              43,
		Title:           "Золотой клоун: Смех и трюки",
		Description:     "Веселое семейное шоу с клоунами и трюками.",
		Genre:           "Клоунада",
		DurationMinutes: 110,
		AgeRestriction:  0,
		BasePrice:       1000.00,
		PosterURL:       "https://cdn.circus-theatre.ru/posters/show_43.jpg",
		Status:          "ACTIVE",
	}

	repo.sessions[105] = Session{
		ID:        105,
		ShowID:    42,
		ShowTitle: "Магия цирка: Полет над манежем",
		StartTime: "2026-11-20T18:00:00Z",
		EndTime:   "2026-11-20T20:10:00Z",
		HallName:  "Главный манеж",
		Status:    "SCHEDULED",
	}
	repo.sessions[106] = Session{
		ID:        106,
		ShowID:    42,
		ShowTitle: "Магия цирка: Полет над манежем",
		StartTime: "2026-11-21T15:00:00Z",
		EndTime:   "2026-11-21T17:10:00Z",
		HallName:  "Главный манеж",
		Status:    "SCHEDULED",
	}

	repo.sectors[105] = []PricingSector{
		{SectorCode: "PARTERRE_A", SectorName: "Партер, сектор А (у манежа)", Price: 2500.00},
		{SectorCode: "PARTERRE_B", SectorName: "Партер, сектор Б", Price: 2000.00},
		{SectorCode: "AMPHITHEATER", SectorName: "Амфитеатр", Price: 1500.00},
		{SectorCode: "BALCONY", SectorName: "Балкон", Price: 1000.00},
	}
	repo.sectors[106] = []PricingSector{
		{SectorCode: "PARTERRE_A", SectorName: "Партер, сектор А (у манежа)", Price: 2500.00},
		{SectorCode: "AMPHITHEATER", SectorName: "Амфитеатр", Price: 1500.00},
	}

	return repo
}

func (m *InMemoryRepository) GetShows(genre string, ageRestriction int, page, size int) ([]Show, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Show
	for _, s := range m.shows {
		if s.Status != "ACTIVE" {
			continue
		}
		if genre != "" && s.Genre != genre {
			continue
		}
		if ageRestriction > 0 && s.AgeRestriction > ageRestriction {
			continue
		}
		result = append(result, s)
	}
	return result, len(result), nil
}

func (m *InMemoryRepository) GetShowByID(id int) (*Show, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.shows[id]
	if !ok {
		return nil, nil
	}
	return &s, nil
}

func (m *InMemoryRepository) GetSessions(showID int, dateFrom, dateTo string, page, size int) ([]Session, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Session
	for _, sess := range m.sessions {
		if sess.Status != "SCHEDULED" {
			continue
		}
		if showID > 0 && sess.ShowID != showID {
			continue
		}
		result = append(result, sess)
	}
	return result, len(result), nil
}

func (m *InMemoryRepository) GetSessionDetail(id int) (*SessionDetail, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sess, ok := m.sessions[id]
	if !ok {
		return nil, nil
	}
	sectors := m.sectors[id]
	return &SessionDetail{
		Session:        sess,
		PricingSectors: sectors,
	}, nil
}

func (m *InMemoryRepository) GetInternalSessionVerification(id int) (*InternalSessionVerificationResponse, error) {
	detail, err := m.GetSessionDetail(id)
	if err != nil || detail == nil {
		return nil, err
	}
	prices := make(map[string]float64)
	for _, sec := range detail.PricingSectors {
		prices[sec.SectorCode] = sec.Price
	}
	return &InternalSessionVerificationResponse{
		SessionID:             detail.Session.ID,
		ShowID:                detail.Session.ShowID,
		IsAvailableForBooking: detail.Session.Status == "SCHEDULED",
		StartTime:             detail.Session.StartTime,
		SectorPrices:          prices,
	}, nil
}

func (m *InMemoryRepository) CreateShow(req CreateShowRequest) (*Show, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextID++
	s := Show{
		ID:              m.nextID,
		Title:           req.Title,
		Description:     req.Description,
		Genre:           req.Genre,
		DurationMinutes: req.DurationMinutes,
		AgeRestriction:  req.AgeRestriction,
		BasePrice:       req.BasePrice,
		PosterURL:       req.PosterURL,
		Status:          "ACTIVE",
	}
	m.shows[s.ID] = s
	return &s, nil
}

func (m *InMemoryRepository) ArchiveShow(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.shows[id]; ok {
		s.Status = "ARCHIVED"
		m.shows[id] = s
	}
	return nil
}
