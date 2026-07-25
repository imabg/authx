package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/imabg/authx/pkg/db"
	"github.com/imabg/authx/pkg/config"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type Server struct {
	Router *mux.Router
	Logger *zap.Logger
	Config config.ApplicationConfig
	server *http.Server
	db *pgx.Conn
}

func Setup(config config.ApplicationConfig, db *db.DB) *Server {
	return  &Server{
		Logger: zap.L(),
		Config: config,
		db: db.Connection(),
	}
}

func(srv *Server) Run() {
	srv.setupRouter()
	srv.server = &http.Server{
		Handler: srv.Router,
		WriteTimeout: 10 * time.Second,
		ReadTimeout: 15 * time.Second,
		Addr: srv.Config.App.PORT,
	}
	go func ()  {
		if err := srv.server.ListenAndServe(); err != nil {
		srv.Logger.Error("Error while starting server", zap.Error(err))
	}
	}()
	srv.Logger.Info("server is started: %s", zap.String("addr", srv.server.Addr))
}

// setupMiddlewares set middlewares in the router
func (srv *Server) setupMiddlewares() {
	srv.Router.Use(mux.CORSMethodMiddleware(srv.Router))
}

// setupRouter bind middlewares and routes with router
func(srv *Server) setupRouter()  {
	srv.Router = mux.NewRouter()
	srv.setupMiddlewares()
	srv.Router = srv.Router.PathPrefix("/api").Subrouter()
	srv.Router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
}

func (srv *Server) Close(ctx context.Context) {
	srv.server.Shutdown(ctx)
	srv.Logger.Info("Shutting down server")
}