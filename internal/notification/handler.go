package notification

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"circus-microservices/internal/common"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo   Repository
	broker *common.Broker
}

func NewHandler(repo Repository, broker *common.Broker) *Handler {
	h := &Handler{repo: repo, broker: broker}
	h.registerEventSubscriptions()
	return h
}

func (h *Handler) registerEventSubscriptions() {
	// 1. Подтверждение брони -> отправка электронных билетов с QR-кодом
	h.broker.Subscribe("circus.bookings.events", func(ev common.BaseEvent) {
		switch ev.EventType {
		case "booking.confirmed":
			userID, _ := ev.Payload["userId"].(string)
			bookingID, _ := ev.Payload["bookingId"].(string)
			title := "Ваши электронные билеты в цирк"
			body := fmt.Sprintf("Оплата заказа %s подтверждена! Ваши электронные билеты с QR-кодом готовы для входа на манеж.", bookingID)

			rec, err := h.repo.SaveNotification(userID, "EMAIL", title, body)
			if err == nil {
				log.Printf("[notification-service] [DISPATCH] Sent email ticket confirmation to user %s (ID: %s)", userID, rec.ID)
			}

		case "booking.expired":
			userID, _ := ev.Payload["userId"].(string)
			bookingID, _ := ev.Payload["bookingId"].(string)
			title := "Время бронирования истекло"
			body := fmt.Sprintf("15 минут на оплату брони %s истекли. Места возвращены в свободную продажу.", bookingID)

			_, _ = h.repo.SaveNotification(userID, "EMAIL", title, body)
			log.Printf("[notification-service] [DISPATCH] Sent booking expired alert to user %s", userID)
		}
	})

	// 2. Успешный возврат средств
	h.broker.Subscribe("circus.refunds.events", func(ev common.BaseEvent) {
		if ev.EventType == "refund.completed" {
			bookingID, _ := ev.Payload["bookingId"].(string)
			amount, _ := ev.Payload["refundedAmount"].(float64)
			title := "Возврат денежных средств выполнен"
			body := fmt.Sprintf("Возврат %.2f руб. по заказу %s успешно зачислен на вашу карту.", amount, bookingID)

			_, _ = h.repo.SaveNotification("45a8b9e1-12c4-4b56-8e12-32049e210a55", "EMAIL", title, body)
			log.Printf("[notification-service] [DISPATCH] Sent refund receipt for booking %s", bookingID)
		}
	})

	// 3. Повышение ранга лояльности
	h.broker.Subscribe("circus.loyalty.events", func(ev common.BaseEvent) {
		if ev.EventType == "loyalty.tier.upgraded" {
			userID, _ := ev.Payload["userId"].(string)
			newTier, _ := ev.Payload["newTier"].(string)
			title := "Поздравляем с новым статусом!"
			body := fmt.Sprintf("Вы достигли нового уровня цирковых привилегий: %s. Вам доступен повышенный кэшбэк!", newTier)

			_, _ = h.repo.SaveNotification(userID, "PUSH", title, body)
			log.Printf("[notification-service] [DISPATCH] Sent tier upgraded notification to user %s (%s)", userID, newTier)
		}
	})
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/notifications")
	{
		api.GET("/my", h.GetMyNotifications)
		api.PUT("/preferences", h.UpdatePreferences)
	}

	internal := r.Group("/internal/v1/notifications")
	{
		internal.POST("/send", h.SendDirectNotification)
	}
}

func (h *Handler) GetMyNotifications(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		userID = "45a8b9e1-12c4-4b56-8e12-32049e210a55"
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if size <= 0 {
		size = 10
	}

	items, total, err := h.repo.GetMyNotifications(userID, page, size)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Database Error", err.Error(), c.Request.URL.Path)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":         items,
		"totalElements": total,
		"page":          page,
		"size":          size,
	})
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		userID = "45a8b9e1-12c4-4b56-8e12-32049e210a55"
	}

	var pref NotificationPreferences
	if err := c.ShouldBindJSON(&pref); err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Validation Error", err.Error(), c.Request.URL.Path)
		return
	}

	if err := h.repo.UpdatePreferences(userID, pref); err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Update Error", err.Error(), c.Request.URL.Path)
		return
	}

	c.JSON(http.StatusOK, pref)
}

func (h *Handler) SendDirectNotification(c *gin.Context) {
	var req SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Validation Error", err.Error(), c.Request.URL.Path)
		return
	}

	title := fmt.Sprintf("Уведомление: %s", req.TemplateCode)
	body := fmt.Sprintf("Сообщение для %s по шаблону %s", req.Recipient, req.TemplateCode)
	rec, err := h.repo.SaveNotification("system", req.Channel, title, body)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Send Error", err.Error(), c.Request.URL.Path)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"dispatchId": rec.ID,
		"status":     "QUEUED",
	})
}
