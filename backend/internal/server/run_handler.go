package server

import (
	"encoding/json"
	"net/http"

	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/jobs"
	"github.com/codera/code-executor/internal/language"
)

type RunHandler struct {
	jobService *jobs.Service
	registry language.Registry
}

func NewRunHandler(jobService *jobs.Service, registry language.Registry) *RunHandler {
	return &RunHandler{
		jobService: jobService,
		registry:   registry,
	}
}

func (h *RunHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /run", h.createRun)
	mux.HandleFunc("GET /run/{id}", h.getRun)
}

type CreateRunRequest struct {
	Language   domain.Language `json:"language"`
	SourceCode string          `json:"source_code"`
	Input      string          `json:"input"`
}

func (h *RunHandler) createRun(w http.ResponseWriter, r *http.Request) {
	var req CreateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return
	}

	_, err := h.registry.Get(req.Language)
	if err != nil {
		http.Error(w, "unsupported language", http.StatusBadRequest)
		return
	}

	job, err := h.jobService.CreateExecution(r.Context(), domain.ExecutionRequest{
		Language:   req.Language,
		SourceCode: req.SourceCode,
		Input:      req.Input,
	})
	if err != nil {
		http.Error(w, "failed to create run", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"id": job.ID, "status": string(job.Status)})
}

func (h *RunHandler) getRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	
	job, err := h.jobService.GetExecution(r.Context(), id)
	if err != nil {
		if err == jobs.ErrJobNotFound {
			http.Error(w, "run not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}
