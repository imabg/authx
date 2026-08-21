package server

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (srv *Server) publicRoute(r *mux.Router) {
	auth := r.NewRoute().Subrouter()
	auth.Use(srv.requireApplication)
	auth.HandleFunc("/auth", srv.handleAuthenticate).Methods(http.MethodPost)
	auth.HandleFunc("/auth/refresh", srv.handleRefresh).Methods(http.MethodPost)
	auth.HandleFunc("/auth/logout", srv.handleLogout).Methods(http.MethodPost)
}

func (srv *Server) privateRoute(r *mux.Router) {
	me := r.NewRoute().Subrouter()
	me.Use(srv.requireApplication)
	me.Use(srv.requireUser)
	me.HandleFunc("/me", srv.handleMe).Methods(http.MethodGet)
	me.HandleFunc("/me", srv.handleUpdateMe).Methods(http.MethodPatch)
}

func (srv *Server) adminRoute(r *mux.Router) {
	admin := r.PathPrefix("/admin").Subrouter()
	admin.Use(srv.requireAdmin)
	admin.HandleFunc("/applications", srv.handleCreateApplication).Methods(http.MethodPost)
	admin.HandleFunc("/applications/{id}", srv.handleGetApplication).Methods(http.MethodGet)
	admin.HandleFunc("/applications/{id}", srv.handleUpdateApplication).Methods(http.MethodPatch)
	admin.HandleFunc("/applications/{id}/smtp-configs", srv.handleListSMTPConfigs).Methods(http.MethodGet)
	admin.HandleFunc("/applications/{id}/smtp-configs", srv.handleCreateSMTPConfig).Methods(http.MethodPost)
	admin.HandleFunc("/applications/{id}/smtp-configs/{sid}", srv.handleGetSMTPConfig).Methods(http.MethodGet)
	admin.HandleFunc("/applications/{id}/smtp-configs/{sid}", srv.handleUpdateSMTPConfig).Methods(http.MethodPatch)
	admin.HandleFunc("/applications/{id}/smtp-configs/{sid}", srv.handleDeleteSMTPConfig).Methods(http.MethodDelete)
	admin.HandleFunc("/applications/{id}/smtp-configs/{sid}/activate", srv.handleActivateSMTPConfig).Methods(http.MethodPost)
	admin.HandleFunc("/users/{id}", srv.handleAdminUpdateUser).Methods(http.MethodPatch)
}
