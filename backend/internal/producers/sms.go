package producers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"planningpoker/internal/config"
	"time"
)

type SMS interface {
	Send(context.Context, string, string) error
}
type DevSMS struct{}

func (DevSMS) Send(context.Context, string, string) error { return nil }

type WebhookSMS struct {
	URL, Token string
	client     *http.Client
}

func NewSMS(c *config.Config) SMS {
	if c.SMSURL == "" {
		return DevSMS{}
	}
	return &WebhookSMS{c.SMSURL, c.SMSToken, &http.Client{Timeout: 8 * time.Second}}
}
func (s *WebhookSMS) Send(ctx context.Context, phone, code string) error {
	b, _ := json.Marshal(map[string]string{"phone": phone, "code": code, "purpose": "login", "message": "کد ورود هم‌تخمین: " + code})
	req, err := http.NewRequestWithContext(ctx, "POST", s.URL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.Token)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("SMS gateway rejected request")
	}
	return nil
}
