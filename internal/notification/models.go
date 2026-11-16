package notification

type NotificationRecord struct {
	ID             string `json:"id"`
	UserID         string `json:"userId"`
	Channel        string `json:"channel"` // EMAIL, SMS, PUSH
	Title          string `json:"title"`
	Body           string `json:"body"`
	DeliveryStatus string `json:"deliveryStatus"` // DELIVERED, SENT, FAILED
	SentAt         string `json:"sentAt"`
}

type NotificationPreferences struct {
	Email            string `json:"email"`
	Phone            string `json:"phone"`
	EmailEnabled     bool   `json:"emailEnabled"`
	SmsEnabled       bool   `json:"smsEnabled"`
	MarketingAllowed bool   `json:"marketingAllowed"`
}

type SendNotificationRequest struct {
	Recipient    string            `json:"recipient" binding:"required"`
	Channel      string            `json:"channel" binding:"required"`
	TemplateCode string            `json:"templateCode" binding:"required"`
	Parameters   map[string]string `json:"parameters"`
}

type NotificationTemplate struct {
	Code         string `json:"code"`
	Title        string `json:"title"`
	Channel      string `json:"channel"`
	BodyTemplate string `json:"bodyTemplate"`
}
