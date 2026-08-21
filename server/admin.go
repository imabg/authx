package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/imabg/authx/internal/app"
	"github.com/imabg/authx/internal/httpx"
	"github.com/imabg/authx/internal/users"
	"github.com/imabg/authx/internal/validate"
)

type createApplicationBody struct {
	Name        string          `json:"name" validate:"required"`
	Description string          `json:"description"`
	Settings    json.RawMessage `json:"settings"`
}

type updateApplicationBody struct {
	Name        *string         `json:"name" validate:"omitempty,min=1"`
	Description *string         `json:"description"`
	Status      *string         `json:"status" validate:"omitempty,oneof=active disabled"`
	Settings    json.RawMessage `json:"settings"`
}

type applicationResponse struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	ClientID    string       `json:"client_id"`
	Settings    app.Settings `json:"settings"`
	Status      string       `json:"status"`
}

type createApplicationResponse struct {
	applicationResponse
	ClientSecret string `json:"client_secret"`
}

type updateUserBody = users.ProfileUpdate

func publicApplication(a app.Application) applicationResponse {
	return applicationResponse{
		ID:          a.ID.String(),
		Name:        a.Name,
		Description: a.Description,
		ClientID:    a.ClientID,
		Settings:    a.Settings.Public(),
		Status:      a.Status,
	}
}

func (srv *Server) handleCreateApplication(w http.ResponseWriter, r *http.Request) {
	var body createApplicationBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if err := validate.Struct(body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", validationErrorMessage(err))
		return
	}
	settings, err := app.DecodeSettings(body.Settings)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	created, err := srv.apps.Create(r.Context(), app.CreateInput{
		Name:        body.Name,
		Description: body.Description,
		Settings:    settings,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, createApplicationResponse{
		applicationResponse: publicApplication(created.Application),
		ClientSecret:        created.ClientSecret,
	})
}

func (srv *Server) handleGetApplication(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid application id")
		return
	}
	application, err := srv.apps.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "application not found")
			return
		}
		srv.logInternalError(r, err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal error")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, publicApplication(application))
}

func (srv *Server) handleUpdateApplication(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid application id")
		return
	}
	var body updateApplicationBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if err := validate.Struct(body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", validationErrorMessage(err))
		return
	}
	updated, err := srv.apps.Update(r.Context(), id, app.UpdateInput{
		Name:         body.Name,
		Description:  body.Description,
		Status:       body.Status,
		SettingsJSON: body.Settings,
	})
	if err != nil {
		if errors.Is(err, app.ErrNotFound) {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "application not found")
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, publicApplication(updated))
}

func (srv *Server) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}
	var body updateUserBody
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	updated, err := srv.users.UpdateProfile(r.Context(), id, body)
	if err != nil {
		writeUserUpdateError(w, r, srv, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, updated)
}

func writeUserUpdateError(w http.ResponseWriter, r *http.Request, srv *Server, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, users.ErrValidation) {
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if errors.Is(err, users.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	srv.logInternalError(r, err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "internal error")
}
