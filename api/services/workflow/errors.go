package workflow

import "errors"

// this file errors.go will contains custom workflow related errors

var (
	// generic errors
	ErrInternalServerError  = errors.New("internal server error")
	ErrResponseDecodeFailed = errors.New("failed to decode response")
	ErrMarshalFailed        = errors.New("failed to marshal results")

	// Workflow-level errors
	ErrWorkflowNotFound      = errors.New("workflow not found")
	ErrInvalidWorkflowFormat = errors.New("invalid workflow format")
	ErrMissingStartNode      = errors.New("missing 'start' node")
	ErrMissingEndNode        = errors.New("missing 'end' node")

	// Request validation errors
	ErrInvalidJSON           = errors.New("invalid JSON")
	ErrMissingFormFieldName  = errors.New("name is required")
	ErrMissingFormFieldEmail = errors.New("email is required")
	ErrMissingFormFieldCity  = errors.New("city is required")
)

func errorToJSON(err error) string {
	return `{"error":"` + err.Error() + `"}`
}
