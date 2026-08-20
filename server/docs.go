package server

import (
	"net/http"

	"github.com/imabg/authx/docs"
)

func (srv *Server) setupDocsRoutes() {
	srv.Router.HandleFunc(docs.SpecPath, srv.handleOpenAPISpec).Methods(http.MethodGet)
	srv.Router.HandleFunc(docs.UIPath, srv.handleDocsUI).Methods(http.MethodGet)
	srv.Router.HandleFunc(docs.UIPath+"/", srv.handleDocsUI).Methods(http.MethodGet)
	srv.Router.HandleFunc("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, docs.UIPath, http.StatusFound)
	}).Methods(http.MethodGet)
}

func (srv *Server) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(docs.OpenAPIYAML())
}

func (srv *Server) handleDocsUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(docs.SwaggerHTML())
}
