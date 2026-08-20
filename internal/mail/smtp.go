package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"strings"
)

type SMTPSender struct {
	DialContext func(ctx context.Context, network, address string) (net.Conn, error)
}

func NewSMTPSender() *SMTPSender {
	return &SMTPSender{}
}

func (s *SMTPSender) Send(ctx context.Context, cfg Config, msg Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	host := strings.TrimSpace(cfg.SMTP.Host)
	if host == "" {
		return fmt.Errorf("smtp host is not configured")
	}
	port := cfg.SMTP.Port
	if port == 0 {
		port = 587
	}
	from := strings.TrimSpace(cfg.FromEmail)
	if from == "" {
		return fmt.Errorf("smtp from_email is not configured")
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	raw, err := s.dial(ctx, addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}

	tlsCfg := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: cfg.SMTP.SkipVerify, //nolint:gosec // opt-in for self-signed lab servers
		MinVersion:         tls.VersionTLS12,
	}

	var conn net.Conn = raw
	implicitTLS := cfg.SMTP.TLS && port == 465
	if implicitTLS {
		tlsConn := tls.Client(raw, tlsCfg)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = raw.Close()
			return fmt.Errorf("smtp tls handshake: %w", err)
		}
		conn = tlsConn
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if cfg.SMTP.TLS && !implicitTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsCfg); err != nil {
				return fmt.Errorf("smtp starttls: %w", err)
			}
		} else {
			return fmt.Errorf("smtp server does not support STARTTLS")
		}
	}

	if cfg.SMTP.Username != "" {
		auth := smtp.PlainAuth("", cfg.SMTP.Username, cfg.SMTP.Password, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := io.WriteString(wc, formatSMTPMessage(cfg, msg)); err != nil {
		_ = wc.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return client.Quit()
}

func (s *SMTPSender) dial(ctx context.Context, addr string) (net.Conn, error) {
	if s != nil && s.DialContext != nil {
		return s.DialContext(ctx, "tcp", addr)
	}
	var d net.Dialer
	return d.DialContext(ctx, "tcp", addr)
}

func formatSMTPMessage(cfg Config, msg Message) string {
	from := strings.TrimSpace(cfg.FromEmail)
	fromHeader := from
	if name := strings.TrimSpace(cfg.FromName); name != "" {
		fromHeader = fmt.Sprintf("%s <%s>", name, from)
	}
	var b strings.Builder
	b.WriteString("From: ")
	b.WriteString(fromHeader)
	b.WriteString("\r\nTo: ")
	b.WriteString(msg.To)
	b.WriteString("\r\nSubject: ")
	b.WriteString(msg.Subject)
	b.WriteString("\r\nMIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(msg.Text, "\n", "\r\n"))
	if !strings.HasSuffix(msg.Text, "\n") {
		b.WriteString("\r\n")
	}
	return b.String()
}
