package payment

type Payment struct {
	ID             string  `json:"id"`
	BookingID      string  `json:"bookingId"`
	UserID         string  `json:"userId"`
	Amount         float64 `json:"amount"`
	Currency       string  `json:"currency"`
	Status         string  `json:"status"` // PENDING, COMPLETED, FAILED, REFUNDED
	PaymentURL     string  `json:"paymentUrl"`
	ReceiptURL     string  `json:"receiptUrl,omitempty"`
	IdempotencyKey string  `json:"idempotencyKey,omitempty"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

type CreatePaymentRequest struct {
	BookingID   string  `json:"bookingId" binding:"required"`
	UserID      string  `json:"userId" binding:"required"`
	Amount      float64 `json:"amount" binding:"required"`
	Currency    string  `json:"currency"`
	Description string  `json:"description"`
}

type ProcessRefundRequest struct {
	RefundAmount float64 `json:"refundAmount" binding:"required"`
	Reason       string  `json:"reason"`
}

type RefundResultResponse struct {
	RefundID       string  `json:"refundId"`
	PaymentID      string  `json:"paymentId"`
	RefundedAmount float64 `json:"refundedAmount"`
	Status         string  `json:"status"`
	ProcessedAt    string  `json:"processedAt"`
}

type AcquiringWebhookPayload struct {
	ExternalTransactionID string  `json:"externalTransactionId" binding:"required"`
	InternalPaymentID     string  `json:"internalPaymentId" binding:"required"`
	Status                string  `json:"status" binding:"required"` // SUCCESS, FAIL
	Amount                float64 `json:"amount"`
	FailureReason         string  `json:"failureReason,omitempty"`
}
