package mail

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"
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
	if err := svc.SendMagicLink(context.Background(), Config{Provider: ProviderSMTP, SMTP: SMTPConfig{Host: "smtp.example.com"}}, "user@example.com", "https://app.example/auth/callback?token=abc"); err != nil {
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

func TestServiceRequiresActiveSMTP(t *testing.T) {
	svc := NewServiceWithSenders(zap.NewNop(), &stubSender{}, &stubSender{})
	err := svc.SendOTP(context.Background(), Config{Provider: ProviderSMTP}, "user@example.com", "123456")
	if err == nil || !strings.Contains(err.Error(), "no active smtp configuration") {
		t.Fatalf("error = %v", err)
	}
}
