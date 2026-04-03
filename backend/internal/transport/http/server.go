package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"backend/internal/application"
	"backend/internal/config"
	"backend/internal/transport/http/handlers"
	mw "backend/internal/transport/http/middleware"
)

type Server struct {
	http *http.Server
}

const requestBodyLimitBytes int64 = 1 << 20

func NewServer(cfg config.Config, logger zerolog.Logger, services *application.Services) *Server {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(mw.RequestBodyLimit(requestBodyLimitBytes))
	r.Use(mw.Recoverer(logger))
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(mw.RequestLogging(logger))
	r.Use(mw.AdminAuth(cfg.Authz.AdminToken))

	h := handlers.New(services, validator.New())
	h.RegisterRoutes(r)

	srv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      r,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}
	return &Server{http: srv}
}

func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
