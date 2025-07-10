package workflow

// this file node.go contains the the struct definition of the workflow graph (nodes and edges).

// worflow definition holds the id and nodes + edges
type WorkflowDefinition struct {
	ID    string `json:"id"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Node struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Position Position `json:"position"`
	Data     NodeData `json:"data"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type NodeData struct {
	Label       string       `json:"label"`
	Description string       `json:"description"`
	Metadata    NodeMetadata `json:"metadata"`
}

type NodeMetadata struct {
	HasHandles      HasHandles        `json:"hasHandles"`
	InputFields     []string          `json:"inputFields,omitempty"`
	OutputVariables []string          `json:"outputVariables,omitempty"`
	InputVariables  []string          `json:"inputVariables,omitempty"`
	EmailTemplate   *EmailTemplate    `json:"emailTemplate,omitempty"`
	APIEndpoint     string            `json:"apiEndpoint,omitempty"`
	Options         []CityCoordinates `json:"options,omitempty"`
	ConditionExpr   string            `json:"conditionExpression,omitempty"`
}

type HasHandles struct {
	Source interface{} `json:"source"` // Can be bool or []string
	Target interface{} `json:"target"` // Can be bool or string
}

type EmailTemplate struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type CityCoordinates struct {
	City string  `json:"city"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
}

type Edge struct {
	ID           string                 `json:"id"`
	Source       string                 `json:"source"`
	Target       string                 `json:"target"`
	Type         string                 `json:"type"`
	Animated     bool                   `json:"animated"`
	SourceHandle string                 `json:"sourceHandle,omitempty"`
	Style        map[string]interface{} `json:"style"`
	Label        string                 `json:"label,omitempty"`
	LabelStyle   map[string]interface{} `json:"labelStyle,omitempty"`
}
