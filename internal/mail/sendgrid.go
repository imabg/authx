package mail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const sendGridDefaultBaseURL = "https://api.sendgrid.com"

type SendGridSender struct {
	HTTPClient *http.Client
	BaseURL    string
}

func NewSendGridSender(client *http.Client) Sender {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &SendGridSender{HTTPClient: client, BaseURL: sendGridDefaultBaseURL}
}

func (s *SendGridSender) Send(ctx context.Context, cfg Config, msg Message) error {
	if strings.TrimSpace(cfg.SendGrid.APIKey) == "" {
		return fmt.Errorf("sendgrid api key is not configured")
	}
	fromEmail := strings.TrimSpace(cfg.FromEmail)
	if fromEmail == "" {
		return fmt.Errorf("sendgrid from_email is not configured")
	}
	payload := sendGridMail{
		Personalizations: []sendGridPersonalization{{
			To: []sendGridEmail{{Email: msg.To}},
		}},
		From: sendGridEmail{
			Email: fromEmail,
			Name:  cfg.FromName,
		},
		Subject: msg.Subject,
		Content: []sendGridContent{{
			Type:  "text/plain",
			Value: msg.Text,
		}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode sendgrid payload: %w", err)
	}
	base := s.BaseURL
	if base == "" {
		base = sendGridDefaultBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.SendGrid.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("sendgrid request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 2048))
	return fmt.Errorf("sendgrid: status %d", resp.StatusCode)
}

type sendGridMail struct {
	Personalizations []sendGridPersonalization `json:"personalizations"`
	From             sendGridEmail             `json:"from"`
	Subject          string                    `json:"subject"`
	Content          []sendGridContent         `json:"content"`
}

type sendGridPersonalization struct {
	To []sendGridEmail `json:"to"`
}

type sendGridEmail struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type sendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
