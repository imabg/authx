package mail

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type stubSender struct {
	called bool
	cfg    Config
	msg    Message
	err    error
}

func (s *stubSender) Send(_ context.Context, cfg Config, msg Message) error {
	s.called = true
	s.cfg = cfg
	s.msg = msg
	return s.err
}

func TestServiceDispatchesByProvider(t *testing.T) {
	sg := &stubSender{}
	smtp := &stubSender{}
	svc := NewServiceWithSenders(zap.NewNop(), sg, smtp)

	if err := svc.SendOTP(context.Background(), Config{Provider: ProviderSendGrid, FromName: "Acme"}, "user@example.com", "123456"); err != nil {
		t.Fatalf("sendgrid: %v", err)
	}
	if !sg.called || smtp.called {
		t.Fatalf("sendgrid called=%v smtp called=%v", sg.called, smtp.called)
	}
	if sg.cfg.Provider != ProviderSendGrid || sg.msg.To != "user@example.com" || !strings.Contains(sg.msg.Text, "123456") {
		t.Fatalf("sendgrid msg = %+v cfg = %+v", sg.msg, sg.cfg)
	}

	sg.called = false
	if err := svc.SendMagicLink(context.Background(), Config{Provider: ProviderSMTP}, "user@example.com", "https://app.example/auth/callback?token=abc"); err != nil {
		t.Fatalf("smtp: %v", err)
	}
	if sg.called || !smtp.called {
		t.Fatalf("sendgrid called=%v smtp called=%v", sg.called, smtp.called)
	}
	if !strings.Contains(smtp.msg.Text, "https://app.example/auth/callback?token=abc") {
		t.Fatalf("magic link body = %q", smtp.msg.Text)
	}

	sg.called = false
	smtp.called = false
	if err := svc.SendOTP(context.Background(), Config{Provider: ProviderLog}, "user@example.com", "999999"); err != nil {
		t.Fatalf("log: %v", err)
	}
	if sg.called || smtp.called {
		t.Fatal("log provider should not call sendgrid or smtp")
	}
}

func TestLogMailerImplementsMailer(t *testing.T) {
	m := NewLogMailer(zap.NewNop())
	if err := m.SendOTP(context.Background(), Config{}, "user@example.com", "123"); err != nil {
		t.Fatalf("SendOTP: %v", err)
	}
	if err := m.SendMagicLink(context.Background(), Config{}, "user@example.com", "https://example/link"); err != nil {
		t.Fatalf("SendMagicLink: %v", err)
	}
}

func TestServiceLogsSuccessfulSend(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	sg := &stubSender{}
	smtp := &stubSender{}
	svc := NewServiceWithSenders(zap.New(core), sg, smtp)

	ctx := WithApplicationID(context.Background(), "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	cfg := Config{
		Provider: ProviderSMTP,
		SMTP:     SMTPConfig{Host: "smtp.example.com", Username: "smtp-user", Password: "smtp-password"},
	}
	if err := svc.SendOTP(ctx, cfg, "User@Example.COM", "123456"); err != nil {
		t.Fatalf("SendOTP: %v", err)
	}

	sent := logs.FilterMessage("mail: sent").All()
	if len(sent) != 1 {
		t.Fatalf("sent logs = %d, want 1", len(sent))
	}
	fields := sent[0].ContextMap()
	if fields["provider"] != "smtp" {
		t.Fatalf("provider = %v", fields["provider"])
	}
	if fields["recipient_domain"] != "example.com" {
		t.Fatalf("recipient_domain = %v", fields["recipient_domain"])
	}
	if fields["application_id"] != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("application_id = %v", fields["application_id"])
	}
	if fields["smtp_host"] != "smtp.example.com" {
		t.Fatalf("smtp_host = %v", fields["smtp_host"])
	}
	assertLogOmitsSecrets(t, fields, "smtp-user", "smtp-password", "123456", "user@example.com")
}

func TestServiceLogsProviderError(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	providerErr := errors.New("535 authentication failed")
	svc := NewServiceWithSenders(zap.New(core), &stubSender{}, &stubSender{err: providerErr})

	ctx := WithApplicationID(context.Background(), "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	cfg := Config{
		Provider: ProviderSMTP,
		SMTP:     SMTPConfig{Host: "smtp.example.com", Username: "smtp-user", Password: "smtp-password"},
	}
	err := svc.SendMagicLink(ctx, cfg, "other@acme.test", "https://app.example/auth/callback?token=abc")
	if !errors.Is(err, providerErr) {
		t.Fatalf("error = %v", err)
	}

	logged := logs.FilterMessage("mail: provider error").All()
	if len(logged) != 1 {
		t.Fatalf("error logs = %d, want 1", len(logged))
	}
	if logged[0].Level != zap.ErrorLevel {
		t.Fatalf("level = %s", logged[0].Level)
	}
	fields := logged[0].ContextMap()
	if fields["provider"] != "smtp" || fields["recipient_domain"] != "acme.test" {
		t.Fatalf("fields = %+v", fields)
	}
	if fields["application_id"] != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("application_id = %v", fields["application_id"])
	}
	if !strings.Contains(fmtErr(fields["error"]), "535 authentication failed") {
		t.Fatalf("error field = %v", fields["error"])
	}
	assertLogOmitsSecrets(t, fields, "smtp-user", "smtp-password")
}

func TestRecipientDomain(t *testing.T) {
	tests := []struct {
		to   string
		want string
	}{
		{to: "User@Example.COM", want: "example.com"},
		{to: " not-an-email ", want: ""},
		{to: "", want: ""},
	}
	for _, tt := range tests {
		if got := recipientDomain(tt.to); got != tt.want {
			t.Fatalf("recipientDomain(%q) = %q, want %q", tt.to, got, tt.want)
		}
	}
}

func assertLogOmitsSecrets(t *testing.T, fields map[string]any, secrets ...string) {
	t.Helper()
	raw := fmt.Sprintf("%v", fields)
	for _, secret := range secrets {
		if strings.Contains(raw, secret) {
			t.Fatalf("log leaked %q: %s", secret, raw)
		}
	}
}

func fmtErr(v any) string {
	if err, ok := v.(error); ok {
		return err.Error()
	}
	return fmt.Sprintf("%v", v)
}
