package server

import (
	"encoding/json"
	"net/http"

	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/repository"
)

type ProblemHandler struct {
	probRepo     repository.ProblemRepository
	testCaseRepo repository.TestCaseRepository
}

func NewProblemHandler(probRepo repository.ProblemRepository, testCaseRepo repository.TestCaseRepository) *ProblemHandler {
	return &ProblemHandler{probRepo: probRepo, testCaseRepo: testCaseRepo}
}

func (h *ProblemHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /admin/problems", h.createProblem)
	mux.HandleFunc("POST /admin/problems/full", h.createFullProblem)
	mux.HandleFunc("POST /admin/problems/{id}/test-cases", h.createTestCase)
	mux.HandleFunc("GET /problems", h.listProblems)
	mux.HandleFunc("GET /problems/{id}", h.getProblem)
}

func (h *ProblemHandler) createProblem(w http.ResponseWriter, r *http.Request) {
	var p domain.Problem
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.probRepo.Create(r.Context(), &p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

type CreateFullProblemRequest struct {
	Problem   domain.Problem    `json:"problem"`
	TestCases []domain.TestCase `json:"test_cases"`
}

func (h *ProblemHandler) createFullProblem(w http.ResponseWriter, r *http.Request) {
	var req CreateFullProblemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 1. Create Problem
	if err := h.probRepo.Create(r.Context(), &req.Problem); err != nil {
		http.Error(w, "failed to create problem: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Create Test Cases
	for i := range req.TestCases {
		req.TestCases[i].ProblemID = req.Problem.ID
		if req.TestCases[i].SortOrder == 0 {
			req.TestCases[i].SortOrder = i + 1 // Ensure some sort order
		}
		if err := h.testCaseRepo.Create(r.Context(), &req.TestCases[i]); err != nil {
			http.Error(w, "failed to create test case: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"problem":    req.Problem,
		"test_cases": req.TestCases,
	})
}

func (h *ProblemHandler) createTestCase(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	
	var tc domain.TestCase
	if err := json.NewDecoder(r.Body).Decode(&tc); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tc.ProblemID = id

	if err := h.testCaseRepo.Create(r.Context(), &tc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tc)
}

func (h *ProblemHandler) getProblem(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	
	prob, err := h.probRepo.GetByID(r.Context(), id)
	if err != nil {
		if err == repository.ErrProblemNotFound {
			http.Error(w, "problem not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch only public test cases!
	testCases, err := h.testCaseRepo.GetByProblemID(r.Context(), id, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"problem":    prob,
		"test_cases": testCases,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ProblemHandler) listProblems(w http.ResponseWriter, r *http.Request) {
	problems, err := h.probRepo.ListAll(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(problems)
}
