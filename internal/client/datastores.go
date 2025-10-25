package client

import (
	"fmt"
	"time"

	"google.golang.org/api/discoveryengine/v1"
)

// ListDataStores lists all data stores in the project
func (c *GeminiClient) ListDataStores() ([]*DataStore, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", c.config.ProjectID, c.config.Location)
	
	call := c.service.Projects.Locations.DataStores.List(parent)
	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list data stores: %w", err)
	}
	
	var dataStores []*DataStore
	for _, ds := range response.DataStores {
		dataStores = append(dataStores, convertDataStore(ds))
	}
	
	return dataStores, nil
}

// GetDataStoreDetails gets detailed information about a specific data store
func (c *GeminiClient) GetDataStoreDetails(dataStoreName string) (*DataStore, error) {
	call := c.service.Projects.Locations.DataStores.Get(dataStoreName)
	dataStore, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get data store details: %w", err)
	}
	
	return convertDataStore(dataStore), nil
}

// GetDataStoreSchema gets the schema for a data store
func (c *GeminiClient) GetDataStoreSchema(dataStoreName string) (map[string]interface{}, error) {
	schemaName := fmt.Sprintf("%s/schemas/default_schema", dataStoreName)
	call := c.service.Projects.Locations.DataStores.Schemas.Get(schemaName)
	schema, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get data store schema: %w", err)
	}
	
	// Convert schema to map for easier handling
	schemaMap := make(map[string]interface{})
	if schema.Name != "" {
		schemaMap["name"] = schema.Name
	}
	// Note: FieldConfigs field may not exist in current API version
	// if schema.FieldConfigs != nil {
	//     schemaMap["fieldConfigs"] = schema.FieldConfigs
	// }
	
	return schemaMap, nil
}

// CreateDataStoreFromGCS creates a data store and imports data from GCS bucket
func (c *GeminiClient) CreateDataStoreFromGCS(dataStoreID, displayName, gcsURI, dataSchema, reconciliationMode string) (*CreateResult, error) {
	collectionName := fmt.Sprintf("projects/%s/locations/%s/collections/%s",
		c.config.ProjectID, c.config.Location, c.config.Collection)
	
	// Step 1: Create the data store
	dataStoreConfig := &discoveryengine.GoogleCloudDiscoveryengineV1DataStore{
		DisplayName:      displayName,
		IndustryVertical: "GENERIC",
		SolutionTypes:    []string{"SOLUTION_TYPE_SEARCH"},
		ContentConfig:    "CONTENT_REQUIRED",
	}
	
	call := c.service.Projects.Locations.Collections.DataStores.Create(collectionName, dataStoreConfig)
	call.DataStoreId(dataStoreID)
	
	_, err := call.Do()
	if err != nil {
		return &CreateResult{
			Status: "error",
			Error:  fmt.Sprintf("Failed to create data store: %v", err),
		}, nil
	}
	
	// For now, construct the expected data store name
	actualDataStoreName := fmt.Sprintf("projects/%s/locations/%s/collections/%s/dataStores/%s",
		c.config.ProjectID, c.config.Location, c.config.Collection, dataStoreID)
	
	// Step 3: Import documents from GCS
	branchName := fmt.Sprintf("%s/branches/default_branch", actualDataStoreName)
	
	importConfig := &discoveryengine.GoogleCloudDiscoveryengineV1ImportDocumentsRequest{
		GcsSource: &discoveryengine.GoogleCloudDiscoveryengineV1GcsSource{
			InputUris:  []string{gcsURI},
			DataSchema: dataSchema,
		},
		ReconciliationMode: reconciliationMode,
	}
	
	importCall := c.service.Projects.Locations.DataStores.Branches.Documents.Import(branchName, importConfig)
	_, err = importCall.Do()
	if err != nil {
		return &CreateResult{
			Status: "error",
			Error:  fmt.Sprintf("Failed to import documents: %v", err),
		}, nil
	}
	
	return &CreateResult{
		DataStoreName: actualDataStoreName,
		ImportOperation: map[string]interface{}{
			"name": "import-operation",
		},
		Status: "success",
	}, nil
}

// ListDocuments lists documents in a data store branch
func (c *GeminiClient) ListDocuments(dataStoreName, branch string) ([]*Document, error) {
	branchName := fmt.Sprintf("%s/branches/%s", dataStoreName, branch)
	
	call := c.service.Projects.Locations.DataStores.Branches.Documents.List(branchName)
	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	
	var documents []*Document
	for _, doc := range response.Documents {
		documents = append(documents, convertDocument(doc))
	}
	
	return documents, nil
}

// DeleteDataStore deletes a data store
func (c *GeminiClient) DeleteDataStore(dataStoreName string) (*DeleteResult, error) {
	call := c.service.Projects.Locations.DataStores.Delete(dataStoreName)
	_, err := call.Do()
	if err != nil {
		return &DeleteResult{
			Status:  "error",
			Message: fmt.Sprintf("Failed to delete data store: %v", err),
		}, nil
	}
	
	return &DeleteResult{
		Status:  "success",
		Message: "Data store deleted successfully",
	}, nil
}

// waitForDataStoreCreation waits for data store creation operation to complete
func (c *GeminiClient) waitForDataStoreCreation(operationName, dataStoreID string) (string, error) {
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
				return "", fmt.Errorf("data store creation failed: %v", operation.Error)
			}
			
		// Extract data store name from the response
		if operation.Response != nil {
			// Response is a byte slice, we need to handle it differently
			// For now, construct the expected name
			return fmt.Sprintf("projects/%s/locations/%s/collections/%s/dataStores/%s",
				c.config.ProjectID, c.config.Location, c.config.Collection, dataStoreID), nil
		}
			
			// Fallback: construct the expected data store name
			return fmt.Sprintf("projects/%s/locations/%s/collections/%s/dataStores/%s",
				c.config.ProjectID, c.config.Location, c.config.Collection, dataStoreID), nil
		}
		
		time.Sleep(checkInterval)
	}
	
	return "", fmt.Errorf("timeout waiting for data store creation")
}

// convertDataStore converts a Discovery Engine API data store to our DataStore struct
func convertDataStore(ds *discoveryengine.GoogleCloudDiscoveryengineV1DataStore) *DataStore {
	result := &DataStore{
		Name:             ds.Name,
		DisplayName:      ds.DisplayName,
		IndustryVertical: ds.IndustryVertical,
		ContentConfig:    ds.ContentConfig,
		CreateTime:       ds.CreateTime,
		SolutionTypes:    ds.SolutionTypes,
		AclEnabled:       ds.AclEnabled,
		DocumentProcessingConfig: make(map[string]interface{}),
	}
	
	if ds.BillingEstimation != nil {
		result.BillingEstimation = &BillingEstimation{
			UnstructuredDataSize:       ds.BillingEstimation.UnstructuredDataSize,
			UnstructuredDataUpdateTime: ds.BillingEstimation.UnstructuredDataUpdateTime,
		}
	}
	
	return result
}

// convertDocument converts a Discovery Engine API document to our Document struct
func convertDocument(doc *discoveryengine.GoogleCloudDiscoveryengineV1Document) *Document {
	return &Document{
		ID:        doc.Id,
		Content:   make(map[string]interface{}),
		IndexTime: doc.IndexTime,
	}
}
