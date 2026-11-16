package catalog

import (
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
	return &Handler{repo: repo, broker: broker}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		api.GET("/shows", h.GetShows)
		api.GET("/shows/:id", h.GetShowByID)
		api.GET("/sessions", h.GetSessions)
		api.GET("/sessions/:id", h.GetSessionByID)
	}

	internal := r.Group("/internal/v1")
	{
		internal.GET("/sessions/:id", h.GetInternalSessionVerification)
		internal.POST("/shows", h.CreateShow)
		internal.DELETE("/shows/:id", h.ArchiveShow)
	}
}

func (h *Handler) GetShows(c *gin.Context) {
	genre := c.Query("genre")
	ageRestriction, _ := strconv.Atoi(c.Query("ageRestriction"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if size <= 0 {
		size = 10
	}

	items, total, err := h.repo.GetShows(genre, ageRestriction, page, size)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Database Error", err.Error(), c.Request.URL.Path)
		return
	}

	totalPages := (total + size - 1) / size
	c.JSON(http.StatusOK, ShowPageResponse{
		Items:         items,
		TotalElements: total,
		TotalPages:    totalPages,
		Page:          page,
		Size:          size,
	})
}

func (h *Handler) GetShowByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Invalid ID", "ID must be an integer", c.Request.URL.Path)
		return
	}

	show, err := h.repo.GetShowByID(id)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Database Error", err.Error(), c.Request.URL.Path)
		return
	}
	if show == nil {
		common.RenderProblem(c, http.StatusNotFound, "Show Not Found", "Цирковое представление не найдено", c.Request.URL.Path)
		return
	}

	c.JSON(http.StatusOK, show)
}

func (h *Handler) GetSessions(c *gin.Context) {
	showID, _ := strconv.Atoi(c.Query("showId"))
	dateFrom := c.Query("dateFrom")
	dateTo := c.Query("dateTo")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if size <= 0 {
		size = 20
	}

	sessions, total, err := h.repo.GetSessions(showID, dateFrom, dateTo, page, size)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Database Error", err.Error(), c.Request.URL.Path)
		return
	}

	totalPages := (total + size - 1) / size
	c.JSON(http.StatusOK, SessionPageResponse{
		Items:         sessions,
		TotalElements: total,
		TotalPages:    totalPages,
		Page:          page,
		Size:          size,
	})
}

func (h *Handler) GetSessionByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Invalid ID", "ID must be an integer", c.Request.URL.Path)
		return
	}

	detail, err := h.repo.GetSessionDetail(id)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Database Error", err.Error(), c.Request.URL.Path)
		return
	}
	if detail == nil {
		common.RenderProblem(c, http.StatusNotFound, "Session Not Found", "Сеанс не найден", c.Request.URL.Path)
		return
	}

	c.JSON(http.StatusOK, detail)
}

func (h *Handler) GetInternalSessionVerification(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Invalid ID", "ID must be an integer", c.Request.URL.Path)
		return
	}

	resp, err := h.repo.GetInternalSessionVerification(id)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Database Error", err.Error(), c.Request.URL.Path)
		return
	}
	if resp == nil {
		common.RenderProblem(c, http.StatusNotFound, "Session Not Found", "Сеанс для бронирования не найден", c.Request.URL.Path)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) CreateShow(c *gin.Context) {
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		common.RenderProblem(c, http.StatusBadRequest, "Missing Header", "Idempotency-Key header is required", c.Request.URL.Path)
		return
	}

	var req CreateShowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Validation Error", err.Error(), c.Request.URL.Path)
		return
	}

	show, err := h.repo.CreateShow(req)
	if err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Create Error", err.Error(), c.Request.URL.Path)
		return
	}

	// Публикация события в брокер
	event := common.NewEvent("catalog.show.created", map[string]any{
		"showId":    show.ID,
		"title":     show.Title,
		"genre":     show.Genre,
		"basePrice": show.BasePrice,
	})
	_ = h.broker.Publish("circus.catalog.events", event)

	c.JSON(http.StatusCreated, show)
}

func (h *Handler) ArchiveShow(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.RenderProblem(c, http.StatusBadRequest, "Invalid ID", "ID must be an integer", c.Request.URL.Path)
		return
	}

	if err := h.repo.ArchiveShow(id); err != nil {
		common.RenderProblem(c, http.StatusInternalServerError, "Archive Error", err.Error(), c.Request.URL.Path)
		return
	}

	event := common.NewEvent("catalog.show.archived", map[string]any{
		"showId": id,
	})
	_ = h.broker.Publish("circus.catalog.events", event)

	c.Status(http.StatusNoContent)
}
