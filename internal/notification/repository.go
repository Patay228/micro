package notification

import (
	"database/sql"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	SaveNotification(userID, channel, title, body string) (*NotificationRecord, error)
	GetMyNotifications(userID string, page, size int) ([]NotificationRecord, int, error)
	GetPreferences(userID string) (*NotificationPreferences, error)
	UpdatePreferences(userID string, pref NotificationPreferences) error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) SaveNotification(userID, channel, title, body string) (*NotificationRecord, error) {
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	query := `INSERT INTO notifications (id, user_id, channel, title, body, delivery_status, sent_at)
	          VALUES ($1, $2, $3, $4, $5, 'DELIVERED', $6)`
	_, err := r.db.Exec(query, id, userID, channel, title, body, now)
	if err != nil {
		return nil, err
	}
	return &NotificationRecord{
		ID:             id,
		UserID:         userID,
		Channel:        channel,
		Title:          title,
		Body:           body,
		DeliveryStatus: "DELIVERED",
		SentAt:         now,
	}, nil
}

func (r *PostgresRepository) GetMyNotifications(userID string, page, size int) ([]NotificationRecord, int, error) {
	offset := page * size
	rows, err := r.db.Query(`SELECT id, user_id, channel, title, body, delivery_status, sent_at 
	                         FROM notifications WHERE user_id = $1 ORDER BY sent_at DESC LIMIT $2 OFFSET $3`,
		userID, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []NotificationRecord
	for rows.Next() {
		var n NotificationRecord
		if err := rows.Scan(&n.ID, &n.UserID, &n.Channel, &n.Title, &n.Body, &n.DeliveryStatus, &n.SentAt); err == nil {
			list = append(list, n)
		}
	}

	var total int
	_ = r.db.QueryRow("SELECT COUNT(*) FROM notifications WHERE user_id = $1", userID).Scan(&total)
	return list, total, nil
}

func (r *PostgresRepository) GetPreferences(userID string) (*NotificationPreferences, error) {
	var pref NotificationPreferences
	var email, phone sql.NullString
	query := `SELECT email, phone, email_enabled, sms_enabled, marketing_allowed FROM notification_preferences WHERE user_id = $1`
	err := r.db.QueryRow(query, userID).Scan(&email, &phone, &pref.EmailEnabled, &pref.SmsEnabled, &pref.MarketingAllowed)
	if err == sql.ErrNoRows {
		return &NotificationPreferences{
			Email:            "viewer@circus.ru",
			Phone:            "+79991234567",
			EmailEnabled:     true,
			SmsEnabled:       true,
			MarketingAllowed: false,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	if email.Valid {
		pref.Email = email.String
	}
	if phone.Valid {
		pref.Phone = phone.String
	}
	return &pref, nil
}

func (r *PostgresRepository) UpdatePreferences(userID string, pref NotificationPreferences) error {
	query := `INSERT INTO notification_preferences (user_id, email, phone, email_enabled, sms_enabled, marketing_allowed)
	          VALUES ($1, $2, $3, $4, $5, $6)
	          ON CONFLICT (user_id) DO UPDATE SET 
	          email_enabled = EXCLUDED.email_enabled, sms_enabled = EXCLUDED.sms_enabled, marketing_allowed = EXCLUDED.marketing_allowed`
	_, err := r.db.Exec(query, userID, pref.Email, pref.Phone, pref.EmailEnabled, pref.SmsEnabled, pref.MarketingAllowed)
	return err
}

type InMemoryRepository struct {
	mu            sync.RWMutex
	notifications []NotificationRecord
	preferences   map[string]NotificationPreferences
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		notifications: make([]NotificationRecord, 0),
		preferences:   make(map[string]NotificationPreferences),
	}
}

func (m *InMemoryRepository) SaveNotification(userID, channel, title, body string) (*NotificationRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec := NotificationRecord{
		ID:             uuid.NewString(),
		UserID:         userID,
		Channel:        channel,
		Title:          title,
		Body:           body,
		DeliveryStatus: "DELIVERED",
		SentAt:         time.Now().UTC().Format(time.RFC3339),
	}
	m.notifications = append(m.notifications, rec)
	return &rec, nil
}

func (m *InMemoryRepository) GetMyNotifications(userID string, page, size int) ([]NotificationRecord, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []NotificationRecord
	for i := len(m.notifications) - 1; i >= 0; i-- {
		if m.notifications[i].UserID == userID {
			result = append(result, m.notifications[i])
		}
	}
	return result, len(result), nil
}

func (m *InMemoryRepository) GetPreferences(userID string) (*NotificationPreferences, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pref, ok := m.preferences[userID]
	if !ok {
		return &NotificationPreferences{
			Email:            "viewer@circus.ru",
			Phone:            "+79991234567",
			EmailEnabled:     true,
			SmsEnabled:       true,
			MarketingAllowed: false,
		}, nil
	}
	return &pref, nil
}

func (m *InMemoryRepository) UpdatePreferences(userID string, pref NotificationPreferences) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.preferences[userID] = pref
	return nil
}
