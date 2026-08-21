package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDocsRoutes(t *testing.T) {
	srv := &Server{}
	srv.setupRouter()

	t.Run("openapi yaml", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/docs/openapi.yaml", nil)
		rr := httptest.NewRecorder()
		srv.Router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		body := rr.Body.String()
		if !strings.HasPrefix(body, "openapi: 3.0.3") {
			t.Fatalf("spec prefix = %q", body[:min(40, len(body))])
		}
		for _, path := range []string{
			"/api/health",
			"/api/v1/auth",
			"/api/v1/auth/refresh",
			"/api/v1/auth/logout",
			"/api/v1/me",
			"/api/v1/admin/applications",
			"/api/v1/admin/applications/{id}",
			"/api/v1/admin/applications/{id}/smtp-configs",
			"/api/v1/admin/users/{id}",
		} {
			if !strings.Contains(body, path) {
				t.Errorf("spec missing path %s", path)
			}
		}
		for _, needle := range []string{"sendgrid", "smtp", "first_name", "blocked_domains", "email_domain_blocked"} {
			if !strings.Contains(body, needle) {
				t.Errorf("spec missing %q", needle)
			}
		}
	})

	t.Run("swagger ui", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/docs", nil)
		rr := httptest.NewRecorder()
		srv.Router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
			t.Fatalf("content-type = %q", ct)
		}
	})

	t.Run("swagger redirect", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
		rr := httptest.NewRecorder()
		srv.Router.ServeHTTP(rr, req)
		if rr.Code != http.StatusFound {
			t.Fatalf("status = %d, want 302", rr.Code)
		}
		if loc := rr.Header().Get("Location"); loc != "/api/docs" {
			t.Fatalf("location = %q", loc)
		}
	})

	t.Run("health still served", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		rr := httptest.NewRecorder()
		srv.Router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rr.Code)
		}
		if !strings.Contains(rr.Body.String(), `"ok":true`) {
			t.Fatalf("body = %q", rr.Body.String())
		}
	})
}
