package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/imabg/authx/internal/app"
	"github.com/imabg/authx/internal/httpx"
	"github.com/imabg/authx/internal/users"
	"go.uber.org/zap"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zap.L().Info("request",
			zap.String("uri", r.RequestURI),
			zap.String("method", r.Method),
			zap.String("request_id", httpx.RequestIDFrom(r.Context())),
		)
		next.ServeHTTP(w, r)
	})
}

func (srv *Server) requireApplication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID := strings.TrimSpace(r.Header.Get("X-Authx-Client-Id"))
		clientSecret := strings.TrimSpace(r.Header.Get("X-Authx-Client-Secret"))
		if clientID == "" || clientSecret == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing application credentials")
			return
		}
		application, err := srv.apps.GetByClientCredentials(r.Context(), clientID, clientSecret)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid application credentials")
			return
		}
		if !application.Active() {
			httpx.WriteError(w, http.StatusForbidden, "application_disabled", "application is disabled")
			return
		}
		next.ServeHTTP(w, r.WithContext(app.WithApplication(r.Context(), &application)))
	})
}

func (srv *Server) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		application := app.FromContext(r.Context())
		if application == nil {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing application")
			return
		}
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
			return
		}
		raw := strings.TrimSpace(header[7:])
		claims, err := srv.tokens.ParseAccess(raw)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_token", "invalid access token")
			return
		}
		if claims.AppID != application.ID.String() {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_token", "invalid access token")
			return
		}
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_token", "invalid access token")
			return
		}
		user, err := srv.users.GetByID(r.Context(), userID)
		if err != nil || !user.Active() {
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_token", "invalid access token")
			return
		}
		if application.Settings.EmailDomainBlocked(user.Email) {
			httpx.WriteError(w, http.StatusForbidden, "email_domain_blocked", "email domain is not allowed")
			return
		}
		next.ServeHTTP(w, r.WithContext(withUser(r.Context(), user)))
	})
}

func (srv *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.Header.Get("X-Authx-Admin-Key"))
		if srv.Config.AdminAPIKey == "" || key == "" || key != srv.Config.AdminAPIKey {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid admin credentials")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type userCtxKey struct{}

func withUser(ctx context.Context, user users.User) context.Context {
	return context.WithValue(ctx, userCtxKey{}, user)
}

func userFrom(ctx context.Context) (users.User, bool) {
	user, ok := ctx.Value(userCtxKey{}).(users.User)
	return user, ok
}
