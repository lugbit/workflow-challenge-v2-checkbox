package workflow

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// execution result structs
type ExecutionResult struct {
	ExecutedAt string       `json:"executedAt"`
	Status     string       `json:"status"`
	Steps      []StepResult `json:"steps"`
}

type StepResult struct {
	NodeID      string                 `json:"nodeId"`
	Type        string                 `json:"type"`
	Label       string                 `json:"label"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	Output      map[string]interface{} `json:"output,omitempty"`
}

const (
	// valid node IDs (types)
	StartNodeID      = "start"
	EndNodeID        = "end"
	FormNodeID       = "form"
	WeatherAPINodeID = "weather-api"
	ConditionNodeID  = "condition"
	EmailNodeID      = "email"

	// node status
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// processNodes processes each node in sequence from the workflow.
func processNodes(wf *WorkflowDefinition, payload *ExecutePayload) (*ExecutionResult, error) {
	// record the each node execution in steps
	steps := []StepResult{}
	// this stores node outputs (e.g temperature from the weather check node)
	contextData := make(map[string]any)

	nodeMap := make(map[string]Node)
	for _, node := range wf.Nodes {
		nodeMap[node.ID] = node
	}

	// validate that the workflow graph contains start and end nodes
	if _, ok := nodeMap[StartNodeID]; !ok {
		return nil, ErrMissingStartNode
	}
	if _, ok := nodeMap[EndNodeID]; !ok {
		return nil, ErrMissingEndNode
	}

	// build adjacency map: sourceID > list of targetIDs
	adj := make(map[string][]string)
	for _, edge := range wf.Edges {
		adj[edge.Source] = append(adj[edge.Source], edge.Target)
	}

	visited := make(map[string]bool)

	var traverse func(id string) error
	traverse = func(id string) error {
		if visited[id] {
			return nil
		}
		visited[id] = true

		node, ok := nodeMap[id]
		if !ok {
			return fmt.Errorf("node %s not found in nodeMap", id)
		}

		switch node.ID {
		case StartNodeID:
			startTime := time.Now()
			err := processStartNode(node)
			duration := time.Since(startTime).Milliseconds()

			if err != nil {
				steps = append(steps, StepResult{
					NodeID:      node.ID,
					Type:        node.Type,
					Label:       node.Data.Label,
					Description: node.Data.Description,
					Status:      StatusFailed,
					Output: map[string]interface{}{
						"error":    err.Error(),
						"duration": duration,
					},
				})
				return nil
			}

			output := map[string]interface{}{
				"duration": duration,
			}

			steps = append(steps, StepResult{
				NodeID:      node.ID,
				Type:        node.Type,
				Label:       node.Data.Label,
				Description: node.Data.Description,
				Status:      StatusCompleted,
				Output:      output,
			})

			fmt.Println(node.Data.Description)
		case EndNodeID:
			startTime := time.Now()
			err := processEndNode(node)
			duration := time.Since(startTime).Milliseconds()

			if err != nil {
				steps = append(steps, StepResult{
					NodeID:      node.ID,
					Type:        node.Type,
					Label:       node.Data.Label,
					Description: node.Data.Description,
					Status:      StatusFailed,
					Output: map[string]interface{}{
						"error":    err.Error(),
						"duration": duration,
					},
				})
				return nil
			}

			output := map[string]interface{}{
				"duration": duration,
			}

			steps = append(steps, StepResult{
				NodeID:      node.ID,
				Type:        node.Type,
				Label:       node.Data.Label,
				Description: node.Data.Description,
				Status:      StatusCompleted,
				Output:      output,
			})
		case FormNodeID:
			startTime := time.Now()
			err := processFormNode(node, payload)
			duration := time.Since(startTime).Milliseconds()

			// if there's an error with this node execution, record as failed step
			if err != nil {
				steps = append(steps, StepResult{
					NodeID:      node.ID,
					Type:        node.Type,
					Label:       node.Data.Label,
					Description: node.Data.Description,
					Status:      StatusFailed,
					Output: map[string]interface{}{
						"error":    err.Error(),
						"duration": duration,
					},
				})
				return nil
			}

			// success
			output := map[string]interface{}{
				"name":     payload.FormData.Name,
				"email":    payload.FormData.Email,
				"city":     payload.FormData.City,
				"duration": duration,
			}
			steps = append(steps, StepResult{
				NodeID:      node.ID,
				Type:        node.Type,
				Label:       node.Data.Label,
				Description: node.Data.Description,
				Status:      StatusCompleted,
				Output:      output,
			})
		case WeatherAPINodeID:
			startTime := time.Now()
			err := processWeatherNode(node, payload, contextData)
			duration := time.Since(startTime).Milliseconds()

			if err != nil {
				steps = append(steps, StepResult{
					NodeID:      node.ID,
					Type:        node.Type,
					Label:       node.Data.Label,
					Description: node.Data.Description,
					Status:      StatusFailed,
					Output: map[string]interface{}{
						"error":    err.Error(),
						"duration": duration,
					},
				})
				return nil
			}

			// success
			output := map[string]interface{}{
				"temperature": contextData["weather.temperature"],
				"location":    payload.FormData.City,
				"duration":    duration,
			}
			steps = append(steps, StepResult{
				NodeID:      node.ID,
				Type:        node.Type,
				Label:       node.Data.Label,
				Description: node.Data.Description,
				Status:      StatusCompleted,
				Output:      output,
			})
		case ConditionNodeID:
			startTime := time.Now()
			conditionMet, err := processConditionNode(node, payload, contextData)
			duration := time.Since(startTime).Milliseconds()

			if err != nil {
				steps = append(steps, StepResult{
					NodeID:      node.ID,
					Type:        node.Type,
					Label:       node.Data.Label,
					Description: node.Data.Description,
					Status:      StatusFailed,
					Output: map[string]interface{}{
						"error":    err.Error(),
						"duration": duration,
					},
				})
				return nil
			}

			// success
			operatorReadable := strings.ReplaceAll(payload.Condition.Operator, "_", " ")
			actualValue := contextData["weather.temperature"].(float64)
			threshold := payload.Condition.Threshold

			conditionText := "condition not met"
			if conditionMet {
				conditionText = "condition met"
			}

			output := map[string]interface{}{
				"conditionMet": conditionMet,
				"threshold":    payload.Condition.Threshold,
				"operator":     payload.Condition.Operator,
				"actualValue":  contextData["weather.temperature"],
				"message":      fmt.Sprintf("Temperature %.1f°C is %s %.1f°C - %s", actualValue, operatorReadable, threshold, conditionText),
				"duration":     duration,
			}
			steps = append(steps, StepResult{
				NodeID:      node.ID,
				Type:        node.Type,
				Label:       node.Data.Label,
				Description: node.Data.Description,
				Status:      StatusCompleted,
				Output:      output,
			})

			// route based on conditionMet and edge label
			for _, edge := range wf.Edges {
				if edge.Source != node.ID {
					continue
				}
				if conditionMet && edge.Label == "✓ Condition Met" {
					return traverse(edge.Target)
				}
				if !conditionMet && edge.Label == "✗ No Alert Needed" {
					return traverse(edge.Target)
				}
			}
			return fmt.Errorf("no matching conditional edge for node %s", node.ID)
		case EmailNodeID:
			startTime := time.Now()
			err := processEmailNode(node, payload, contextData)
			duration := time.Since(startTime).Milliseconds()

			if err != nil {
				steps = append(steps, StepResult{
					NodeID:      node.ID,
					Type:        node.Type,
					Label:       node.Data.Label,
					Description: node.Data.Description,
					Status:      StatusFailed,
					Output: map[string]interface{}{
						"error":    err.Error(),
						"duration": duration,
					},
				})
				return nil
			}

			// mock email output
			output := map[string]interface{}{
				"emailDraft": map[string]interface{}{
					"to":      payload.FormData.Email,
					"from":    "weather-alerts@example.com",
					"subject": node.Data.Metadata.EmailTemplate.Subject,
					"body": strings.ReplaceAll(
						strings.ReplaceAll(
							node.Data.Metadata.EmailTemplate.Body,
							"{{city}}", payload.FormData.City,
						),
						"{{temperature}}", fmt.Sprintf("%.1f", contextData["weather.temperature"]),
					),
					"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
				},
				"deliveryStatus": "sent",
				"messageId":      "msg_abc123def456",
				"emailSent":      true,
				"duration":       duration,
			}

			steps = append(steps, StepResult{
				NodeID:      node.ID,
				Type:        node.Type,
				Label:       node.Data.Label,
				Description: node.Data.Description,
				Status:      StatusCompleted,
				Output:      output,
			})
		}

		// Default linear traversal
		for _, next := range adj[id] {
			if err := traverse(next); err != nil {
				return err
			}
		}
		return nil
	}

	if err := traverse("start"); err != nil {
		return &ExecutionResult{
			ExecutedAt: time.Now().UTC().Format(time.RFC3339Nano),
			Status:     StatusFailed,
			Steps:      steps,
		}, err
	}

	return &ExecutionResult{
		ExecutedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Status:     StatusCompleted,
		Steps:      steps,
	}, nil
}

// node handlers

// processStartNode doesn't do much but custom logic can be added later (e.g metrics?)
func processStartNode(node Node) error {
	slog.Debug("Processing node", "node id", node.ID)
	return nil
}

// processEndNode is similar to the the start node
func processEndNode(node Node) error {
	slog.Debug("Processing node", "node id", node.ID)
	return nil
}

// processFormNode ensures the required fields are not empty.
func processFormNode(node Node, payload *ExecutePayload) error {
	slog.Debug("Processing node", "node id", node.ID)

	if payload.FormData.Name == "" {
		return ErrMissingFormFieldName
	}
	if payload.FormData.Email == "" {
		return ErrMissingFormFieldEmail
	}
	// can also add to check email is in email format
	if payload.FormData.City == "" {
		return ErrMissingFormFieldCity
	}

	return nil
}

// structs for geocoding response
type GeoCodingResponse struct {
	Results []struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"results"`
}

// struct for Open-Meteo weather response
type WeatherResponse struct {
	CurrentWeather struct {
		Temperature float64 `json:"temperature"`
	} `json:"current_weather"`
}

// processWeatherNode calls an external API to retrieve the current weather for the input city.
func processWeatherNode(node Node, payload *ExecutePayload, contextData map[string]any) error {
	slog.Debug("Processing node", "node id", node.ID)

	city := payload.FormData.City
	if city == "" {
		return ErrMissingFormFieldCity
	}

	// get coordinates from city (required in the weather check API)
	geoURL := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1", city)
	resp, err := http.Get(geoURL)
	if err != nil {
		return fmt.Errorf("geocoding API request failed: %w", err)
	}
	defer resp.Body.Close()

	var geoData GeoCodingResponse
	if err := json.NewDecoder(resp.Body).Decode(&geoData); err != nil {
		return ErrResponseDecodeFailed
	}
	if len(geoData.Results) == 0 {
		return fmt.Errorf("no results found for city: %s", city)
	}

	lat := geoData.Results[0].Latitude
	lon := geoData.Results[0].Longitude

	// replace placeholders in definition API URL
	apiEndpoint := node.Data.Metadata.APIEndpoint
	apiEndpoint = strings.ReplaceAll(apiEndpoint, "{lat}", fmt.Sprintf("%f", lat))
	apiEndpoint = strings.ReplaceAll(apiEndpoint, "{lon}", fmt.Sprintf("%f", lon))

	// fetch weather data from API URL
	weatherResp, err := http.Get(apiEndpoint)
	if err != nil {
		return fmt.Errorf("failed to fetch weather data: %w", err)
	}
	defer weatherResp.Body.Close()

	if weatherResp.StatusCode != http.StatusOK {
		return fmt.Errorf("weather API returned status: %d", weatherResp.StatusCode)
	}

	var weather WeatherResponse
	if err := json.NewDecoder(weatherResp.Body).Decode(&weather); err != nil {
		return ErrResponseDecodeFailed
	}

	// put temperature to contextData map
	contextData["weather.temperature"] = weather.CurrentWeather.Temperature

	return nil
}

func processConditionNode(node Node, payload *ExecutePayload, contextData map[string]any) (bool, error) {
	slog.Debug("Processing node", "node id", node.ID)

	tempVal, ok := contextData["weather.temperature"]
	if !ok {
		return false, fmt.Errorf("weather temp not in map")
	}

	temperature, ok := tempVal.(float64)
	if !ok {
		return false, fmt.Errorf("weather temp is not a float64")
	}

	operator := payload.Condition.Operator
	threshold := payload.Condition.Threshold

	switch operator {
	case "greater_than":
		return temperature > threshold, nil
	case "less_than":
		return temperature < threshold, nil
	case "equals":
		return temperature == threshold, nil
	case "greater_than_or_equal":
		return temperature >= threshold, nil
	case "less_than_or_equal":
		return temperature <= threshold, nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

// processEmailNode is suppose to send emails but this is just a placeholder as no live emails are sent.
func processEmailNode(node Node, payload *ExecutePayload, contextData map[string]any) error {
	slog.Debug("Processing node", "node id", node.ID)
	slog.Debug("Sending email", "email", payload.FormData.Email)

	return nil
}
