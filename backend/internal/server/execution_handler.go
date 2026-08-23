package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/jobs"
	"github.com/codera/code-executor/internal/queue"
)

type ExecutionHandler struct {
	jobService *jobs.Service
}

func NewExecutionHandler(js *jobs.Service) *ExecutionHandler {
	return &ExecutionHandler{
		jobService: js,
	}
}

func (h *ExecutionHandler) HandleExecute() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req domain.ExecutionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		job, err := h.jobService.CreateExecution(r.Context(), req)
		if err != nil {
			if errors.Is(err, queue.ErrQueueFull) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]string{
						"code":    "EXECUTION_QUEUE_FULL",
						"message": "execution queue is full",
					},
				})
				return
			}
			
			if strings.Contains(err.Error(), "required") {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"execution_id": job.ID,
			"status":       job.Status,
		})
	}
}

func (h *ExecutionHandler) HandleGetExecution() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Very basic path param parsing since we don't have a robust router right now
		// Assuming path is /api/v1/executions/{id}
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 5 {
			http.Error(w, "Missing execution ID", http.StatusBadRequest)
			return
		}
		jobID := parts[4]

		job, err := h.jobService.GetExecution(r.Context(), jobID)
		if err != nil {
			if errors.Is(err, jobs.ErrJobNotFound) {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		resp := map[string]interface{}{
			"execution_id": job.ID,
			"status":       job.Status,
			"created_at":   job.CreatedAt,
		}
		if job.StartedAt != nil {
			resp["started_at"] = job.StartedAt
		}
		if job.CompletedAt != nil {
			resp["completed_at"] = job.CompletedAt
		}
		if job.Result != nil {
			resp["result"] = job.Result
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}
