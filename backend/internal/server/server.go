package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/codera/code-executor/internal/config"
	"github.com/codera/code-executor/internal/jobs"
	phttp "github.com/codera/code-executor/internal/platform/http"
)

type Server struct {
	server      *http.Server
	logger      *slog.Logger
	config      *config.Config
	execHandler *ExecutionHandler
	shuttingDown atomic.Bool
}

func New(cfg *config.Config, logger *slog.Logger, jobService *jobs.Service) *Server {
	mux := http.NewServeMux()

	s := &Server{
		logger:      logger,
		config:      cfg,
		execHandler: NewExecutionHandler(jobService),
	}

	mux.HandleFunc("/health/live", s.handleHealthLive())
	mux.HandleFunc("/health/ready", s.handleHealthReady())
	mux.HandleFunc("/api/v1/version", s.handleVersion())
	mux.HandleFunc("/api/v1/executions", func(w http.ResponseWriter, r *http.Request) {
		if s.shuttingDown.Load() {
			http.Error(w, "Service Unavailable - Shutting Down", http.StatusServiceUnavailable)
			return
		}
		s.execHandler.HandleExecute()(w, r)
	})
	mux.HandleFunc("/api/v1/executions/", s.execHandler.HandleGetExecution())

	// Apply Middlewares: Recovery -> RequestID -> RequestLogger
	handler := phttp.Chain(mux,
		phttp.RecoverPanic(logger),
		phttp.RequestID(),
		phttp.RequestLogger(logger),
	)

	s.server = &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return s
}

func (s *Server) handleHealthLive() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

func (s *Server) handleHealthReady() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.shuttingDown.Load() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "shutting_down"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

func (s *Server) handleVersion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"version": "1.0.0"})
	}
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.shuttingDown.Store(true)
	return s.server.Shutdown(ctx)
}
