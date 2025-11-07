package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"google.golang.org/api/googleapi"
)

const defaultAssistantID = "default_assistant"

// Agent represents a Gemini Enterprise Dialogflow agent registration
type Agent struct {
	Name                      string                     `json:"name,omitempty"`
	DisplayName               string                     `json:"displayName,omitempty"`
	Description               string                     `json:"description,omitempty"`
	Icon                      *AgentIcon                 `json:"icon,omitempty"`
	DialogflowAgentDefinition *DialogflowAgentDefinition `json:"dialogflowAgentDefinition,omitempty"`
	ReasoningEngine           string                     `json:"reasoningEngine,omitempty"`
	CreateTime                string                     `json:"createTime,omitempty"`
	UpdateTime                string                     `json:"updateTime,omitempty"`
	ConnectorDefinition       map[string]interface{}     `json:"connectorDefinition,omitempty"`
	AdditionalAgentProperties map[string]interface{}     `json:"additionalAgentProperties,omitempty"`
	Annotations               map[string]string          `json:"annotations,omitempty"`
	Labels                    map[string]string          `json:"labels,omitempty"`
	EnvironmentConfigurations []map[string]interface{}   `json:"environmentConfigurations,omitempty"`
	AgentMonitoringState      map[string]interface{}     `json:"agentMonitoringState,omitempty"`
	Capabilities              []string                   `json:"capabilities,omitempty"`
	Metadata                  map[string]interface{}     `json:"metadata,omitempty"`
}

// AgentIcon represents the icon associated with an agent
type AgentIcon struct {
	URI     string `json:"uri,omitempty"`
	Content string `json:"content,omitempty"`
}

// DialogflowAgentDefinition represents Dialogflow linkage details
type DialogflowAgentDefinition struct {
	DialogflowAgent string `json:"dialogflowAgent,omitempty"`
}

// AgentCreateInput represents the payload for creating an agent
type AgentCreateInput struct {
	DisplayName               string                     `json:"displayName,omitempty"`
	Description               string                     `json:"description,omitempty"`
	Icon                      *AgentIcon                 `json:"icon,omitempty"`
	DialogflowAgentDefinition *DialogflowAgentDefinition `json:"dialogflowAgentDefinition,omitempty"`
	ReasoningEngine           string                     `json:"reasoningEngine,omitempty"`
}

// AgentUpdateInput represents the payload for updating an agent
type AgentUpdateInput struct {
	DisplayName               string                     `json:"displayName,omitempty"`
	Description               string                     `json:"description,omitempty"`
	Icon                      *AgentIcon                 `json:"icon,omitempty"`
	DialogflowAgentDefinition *DialogflowAgentDefinition `json:"dialogflowAgentDefinition,omitempty"`
	ReasoningEngine           string                     `json:"reasoningEngine,omitempty"`
}

// ListAgents retrieves all agents registered to a given engine assistant
func (c *GeminiClient) ListAgents(engineName string) ([]*Agent, error) {
	url, err := c.agentCollectionURL(engineName)
	if err != nil {
		return nil, err
	}

	body, err := c.doAgentsRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Agents []*Agent `json:"agents"`
	}

	if len(body) == 0 {
		return []*Agent{}, nil
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to decode agents response: %w", err)
	}

	if response.Agents == nil {
		return []*Agent{}, nil
	}

	return response.Agents, nil
}

// GetAgent retrieves details for a specific agent registration
func (c *GeminiClient) GetAgent(agentName string) (*Agent, error) {
	url, err := c.agentResourceURL(agentName)
	if err != nil {
		return nil, err
	}

	body, err := c.doAgentsRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if len(body) == 0 {
		return nil, nil
	}

	var agent Agent
	if err := json.Unmarshal(body, &agent); err != nil {
		return nil, fmt.Errorf("failed to decode agent response: %w", err)
	}

	return &agent, nil
}

// CreateAgent registers a Dialogflow agent against the engine assistant
func (c *GeminiClient) CreateAgent(engineName string, input *AgentCreateInput) (*Agent, error) {
	if input == nil {
		return nil, fmt.Errorf("agent create payload is required")
	}

	url, err := c.agentCollectionURL(engineName)
	if err != nil {
		return nil, err
	}

	body, err := c.doAgentsRequest(http.MethodPost, url, input)
	if err != nil {
		return nil, err
	}

	var agent Agent
	if err := json.Unmarshal(body, &agent); err != nil {
		return nil, fmt.Errorf("failed to decode created agent: %w", err)
	}

	return &agent, nil
}

// UpdateAgent updates a specific agent registration using the provided update mask
func (c *GeminiClient) UpdateAgent(agentName string, input *AgentUpdateInput, updateMask []string) (*Agent, error) {
	if input == nil {
		return nil, fmt.Errorf("agent update payload is required")
	}

	url, err := c.agentResourceURL(agentName)
	if err != nil {
		return nil, err
	}

	if len(updateMask) > 0 {
		query := urlValuesFromMask(updateMask)
		if strings.Contains(url, "?") {
			url = fmt.Sprintf("%s&%s", url, query.Encode())
		} else {
			url = fmt.Sprintf("%s?%s", url, query.Encode())
		}
	}

	body, err := c.doAgentsRequest(http.MethodPatch, url, input)
	if err != nil {
		return nil, err
	}

	var agent Agent
	if err := json.Unmarshal(body, &agent); err != nil {
		return nil, fmt.Errorf("failed to decode updated agent: %w", err)
	}

	return &agent, nil
}

// DeleteAgent removes the agent registration from the engine assistant
func (c *GeminiClient) DeleteAgent(agentName string) (*DeleteResult, error) {
	url, err := c.agentResourceURL(agentName)
	if err != nil {
		return nil, err
	}

	_, err = c.doAgentsRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	return &DeleteResult{
		Status:  "success",
		Message: "Agent deleted successfully",
	}, nil
}

// ConstructAgentName constructs the fully-qualified agent resource name
func ConstructAgentName(engineName, agentID string) string {
	if strings.Contains(agentID, "/") {
		return agentID
	}
	return fmt.Sprintf("%s/assistants/%s/agents/%s", engineName, defaultAssistantID, agentID)
}

func (c *GeminiClient) agentCollectionURL(engineName string) (string, error) {
	engineName = strings.TrimPrefix(engineName, "/")
	if engineName == "" {
		return "", fmt.Errorf("engine name is required")
	}

	base := strings.TrimRight(c.service.BasePath, "/")
	return fmt.Sprintf("%s/v1alpha/%s/assistants/%s/agents", base, engineName, defaultAssistantID), nil
}

func (c *GeminiClient) agentResourceURL(agentName string) (string, error) {
	agentName = strings.TrimPrefix(agentName, "/")
	if agentName == "" {
		return "", fmt.Errorf("agent name is required")
	}

	base := strings.TrimRight(c.service.BasePath, "/")
	return fmt.Sprintf("%s/v1alpha/%s", base, agentName), nil
}

func (c *GeminiClient) doAgentsRequest(method, urlStr string, payload interface{}) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(payload); err != nil {
			return nil, fmt.Errorf("failed to encode agent payload: %w", err)
		}
		body = buf
	}

	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if ua := c.agentsUserAgent(); ua != "" {
		req.Header.Set("User-Agent", ua)
	}

	if c.config != nil && c.config.ProjectID != "" {
		req.Header.Set("X-Goog-User-Project", c.config.ProjectID)
	}

	client := c.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	req = req.WithContext(context.Background())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("agents API request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read agents API response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("agents API %s %s returned status %d: %s",
			method, urlStr, resp.StatusCode, strings.TrimSpace(string(data)))
	}

	return data, nil
}

func (c *GeminiClient) agentsUserAgent() string {
	if c.service == nil {
		return googleapi.UserAgent
	}
	if c.service.UserAgent == "" {
		return googleapi.UserAgent
	}
	return googleapi.UserAgent + " " + c.service.UserAgent
}

func urlValuesFromMask(fields []string) url.Values {
	values := url.Values{}
	if len(fields) == 0 {
		return values
	}
	values.Set("updateMask", strings.Join(fields, ","))
	return values
}
