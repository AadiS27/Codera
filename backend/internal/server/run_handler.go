package server

import (
	"encoding/json"
	"net/http"

	"github.com/codera/code-executor/internal/ai"
	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/jobs"
	"github.com/codera/code-executor/internal/language"
)

type RunHandler struct {
	jobService *jobs.Service
	registry   language.Registry
	analyzer   *ai.Analyzer
}

func NewRunHandler(jobService *jobs.Service, registry language.Registry, analyzer *ai.Analyzer) *RunHandler {
	return &RunHandler{
		jobService: jobService,
		registry:   registry,
		analyzer:   analyzer,
	}
}

func (h *RunHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /run", h.createRun)
	mux.HandleFunc("GET /run/{id}", h.getRun)
	mux.HandleFunc("POST /analyze", h.analyzeCode)
}

type CreateRunRequest struct {
	Language   domain.Language `json:"language"`
	SourceCode string          `json:"source_code"`
	Inputs     []string        `json:"inputs"`
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

	job, err := h.jobService.CreateExecution(r.Context(), domain.JobRequest{
		Language:   req.Language,
		SourceCode: req.SourceCode,
		Inputs:     req.Inputs,
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

func (h *RunHandler) analyzeCode(w http.ResponseWriter, r *http.Request) {
	if h.analyzer == nil {
		http.Error(w, "AI Analysis is not configured on this server", http.StatusNotImplemented)
		return
	}

	var req CreateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return
	}

	analysis, err := h.analyzer.AnalyzeComplexity(r.Context(), req.SourceCode, string(req.Language))
	if err != nil {
		http.Error(w, "failed to analyze code: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}
