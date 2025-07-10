package workflow

import "context"

// This file repository.go contains workflow related DB methods.
// Note: The queries currently uses raw SQL and manual scanning.
// It could be improved by leveraging SQLBoiler for type safety and maintainability.

// GetWorkflowDefinitionByID retuens a workflow by id.
func (s *Service) GetWorkflowDefinitionByID(ctx context.Context, id string) ([]byte, error) {
	var definition []byte

	err := s.db.QueryRow(ctx, `
		SELECT definition
		FROM workflows
		WHERE definition->>'id' = $1
	`, id).Scan(&definition)

	if err != nil {
		return nil, err
	}

	return definition, nil
}

// UpdateWorkflowDefinitionByID is a helper method to update a workflow definition by id.
func (s *Service) UpdateWorkflowDefinitionByID(ctx context.Context, id string, newDefinition []byte) error {
	_, err := s.db.Exec(ctx, `
		UPDATE workflows
		SET definition = $1,
		    updated_at = now()
		WHERE definition->>'id' = $2
	`, newDefinition, id)
	return err
}
