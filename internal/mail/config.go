package mail

import (
	"fmt"
	"strings"
)

type Provider string

const (
	ProviderLog      Provider = "log"
	ProviderSendGrid Provider = "sendgrid"
	ProviderSMTP     Provider = "smtp"

	// MaskedSecret is returned in API responses in place of API keys and SMTP passwords.
	MaskedSecret = "********"
)

type Config struct {
	Provider  Provider       `json:"provider"`
	FromEmail string         `json:"from_email"`
	FromName  string         `json:"from_name"`
	SendGrid  SendGridConfig `json:"sendgrid"`
	SMTP      SMTPConfig     `json:"smtp"`
}

type SendGridConfig struct {
	APIKey string `json:"api_key,omitempty"`
}

type SMTPConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	TLS        bool   `json:"tls"`
	SkipVerify bool   `json:"skip_verify"`
}

func (c Config) Public() Config {
	out := c
	if out.SendGrid.APIKey != "" {
		out.SendGrid.APIKey = MaskedSecret
	}
	if out.SMTP.Password != "" {
		out.SMTP.Password = MaskedSecret
	}
	return out
}

func SecretUnchanged(v string) bool {
	return v == "" || v == MaskedSecret
}

func (c Config) Validate() error {
	provider := Provider(strings.ToLower(strings.TrimSpace(string(c.Provider))))
	switch provider {
	case "", ProviderLog:
		return nil
	case ProviderSendGrid:
		if strings.TrimSpace(c.FromEmail) == "" {
			return fmt.Errorf("mail.from_email is required for sendgrid")
		}
		if SecretUnchanged(c.SendGrid.APIKey) {
			return fmt.Errorf("mail.sendgrid.api_key is required")
		}
		return nil
	case ProviderSMTP:
		if strings.TrimSpace(c.SMTP.Host) == "" {
			return fmt.Errorf("mail.smtp.host is required")
		}
		if c.SMTP.Port < 1 || c.SMTP.Port > 65535 {
			return fmt.Errorf("mail.smtp.port must be between 1 and 65535")
		}
		if strings.TrimSpace(c.FromEmail) == "" {
			return fmt.Errorf("mail.from_email is required for smtp")
		}
		return nil
	default:
		return fmt.Errorf("mail.provider must be log, sendgrid, or smtp")
	}
}
