package client

import (
	"fmt"
	"time"

	"google.golang.org/api/discoveryengine/v1"
)

// ListEngines lists all engines in a collection
func (c *GeminiClient) ListEngines(collectionID string) ([]*Engine, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s/collections/%s",
		c.config.ProjectID, c.config.Location, collectionID)

	call := c.service.Projects.Locations.Collections.Engines.List(parent)
	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list engines: %w", err)
	}

	var engines []*Engine
	for _, engine := range response.Engines {
		engines = append(engines, convertEngine(engine))
	}

	return engines, nil
}

// GetEngineDetails gets detailed information about a specific engine
func (c *GeminiClient) GetEngineDetails(engineName string) (*Engine, error) {
	call := c.service.Projects.Locations.Collections.Engines.Get(engineName)
	engine, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get engine details: %w", err)
	}

	return convertEngine(engine), nil
}

// GetEngineFullConfig gets complete configuration for an engine including all data stores
func (c *GeminiClient) GetEngineFullConfig(engineName string) (map[string]interface{}, error) {
	engine, err := c.GetEngineDetails(engineName)
	if err != nil {
		return nil, err
	}

	config := map[string]interface{}{
		"engine":      engine,
		"data_stores": []interface{}{},
	}

	// Get details for each data store
	for _, dsID := range engine.DataStoreIds {
		dsName := fmt.Sprintf("projects/%s/locations/%s/collections/%s/dataStores/%s",
			c.config.ProjectID, c.config.Location, c.config.Collection, dsID)

		ds, err := c.GetDataStoreDetails(dsName)
		if err != nil {
			continue // Skip failed data stores
		}

		// Try to get schema as well
		schema, err := c.GetDataStoreSchema(dsName)
		if err == nil && schema != nil {
			ds.Schema = schema
		}

		config["data_stores"] = append(config["data_stores"].([]interface{}), ds)
	}

	return config, nil
}

// CreateSearchEngine creates a search engine connected to data stores
func (c *GeminiClient) CreateSearchEngine(engineID, displayName string, dataStoreIDs []string, searchTier string) (*CreateResult, error) {
	collectionName := fmt.Sprintf("projects/%s/locations/%s/collections/%s",
		c.config.ProjectID, c.config.Location, c.config.Collection)

	engineConfig := &discoveryengine.GoogleCloudDiscoveryengineV1Engine{
		DisplayName:      displayName,
		SolutionType:     "SOLUTION_TYPE_SEARCH",
		IndustryVertical: "GENERIC",
		AppType:          "APP_TYPE_INTRANET",
		CommonConfig: &discoveryengine.GoogleCloudDiscoveryengineV1EngineCommonConfig{
			CompanyName: "BCBSMA",
		},
	}

	// Only add dataStoreIds if data stores are provided
	if len(dataStoreIDs) > 0 {
		engineConfig.DataStoreIds = dataStoreIDs
	}

	call := c.service.Projects.Locations.Collections.Engines.Create(collectionName, engineConfig)
	call.EngineId(engineID)

	_, err := call.Do()
	if err != nil {
		return &CreateResult{
			Status: "error",
			Error:  fmt.Sprintf("Failed to create engine: %v", err),
		}, nil
	}

	// For now, construct the expected engine name
	actualEngineName := fmt.Sprintf("projects/%s/locations/%s/collections/%s/engines/%s",
		c.config.ProjectID, c.config.Location, c.config.Collection, engineID)

	return &CreateResult{
		EngineName: actualEngineName,
		Status:     "success",
	}, nil
}

// DeleteEngine deletes a search engine
func (c *GeminiClient) DeleteEngine(engineName string) (*DeleteResult, error) {
	call := c.service.Projects.Locations.Collections.Engines.Delete(engineName)
	_, err := call.Do()
	if err != nil {
		return &DeleteResult{
			Status:  "error",
			Message: fmt.Sprintf("Failed to delete engine: %v", err),
		}, nil
	}

	return &DeleteResult{
		Status:  "success",
		Message: "Engine deleted successfully",
	}, nil
}

const (
	engineFeatureStateOn  = "FEATURE_STATE_ON"
	engineFeatureStateOff = "FEATURE_STATE_OFF"
)

const (
	// EngineFeatureStateOn represents enabled feature state in Discovery Engine.
	EngineFeatureStateOn = engineFeatureStateOn
	// EngineFeatureStateOff represents disabled feature state in Discovery Engine.
	EngineFeatureStateOff = engineFeatureStateOff
)

// UpdateEngineFeatures toggles engine features while preserving unspecified states.
func (c *GeminiClient) UpdateEngineFeatures(engineName string, updates map[string]string) (*Engine, error) {
	if len(updates) == 0 {
		return nil, fmt.Errorf("no feature updates provided")
	}

	current, err := c.GetEngineDetails(engineName)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch engine before update: %w", err)
	}

	currentFeatures := map[string]string{}
	if current != nil && current.Features != nil {
		for key, value := range current.Features {
			currentFeatures[key] = value
		}
	}

	for key, value := range updates {
		currentFeatures[key] = value
	}

	enginePatch := &discoveryengine.GoogleCloudDiscoveryengineV1Engine{
		Features: currentFeatures,
	}

	call := c.service.Projects.Locations.Collections.Engines.Patch(engineName, enginePatch)
	call.UpdateMask("features")

	updatedEngine, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update engine features: %w", err)
	}

	return convertEngine(updatedEngine), nil
}

// waitForEngineCreation waits for engine creation operation to complete
func (c *GeminiClient) waitForEngineCreation(operationName, engineID string) (string, error) {
	maxWaitTime := 5 * time.Minute
	checkInterval := 5 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < maxWaitTime {
		operation, err := c.service.Projects.Locations.Operations.Get(operationName).Do()
		if err != nil {
			return "", fmt.Errorf("failed to check operation status: %w", err)
		}

		if operation.Done {
			if operation.Error != nil {
				return "", fmt.Errorf("engine creation failed: %v", operation.Error)
			}

			// Extract engine name from the response
			if operation.Response != nil {
				// Response is a byte slice, we need to handle it differently
				// For now, construct the expected name
				return fmt.Sprintf("projects/%s/locations/%s/collections/%s/engines/%s",
					c.config.ProjectID, c.config.Location, c.config.Collection, engineID), nil
			}

			// Fallback: construct the expected engine name
			return fmt.Sprintf("projects/%s/locations/%s/collections/%s/engines/%s",
				c.config.ProjectID, c.config.Location, c.config.Collection, engineID), nil
		}

		time.Sleep(checkInterval)
	}

	return "", fmt.Errorf("timeout waiting for engine creation")
}

// convertEngine converts a Discovery Engine API engine to our Engine struct
func convertEngine(engine *discoveryengine.GoogleCloudDiscoveryengineV1Engine) *Engine {
	result := &Engine{
		Name:             engine.Name,
		DisplayName:      engine.DisplayName,
		SolutionType:     engine.SolutionType,
		IndustryVertical: engine.IndustryVertical,
		AppType:          engine.AppType,
		CreateTime:       engine.CreateTime,
		DataStoreIds:     engine.DataStoreIds,
		CommonConfig:     make(map[string]interface{}),
		Features:         engine.Features,
	}

	// Convert CommonConfig if it exists
	if engine.CommonConfig != nil {
		result.CommonConfig["companyName"] = engine.CommonConfig.CompanyName
	}

	return result
}
