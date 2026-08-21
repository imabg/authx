package app

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/imabg/authx/internal/mail"
)

var (
	ErrSMTPNotFound      = fmt.Errorf("smtp configuration not found")
	ErrSMTPNotConfigured = fmt.Errorf("smtp repository is not configured")
)

// SMTPConfig is an application-scoped SMTP transport. Username and Password
// are plaintext in memory after decrypt, and ciphertext in the database.
type SMTPConfig struct {
	ID                uuid.UUID
	ApplicationID     uuid.UUID
	Name              string
	Host              string
	Port              int
	Username          string
	Password          string
	TLS               bool
	SkipVerify        bool
	Active            bool
	UpdatedBy         string
	CreationTimestamp time.Time
	UpdatedTimestamp  time.Time
}

func (c SMTPConfig) toMailSMTP() mail.SMTPConfig {
	return mail.SMTPConfig{
		Host:       c.Host,
		Port:       c.Port,
		Username:   c.Username,
		Password:   c.Password,
		TLS:        c.TLS,
		SkipVerify: c.SkipVerify,
	}
}

func (c SMTPConfig) Public() SMTPConfigPublic {
	return SMTPConfigPublic{
		ID:         c.ID.String(),
		Name:       c.Name,
		Host:       c.Host,
		Port:       c.Port,
		TLS:        c.TLS,
		SkipVerify: c.SkipVerify,
		Active:     c.Active,
	}
}

// SMTPConfigPublic is the API representation. Username and password are never included.
type SMTPConfigPublic struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	TLS        bool   `json:"tls"`
	SkipVerify bool   `json:"skip_verify"`
	Active     bool   `json:"active"`
}

type SMTPCreateInput struct {
	Name       string `json:"name"`
	Host       string `json:"host" validate:"required"`
	Port       int    `json:"port" validate:"required,gte=1,lte=65535"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	TLS        *bool  `json:"tls"`
	SkipVerify *bool  `json:"skip_verify"`
}

type SMTPUpdateInput struct {
	Name       *string `json:"name"`
	Host       *string `json:"host" validate:"omitempty,min=1"`
	Port       *int    `json:"port" validate:"omitempty,gte=1,lte=65535"`
	Username   *string `json:"username"`
	Password   *string `json:"password"`
	TLS        *bool   `json:"tls"`
	SkipVerify *bool   `json:"skip_verify"`
}
