package mail

import (
	"strings"

	"github.com/imabg/authx/internal/validate"
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
	Provider  Provider       `json:"provider" validate:"omitempty,oneof=log sendgrid smtp"`
	FromEmail string         `json:"from_email" validate:"sendgrid_from,smtp_from"`
	FromName  string         `json:"from_name"`
	SendGrid  SendGridConfig `json:"sendgrid"`
	SMTP      SMTPConfig     `json:"smtp,omitzero"`
}

type SendGridConfig struct {
	APIKey string `json:"api_key,omitempty" validate:"sendgrid_key"`
}

type SMTPConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	TLS        bool   `json:"tls"`
	SkipVerify bool   `json:"skip_verify"`
}

func (c Config) Public() Config {
	out := c
	if out.SendGrid.APIKey != "" {
		out.SendGrid.APIKey = MaskedSecret
	}
	// SMTP username and password are write-only; never return them (masked or not).
	out.SMTP.Username = ""
	out.SMTP.Password = ""
	return out
}

func SecretUnchanged(v string) bool {
	return v == "" || v == MaskedSecret
}

func (c Config) Validate() error {
	if err := validate.Struct(c); err != nil {
		return mapMailValidationError(err)
	}
	return nil
}

func mapMailValidationError(err error) error {
	return validate.Map(err, nil, mailValidationMessage)
}

func mailValidationMessage(path, tag, param string) (string, bool) {
	switch path {
	case "provider":
		if tag == "oneof" {
			return "mail.provider must be log, sendgrid, or smtp", true
		}
	case "from_email":
		switch tag {
		case "sendgrid_from":
			return "mail.from_email is required for sendgrid", true
		case "smtp_from":
			return "mail.from_email is required for smtp", true
		}
	case "sendgrid.api_key":
		if tag == "sendgrid_key" {
			return "mail.sendgrid.api_key is required", true
		}
	case "smtp.host":
		if tag == "smtp_host" {
			return "mail.smtp.host is required", true
		}
	case "smtp.port":
		if tag == "smtp_port" {
			return "mail.smtp.port must be between 1 and 65535", true
		}
	}
	full := path
	if !strings.HasPrefix(path, "mail.") {
		full = "mail." + path
	}
	return validate.StandardMessages(full, tag, param)
}
