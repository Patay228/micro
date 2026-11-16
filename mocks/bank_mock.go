package mocks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BankMock имитирует банковский эквайринг (платёжный шлюз банка).
type BankMock struct {
	paymentServiceURL string
	httpClient        *http.Client
}

func NewBankMock(paymentServiceURL string) *BankMock {
	return &BankMock{
		paymentServiceURL: paymentServiceURL,
		httpClient:        &http.Client{Timeout: 3 * time.Second},
	}
}

// SimulateSuccessfulPayment отправляет вебхук успешной оплаты в payment-service.
func (b *BankMock) SimulateSuccessfulPayment(internalPaymentID string, amount float64) error {
	payload := map[string]any{
		"externalTransactionId": fmt.Sprintf("BANK_TX_%d", time.Now().UnixNano()),
		"internalPaymentId":     internalPaymentID,
		"status":                "SUCCESS",
		"amount":                amount,
	}
	bodyBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, b.paymentServiceURL+"/api/v1/payments/webhooks/acquiring", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", "mock_hmac_sha256_signature")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bank webhook failed with status: %d", resp.StatusCode)
	}
	return nil
}

// SimulateFailedPayment отправляет вебхук отказа в оплате (недостаток средств).
func (b *BankMock) SimulateFailedPayment(internalPaymentID string, reason string) error {
	payload := map[string]any{
		"externalTransactionId": fmt.Sprintf("BANK_TX_FAIL_%d", time.Now().UnixNano()),
		"internalPaymentId":     internalPaymentID,
		"status":                "FAIL",
		"amount":                0,
		"failureReason":         reason,
	}
	bodyBytes, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPost, b.paymentServiceURL+"/api/v1/payments/webhooks/acquiring", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", "mock_hmac_sha256_signature")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
