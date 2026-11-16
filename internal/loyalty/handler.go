package loyalty

import (
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
	// Подписка на подтверждение бронирования: финализация списания баллов и начисление кэшбэка
	h.broker.Subscribe("circus.bookings.events", func(ev common.BaseEvent) {
		if ev.EventType == "booking.confirmed" {
			bookingID, _ := ev.Payload["bookingId"].(string)
			userID, _ := ev.Payload["userId"].(string)
			totalAmount, _ := ev.Payload["totalAmount"].(float64)

			log.Printf("[loyalty-service] Handling booking.confirmed for bookingId=%s, userId=%s", bookingID, userID)

			// 1. Подтверждаем списание замороженных баллов
			if res, err := h.repo.FindReservationByBookingID(bookingID); err == nil && res != nil {
				_, _ = h.repo.ConfirmReservation(res.ReservationID)
				_ = h.broker.Publish("circus.loyalty.events", common.NewEvent("loyalty.points.redeemed", map[string]any{
					"userId":         userID,
					"bookingId":      bookingID,
					"pointsDeducted": res.ReservedPoints,
				}))
			}

			// 2. Начисляем кэшбэк за оплаченные рубли
			if totalAmount > 0 && userID != "" {
				accRes, err := h.repo.AccruePoints(AccruePointsRequest{
					BookingID:       bookingID,
					UserID:          userID,
					PaidMoneyAmount: totalAmount,
				})
				if err == nil {
					_ = h.broker.Publish("circus.loyalty.events", common.NewEvent("loyalty.points.accrued", map[string]any{
						"userId":      userID,
						"bookingId":   bookingID,
						"pointsAdded": accRes.AccruedPoints,
						"newBalance":  accRes.NewBalance,
					}))

					if accRes.TierChanged {
						_ = h.broker.Publish("circus.loyalty.events", common.NewEvent("loyalty.tier.upgraded", map[string]any{
							"userId":  userID,
							"newTier": accRes.CurrentTier,
						}))
					}
					log.Printf("[loyalty-service] Accrued %d points to user %s. New balance: %d",
						accRes.AccruedPoints, userID, accRes.NewBalance)
				}
			}
		}
	})

	// Подписка на запрос возврата: штраф и списание бонусных баллов
	h.broker.Subscribe("circus.refunds.events", func(ev common.BaseEvent) {
		if ev.EventType == "refund.requested" {
			bookingID, _ := ev.Payload["bookingId"].(string)
			userID, _ := ev.Payload["userId"].(string)
			if userID != "" {
				res, err := h.repo.ApplyPenalty(PenaltyRequest{
					BookingID: bookingID,
					UserID:    userID,
				})
				if err == nil {
					_ = h.broker.Publish("circus.loyalty.events", common.NewEvent("loyalty.penalty.applied", map[string]any{
						"userId":         userID,
						"bookingId":      bookingID,
						"pointsDeducted": res.DeductedPoints,
					}))
					log.Printf("[loyalty-service] Applied refund penalty for user %s: deducted %d points", userID, res.DeductedPoints)
				}
			}
		}
	})
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/loyalty")
	{
		api.GET("/me", h.GetMyProfile)
		api.GET("/tiers", h.GetTiers)
		api.GET("/rules", h.GetMarketingRules)
	}

	internal := r.Group("/internal/v1/loyalty")
	{
		internal.POST("/reservations", h.ReservePoints)
		internal.DELETE("/reservations/:id", h.ReleaseReservation)
		internal.POST("/reservations/:id/confirm", h.ConfirmReservation)
		internal.POST("/accrual", h.AccruePoints)
		internal.POST("/penalty", h.ApplyPenalty)
	}
}

func (h *Handler) GetMyProfile(c *gin.Context) {
	userID := c.GetHeader("X-User-Id")
	if userID == "" {
		userID = "45a8b9e1-12c4-4b56-8e12-32049e210a55" // default demo user
	}

	prof, err := h.repo.GetProfile(userID)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Database Error", err.Error(), c.Request.URL.Path)
		return
	}

	c.JSON(http.StatusOK, prof)
}

func (h *Handler) GetTiers(c *gin.Context) {
	tiers, err := h.repo.GetTiers()
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Database Error", err.Error(), c.Request.URL.Path)
		return
	}
	c.JSON(http.StatusOK, tiers)
}

func (h *Handler) GetMarketingRules(c *gin.Context) {
	rules, err := h.repo.GetMarketingRules()
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Database Error", err.Error(), c.Request.URL.Path)
		return
	}
	c.JSON(http.StatusOK, rules)
}

func (h *Handler) ReservePoints(c *gin.Context) {
	var req ReservePointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Validation Error", err.Error(), c.Request.URL.Path)
		return
	}

	res, err := h.repo.ReservePoints(req)
	if err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Reservation Failed", err.Error(), c.Request.URL.Path)
		return
	}

	c.JSON(http.StatusCreated, res)
}

func (h *Handler) ReleaseReservation(c *gin.Context) {
	id := c.Param("id")
	res, err := h.repo.ReleaseReservation(id)
	if err != nil {
		common.RenderProblem(c, http.StatusNotFound, "Reservation Not Found", err.Error(), c.Request.URL.Path)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ConfirmReservation(c *gin.Context) {
	id := c.Param("id")
	res, err := h.repo.ConfirmReservation(id)
	if err != nil {
		common.RenderProblem(c, http.StatusNotFound, "Reservation Not Found", err.Error(), c.Request.URL.Path)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AccruePoints(c *gin.Context) {
	var req AccruePointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Validation Error", err.Error(), c.Request.URL.Path)
		return
	}

	res, err := h.repo.AccruePoints(req)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Accrual Failed", err.Error(), c.Request.URL.Path)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ApplyPenalty(c *gin.Context) {
	var req PenaltyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Validation Error", err.Error(), c.Request.URL.Path)
		return
	}

	res, err := h.repo.ApplyPenalty(req)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Penalty Failed", err.Error(), c.Request.URL.Path)
		return
	}
	c.JSON(http.StatusOK, res)
}
