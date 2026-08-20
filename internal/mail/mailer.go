package mail

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

//go:generate mockgen -destination=mock/mail_mock.go -package=mock github.com/imabg/authx/internal/mail Mailer

type Mailer interface {
	SendOTP(ctx context.Context, cfg Config, to, code string) error
	SendMagicLink(ctx context.Context, cfg Config, to, link string) error
}

type Sender interface {
	Send(ctx context.Context, cfg Config, msg Message) error
}

type Service struct {
	log      *LogMailer
	sendgrid Sender
	smtp     Sender
}

func NewService(logger *zap.Logger) *Service {
	return &Service{
		log:      NewLogMailer(logger),
		sendgrid: NewSendGridSender(nil),
		smtp:     NewSMTPSender(),
	}
}

func NewServiceWithSenders(logger *zap.Logger, sendgrid, smtp Sender) *Service {
	return &Service{
		log:      NewLogMailer(logger),
		sendgrid: sendgrid,
		smtp:     smtp,
	}
}

var _ Mailer = (*Service)(nil)

func (s *Service) SendOTP(ctx context.Context, cfg Config, to, code string) error {
	return s.dispatch(ctx, cfg, otpMessage(cfg, to, code))
}

func (s *Service) SendMagicLink(ctx context.Context, cfg Config, to, link string) error {
	return s.dispatch(ctx, cfg, magicLinkMessage(cfg, to, link))
}

func (s *Service) dispatch(ctx context.Context, cfg Config, msg Message) error {
	switch Provider(strings.ToLower(strings.TrimSpace(string(cfg.Provider)))) {
	case ProviderSendGrid:
		if s.sendgrid == nil {
			return fmt.Errorf("sendgrid sender is not configured")
		}
		return s.sendgrid.Send(ctx, cfg, msg)
	case ProviderSMTP:
		if s.smtp == nil {
			return fmt.Errorf("smtp sender is not configured")
		}
		return s.smtp.Send(ctx, cfg, msg)
	default:
		return s.log.Send(ctx, cfg, msg)
	}
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

var _ Mailer = (*LogMailer)(nil)
var _ Sender = (*LogMailer)(nil)

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
