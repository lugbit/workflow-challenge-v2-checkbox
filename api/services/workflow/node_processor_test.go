package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessNodes(t *testing.T) {
	tests := []struct {
		label              string
		workflow           *WorkflowDefinition
		payload            *ExecutePayload
		wantStatus         string
		wantStepLen        int
		expectErr          bool
		missingNode        string
		setup              func()
		teardown           func()
		processEmailNodeFn func()
	}{
		{
			label: "success: minimal start -> end",
			workflow: &WorkflowDefinition{
				Nodes: []Node{
					{ID: StartNodeID, Type: "start", Data: NodeData{Label: "Start", Description: "Begin"}},
					{ID: EndNodeID, Type: "end", Data: NodeData{Label: "End", Description: "Finish"}},
				},
				Edges: []Edge{
					{Source: StartNodeID, Target: EndNodeID},
				},
			},
			payload:     &ExecutePayload{},
			wantStatus:  StatusCompleted,
			wantStepLen: 2,
			expectErr:   false,
		},
		{
			label: "error: missing start node",
			workflow: &WorkflowDefinition{
				Nodes: []Node{
					{ID: EndNodeID, Type: "end", Data: NodeData{Label: "End", Description: "Finish"}},
				},
			},
			payload:     &ExecutePayload{},
			wantStatus:  StatusFailed,
			wantStepLen: 0,
			expectErr:   true,
			missingNode: StartNodeID,
		},
		{
			label: "error: missing end node",
			workflow: &WorkflowDefinition{
				Nodes: []Node{
					{ID: StartNodeID, Type: "start", Data: NodeData{Label: "Start", Description: "Begin"}},
				},
			},
			payload:     &ExecutePayload{},
			wantStatus:  StatusFailed,
			wantStepLen: 0,
			expectErr:   true,
			missingNode: EndNodeID,
		},
		{
			label: "happy path: full workflow with mocked process weather-api node and email node",
			workflow: &WorkflowDefinition{
				Nodes: []Node{
					{ID: StartNodeID, Type: "start", Data: NodeData{Label: "Start", Description: "Begin"}},
					{ID: FormNodeID, Type: "form", Data: NodeData{Label: "Form", Description: "User Input"}},
					{ID: WeatherAPINodeID, Type: "weather-api", Data: NodeData{Label: "Weather", Description: "Mocked weather"}},
					{ID: ConditionNodeID, Type: "condition", Data: NodeData{Label: "Check", Description: "Temp Check"}},
					{ID: EmailNodeID, Type: "email", Data: NodeData{
						Label:       "Send Email",
						Description: "Send alert email",
						Metadata: NodeMetadata{
							EmailTemplate: &EmailTemplate{
								Subject: "Weather Alert",
								Body:    "Hello, the temperature in {{city}} is {{temperature}}°C.",
							},
						},
					}},
					{ID: EndNodeID, Type: "end", Data: NodeData{Label: "End", Description: "Finish"}},
				},
				Edges: []Edge{
					{Source: StartNodeID, Target: FormNodeID},
					{Source: FormNodeID, Target: WeatherAPINodeID},
					{Source: WeatherAPINodeID, Target: ConditionNodeID},
					{Source: ConditionNodeID, Target: EmailNodeID, Label: "✓ Condition Met"},
					{Source: EmailNodeID, Target: EndNodeID},
				},
			},
			payload: &ExecutePayload{
				FormData: FormData{
					Name:  "Jane",
					Email: "jane@example.com",
					City:  "Melbourne",
				},
				Condition: Condition{
					Operator:  "equals",
					Threshold: 21.0,
				},
			},
			wantStatus:  StatusCompleted,
			wantStepLen: 6,
			expectErr:   false,
			setup: func() {
				processWeatherNodeFn = func(node Node, payload *ExecutePayload, contextData map[string]any) error {
					contextData["weather.temperature"] = 21.0
					return nil
				}
				processEmailNodeFn = func(node Node, payload *ExecutePayload) error {
					// mock email send success
					return nil
				}
			},
			teardown: func() {
				processWeatherNodeFn = processWeatherNode
				processEmailNodeFn = processEmailNode
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {

			if tt.setup != nil {
				tt.setup()
			}
			defer func() {
				if tt.teardown != nil {
					tt.teardown()
				}
			}()

			got, err := processNodes(tt.workflow, tt.payload)

			if tt.expectErr {
				require.Error(t, err)
				if tt.missingNode != "" {
					require.Contains(t, err.Error(), tt.missingNode)
				}
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				require.Equal(t, tt.wantStatus, got.Status)
				require.Len(t, got.Steps, tt.wantStepLen)
			}
		})
	}
}

func TestProcessConditionNode(t *testing.T) {
	tests := []struct {
		label       string
		node        Node
		payload     *ExecutePayload
		contextData map[string]any
		wantResult  bool
		expectErr   bool
		errContains string
	}{
		{
			label: "greater_than true",
			payload: &ExecutePayload{
				Condition: Condition{
					Operator:  "greater_than",
					Threshold: 10,
				},
			},
			contextData: map[string]any{"weather.temperature": 15.5},
			wantResult:  true,
		},
		{
			label: "greater_than false",
			payload: &ExecutePayload{
				Condition: Condition{
					Operator:  "greater_than",
					Threshold: 20,
				},
			},
			contextData: map[string]any{"weather.temperature": 15.5},
			wantResult:  false,
		},
		{
			label: "less_than true",
			payload: &ExecutePayload{
				Condition: Condition{
					Operator:  "less_than",
					Threshold: 20,
				},
			},
			contextData: map[string]any{"weather.temperature": 15.5},
			wantResult:  true,
		},
		{
			label: "less_than false",
			payload: &ExecutePayload{
				Condition: Condition{
					Operator:  "less_than",
					Threshold: 10,
				},
			},
			contextData: map[string]any{"weather.temperature": 15.5},
			wantResult:  false,
		},
		{
			label: "equals true",
			payload: &ExecutePayload{
				Condition: Condition{
					Operator:  "equals",
					Threshold: 15.5,
				},
			},
			contextData: map[string]any{"weather.temperature": 15.5},
			wantResult:  true,
		},
		{
			label: "equals false",
			payload: &ExecutePayload{
				Condition: Condition{
					Operator:  "equals",
					Threshold: 10,
				},
			},
			contextData: map[string]any{"weather.temperature": 15.5},
			wantResult:  false,
		},
		{
			label: "greater_than_or_equal true (equal)",
			payload: &ExecutePayload{
				Condition: Condition{
					Operator:  "greater_than_or_equal",
					Threshold: 15.5,
				},
			},
			contextData: map[string]any{"weather.temperature": 15.5},
			wantResult:  true,
		},
		{
			label: "greater_than_or_equal true (greater)",
			payload: &ExecutePayload{
				Condition: Condition{
					Operator:  "greater_than_or_equal",
					Threshold: 10,
				},
			},
			contextData: map[string]any{"weather.temperature": 15.5},
			wantResult:  true,
		},
		{
			label: "greater_than_or_equal false",
			payload: &ExecutePayload{
				Condition: Condition{
					Operator:  "greater_than_or_equal",
					Threshold: 20,
				},
			},
			contextData: map[string]any{"weather.temperature": 15.5},
			wantResult:  false,
		},
		{
			label: "less_than_or_equal true (equal)",
			payload: &ExecutePayload{
				Condition: Condition{
					Operator:  "less_than_or_equal",
					Threshold: 15.5,
				},
			},
			contextData: map[string]any{"weather.temperature": 15.5},
			wantResult:  true,
		},
		{
			label: "less_than_or_equal true (less)",
			payload: &ExecutePayload{
				Condition: Condition{
					Operator:  "less_than_or_equal",
					Threshold: 20,
				},
			},
			contextData: map[string]any{"weather.temperature": 15.5},
			wantResult:  true,
		},
		{
			label: "less_than_or_equal false",
			payload: &ExecutePayload{
				Condition: Condition{
					Operator:  "less_than_or_equal",
					Threshold: 10,
				},
			},
			contextData: map[string]any{"weather.temperature": 15.5},
			wantResult:  false,
		},
		{
			label:       "error: missing temperature",
			payload:     &ExecutePayload{},
			contextData: map[string]any{},
			expectErr:   true,
			errContains: "weather temp not in map",
		},
		{
			label:       "error: temperature wrong type",
			payload:     &ExecutePayload{},
			contextData: map[string]any{"weather.temperature": "not a float"},
			expectErr:   true,
			errContains: "weather temp is not a float64",
		},
		{
			label: "error: unsupported operator",
			payload: &ExecutePayload{
				Condition: Condition{
					Operator:  "unsupported",
					Threshold: 10,
				},
			},
			contextData: map[string]any{"weather.temperature": 15.5},
			expectErr:   true,
			errContains: "unsupported operator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got, err := processConditionNode(tt.node, tt.payload, tt.contextData)

			if tt.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantResult, got)
			}
		})
	}
}

func TestProcessFormNode(t *testing.T) {
	tests := []struct {
		label       string
		payload     *ExecutePayload
		expectErr   bool
		errExpected error
	}{
		{
			label: "success: all fields present",
			payload: &ExecutePayload{
				FormData: FormData{
					Name:  "Alice",
					Email: "alice@example.com",
					City:  "Sydney",
				},
			},
			expectErr: false,
		},
		{
			label: "error: missing name",
			payload: &ExecutePayload{
				FormData: FormData{
					Email: "alice@example.com",
					City:  "Sydney",
				},
			},
			expectErr:   true,
			errExpected: ErrMissingFormFieldName,
		},
		{
			label: "error: missing email",
			payload: &ExecutePayload{
				FormData: FormData{
					Name: "Alice",
					City: "Sydney",
				},
			},
			expectErr:   true,
			errExpected: ErrMissingFormFieldEmail,
		},
		{
			label: "error: missing city",
			payload: &ExecutePayload{
				FormData: FormData{
					Name:  "Alice",
					Email: "alice@example.com",
				},
			},
			expectErr:   true,
			errExpected: ErrMissingFormFieldCity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			err := processFormNode(Node{ID: FormNodeID}, tt.payload)
			if tt.expectErr {
				require.Error(t, err)
				require.Equal(t, tt.errExpected, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TODO: Add unit test for the rest of node processors.
