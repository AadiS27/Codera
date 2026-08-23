package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/codera/code-executor/internal/config"
	"github.com/codera/code-executor/internal/execution"
	"github.com/codera/code-executor/internal/platform/logger"
	"github.com/codera/code-executor/internal/sandbox/docker"
	"github.com/codera/code-executor/internal/server"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		// Cannot use slog yet, use standard print for bootstrap failure
		os.Stdout.WriteString("Failed to load configuration: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Initialize structured logging
	log := logger.New(cfg.Env, cfg.LogLevel)
	slog.SetDefault(log)

	// Initialize Sandbox
	sb := docker.NewRuntime(cfg)

	// Initialize services
	execService := execution.NewService(cfg, sb)

	// Initialize server
	srv := server.New(cfg, log, execService)

	// Channel to listen for errors coming from the listener.
	serverErrors := make(chan error, 1)

	// Start the API in a goroutine
	go func() {
		log.Info("Starting server", "port", cfg.Port, "env", cfg.Env)
		serverErrors <- srv.Start()
	}()

	// Channel to listen for an interrupt or terminate signal from the OS.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("Server error", "error", err)
			os.Exit(1)
		}

	case sig := <-shutdown:
		log.Info("Graceful shutdown started", "signal", sig, "timeout", cfg.ShutdownTimeout)

		// Create context for graceful shutdown timeout
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Error("Graceful shutdown did not complete in time", "error", err)
			if err := srv.Shutdown(context.Background()); err != nil {
				log.Error("Could not stop server gracefully", "error", err)
			}
			os.Exit(1)
		}
		log.Info("Graceful shutdown completed")
	}
}
