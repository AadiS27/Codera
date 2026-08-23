package server

import (
	"encoding/json"
	"net/http"

	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/language"
	"github.com/codera/code-executor/internal/queue"
	"github.com/codera/code-executor/internal/repository"
)

type SubmissionHandler struct {
	subRepo  repository.SubmissionRepository
	probRepo repository.ProblemRepository
	queue    queue.JobQueue
	registry language.Registry
}

func NewSubmissionHandler(subRepo repository.SubmissionRepository, probRepo repository.ProblemRepository, q queue.JobQueue, registry language.Registry) *SubmissionHandler {
	return &SubmissionHandler{
		subRepo:  subRepo,
		probRepo: probRepo,
		queue:    q,
		registry: registry,
	}
}

func (h *SubmissionHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /problems/{id}/submissions", h.createSubmission)
	mux.HandleFunc("GET /submissions/{id}", h.getSubmission)
}

type CreateSubmissionRequest struct {
	UserID     string          `json:"user_id"`
	Language   domain.Language `json:"language"`
	SourceCode string          `json:"source_code"`
}

func (h *SubmissionHandler) createSubmission(w http.ResponseWriter, r *http.Request) {
	problemID := r.PathValue("id")

	var req CreateSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request format", http.StatusBadRequest)
		return
	}

	// Validate problem
	prob, err := h.probRepo.GetByID(r.Context(), problemID)
	if err != nil {
		if err == repository.ErrProblemNotFound {
			http.Error(w, "problem not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error checking problem", http.StatusInternalServerError)
		return
	}
	if prob.Status != domain.ProblemStatusPublished {
		http.Error(w, "problem is not published", http.StatusBadRequest)
		return
	}

	// Validate language
	_, err = h.registry.Get(req.Language)
	if err != nil {
		http.Error(w, "unsupported language", http.StatusBadRequest)
		return
	}

	// Create submission
	sub := domain.Submission{
		UserID:     req.UserID,
		ProblemID:  prob.ID,
		Language:   req.Language,
		SourceCode: req.SourceCode,
		Status:     domain.SubmissionStatusQueued,
		Verdict:    domain.VerdictPending,
	}

	if err := h.subRepo.Create(r.Context(), &sub); err != nil {
		http.Error(w, "failed to create submission", http.StatusInternalServerError)
		return
	}

	if err := h.queue.Enqueue(r.Context(), sub.ID, "submission"); err != nil {
		// Log error, but submission exists. Ideally we'd rollback or DLQ.
		http.Error(w, "failed to queue submission", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(sub)
}

func (h *SubmissionHandler) getSubmission(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sub, err := h.subRepo.GetByID(r.Context(), id)
	if err != nil {
		if err == repository.ErrSubmissionNotFound {
			http.Error(w, "submission not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}
