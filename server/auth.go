package server

import (
	"errors"
	"net/http"

	"github.com/imabg/authx/internal/app"
	"github.com/imabg/authx/internal/auth"
	"github.com/imabg/authx/internal/httpx"
	"github.com/imabg/authx/internal/mail"
	"github.com/imabg/authx/internal/validate"
)

type authenticateBody struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	Code      string `json:"code"`
	Token     string `json:"token"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type refreshBody struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (srv *Server) handleAuthenticate(w http.ResponseWriter, r *http.Request) {
	application := app.FromContext(r.Context())
	var body authenticateBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	result, err := srv.auth.Authenticate(r.Context(), application, auth.Request{
		Email:     body.Email,
		Password:  body.Password,
		Code:      body.Code,
		Token:     body.Token,
		FirstName: body.FirstName,
		LastName:  body.LastName,
		IP:        clientIP(r),
	})
	if err != nil {
		srv.writeAuthError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (srv *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	application := app.FromContext(r.Context())
	var body refreshBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if err := validate.Struct(body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request")
		return
	}
	result, err := srv.auth.Refresh(r.Context(), application, body.RefreshToken)
	if err != nil {
		srv.writeAuthError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (srv *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var body refreshBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if err := validate.Struct(body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request")
		return
	}
	if err := srv.auth.Logout(r.Context(), body.RefreshToken); err != nil {
		srv.writeAuthError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

func (srv *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, user)
}

func (srv *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing user")
		return
	}
	var body updateUserBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	updated, err := srv.users.UpdateProfile(r.Context(), user.ID, body)
	if err != nil {
		writeUserUpdateError(w, r, srv, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

func (srv *Server) writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidPayload):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_payload_for_auth_method", "payload does not match application auth method")
	case errors.Is(err, auth.ErrPasswordPolicy):
		httpx.WriteError(w, http.StatusBadRequest, "password_policy", err.Error())
	case errors.Is(err, auth.ErrValidation):
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid request")
	case errors.Is(err, auth.ErrRateLimited):
		httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
	case errors.Is(err, auth.ErrEmailDomainBlocked):
		httpx.WriteError(w, http.StatusForbidden, "email_domain_blocked", "email domain is not allowed")
	case errors.Is(err, auth.ErrInvalidCredentials):
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
	case errors.Is(err, mail.ErrNoActiveSMTP):
		httpx.WriteError(w, http.StatusBadRequest, "mail_not_configured", "no active smtp configuration")
	default:
		srv.logInternalError(r, err)
		msg := "internal error"
		if srv.Config.IsDevelopment() {
			msg = err.Error()
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", msg)
	}
}
