package app

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/imabg/authx/internal/mail"
	"github.com/imabg/authx/internal/secret"
	"github.com/imabg/authx/internal/validate"
)

func (s *Service) CreateSMTPConfig(ctx context.Context, applicationID uuid.UUID, input SMTPCreateInput) (SMTPConfig, error) {
	if err := s.requireApplication(ctx, applicationID); err != nil {
		return SMTPConfig{}, err
	}
	if err := s.ensureSMTPRepo(); err != nil {
		return SMTPConfig{}, err
	}
	if err := validate.Struct(input); err != nil {
		return SMTPConfig{}, mapSMTPValidationError(err)
	}
	cfg := SMTPConfig{
		ApplicationID: applicationID,
		Name:          strings.TrimSpace(input.Name),
		Host:          strings.TrimSpace(input.Host),
		Port:          input.Port,
		Username:      strings.TrimSpace(input.Username),
		Password:      input.Password,
		UpdatedBy:     "admin",
	}
	if input.TLS != nil {
		cfg.TLS = *input.TLS
	}
	if input.SkipVerify != nil {
		cfg.SkipVerify = *input.SkipVerify
	}
	if err := decodeAndEncryptSMTP(&cfg, s.secrets); err != nil {
		return SMTPConfig{}, err
	}
	created, err := s.smtp.Create(ctx, cfg)
	if err != nil {
		return SMTPConfig{}, err
	}
	return redactSMTP(created), nil
}

func (s *Service) ListSMTPConfigs(ctx context.Context, applicationID uuid.UUID) ([]SMTPConfig, error) {
	if err := s.requireApplication(ctx, applicationID); err != nil {
		return nil, err
	}
	if err := s.ensureSMTPRepo(); err != nil {
		return nil, err
	}
	cfgs, err := s.smtp.ListByApplication(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	for i := range cfgs {
		cfgs[i] = redactSMTP(cfgs[i])
	}
	return cfgs, nil
}

func (s *Service) GetSMTPConfig(ctx context.Context, applicationID, id uuid.UUID) (SMTPConfig, error) {
	if err := s.requireApplication(ctx, applicationID); err != nil {
		return SMTPConfig{}, err
	}
	if err := s.ensureSMTPRepo(); err != nil {
		return SMTPConfig{}, err
	}
	cfg, err := s.smtp.GetByID(ctx, applicationID, id)
	if err != nil {
		return SMTPConfig{}, err
	}
	return redactSMTP(cfg), nil
}

func (s *Service) UpdateSMTPConfig(ctx context.Context, applicationID, id uuid.UUID, input SMTPUpdateInput) (SMTPConfig, error) {
	if err := s.requireApplication(ctx, applicationID); err != nil {
		return SMTPConfig{}, err
	}
	if err := s.ensureSMTPRepo(); err != nil {
		return SMTPConfig{}, err
	}
	if err := validate.Struct(input); err != nil {
		return SMTPConfig{}, mapSMTPValidationError(err)
	}
	existing, err := s.smtp.GetByID(ctx, applicationID, id)
	if err != nil {
		return SMTPConfig{}, err
	}
	if input.Name != nil {
		existing.Name = strings.TrimSpace(*input.Name)
	}
	if input.Host != nil {
		existing.Host = strings.TrimSpace(*input.Host)
	}
	if input.Port != nil {
		existing.Port = *input.Port
	}
	if input.TLS != nil {
		existing.TLS = *input.TLS
	}
	if input.SkipVerify != nil {
		existing.SkipVerify = *input.SkipVerify
	}
	if input.Username != nil {
		existing.Username = strings.TrimSpace(*input.Username)
	}
	if input.Password != nil && !mail.SecretUnchanged(*input.Password) {
		decoded, err := mail.DecodeSMTPPassword(*input.Password)
		if err != nil {
			return SMTPConfig{}, err
		}
		existing.Password = decoded
	}
	if err := encryptSMTP(existing.Username, existing.Password, s.secrets, &existing.Username, &existing.Password); err != nil {
		return SMTPConfig{}, err
	}
	existing.UpdatedBy = "admin"
	updated, err := s.smtp.Update(ctx, existing)
	if err != nil {
		return SMTPConfig{}, err
	}
	return redactSMTP(updated), nil
}

func (s *Service) ActivateSMTPConfig(ctx context.Context, applicationID, id uuid.UUID) (SMTPConfig, error) {
	if err := s.requireApplication(ctx, applicationID); err != nil {
		return SMTPConfig{}, err
	}
	if err := s.ensureSMTPRepo(); err != nil {
		return SMTPConfig{}, err
	}
	cfg, err := s.smtp.Activate(ctx, applicationID, id)
	if err != nil {
		return SMTPConfig{}, err
	}
	return redactSMTP(cfg), nil
}

func (s *Service) DeleteSMTPConfig(ctx context.Context, applicationID, id uuid.UUID) error {
	if err := s.requireApplication(ctx, applicationID); err != nil {
		return err
	}
	if err := s.ensureSMTPRepo(); err != nil {
		return err
	}
	return s.smtp.Delete(ctx, applicationID, id)
}

func (s *Service) requireApplication(ctx context.Context, applicationID uuid.UUID) error {
	_, err := s.repo.GetByID(ctx, applicationID)
	return err
}

func (s *Service) ensureSMTPRepo() error {
	if s.smtp == nil {
		return ErrSMTPNotConfigured
	}
	return nil
}

func (s *Service) attachActiveSMTP(ctx context.Context, application *Application) error {
	if application == nil || s.smtp == nil {
		return nil
	}
	if mail.Provider(strings.ToLower(strings.TrimSpace(string(application.Settings.Mail.Provider)))) != mail.ProviderSMTP {
		return nil
	}
	cfg, err := s.smtp.GetActive(ctx, application.ID)
	if errors.Is(err, ErrSMTPNotFound) {
		application.Settings.Mail.SMTP = mail.SMTPConfig{}
		return nil
	}
	if err != nil {
		return err
	}
	username, password := cfg.Username, cfg.Password
	if err := decryptSMTP(&username, &password, s.secrets); err != nil {
		return err
	}
	cfg.Username = username
	cfg.Password = password
	application.Settings.Mail.SMTP = cfg.toMailSMTP()
	return nil
}

func redactSMTP(cfg SMTPConfig) SMTPConfig {
	cfg.Username = ""
	cfg.Password = ""
	return cfg
}

func decodeAndEncryptSMTP(cfg *SMTPConfig, box secret.Box) error {
	if cfg.Password != "" {
		decoded, err := mail.DecodeSMTPPassword(cfg.Password)
		if err != nil {
			return err
		}
		cfg.Password = decoded
	}
	return encryptSMTP(cfg.Username, cfg.Password, box, &cfg.Username, &cfg.Password)
}

func encryptSMTP(username, password string, box secret.Box, userOut, passOut *string) error {
	mc := mail.Config{SMTP: mail.SMTPConfig{Username: username, Password: password}}
	if err := mc.EncryptSecrets(box); err != nil {
		return err
	}
	*userOut = mc.SMTP.Username
	*passOut = mc.SMTP.Password
	return nil
}

func decryptSMTP(username, password *string, box secret.Box) error {
	mc := mail.Config{SMTP: mail.SMTPConfig{Username: *username, Password: *password}}
	if err := mc.DecryptSecrets(box); err != nil {
		return err
	}
	*username = mc.SMTP.Username
	*password = mc.SMTP.Password
	return nil
}

func mapSMTPValidationError(err error) error {
	return validate.Map(err, nil, smtpValidationMessage)
}

func smtpValidationMessage(path, tag, param string) (string, bool) {
	switch path {
	case "host":
		if tag == "required" || tag == "min" {
			return "host is required", true
		}
	case "port":
		if tag == "required" || tag == "gte" || tag == "lte" {
			return "port must be between 1 and 65535", true
		}
	}
	return validate.StandardMessages(path, tag, param)
}

func seedSMTPFromSettings(ctx context.Context, smtp ISMTPRepository, applicationID uuid.UUID, mailCfg mail.Config) error {
	if smtp == nil || strings.TrimSpace(mailCfg.SMTP.Host) == "" {
		return nil
	}
	port := mailCfg.SMTP.Port
	if port == 0 {
		port = 587
	}
	_, err := smtp.Create(ctx, SMTPConfig{
		ApplicationID: applicationID,
		Name:          "default",
		Host:          strings.TrimSpace(mailCfg.SMTP.Host),
		Port:          port,
		Username:      mailCfg.SMTP.Username,
		Password:      mailCfg.SMTP.Password,
		TLS:           mailCfg.SMTP.TLS,
		SkipVerify:    mailCfg.SMTP.SkipVerify,
		UpdatedBy:     "admin",
	})
	return err
}

func stripNestedSMTP(settings *Settings) {
	if settings == nil {
		return
	}
	settings.Mail.SMTP = mail.SMTPConfig{}
}
