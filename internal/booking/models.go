package booking

type SeatIdentifier struct {
	SectorCode string `json:"sectorCode" binding:"required"`
	Row        int    `json:"row" binding:"required,gt=0"`
	SeatNumber int    `json:"seatNumber" binding:"required,gt=0"`
}

type Ticket struct {
	TicketNumber string  `json:"ticketNumber"`
	SectorCode   string  `json:"sectorCode"`
	Row          int     `json:"row"`
	SeatNumber   int     `json:"seatNumber"`
	Price        float64 `json:"price"`
	QRCodeURL    string  `json:"qrCodeUrl"`
}

type BookingResponse struct {
	ID             string           `json:"id"`
	SessionID      int              `json:"sessionId"`
	UserID         string           `json:"userId"`
	Status         string           `json:"status"` // PENDING_PAYMENT, CONFIRMED, CANCELLED, EXPIRED, REFUNDED
	TotalPrice     float64          `json:"totalPrice"`
	PointsDiscount float64          `json:"pointsDiscount"`
	PayableAmount  float64          `json:"payableAmount"`
	PaymentURL     string           `json:"paymentUrl,omitempty"`
	CreatedAt      string           `json:"createdAt"`
	ExpiresAt      string           `json:"expiresAt"`
	Seats          []SeatIdentifier `json:"seats"`
	Tickets        []Ticket         `json:"tickets,omitempty"`
}

type CreateBookingRequest struct {
	SessionID             int              `json:"sessionId" binding:"required"`
	Seats                 []SeatIdentifier `json:"seats" binding:"required,min=1"`
	LoyaltyPointsToRedeem int              `json:"loyaltyPointsToRedeem"`
}

type RefundRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type RefundResponse struct {
	BookingID      string  `json:"bookingId"`
	RefundStatus   string  `json:"refundStatus"` // REQUESTED, PROCESSED, REJECTED
	RefundAmount   float64 `json:"refundAmount"`
	PenaltyPercent int     `json:"penaltyPercent"`
}

type SessionSeatsStatusResponse struct {
	SessionID     int              `json:"sessionId"`
	OccupiedSeats []SeatIdentifier `json:"occupiedSeats"`
	HeldSeats     []SeatIdentifier `json:"heldSeats"`
}
