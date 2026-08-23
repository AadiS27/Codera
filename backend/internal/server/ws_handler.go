package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/jobs"
	"github.com/codera/code-executor/internal/repository"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev
	},
}

type WSHandler struct {
	jobService *jobs.Service
	subRepo    repository.SubmissionRepository
	redis      *redis.Client
}

func NewWSHandler(jobService *jobs.Service, subRepo repository.SubmissionRepository, redisClient *redis.Client) *WSHandler {
	return &WSHandler{
		jobService: jobService,
		subRepo:    subRepo,
		redis:      redisClient,
	}
}

func (h *WSHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /ws/run/{id}", h.handleRunWS)
	mux.HandleFunc("GET /ws/submissions/{id}", h.handleSubmissionWS)
}

func (h *WSHandler) handleRunWS(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Initial fetch to see if it's already done
	job, err := h.jobService.GetExecution(r.Context(), id)
	if err != nil {
		conn.WriteJSON(map[string]interface{}{"error": "not found"})
		return
	}

	// Send current state
	conn.WriteJSON(job)

	if job.Status == jobs.StatusCompleted || job.Status == jobs.StatusDeadLettered {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		return // Already finished
	}

	// Subscribe to Redis PubSub for updates
	channel := "job-updates:" + id
	pubsub := h.redis.Subscribe(r.Context(), channel)
	defer pubsub.Close()

	// Use a channel to wait for updates or client disconnect
	ch := pubsub.Channel()

	// Ping ticker to keep connection alive
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Listen for close message from client to exit gracefully
	go func() {
		for {
			if _, _, err := conn.NextReader(); err != nil {
				pubsub.Close()
				break
			}
		}
	}()

	for {
		select {
		case msg := <-ch:
			// The message payload should be the updated job JSON
			var updatedJob jobs.ExecutionJob
			if err := json.Unmarshal([]byte(msg.Payload), &updatedJob); err == nil {
				conn.WriteJSON(updatedJob)
				if updatedJob.Status == jobs.StatusCompleted || updatedJob.Status == jobs.StatusDeadLettered {
					conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
					return
				}
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}

func (h *WSHandler) handleSubmissionWS(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Initial fetch
	sub, err := h.subRepo.GetByID(r.Context(), id)
	if err != nil {
		conn.WriteJSON(map[string]interface{}{"error": "not found"})
		return
	}

	conn.WriteJSON(sub)

	if sub.Status == domain.SubmissionStatusCompleted {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		return // Already finished
	}

	channel := "job-updates:" + id
	pubsub := h.redis.Subscribe(r.Context(), channel)
	defer pubsub.Close()

	ch := pubsub.Channel()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			if _, _, err := conn.NextReader(); err != nil {
				pubsub.Close()
				break
			}
		}
	}()

	for {
		select {
		case msg := <-ch:
			var updatedSub domain.Submission
			if err := json.Unmarshal([]byte(msg.Payload), &updatedSub); err == nil {
				conn.WriteJSON(updatedSub)
				if updatedSub.Status == domain.SubmissionStatusCompleted {
					conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
					return
				}
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}

