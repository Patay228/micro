package common

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ProblemDetails реализует единую модель ошибок согласно RFC 7807.
type ProblemDetails struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance,omitempty"`
}

// RenderProblem отправляет ответ с кодом status и заголовком application/problem+json.
func RenderProblem(c *gin.Context, status int, title, detail, instance string) {
	problem := ProblemDetails{
		Type:     fmt.Sprintf("https://circus-theatre.ru/errors/%d", status),
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: instance,
	}
	c.Header("Content-Type", "application/problem+json")
	c.JSON(status, problem)
}

// BaseEvent представляет собой стандартное доменное событие для брокера (RabbitMQ / EventBus).
type BaseEvent struct {
	EventID       string         `json:"eventId"`
	EventType     string         `json:"eventType"`
	SchemaVersion int            `json:"schemaVersion"`
	OccurredAt    string         `json:"occurredAt"`
	Payload       map[string]any `json:"payload"`
}

// NewEvent создает новое событие со сгенерированным UUID и текущим временем UTC.
func NewEvent(eventType string, payload map[string]any) BaseEvent {
	return BaseEvent{
		EventID:       uuid.NewString(),
		EventType:     eventType,
		SchemaVersion: 1,
		OccurredAt:    time.Now().UTC().Format(time.RFC3339),
		Payload:       payload,
	}
}
