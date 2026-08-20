package mail

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendGridSenderSendsMail(t *testing.T) {
	var gotAuth, gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/mail/send" {
			t.Errorf("path = %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()

	sender := NewSendGridSender(ts.Client())
	sender.BaseURL = ts.URL
	cfg := Config{
		FromEmail: "noreply@example.com",
		FromName:  "Acme",
		SendGrid:  SendGridConfig{APIKey: "sg-test-key"},
	}
	err := sender.Send(context.Background(), cfg, Message{To: "user@example.com", Subject: "code", Text: "123456"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotAuth != "Bearer sg-test-key" {
		t.Fatalf("authorization = %q", gotAuth)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(gotBody), &payload); err != nil {
		t.Fatalf("payload json: %v", err)
	}
	from := payload["from"].(map[string]any)
	if from["email"] != "noreply@example.com" || from["name"] != "Acme" {
		t.Fatalf("from = %+v", from)
	}
}

func TestSendGridSenderErrorDoesNotIncludeAPIKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errors":[{"message":"invalid key sg-test-key"}]}`))
	}))
	defer ts.Close()

	sender := NewSendGridSender(ts.Client())
	sender.BaseURL = ts.URL
	err := sender.Send(context.Background(), Config{
		FromEmail: "noreply@example.com",
		SendGrid:  SendGridConfig{APIKey: "sg-test-key"},
	}, Message{To: "user@example.com", Subject: "x", Text: "y"})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "sg-test-key") {
		t.Fatalf("error leaked api key: %v", err)
	}
}

func TestSendGridSenderRequiresAPIKey(t *testing.T) {
	sender := NewSendGridSender(nil)
	err := sender.Send(context.Background(), Config{FromEmail: "a@b.com"}, Message{To: "c@d.com"})
	if err == nil || !strings.Contains(err.Error(), "api key") {
		t.Fatalf("error = %v", err)
	}
}
