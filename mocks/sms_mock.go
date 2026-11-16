package mocks

import (
	"log"
	"sync"
)

// SMSMock эмулирует внешний SMS и Email шлюз связи.
type SMSMock struct {
	mu       sync.Mutex
	messages []string
}

func NewSMSMock() *SMSMock {
	return &SMSMock{messages: make([]string, 0)}
}

func (s *SMSMock) SendSMS(phone, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Printf("[SMSMock] [TRANSPORT] Delivered SMS to %s: %s", phone, text)
	s.messages = append(s.messages, phone+": "+text)
}

func (s *SMSMock) SendEmail(email, subject, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Printf("[SMSMock] [TRANSPORT] Delivered Email to %s [%s]: %s", email, subject, body)
	s.messages = append(s.messages, email+" ("+subject+"): "+body)
}

func (s *SMSMock) GetDeliveredCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messages)
}
