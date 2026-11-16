package booking

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"circus-microservices/internal/common"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo   Repository
	saga   *SagaCoordinator
	broker *common.Broker
}

func NewHandler(repo Repository, saga *SagaCoordinator, broker *common.Broker) *Handler {
	h := &Handler{repo: repo, saga: saga, broker: broker}
	h.registerEventSubscriptions()
	return h
}

func (h *Handler) registerEventSubscriptions() {
	// Подписка на результаты платежей (из payment-service через брокер RabbitMQ)
	h.broker.Subscribe("circus.payments.events", func(ev common.BaseEvent) {
		bookingID, _ := ev.Payload["bookingId"].(string)
		if bookingID == "" {
			return
		}

		b, err := h.repo.GetBookingByID(bookingID)
		if err != nil || b == nil {
			log.Printf("[booking-service] Booking %s not found for event %s", bookingID, ev.EventType)
			return
		}

		switch ev.EventType {
		case "payment.completed":
			log.Printf("[booking-service] [SAGA STEP] Payment confirmed for booking %s", bookingID)
			if b.Status == "PENDING_PAYMENT" {
				// 1. Выпускаем билеты с уникальными QR-кодами
				tickets, err := h.repo.ConfirmSeatsAndCreateTickets(b)
				if err == nil {
					b.Status = "CONFIRMED"
					b.Tickets = tickets

					// 2. Публикуем событие booking.confirmed в брокер
					confEvent := common.NewEvent("booking.confirmed", map[string]any{
						"bookingId":    b.ID,
						"userId":       b.UserID,
						"sessionId":    b.SessionID,
						"totalAmount":  b.PayableAmount,
						"ticketsCount": len(tickets),
						"tickets":      tickets,
					})
					_ = h.broker.Publish("circus.bookings.events", confEvent)
					log.Printf("[booking-service] [SAGA SUCCESS] Booking %s CONFIRMED. Tickets issued: %d", b.ID, len(tickets))
				}
			} else if b.Status == "EXPIRED" {
				// Запоздалая оплата после истечения 15 минут!
				log.Printf("[booking-service] [SAGA EDGE CASE] Late payment received for expired booking %s. Initiating 100%% refund.", b.ID)
				_ = h.broker.Publish("circus.refunds.events", common.NewEvent("refund.requested", map[string]any{
					"bookingId":    b.ID,
					"userId":       b.UserID,
					"refundAmount": b.PayableAmount, // 100% возврат
					"reason":       "Late payment after 15-minute expiration",
				}))
			}

		case "payment.failed":
			log.Printf("[booking-service] [SAGA COMPENSATION] Payment failed for booking %s. Cancelling booking.", bookingID)
			_ = h.repo.UpdateBookingStatus(bookingID, "CANCELLED")
			_ = h.repo.ReleaseSeats(bookingID)
			_ = h.broker.Publish("circus.bookings.events", common.NewEvent("booking.cancelled", map[string]any{
				"bookingId": bookingID,
				"reason":    ev.Payload["reason"],
			}))
		}
	})
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/bookings")
	{
		api.POST("", h.CreateBooking)
		api.GET("/my", h.GetMyBookings)
		api.GET("/:id", h.GetBookingByID)
		api.POST("/:id/cancel", h.CancelBooking)
		api.POST("/:id/refund", h.RequestRefund)
	}

	internal := r.Group("/internal/v1")
	{
		internal.GET("/sessions/:sessionId/seats/status", h.GetSeatsStatus)
		internal.POST("/bookings/:id/expire", h.ExpireBooking)
	}
}

func (h *Handler) CreateBooking(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		userID = "45a8b9e1-12c4-4b56-8e12-32049e210a55" // default test user
	}

	var req CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Validation Error", err.Error(), c.Request.URL.Path)
		return
	}

	// Запуск Саги бронирования
	booking, err := h.saga.ExecuteBookingSaga(userID, req, idempotencyKey)
	if err != nil {
		if strings.Contains(err.Error(), "conflict") || strings.Contains(err.Error(), "held") || strings.Contains(err.Error(), "sold") {
			common.RenderProblem(c, http.StatusConflict, "Seat Conflict", err.Error(), c.Request.URL.Path)
			return
		}
		common.RenderProblem(c, http.StatusBadRequest, "Booking Failed", err.Error(), c.Request.URL.Path)
		return
	}

	c.JSON(http.StatusCreated, booking)
}

func (h *Handler) GetBookingByID(c *gin.Context) {
	id := c.Param("id")
	b, err := h.repo.GetBookingByID(id)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Database Error", err.Error(), c.Request.URL.Path)
		return
	}
	if b == nil {
		common.RenderProblem(c, http.StatusNotFound, "Booking Not Found", "Бронирование не найдено", c.Request.URL.Path)
		return
	}
	c.JSON(http.StatusOK, b)
}

func (h *Handler) GetMyBookings(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		userID = "45a8b9e1-12c4-4b56-8e12-32049e210a55"
	}
	status := c.Query("status")

	list, err := h.repo.GetBookingsByUserID(userID, status)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Database Error", err.Error(), c.Request.URL.Path)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":         list,
		"totalElements": len(list),
		"page":          0,
		"size":          10,
	})
}

func (h *Handler) CancelBooking(c *gin.Context) {
	id := c.Param("id")
	b, err := h.repo.GetBookingByID(id)
	if err != nil || b == nil {
		common.RenderProblem(c, http.StatusNotFound, "Booking Not Found", "Бронирование не найдено", c.Request.URL.Path)
		return
	}

	if b.Status != "PENDING_PAYMENT" {
		common.RenderProblem(c, http.StatusBadRequest, "Invalid Status", "Невозможно отменить заказ в текущем статусе", c.Request.URL.Path)
		return
	}

	_ = h.repo.UpdateBookingStatus(id, "CANCELLED")
	_ = h.repo.ReleaseSeats(id)
	b.Status = "CANCELLED"

	_ = h.broker.Publish("circus.bookings.events", common.NewEvent("booking.cancelled", map[string]any{
		"bookingId": id,
		"reason":    "Cancelled by user",
	}))

	c.JSON(http.StatusOK, b)
}

func (h *Handler) RequestRefund(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		userID = "45a8b9e1-12c4-4b56-8e12-32049e210a55"
	}

	var req RefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Validation Error", err.Error(), c.Request.URL.Path)
		return
	}

	b, err := h.repo.GetBookingByID(id)
	if err != nil || b == nil {
		common.RenderProblem(c, http.StatusNotFound, "Booking Not Found", "Заказ не найден", c.Request.URL.Path)
		return
	}

	refundAmount := b.PayableAmount * 0.5 // 50% согласно правилам цирка
	refResp, err := h.repo.CreateRefund(id, userID, refundAmount, req.Reason)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Refund Error", err.Error(), c.Request.URL.Path)
		return
	}

	// Запуск саги возврата через Kafka/RabbitMQ
	_ = h.broker.Publish("circus.refunds.events", common.NewEvent("refund.requested", map[string]any{
		"bookingId":      id,
		"userId":         userID,
		"refundAmount":   refundAmount,
		"penaltyPercent": 50,
	}))

	c.JSON(http.StatusAccepted, refResp)
}

func (h *Handler) GetSeatsStatus(c *gin.Context) {
	sID, _ := strconv.Atoi(c.Param("sessionId"))
	status, err := h.repo.GetSeatsStatus(sID)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Database Error", err.Error(), c.Request.URL.Path)
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *Handler) ExpireBooking(c *gin.Context) {
	id := c.Param("id")
	b, err := h.repo.GetBookingByID(id)
	if err != nil || b == nil {
		common.RenderProblem(c, http.StatusNotFound, "Booking Not Found", "Бронирование не найдено", c.Request.URL.Path)
		return
	}

	_ = h.repo.UpdateBookingStatus(id, "EXPIRED")
	_ = h.repo.ReleaseSeats(id)
	b.Status = "EXPIRED"

	_ = h.broker.Publish("circus.bookings.events", common.NewEvent("booking.expired", map[string]any{
		"bookingId": id,
		"userId":    b.UserID,
	}))

	c.JSON(http.StatusOK, b)
}
