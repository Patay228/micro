package catalog

type Show struct {
	ID              int     `json:"id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Genre           string  `json:"genre"`
	DurationMinutes int     `json:"durationMinutes"`
	AgeRestriction  int     `json:"ageRestriction"`
	BasePrice       float64 `json:"basePrice"`
	PosterURL       string  `json:"posterUrl"`
	Status          string  `json:"status"` // ACTIVE, ARCHIVED
}

type Session struct {
	ID        int    `json:"id"`
	ShowID    int    `json:"showId"`
	ShowTitle string `json:"showTitle"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	HallName  string `json:"hallName"`
	Status    string `json:"status"` // SCHEDULED, CANCELLED, COMPLETED
}

type PricingSector struct {
	SectorCode string  `json:"sectorCode"`
	SectorName string  `json:"sectorName"`
	Price      float64 `json:"price"`
}

type SessionDetail struct {
	Session        Session         `json:"session"`
	PricingSectors []PricingSector `json:"pricingSectors"`
}

type InternalSessionVerificationResponse struct {
	SessionID             int                `json:"sessionId"`
	ShowID                int                `json:"showId"`
	IsAvailableForBooking bool               `json:"isAvailableForBooking"`
	StartTime             string             `json:"startTime"`
	SectorPrices          map[string]float64 `json:"sectorPrices"`
}

type ShowPageResponse struct {
	Items         []Show `json:"items"`
	TotalElements int    `json:"totalElements"`
	TotalPages    int    `json:"totalPages"`
	Page          int    `json:"page"`
	Size          int    `json:"size"`
}

type SessionPageResponse struct {
	Items         []Session `json:"items"`
	TotalElements int       `json:"totalElements"`
	TotalPages    int       `json:"totalPages"`
	Page          int       `json:"page"`
	Size          int       `json:"size"`
}

type CreateShowRequest struct {
	Title           string  `json:"title" binding:"required"`
	Description     string  `json:"description"`
	Genre           string  `json:"genre" binding:"required"`
	DurationMinutes int     `json:"durationMinutes" binding:"required"`
	AgeRestriction  int     `json:"ageRestriction"`
	BasePrice       float64 `json:"basePrice" binding:"required"`
	PosterURL       string  `json:"posterUrl"`
}
