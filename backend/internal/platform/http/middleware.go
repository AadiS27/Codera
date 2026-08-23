package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

type contextKey string

const requestIDKey = contextKey("requestID")

func RecoverPanic(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.Header().Set("Connection", "close")
					w.WriteHeader(http.StatusInternalServerError)
					logger.Error("panic recovered", "error", err, "path", r.URL.Path)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func generateRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func RequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get("X-Request-ID")
			if reqID == "" {
				reqID = generateRequestID()
			}
			w.Header().Set("X-Request-ID", reqID)
			ctx := context.WithValue(r.Context(), requestIDKey, reqID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (rr *responseRecorder) WriteHeader(status int) {
	rr.status = status
	rr.ResponseWriter.WriteHeader(status)
}

func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip wrapping for WebSocket upgrades — the responseRecorder
			// hides the http.Hijacker interface the upgrader needs.
			if r.Header.Get("Upgrade") == "websocket" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			rr := &responseRecorder{
				ResponseWriter: w,
				status:         http.StatusOK, // Default to 200
			}

			next.ServeHTTP(rr, r)

			reqID := ""
			if val := r.Context().Value(requestIDKey); val != nil {
				if str, ok := val.(string); ok {
					reqID = str
				}
			}

			logger.Info("HTTP Request",
				slog.String("request_id", reqID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rr.status),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

func Chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
