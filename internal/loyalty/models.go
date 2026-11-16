package loyalty

type LoyaltyProfileResponse struct {
	UserID            string `json:"userId"`
	ActiveBalance     int    `json:"activeBalance"`
	ReservedBalance   int    `json:"reservedBalance"`
	CurrentTier       string `json:"currentTier"`
	CashbackPercent   int    `json:"cashbackPercent"`
	ShowsAttended     int    `json:"showsAttended"`
	NextTierThreshold int    `json:"nextTierThreshold"`
}

type LoyaltyTier struct {
	TierCode        string `json:"tierCode"`
	TierName        string `json:"tierName"`
	MinShows        int    `json:"minShows"`
	CashbackPercent int    `json:"cashbackPercent"`
	Description     string `json:"description"`
}

type ReservePointsRequest struct {
	BookingID       string `json:"bookingId" binding:"required"`
	UserID          string `json:"userId" binding:"required"`
	PointsToReserve int    `json:"pointsToReserve" binding:"required,gt=0"`
}

type PointsReservationResponse struct {
	ReservationID  string `json:"reservationId"`
	BookingID      string `json:"bookingId"`
	UserID         string `json:"userId"`
	ReservedPoints int    `json:"reservedPoints"`
	Status         string `json:"status"` // RESERVED, CONFIRMED, RELEASED
}

type AccruePointsRequest struct {
	BookingID       string  `json:"bookingId" binding:"required"`
	UserID          string  `json:"userId" binding:"required"`
	PaidMoneyAmount float64 `json:"paidMoneyAmount" binding:"required,gt=0"`
}

type AccrualResultResponse struct {
	AccruedPoints int    `json:"accruedPoints"`
	NewBalance    int    `json:"newBalance"`
	TierChanged   bool   `json:"tierChanged"`
	CurrentTier   string `json:"currentTier"`
}

type PenaltyRequest struct {
	BookingID string `json:"bookingId" binding:"required"`
	UserID    string `json:"userId" binding:"required"`
}

type PenaltyResponse struct {
	DeductedPoints int  `json:"deductedPoints"`
	UpdatedBalance int  `json:"updatedBalance"`
	PenaltyApplied bool `json:"penaltyApplied"`
}

type MarketingRule struct {
	ID                   int    `json:"id"`
	Name                 string `json:"name"`
	TargetShowGenre      string `json:"targetShowGenre,omitempty"`
	ExtraCashbackPercent int    `json:"extraCashbackPercent"`
	IsActive             bool   `json:"isActive"`
}
