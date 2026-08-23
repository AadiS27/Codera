package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/codera/code-executor/internal/config"
	"github.com/codera/code-executor/internal/execution"
	"github.com/codera/code-executor/internal/sandbox/docker"
)

func TestEndpoints(t *testing.T) {
	cfg := &config.Config{
		Env:             "test",
		Port:            "8080",
		LogLevel:        "info",
		ShutdownTimeout: 1 * time.Second,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sb := docker.NewRuntime(cfg)
	execService := execution.NewService(cfg, sb)
	srv := New(cfg, logger, execService)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedKey    string
	}{
		{"Health Live", "/health/live", http.StatusOK, "status"},
		{"Health Ready", "/health/ready", http.StatusOK, "status"},
		{"API Version", "/api/v1/version", http.StatusOK, "version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, tt.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			rr := httptest.NewRecorder()
			// The handler we bound to the server
			srv.server.Handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			// Validate Request ID middleware worked
			if rr.Header().Get("X-Request-ID") == "" {
				t.Errorf("Expected X-Request-ID header to be set")
			}

			var response map[string]string
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if _, ok := response[tt.expectedKey]; !ok {
				t.Errorf("Expected key %q in response, got %v", tt.expectedKey, response)
			}
		})
	}
}
