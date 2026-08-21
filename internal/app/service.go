package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/imabg/authx/internal/hasher"
	"github.com/imabg/authx/internal/mail"
	"github.com/imabg/authx/internal/secret"
)

type Service struct {
	repo    IRepository
	smtp    *SMTPStore
	secrets secret.Box
}

func NewService(repo IRepository) *Service {
	return NewServiceWithDeps(repo, nil, nil)
}

func NewServiceWithSecrets(repo IRepository, secrets secret.Box) *Service {
	return NewServiceWithDeps(repo, nil, secrets)
}

func NewServiceWithDeps(repo IRepository, smtp *SMTPStore, secrets secret.Box) *Service {
	return &Service{repo: repo, smtp: smtp, secrets: secrets}
}

type CreateInput struct {
	Name        string `validate:"required"`
	Description string
	Settings    Settings
}

type CreatedApplication struct {
	Application  Application
	ClientSecret string
}

type UpdateInput struct {
	Name         *string
	Description  *string
	Status       *string
	SettingsJSON json.RawMessage
}

func (s *Service) Create(ctx context.Context, input CreateInput) (CreatedApplication, error) {
	if err := validateCreateInput(input); err != nil {
		return CreatedApplication{}, err
	}
	if input.Settings.AuthMethod == "" {
		input.Settings.AuthMethod = AuthMethodPassword
	}
	if err := prepareSettings(&input.Settings); err != nil {
		return CreatedApplication{}, err
	}
	if err := mail.DecodeIncomingSMTP(&input.Settings.Mail); err != nil {
		return CreatedApplication{}, err
	}
	if err := input.Settings.Mail.EncryptSecrets(s.secrets); err != nil {
		return CreatedApplication{}, err
	}
	nestedSMTP := input.Settings.Mail
	stripNestedSMTP(&input.Settings)
	clientID, err := randomHex(16)
	if err != nil {
		return CreatedApplication{}, err
	}
	clientSecret, err := randomHex(32)
	if err != nil {
		return CreatedApplication{}, err
	}
	secretHash, err := hasher.Hash(clientSecret)
	if err != nil {
		return CreatedApplication{}, err
	}
	application, err := s.repo.Create(ctx, Application{
		Name:             input.Name,
		Description:      input.Description,
		ClientID:         "app_" + clientID,
		ClientSecretHash: secretHash,
		Settings:         input.Settings,
		Status:           StatusActive,
		UpdatedBy:        "admin",
	})
	if err != nil {
		return CreatedApplication{}, err
	}
	if err := s.smtp.SeedFromSettings(ctx, application.ID, nestedSMTP); err != nil {
		return CreatedApplication{}, err
	}
	if err := s.decryptMail(&application); err != nil {
		return CreatedApplication{}, err
	}
	return CreatedApplication{Application: application, ClientSecret: clientSecret}, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (Application, error) {
	application, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Application{}, err
	}
	if err := s.decryptMail(&application); err != nil {
		return Application{}, err
	}
	return application, nil
}

func (s *Service) GetByClientCredentials(ctx context.Context, clientID, clientSecret string) (Application, error) {
	application, err := s.repo.GetByClientID(ctx, clientID)
	if err != nil {
		return Application{}, err
	}
	ok, err := hasher.Compare(clientSecret, application.ClientSecretHash)
	if err != nil || !ok {
		return Application{}, ErrNotFound
	}
	if err := s.decryptMail(&application); err != nil {
		return Application{}, err
	}
	if err := s.smtp.AttachActive(ctx, &application); err != nil {
		return Application{}, err
	}
	return application, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (Application, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return Application{}, err
	}
	if err := s.decryptMail(&existing); err != nil {
		return Application{}, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if err := validateUpdateName(name); err != nil {
			return Application{}, err
		}
		existing.Name = name
	}
	if input.Description != nil {
		existing.Description = strings.TrimSpace(*input.Description)
	}
	if input.Status != nil {
		status := strings.ToLower(strings.TrimSpace(*input.Status))
		if err := validateUpdateStatus(status); err != nil {
			return Application{}, err
		}
		existing.Status = status
	}
	if len(input.SettingsJSON) > 0 {
		merged, err := MergeSettings(existing.Settings, input.SettingsJSON)
		if err != nil {
			return Application{}, err
		}
		existing.Settings = merged
	}
	if err := prepareSettings(&existing.Settings); err != nil {
		return Application{}, err
	}
	if err := existing.Settings.Mail.EncryptSecrets(s.secrets); err != nil {
		return Application{}, err
	}
	stripNestedSMTP(&existing.Settings)
	existing.UpdatedBy = "admin"
	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		return Application{}, err
	}
	if err := s.decryptMail(&updated); err != nil {
		return Application{}, err
	}
	return updated, nil
}

func (s *Service) decryptMail(application *Application) error {
	if application == nil {
		return nil
	}
	return application.Settings.Mail.DecryptSecrets(s.secrets)
}

func randomHex(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
