package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/imabg/authx/internal/app"
	"github.com/imabg/authx/internal/auth"
	"github.com/imabg/authx/internal/challenge"
	"github.com/imabg/authx/internal/httpx"
	"github.com/imabg/authx/internal/mail"
	"github.com/imabg/authx/internal/token"
	"github.com/imabg/authx/internal/users"
	"github.com/imabg/authx/pkg/config"
	"github.com/imabg/authx/pkg/db"
	"go.uber.org/zap"
)

type Server struct {
	Router *mux.Router
	Logger *zap.Logger
	Config config.ApplicationConfig
	server *http.Server
	db     *db.DB
	apps   *app.Service
	auth   *auth.Service
	tokens token.IService
	users  users.IUserRepository
}

func Setup(cfg config.ApplicationConfig, database *db.DB) *Server {
	if cfg.JWT.Secret == "" {
		zap.L().Fatal("jwt.secret is required")
	}
	pool := database.Pool()
	appRepo := app.NewRepository(pool)
	appSvc := app.NewService(appRepo)
	userRepo := users.NewUserRepository(pool)
	challengeSvc := challenge.NewService(challenge.NewRepository(pool))
	tokenSvc := token.NewService(cfg, pool)
	mailer := mail.NewService(zap.L())
	authSvc := auth.NewService(userRepo, challengeSvc, tokenSvc, mailer, cfg.PublicBaseURL)

	return &Server{
		Logger: zap.L(),
		Config: cfg,
		db:     database,
		apps:   appSvc,
		auth:   authSvc,
		tokens: tokenSvc,
		users:  userRepo,
	}
}

func (srv *Server) Run() {
	srv.setupRouter()
	addr := srv.Config.App.PORT
	if addr != "" && !strings.Contains(addr, ":") {
		addr = ":" + addr
	}
	srv.server = &http.Server{
		Handler:      srv.Router,
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  15 * time.Second,
		Addr:         addr,
	}
	go func() {
		if err := srv.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			srv.Logger.Error("Error while starting server", zap.Error(err))
		}
	}()
	srv.Logger.Info("server is started", zap.String("addr", srv.server.Addr))
}

func (srv *Server) setupMiddlewares() {
	srv.Router.Use(httpx.RequestID)
	srv.Router.Use(loggingMiddleware)
	srv.Router.Use(mux.CORSMethodMiddleware(srv.Router))
}

func (srv *Server) setupRouter() {
	srv.Router = mux.NewRouter()
	srv.setupMiddlewares()
	srv.setupDocsRoutes()
	api := srv.Router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}).Methods(http.MethodGet)

	v1 := api.PathPrefix("/v1").Subrouter()
	srv.publicRoute(v1)
	srv.privateRoute(v1)
	srv.adminRoute(v1)
}

func (srv *Server) Close(ctx context.Context) {
	if srv.server != nil {
		_ = srv.server.Shutdown(ctx)
	}
	srv.Logger.Info("Shutting down server")
}

func (srv *Server) logInternalError(r *http.Request, err error) {
	fields := []zap.Field{
		zap.Error(err),
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("request_id", httpx.RequestIDFrom(r.Context())),
	}
	zap.L().Error("internal error", fields...)
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
