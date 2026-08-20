package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/imabg/authx/internal/auth"
	"github.com/imabg/authx/internal/httpx"
	"github.com/imabg/authx/pkg/config"
)

func TestWriteAuthErrorInternalHidesDetailsOutsideDev(t *testing.T) {
	for _, env := range []string{"production", "staging", "prod", "test"} {
		t.Run(env, func(t *testing.T) {
			srv := newServerWithEnv(env)
			rr, req := authErrorRecorder()
			internal := errors.New(`pq: column "otp_code" does not exist`)
			srv.writeAuthError(rr, req, internal)

			if rr.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", rr.Code)
			}
			body := rr.Body.String()
			if strings.Contains(body, "pq:") || strings.Contains(body, "otp_code") {
				t.Fatalf("response leaked internal error: %s", body)
			}
			var parsed httpx.ErrorBody
			if err := json.Unmarshal(rr.Body.Bytes(), &parsed); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if parsed.Code != "internal_error" {
				t.Fatalf("code = %q", parsed.Code)
			}
			if parsed.Error != "internal error" {
				t.Fatalf("error = %q", parsed.Error)
			}
		})
	}
}

func TestWriteAuthErrorInternalIncludesDetailsInDev(t *testing.T) {
	for _, env := range []string{"dev", "development", "DEV"} {
		t.Run(env, func(t *testing.T) {
			srv := newServerWithEnv(env)
			rr, req := authErrorRecorder()
			internal := errors.New(`pq: column "otp_code" does not exist`)
			srv.writeAuthError(rr, req, internal)

			if rr.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want 500", rr.Code)
			}
			var parsed httpx.ErrorBody
			if err := json.Unmarshal(rr.Body.Bytes(), &parsed); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if parsed.Code != "internal_error" {
				t.Fatalf("code = %q", parsed.Code)
			}
			if parsed.Error != internal.Error() {
				t.Fatalf("error = %q, want raw error", parsed.Error)
			}
		})
	}
}

func TestWriteAuthErrorMappedDoesNotLeak(t *testing.T) {
	srv := newServerWithEnv("development")
	rr, req := authErrorRecorder()
	srv.writeAuthError(rr, req, auth.ErrInvalidCredentials)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
	if strings.Contains(rr.Body.String(), "internal") {
		t.Fatalf("unexpected body %s", rr.Body.String())
	}
}

func TestWriteAuthErrorBlockedDomain(t *testing.T) {
	srv := newServerWithEnv("development")
	rr, req := authErrorRecorder()
	srv.writeAuthError(rr, req, auth.ErrEmailDomainBlocked)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
	var parsed httpx.ErrorBody
	if err := json.Unmarshal(rr.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if parsed.Code != "email_domain_blocked" {
		t.Fatalf("code = %q", parsed.Code)
	}
	if parsed.Error != "email domain is not allowed" {
		t.Fatalf("error = %q", parsed.Error)
	}
}

func newServerWithEnv(env string) *Server {
	cfg := config.ApplicationConfig{}
	cfg.App.ENV = env
	return &Server{Config: cfg}
}

func authErrorRecorder() (*httptest.ResponseRecorder, *http.Request) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth", nil)
	req = req.WithContext(httpx.WithRequestID(req.Context(), "req-test"))
	return rr, req
}
