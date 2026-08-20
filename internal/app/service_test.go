package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/imabg/authx/internal/app"
	"github.com/imabg/authx/internal/app/mock"
	"github.com/imabg/authx/internal/hasher"
	"go.uber.org/mock/gomock"
)

func TestServiceCreate(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		input   app.CreateInput
		setup   func(*mock.MockAppRepository)
		wantErr string
	}{
		{
			name:    "requires name",
			input:   app.CreateInput{Settings: app.DefaultSettings()},
			setup:   func(*mock.MockAppRepository) {},
			wantErr: "name is required",
		},
		{
			name: "rejects invalid auth method",
			input: app.CreateInput{
				Name:     "demo",
				Settings: app.Settings{AuthMethod: "saml"},
			},
			setup:   func(*mock.MockAppRepository) {},
			wantErr: "auth_method must be password, otp, or magic_link",
		},
		{
			name: "creates application",
			input: app.CreateInput{
				Name:     "demo",
				Settings: app.DefaultSettings(),
			},
			setup: func(repo *mock.MockAppRepository) {
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, application app.Application) (app.Application, error) {
						application.ID = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
						return application, nil
					},
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mock.NewMockAppRepository(ctrl)
			tt.setup(repo)
			svc := app.NewService(repo)
			got, err := svc.Create(ctx, tt.input)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %s", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			if got.Application.Name != tt.input.Name || got.ClientSecret == "" {
				t.Fatalf("created = %+v", got)
			}
		})
	}
}

func TestServiceGetByClientCredentials(t *testing.T) {
	ctx := context.Background()
	secret := "s3cret"
	hash, err := hasher.Hash(secret)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	stored := app.Application{
		ID:               uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		ClientID:         "app_demo",
		ClientSecretHash: hash,
		Status:           app.StatusActive,
	}

	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		setup        func(*mock.MockAppRepository)
		wantErr      error
	}{
		{
			name:         "valid credentials",
			clientID:     "app_demo",
			clientSecret: secret,
			setup: func(repo *mock.MockAppRepository) {
				repo.EXPECT().GetByClientID(gomock.Any(), "app_demo").Return(stored, nil)
			},
		},
		{
			name:         "wrong secret",
			clientID:     "app_demo",
			clientSecret: "nope",
			setup: func(repo *mock.MockAppRepository) {
				repo.EXPECT().GetByClientID(gomock.Any(), "app_demo").Return(stored, nil)
			},
			wantErr: app.ErrNotFound,
		},
		{
			name:         "unknown client",
			clientID:     "missing",
			clientSecret: secret,
			setup: func(repo *mock.MockAppRepository) {
				repo.EXPECT().GetByClientID(gomock.Any(), "missing").Return(app.Application{}, app.ErrNotFound)
			},
			wantErr: app.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mock.NewMockAppRepository(ctrl)
			tt.setup(repo)
			svc := app.NewService(repo)
			got, err := svc.GetByClientCredentials(ctx, tt.clientID, tt.clientSecret)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetByClientCredentials: %v", err)
			}
			if got.ID != stored.ID {
				t.Fatalf("id = %s, want %s", got.ID, stored.ID)
			}
		})
	}
}

func TestServiceUpdate(t *testing.T) {
	ctx := context.Background()
	id := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	stored := app.Application{
		ID:     id,
		Name:   "demo",
		Status: app.StatusActive,
		Settings: func() app.Settings {
			s := app.DefaultSettings()
			s.Mail.Provider = "sendgrid"
			s.Mail.FromEmail = "noreply@example.com"
			s.Mail.SendGrid.APIKey = "sg-old"
			return s
		}(),
	}

	t.Run("updates otp and sendgrid key", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mock.NewMockAppRepository(ctrl)
		repo.EXPECT().GetByID(gomock.Any(), id).Return(stored, nil)
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, application app.Application) (app.Application, error) {
				if application.Settings.OTP.Length != 8 || application.Settings.OTP.TTLSeconds != 120 {
					t.Fatalf("otp = %+v", application.Settings.OTP)
				}
				if application.Settings.Mail.SendGrid.APIKey != "sg-new" {
					t.Fatalf("api key = %q", application.Settings.Mail.SendGrid.APIKey)
				}
				return application, nil
			},
		)
		svc := app.NewService(repo)
		name := "demo-2"
		got, err := svc.Update(ctx, id, app.UpdateInput{
			Name:         &name,
			SettingsJSON: []byte(`{"otp":{"length":8,"ttl_seconds":120},"mail":{"provider":"sendgrid","from_email":"noreply@example.com","sendgrid":{"api_key":"sg-new"}}}`),
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if got.Name != "demo-2" {
			t.Fatalf("name = %s", got.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mock.NewMockAppRepository(ctrl)
		repo.EXPECT().GetByID(gomock.Any(), id).Return(app.Application{}, app.ErrNotFound)
		svc := app.NewService(repo)
		_, err := svc.Update(ctx, id, app.UpdateInput{})
		if !errors.Is(err, app.ErrNotFound) {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mock.NewMockAppRepository(ctrl)
		repo.EXPECT().GetByID(gomock.Any(), id).Return(stored, nil)
		svc := app.NewService(repo)
		empty := "  "
		_, err := svc.Update(ctx, id, app.UpdateInput{Name: &empty})
		if err == nil || err.Error() != "name is required" {
			t.Fatalf("error = %v", err)
		}
	})
}
