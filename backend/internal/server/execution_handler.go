package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/codera/code-executor/internal/domain"
	"github.com/codera/code-executor/internal/execution"
)

type ExecutionHandler struct {
	service *execution.Service
}

func NewExecutionHandler(service *execution.Service) *ExecutionHandler {
	return &ExecutionHandler{
		service: service,
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

		result, err := h.service.Execute(r.Context(), req)
		if err != nil {
			// Check if it's a validation error
			if errors.Is(err, execution.ErrUnsupportedLanguage) || errors.Is(err, execution.ErrEmptySourceCode) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			// Platform error
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Write successful response (even if execution failed)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}
}
