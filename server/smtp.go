package server

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/imabg/authx/internal/app"
	"github.com/imabg/authx/internal/httpx"
	"github.com/imabg/authx/internal/validate"
)

func (srv *Server) handleCreateSMTPConfig(w http.ResponseWriter, r *http.Request) {
	applicationID, ok := parseApplicationID(w, r)
	if !ok {
		return
	}
	var body app.SMTPCreateInput
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	cfg, err := srv.smtp.Create(r.Context(), applicationID, body)
	if err != nil {
		writeSMTPError(w, r, srv, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, cfg.Public())
}

func (srv *Server) handleListSMTPConfigs(w http.ResponseWriter, r *http.Request) {
	applicationID, ok := parseApplicationID(w, r)
	if !ok {
		return
	}
	cfgs, err := srv.smtp.List(r.Context(), applicationID)
	if err != nil {
		writeSMTPError(w, r, srv, err)
		return
	}
	out := make([]app.SMTPConfigPublic, 0, len(cfgs))
	for _, cfg := range cfgs {
		out = append(out, cfg.Public())
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"smtp_configs": out})
}

func (srv *Server) handleGetSMTPConfig(w http.ResponseWriter, r *http.Request) {
	applicationID, configID, ok := parseSMTPConfigIDs(w, r)
	if !ok {
		return
	}
	cfg, err := srv.smtp.Get(r.Context(), applicationID, configID)
	if err != nil {
		writeSMTPError(w, r, srv, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, cfg.Public())
}

func (srv *Server) handleUpdateSMTPConfig(w http.ResponseWriter, r *http.Request) {
	applicationID, configID, ok := parseSMTPConfigIDs(w, r)
	if !ok {
		return
	}
	var body app.SMTPUpdateInput
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	cfg, err := srv.smtp.Update(r.Context(), applicationID, configID, body)
	if err != nil {
		writeSMTPError(w, r, srv, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, cfg.Public())
}

func (srv *Server) handleActivateSMTPConfig(w http.ResponseWriter, r *http.Request) {
	applicationID, configID, ok := parseSMTPConfigIDs(w, r)
	if !ok {
		return
	}
	cfg, err := srv.smtp.Activate(r.Context(), applicationID, configID)
	if err != nil {
		writeSMTPError(w, r, srv, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, cfg.Public())
}

func (srv *Server) handleDeleteSMTPConfig(w http.ResponseWriter, r *http.Request) {
	applicationID, configID, ok := parseSMTPConfigIDs(w, r)
	if !ok {
		return
	}
	if err := srv.smtp.Delete(r.Context(), applicationID, configID); err != nil {
		writeSMTPError(w, r, srv, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseApplicationID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid application id")
		return uuid.Nil, false
	}
	return id, true
}

func parseSMTPConfigIDs(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	applicationID, ok := parseApplicationID(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	configID, err := uuid.Parse(mux.Vars(r)["sid"])
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid smtp config id")
		return uuid.Nil, uuid.Nil, false
	}
	return applicationID, configID, true
}

func writeSMTPError(w http.ResponseWriter, r *http.Request, srv *Server, err error) {
	switch {
	case errors.Is(err, app.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "application not found")
	case errors.Is(err, app.ErrSMTPNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "smtp configuration not found")
	case errors.Is(err, app.ErrSMTPNotConfigured):
		srv.logInternalError(r, err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal error")
	case containsSMTPValidation(err.Error()):
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		if _, ok := validate.First(err); ok {
			httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		srv.logInternalError(r, err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}

func containsSMTPValidation(msg string) bool {
	return msg == "host is required" ||
		msg == "port must be between 1 and 65535" ||
		msg == "mail.smtp.password must be base64-encoded" ||
		msg == "encryption.key is not configured"
}
