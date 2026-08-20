package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/imabg/authx/internal/hasher"
)

type Service struct {
	repo IRepository
}

func NewService(repo IRepository) *Service {
	return &Service{repo: repo}
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
	return CreatedApplication{Application: application, ClientSecret: clientSecret}, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (Application, error) {
	return s.repo.GetByID(ctx, id)
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
	return application, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (Application, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
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
	existing.UpdatedBy = "admin"
	return s.repo.Update(ctx, existing)
}

func randomHex(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
