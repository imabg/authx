package mail

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

var ErrNoActiveSMTP = errors.New("no active smtp configuration")

//go:generate mockgen -destination=mock/mail_mock.go -package=mock github.com/imabg/authx/internal/mail Mailer

type Mailer interface {
	SendOTP(ctx context.Context, cfg Config, to, code string) error
	SendMagicLink(ctx context.Context, cfg Config, to, link string) error
}

type Sender interface {
	Send(ctx context.Context, cfg Config, msg Message) error
}

type Service struct {
	logger   *zap.Logger
	log      *LogMailer
	sendgrid Sender
	smtp     Sender
}

func NewService(logger *zap.Logger) Mailer {
	return NewServiceWithSenders(logger, NewSendGridSender(nil), NewSMTPSender())
}

func NewServiceWithSenders(logger *zap.Logger, sendgrid, smtp Sender) Mailer {
	if logger == nil {
		logger = zap.L()
	}
	return &Service{
		logger:   logger,
		log:      NewLogMailer(logger),
		sendgrid: sendgrid,
		smtp:     smtp,
	}
}

func (s *Service) SendOTP(ctx context.Context, cfg Config, to, code string) error {
	return s.dispatch(ctx, cfg, otpMessage(cfg, to, code))
}

func (s *Service) SendMagicLink(ctx context.Context, cfg Config, to, link string) error {
	return s.dispatch(ctx, cfg, magicLinkMessage(cfg, to, link))
}

func (s *Service) dispatch(ctx context.Context, cfg Config, msg Message) error {
	err := s.send(ctx, cfg, msg)
	fields := sendLogFields(ctx, cfg, msg)
	if err != nil {
		s.logger.Error("mail: provider error", append(fields, zap.Error(err))...)
		return err
	}
	s.logger.Info("mail: sent", fields...)
	return nil
}

func (s *Service) send(ctx context.Context, cfg Config, msg Message) error {
	switch normalizeProvider(cfg.Provider) {
	case ProviderSendGrid:
		if s.sendgrid == nil {
			return fmt.Errorf("sendgrid sender is not configured")
		}
		return s.sendgrid.Send(ctx, cfg, msg)
	case ProviderSMTP:
		if s.smtp == nil {
			return fmt.Errorf("smtp sender is not configured")
		}
		if strings.TrimSpace(cfg.SMTP.Host) == "" {
			return ErrNoActiveSMTP
		}
		return s.smtp.Send(ctx, cfg, msg)
	default:
		return s.log.Send(ctx, cfg, msg)
	}
}

func normalizeProvider(p Provider) Provider {
	switch Provider(strings.ToLower(strings.TrimSpace(string(p)))) {
	case ProviderSendGrid:
		return ProviderSendGrid
	case ProviderSMTP:
		return ProviderSMTP
	default:
		return ProviderLog
	}
}

func sendLogFields(ctx context.Context, cfg Config, msg Message) []zap.Field {
	provider := normalizeProvider(cfg.Provider)
	fields := []zap.Field{
		zap.String("provider", string(provider)),
		zap.String("recipient_domain", recipientDomain(msg.To)),
	}
	if id := applicationIDFrom(ctx); id != "" {
		fields = append(fields, zap.String("application_id", id))
	}
	if provider == ProviderSMTP {
		if host := strings.TrimSpace(cfg.SMTP.Host); host != "" {
			fields = append(fields, zap.String("smtp_host", host))
		}
	}
	return fields
}

func recipientDomain(to string) string {
	to = strings.ToLower(strings.TrimSpace(to))
	at := strings.LastIndex(to, "@")
	if at <= 0 || at == len(to)-1 {
		return ""
	}
	return to[at+1:]
}

type LogMailer struct {
	logger *zap.Logger
}

func NewLogMailer(logger *zap.Logger) *LogMailer {
	if logger == nil {
		logger = zap.L()
	}
	return &LogMailer{logger: logger}
}

func (m *LogMailer) SendOTP(_ context.Context, _ Config, to, code string) error {
	m.logger.Info("mail: otp", zap.String("to", to), zap.String("code", code))
	return nil
}

func (m *LogMailer) SendMagicLink(_ context.Context, _ Config, to, link string) error {
	m.logger.Info("mail: magic_link", zap.String("to", to), zap.String("link", link))
	return nil
}

func (m *LogMailer) Send(_ context.Context, _ Config, msg Message) error {
	m.logger.Info("mail: send", zap.String("to", msg.To), zap.String("subject", msg.Subject))
	return nil
}
