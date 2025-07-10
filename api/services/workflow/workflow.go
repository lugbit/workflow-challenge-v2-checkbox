package workflow

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/gorilla/mux"
)

func (s *Service) HandleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	ctx := r.Context()

	slog.Debug("Returning workflow definition for id", "id", id)

	definitionBytes, err := s.GetWorkflowDefinitionByID(ctx, id)
	if err != nil {
		var status int
		var msg string

		switch {
		case errors.Is(err, pgx.ErrNoRows):
			status = http.StatusNotFound
			msg = errorToJSON(ErrWorkflowNotFound)
		default:
			status = http.StatusInternalServerError
			msg = errorToJSON(ErrInternalServerError)
		}

		http.Error(w, msg, status)
		return
	}

	var wf WorkflowDefinition
	if err := json.Unmarshal(definitionBytes, &wf); err != nil {
		slog.Error("Invalid workflow format", "id", id, "error", err)
		http.Error(w, errorToJSON(ErrInvalidWorkflowFormat), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(definitionBytes)
}

// form data structs
type Condition struct {
	Operator  string  `json:"operator"`
	Threshold float64 `json:"threshold"`
}

type FormData struct {
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	City      string  `json:"city"`
	Operator  string  `json:"operator"`  // Optional if already in Condition
	Threshold float64 `json:"threshold"` // Optional if already in Condition
}

type ExecutePayload struct {
	FormData  FormData  `json:"formData"`
	Condition Condition `json:"condition"`
}

func (s *Service) HandleExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	ctx := r.Context()
	slog.Debug("Handling workflow execution for id", "id", id)

	// decode form data
	var payload ExecutePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		slog.Error("Invalid JSON payload", "error", err)
		http.Error(w, errorToJSON(ErrInvalidJSON), http.StatusBadRequest)
		return
	}

	definitionBytes, err := s.GetWorkflowDefinitionByID(ctx, id)
	if err != nil {
		var status int
		var msg string

		switch {
		case errors.Is(err, pgx.ErrNoRows):
			status = http.StatusNotFound
			msg = errorToJSON(ErrWorkflowNotFound)
		default:
			status = http.StatusInternalServerError
			msg = errorToJSON(ErrInternalServerError)
		}

		http.Error(w, msg, status)
		return
	}

	var wf WorkflowDefinition
	if err := json.Unmarshal(definitionBytes, &wf); err != nil {
		slog.Error("Invalid workflow format", "id", id, "error", err)
		http.Error(w, errorToJSON(ErrInvalidWorkflowFormat), http.StatusInternalServerError)
		return
	}

	// update workflow definition
	err = s.UpdateWorkflowDefinitionByID(ctx, wf.ID, definitionBytes)
	if err != nil {
		slog.Error("Error updating workflow", "id", id, "error", err)
		http.Error(w, errorToJSON(ErrInternalServerError), http.StatusInternalServerError)
		return
	}

	executionResults, err := processNodes(&wf, &payload)
	if err != nil {
		slog.Error("Error executing workflow", "id", id, "error", err)
		http.Error(w, errorToJSON(ErrInternalServerError), http.StatusInternalServerError)
		return
	}

	jsonBytes, err := json.Marshal(executionResults)
	if err != nil {
		slog.Error("Failed to marshal execution results", "error", err)
		http.Error(w, errorToJSON(ErrMarshalFailed), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}
