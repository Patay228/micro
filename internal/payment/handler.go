package payment

import (
	"fmt"
	"log"
	"net/http"

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
	// Подписка на событие запроса возврата
	h.broker.Subscribe("circus.refunds.events", func(ev common.BaseEvent) {
		if ev.EventType == "refund.requested" {
			bookingID, _ := ev.Payload["bookingId"].(string)
			log.Printf("[payment-service] Received refund.requested for bookingId=%s", bookingID)
			if bookingID != "" {
				p, err := h.repo.GetPaymentByBookingID(bookingID)
				if err == nil && p != nil {
					refundAmount := p.Amount * 0.5 // возврат 50%
					if ra, ok := ev.Payload["refundAmount"].(float64); ok && ra > 0 {
						refundAmount = ra
					}
					ref, err := h.repo.CreateRefund(p.ID, refundAmount, "Возврат по запросу клиента")
					if err == nil {
						// Публикуем refund.completed
						outEv := common.NewEvent("refund.completed", map[string]any{
							"refundId":       ref.RefundID,
							"paymentId":      p.ID,
							"bookingId":      bookingID,
							"refundedAmount": ref.RefundedAmount,
						})
						_ = h.broker.Publish("circus.refunds.events", outEv)
						log.Printf("[payment-service] Successfully processed refund and published refund.completed for bookingId=%s", bookingID)
					}
				}
			}
		}
	})
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	internal := r.Group("/internal/v1")
	{
		internal.POST("/payments", h.CreatePayment)
		internal.GET("/payments/:id", h.GetPaymentByID)
		internal.POST("/payments/:id/refund", h.ProcessRefund)
	}

	api := r.Group("/api/v1/payments")
	{
		api.POST("/webhooks/acquiring", h.HandleWebhook)
	}
}

func (h *Handler) CreatePayment(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")

	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Validation Error", err.Error(), c.Request.URL.Path)
		return
	}

	payment, _, err := h.repo.CreatePayment(req, idempotencyKey)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Payment Error", err.Error(), c.Request.URL.Path)
		return
	}

	c.JSON(http.StatusCreated, payment)
}

func (h *Handler) GetPaymentByID(c *gin.Context) {
	id := c.Param("id")
	p, err := h.repo.GetPaymentByID(id)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Database Error", err.Error(), c.Request.URL.Path)
		return
	}
	if p == nil {
		common.RenderProblem(c, http.StatusNotFound, "Payment Not Found", "Платёж не найден", c.Request.URL.Path)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) ProcessRefund(c *gin.Context) {
	id := c.Param("id")
	var req ProcessRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Validation Error", err.Error(), c.Request.URL.Path)
		return
	}

	res, err := h.repo.CreateRefund(id, req.RefundAmount, req.Reason)
	if err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Refund Error", err.Error(), c.Request.URL.Path)
		return
	}

	p, _ := h.repo.GetPaymentByID(id)
	bookingID := ""
	if p != nil {
		bookingID = p.BookingID
	}

	// Публикация события в брокер
	outEv := common.NewEvent("refund.completed", map[string]any{
		"refundId":       res.RefundID,
		"paymentId":      id,
		"bookingId":      bookingID,
		"refundedAmount": res.RefundedAmount,
	})
	_ = h.broker.Publish("circus.refunds.events", outEv)

	c.JSON(http.StatusOK, res)
}

func (h *Handler) HandleWebhook(c *gin.Context) {
	var payload AcquiringWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Webhook Error", err.Error(), c.Request.URL.Path)
		return
	}

	p, err := h.repo.GetPaymentByID(payload.InternalPaymentID)
	if err == nil && p == nil {
		p, err = h.repo.GetPaymentByBookingID(payload.InternalPaymentID)
	}
	if err != nil || p == nil {
		common.RenderProblem(c, http.StatusNotFound, "Payment Not Found", "Указанный платёж не существует", c.Request.URL.Path)
		return
	}

	if payload.Status == "SUCCESS" {
		receiptURL := fmt.Sprintf("https://receipts.ofd.ru/check/%s", p.ID)
		updated, err := h.repo.UpdatePaymentStatus(p.ID, "COMPLETED", receiptURL)
		if err != nil {
			common.RenderProblem(c, http.StatusInternalServerError, "Update Error", err.Error(), c.Request.URL.Path)
			return
		}

		// Публикация payment.completed
		ev := common.NewEvent("payment.completed", map[string]any{
			"paymentId": updated.ID,
			"bookingId": updated.BookingID,
			"userId":    updated.UserID,
			"amount":    updated.Amount,
			"currency":  updated.Currency,
		})
		_ = h.broker.Publish("circus.payments.events", ev)
		log.Printf("[payment-service] Published payment.completed for bookingId=%s", updated.BookingID)

	} else {
		// Оплата отклонена
		updated, _ := h.repo.UpdatePaymentStatus(p.ID, "FAILED", "")
		ev := common.NewEvent("payment.failed", map[string]any{
			"paymentId": updated.ID,
			"bookingId": updated.BookingID,
			"reason":    payload.FailureReason,
		})
		_ = h.broker.Publish("circus.payments.events", ev)
		log.Printf("[payment-service] Published payment.failed for bookingId=%s, reason=%s", updated.BookingID, payload.FailureReason)
	}

	c.JSON(http.StatusOK, gin.H{"status": "processed"})
}
