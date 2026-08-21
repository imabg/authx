package app_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/imabg/authx/internal/app"
	"github.com/imabg/authx/internal/app/mock"
	"github.com/imabg/authx/internal/hasher"
	"github.com/imabg/authx/internal/mail"
	"github.com/imabg/authx/internal/secret"
	"go.uber.org/mock/gomock"
)

type fakeSMTPRepo struct {
	mu    sync.Mutex
	byApp map[uuid.UUID][]app.SMTPConfig
}

func newFakeSMTPRepo() *fakeSMTPRepo {
	return &fakeSMTPRepo{byApp: map[uuid.UUID][]app.SMTPConfig{}}
}

func (f *fakeSMTPRepo) Create(_ context.Context, cfg app.SMTPConfig) (app.SMTPConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	cfg.ID = uuid.New()
	cfg.Active = false
	f.byApp[cfg.ApplicationID] = append(f.byApp[cfg.ApplicationID], cfg)
	return cfg, nil
}

func (f *fakeSMTPRepo) ListByApplication(_ context.Context, applicationID uuid.UUID) ([]app.SMTPConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := append([]app.SMTPConfig(nil), f.byApp[applicationID]...)
	return out, nil
}

func (f *fakeSMTPRepo) GetByID(_ context.Context, applicationID, id uuid.UUID) (app.SMTPConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, cfg := range f.byApp[applicationID] {
		if cfg.ID == id {
			return cfg, nil
		}
	}
	return app.SMTPConfig{}, app.ErrSMTPNotFound
}

func (f *fakeSMTPRepo) GetActive(_ context.Context, applicationID uuid.UUID) (app.SMTPConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, cfg := range f.byApp[applicationID] {
		if cfg.Active {
			return cfg, nil
		}
	}
	return app.SMTPConfig{}, app.ErrSMTPNotFound
}

func (f *fakeSMTPRepo) Update(_ context.Context, cfg app.SMTPConfig) (app.SMTPConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := f.byApp[cfg.ApplicationID]
	for i, existing := range list {
		if existing.ID == cfg.ID {
			list[i] = cfg
			f.byApp[cfg.ApplicationID] = list
			return cfg, nil
		}
	}
	return app.SMTPConfig{}, app.ErrSMTPNotFound
}

func (f *fakeSMTPRepo) Activate(_ context.Context, applicationID, id uuid.UUID) (app.SMTPConfig, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := f.byApp[applicationID]
	found := false
	var activated app.SMTPConfig
	for i := range list {
		list[i].Active = list[i].ID == id
		if list[i].ID == id {
			found = true
			activated = list[i]
		}
	}
	if !found {
		return app.SMTPConfig{}, app.ErrSMTPNotFound
	}
	f.byApp[applicationID] = list
	return activated, nil
}

func (f *fakeSMTPRepo) Delete(_ context.Context, applicationID, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	list := f.byApp[applicationID]
	out := list[:0]
	found := false
	for _, cfg := range list {
		if cfg.ID == id {
			found = true
			continue
		}
		out = append(out, cfg)
	}
	if !found {
		return app.ErrSMTPNotFound
	}
	f.byApp[applicationID] = out
	return nil
}

func TestSMTPConfigLifecycle(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock.NewMockAppRepository(ctrl)
	smtpRepo := newFakeSMTPRepo()
	box, err := secret.NewBox("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}
	appID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	repo.EXPECT().GetByID(gomock.Any(), appID).Return(app.Application{ID: appID, Settings: app.DefaultSettings()}, nil).AnyTimes()

	svc := app.NewServiceWithDeps(repo, smtpRepo, box)
	ctx := context.Background()

	first, err := svc.CreateSMTPConfig(ctx, appID, app.SMTPCreateInput{
		Name:     "primary",
		Host:     "smtp.example.com",
		Port:     587,
		Username: "acme",
		Password: "czNjcmV0",
	})
	if err != nil {
		t.Fatalf("CreateSMTPConfig: %v", err)
	}
	if first.Active {
		t.Fatal("new configs must default to inactive")
	}
	if first.Username != "" || first.Password != "" {
		t.Fatalf("create leaked credentials: %+v", first)
	}

	second, err := svc.CreateSMTPConfig(ctx, appID, app.SMTPCreateInput{
		Name: "backup",
		Host: "smtp-backup.example.com",
		Port: 465,
	})
	if err != nil {
		t.Fatalf("CreateSMTPConfig backup: %v", err)
	}

	activated, err := svc.ActivateSMTPConfig(ctx, appID, first.ID)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if !activated.Active {
		t.Fatal("expected active")
	}

	listed, err := svc.ListSMTPConfigs(ctx, appID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("len = %d", len(listed))
	}
	activeCount := 0
	for _, cfg := range listed {
		if cfg.Username != "" || cfg.Password != "" {
			t.Fatalf("list leaked credentials: %+v", cfg)
		}
		if cfg.Active {
			activeCount++
			if cfg.ID != first.ID {
				t.Fatalf("active id = %s", cfg.ID)
			}
		}
	}
	if activeCount != 1 {
		t.Fatalf("activeCount = %d", activeCount)
	}

	activated, err = svc.ActivateSMTPConfig(ctx, appID, second.ID)
	if err != nil {
		t.Fatalf("Activate second: %v", err)
	}
	listed, _ = svc.ListSMTPConfigs(ctx, appID)
	activeCount = 0
	for _, cfg := range listed {
		if cfg.Active {
			activeCount++
			if cfg.ID != second.ID {
				t.Fatalf("active id = %s want %s", cfg.ID, second.ID)
			}
		}
	}
	if activeCount != 1 {
		t.Fatalf("activeCount after switch = %d", activeCount)
	}

	_, err = svc.CreateSMTPConfig(ctx, appID, app.SMTPCreateInput{Host: "smtp.example.com", Port: 587, Password: "not-base64!"})
	if err == nil || !strings.Contains(err.Error(), "mail.smtp.password must be base64-encoded") {
		t.Fatalf("error = %v", err)
	}
}

func TestGetByClientCredentialsUsesActiveSMTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock.NewMockAppRepository(ctrl)
	smtpRepo := newFakeSMTPRepo()
	box, err := secret.NewBox("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}
	appID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	settings := app.DefaultSettings()
	settings.Mail.Provider = mail.ProviderSMTP
	settings.Mail.FromEmail = "noreply@example.com"
	hash, err := hasher.Hash("s3cret")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	stored := app.Application{
		ID:               appID,
		ClientID:         "app_demo",
		ClientSecretHash: hash,
		Settings:         settings,
		Status:           app.StatusActive,
	}
	repo.EXPECT().GetByClientID(gomock.Any(), "app_demo").Return(stored, nil)
	repo.EXPECT().GetByID(gomock.Any(), appID).Return(stored, nil).AnyTimes()

	svc := app.NewServiceWithDeps(repo, smtpRepo, box)
	ctx := context.Background()
	created, err := svc.CreateSMTPConfig(ctx, appID, app.SMTPCreateInput{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "acme",
		Password: "czNjcmV0",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.ActivateSMTPConfig(ctx, appID, created.ID); err != nil {
		t.Fatalf("activate: %v", err)
	}

	got, err := svc.GetByClientCredentials(ctx, "app_demo", "s3cret")
	if err != nil {
		t.Fatalf("GetByClientCredentials: %v", err)
	}
	if got.Settings.Mail.SMTP.Host != "smtp.example.com" || got.Settings.Mail.SMTP.Username != "acme" || got.Settings.Mail.SMTP.Password != "s3cret" {
		t.Fatalf("hydrated smtp = %+v", got.Settings.Mail.SMTP)
	}
}
