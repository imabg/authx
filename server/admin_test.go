package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/imabg/authx/internal/app"
	appmock "github.com/imabg/authx/internal/app/mock"
	"github.com/imabg/authx/internal/users"
	usersmock "github.com/imabg/authx/internal/users/mock"
	"github.com/imabg/authx/pkg/config"
	"go.uber.org/mock/gomock"
)

func TestHandleUpdateApplication(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := appmock.NewMockAppRepository(ctrl)
	id := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	stored := app.Application{
		ID:       id,
		Name:     "Acme",
		ClientID: "app_demo",
		Status:   app.StatusActive,
		Settings: func() app.Settings {
			s := app.DefaultSettings()
			s.Mail.Provider = "sendgrid"
			s.Mail.FromEmail = "noreply@example.com"
			s.Mail.SendGrid.APIKey = "sg-real-key"
			return s
		}(),
	}
	repo.EXPECT().GetByID(gomock.Any(), id).Return(stored, nil)
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ any, application app.Application) (app.Application, error) {
			if application.Settings.OTP.Length != 8 {
				t.Fatalf("otp length = %d", application.Settings.OTP.Length)
			}
			if application.Settings.Mail.SendGrid.APIKey != "sg-real-key" {
				t.Fatalf("secret replaced with %q", application.Settings.Mail.SendGrid.APIKey)
			}
			return application, nil
		},
	)

	cfg := config.ApplicationConfig{}
	cfg.AdminAPIKey = "dev-admin-key"
	srv := &Server{apps: app.NewService(repo), Config: cfg}
	srv.setupRouter()

	body := `{"settings":{"otp":{"length":8},"mail":{"provider":"sendgrid","from_email":"noreply@example.com","sendgrid":{"api_key":"********"}}}}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/applications/"+id.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Authx-Admin-Key", "dev-admin-key")
	rr := httptest.NewRecorder()
	srv.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "sg-real-key") {
		t.Fatalf("response leaked api key: %s", rr.Body.String())
	}
	var parsed map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("decode: %v", err)
	}
	settings := parsed["settings"].(map[string]any)
	mail := settings["mail"].(map[string]any)
	sendgrid := mail["sendgrid"].(map[string]any)
	if sendgrid["api_key"] != "********" {
		t.Fatalf("masked key = %+v", sendgrid["api_key"])
	}
}

func TestHandleGetApplication(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := appmock.NewMockAppRepository(ctrl)
	id := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	stored := app.Application{
		ID:       id,
		Name:     "Acme",
		ClientID: "app_demo",
		Status:   app.StatusActive,
		Settings: func() app.Settings {
			s := app.DefaultSettings()
			s.BlockedDomains = []string{"blocked.com"}
			s.Mail.SendGrid.APIKey = "sg-real-key"
			return s
		}(),
	}
	repo.EXPECT().GetByID(gomock.Any(), id).Return(stored, nil)

	cfg := config.ApplicationConfig{}
	cfg.AdminAPIKey = "dev-admin-key"
	srv := &Server{apps: app.NewService(repo), Config: cfg}
	srv.setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/applications/"+id.String(), nil)
	req.Header.Set("X-Authx-Admin-Key", "dev-admin-key")
	rr := httptest.NewRecorder()
	srv.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "sg-real-key") {
		t.Fatalf("response leaked api key: %s", rr.Body.String())
	}
	var parsed map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("decode: %v", err)
	}
	settings := parsed["settings"].(map[string]any)
	domains, ok := settings["blocked_domains"].([]any)
	if !ok || len(domains) != 1 || domains[0] != "blocked.com" {
		t.Fatalf("blocked_domains = %+v", settings["blocked_domains"])
	}
}

func TestHandleGetApplicationOmitsSMTPCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := appmock.NewMockAppRepository(ctrl)
	id := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	stored := app.Application{
		ID:       id,
		Name:     "Acme",
		ClientID: "app_demo",
		Status:   app.StatusActive,
		Settings: func() app.Settings {
			s := app.DefaultSettings()
			s.Mail.Provider = "smtp"
			s.Mail.FromEmail = "noreply@example.com"
			s.Mail.SMTP.Host = "smtp.example.com"
			s.Mail.SMTP.Port = 587
			s.Mail.SMTP.Username = "acme"
			s.Mail.SMTP.Password = "smtp-real"
			return s
		}(),
	}
	repo.EXPECT().GetByID(gomock.Any(), id).Return(stored, nil)

	cfg := config.ApplicationConfig{}
	cfg.AdminAPIKey = "dev-admin-key"
	srv := &Server{apps: app.NewService(repo), Config: cfg}
	srv.setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/applications/"+id.String(), nil)
	req.Header.Set("X-Authx-Admin-Key", "dev-admin-key")
	rr := httptest.NewRecorder()
	srv.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if strings.Contains(body, "acme") || strings.Contains(body, "smtp-real") {
		t.Fatalf("response leaked smtp credentials: %s", body)
	}
	var parsed map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("decode: %v", err)
	}
	mail := parsed["settings"].(map[string]any)["mail"].(map[string]any)
	smtp := mail["smtp"].(map[string]any)
	if _, ok := smtp["username"]; ok {
		t.Fatalf("username present: %+v", smtp)
	}
	if _, ok := smtp["password"]; ok {
		t.Fatalf("password present: %+v", smtp)
	}
	if smtp["host"] != "smtp.example.com" {
		t.Fatalf("host = %+v", smtp["host"])
	}
}

func TestHandleUpdateApplicationBlockedDomains(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := appmock.NewMockAppRepository(ctrl)
	id := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	stored := app.Application{
		ID:     id,
		Name:   "Acme",
		Status: app.StatusActive,
		Settings: func() app.Settings {
			s := app.DefaultSettings()
			s.BlockedDomains = []string{"old.com"}
			return s
		}(),
	}
	repo.EXPECT().GetByID(gomock.Any(), id).Return(stored, nil)
	repo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ any, application app.Application) (app.Application, error) {
			if len(application.Settings.BlockedDomains) != 2 ||
				application.Settings.BlockedDomains[0] != "blocked.com" ||
				application.Settings.BlockedDomains[1] != "evil.org" {
				t.Fatalf("blocked_domains = %#v", application.Settings.BlockedDomains)
			}
			return application, nil
		},
	)

	cfg := config.ApplicationConfig{}
	cfg.AdminAPIKey = "dev-admin-key"
	srv := &Server{apps: app.NewService(repo), Config: cfg}
	srv.setupRouter()

	body := `{"settings":{"blocked_domains":[" Blocked.COM ", "evil.org"]}}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/applications/"+id.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Authx-Admin-Key", "dev-admin-key")
	rr := httptest.NewRecorder()
	srv.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdateApplicationNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := appmock.NewMockAppRepository(ctrl)
	id := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	repo.EXPECT().GetByID(gomock.Any(), id).Return(app.Application{}, app.ErrNotFound)

	cfg := config.ApplicationConfig{}
	cfg.AdminAPIKey = "dev-admin-key"
	srv := &Server{apps: app.NewService(repo), Config: cfg}
	srv.setupRouter()

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/applications/"+id.String(), strings.NewReader(`{"name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Authx-Admin-Key", "dev-admin-key")
	rr := httptest.NewRecorder()
	srv.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
}

func TestHandleGetApplicationNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := appmock.NewMockAppRepository(ctrl)
	id := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	repo.EXPECT().GetByID(gomock.Any(), id).Return(app.Application{}, app.ErrNotFound)

	cfg := config.ApplicationConfig{}
	cfg.AdminAPIKey = "dev-admin-key"
	srv := &Server{apps: app.NewService(repo), Config: cfg}
	srv.setupRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/applications/"+id.String(), nil)
	req.Header.Set("X-Authx-Admin-Key", "dev-admin-key")
	rr := httptest.NewRecorder()
	srv.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
}

func TestHandleAdminUpdateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	usersRepo := usersmock.NewMockIUserRepository(ctrl)
	id := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
	existing := users.User{ID: id, FirstName: "Ada", LastName: "Lovelace", Email: "ada@example.com", Status: users.StatusActive}
	usersRepo.EXPECT().GetByID(gomock.Any(), id).Return(existing, nil)
	usersRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ any, user users.User) (users.User, error) {
			if user.FirstName != "Grace" {
				t.Fatalf("first_name = %s", user.FirstName)
			}
			return user, nil
		},
	)

	cfg := config.ApplicationConfig{}
	cfg.AdminAPIKey = "dev-admin-key"
	srv := &Server{users: users.NewService(usersRepo), Config: cfg}
	srv.setupRouter()

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/"+id.String(), strings.NewReader(`{"first_name":"Grace"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Authx-Admin-Key", "dev-admin-key")
	rr := httptest.NewRecorder()
	srv.Router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdateMe(t *testing.T) {
	ctrl := gomock.NewController(t)
	usersRepo := usersmock.NewMockIUserRepository(ctrl)
	user := users.User{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), FirstName: "Ada", LastName: "Lovelace"}
	usersRepo.EXPECT().GetByID(gomock.Any(), user.ID).Return(user, nil)
	usersRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ any, updated users.User) (users.User, error) {
			if updated.LastName != "Hopper" {
				t.Fatalf("last_name = %s", updated.LastName)
			}
			return updated, nil
		},
	)
	srv := &Server{users: users.NewService(usersRepo)}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/me", strings.NewReader(`{"last_name":"Hopper"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUser(req.Context(), user))
	rr := httptest.NewRecorder()
	srv.handleUpdateMe(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdateMeValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	usersRepo := usersmock.NewMockIUserRepository(ctrl)
	user := users.User{ID: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), FirstName: "Ada"}
	usersRepo.EXPECT().GetByID(gomock.Any(), user.ID).Return(user, nil)
	srv := &Server{users: users.NewService(usersRepo)}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/me", strings.NewReader(`{"first_name":"`+strings.Repeat("a", 26)+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(withUser(req.Context(), user))
	rr := httptest.NewRecorder()
	srv.handleUpdateMe(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
}
