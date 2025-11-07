package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vb140772/gemctl-go/internal/client"
	"gopkg.in/yaml.v3"
)

// getConfigFromFlags extracts configuration from command flags
func getConfigFromFlags(cmd *cobra.Command) (*client.Config, error) {
	config := &client.Config{}

	// Get project ID
	projectID, err := cmd.Flags().GetString("project")
	if err != nil {
		return nil, fmt.Errorf("failed to get project flag: %w", err)
	}
	config.ProjectID = projectID

	// Get location
	location, err := cmd.Flags().GetString("location")
	if err != nil {
		return nil, fmt.Errorf("failed to get location flag: %w", err)
	}
	config.Location = location

	// Get collection
	collection, err := cmd.Flags().GetString("collection")
	if err != nil {
		return nil, fmt.Errorf("failed to get collection flag: %w", err)
	}
	config.Collection = collection

	// Get format
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return nil, fmt.Errorf("failed to get format flag: %w", err)
	}
	config.Format = format

	// Get use service account flag
	useServiceAccount, err := cmd.Flags().GetBool("use-service-account")
	if err != nil {
		return nil, fmt.Errorf("failed to get use-service-account flag: %w", err)
	}
	config.UseServiceAccount = useServiceAccount

	return config, nil
}

// outputEngines outputs engines in the specified format
func outputEngines(engines []*client.Engine, format string) error {
	switch format {
	case "json":
		return outputJSON(engines, format)
	case "yaml":
		return outputYAML(engines)
	default:
		return outputEnginesTable(engines)
	}
}

// outputEnginesTable outputs engines in table format
func outputEnginesTable(engines []*client.Engine) error {
	if len(engines) == 0 {
		fmt.Println("No engines found.")
		return nil
	}

	fmt.Println("=" + strings.Repeat("=", 100))
	fmt.Printf("%-60s %-30s %-10s\n", "NAME", "DISPLAY NAME", "TYPE")
	fmt.Println("=" + strings.Repeat("=", 100))

	for _, engine := range engines {
		name := extractResourceID(engine.Name)
		displayName := engine.DisplayName
		solutionType := strings.Replace(engine.SolutionType, "SOLUTION_TYPE_", "", 1)

		fmt.Printf("%-60s %-30s %-10s\n", name, displayName, solutionType)
	}

	fmt.Printf("\nTotal: %d engine(s)\n", len(engines))
	return nil
}

// outputEngineDetails outputs engine details in the specified format
func outputEngineDetails(engine *client.Engine, format string) error {
	switch format {
	case "json":
		return outputJSON(engine, format)
	case "yaml":
		return outputYAML(engine)
	default:
		return outputEngineDetailsTable(engine)
	}
}

// outputEngineDetailsTable outputs engine details in table format
func outputEngineDetailsTable(engine *client.Engine) error {
	fmt.Println("=" + strings.Repeat("=", 80))
	fmt.Printf("Engine: %s\n", engine.DisplayName)
	fmt.Println("=" + strings.Repeat("=", 80))
	fmt.Printf("Name: %s\n", engine.Name)
	fmt.Printf("Solution Type: %s\n", engine.SolutionType)
	fmt.Printf("Industry Vertical: %s\n", engine.IndustryVertical)
	fmt.Printf("App Type: %s\n", engine.AppType)

	if engine.CommonConfig != nil {
		fmt.Println("\nCommon Config:")
		for key, value := range engine.CommonConfig {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	if engine.SearchEngineConfig != nil {
		fmt.Println("\nSearch Config:")
		fmt.Printf("  Search Tier: %s\n", engine.SearchEngineConfig.SearchTier)
		if len(engine.SearchEngineConfig.SearchAddOns) > 0 {
			fmt.Printf("  Search Add-ons: %s\n", strings.Join(engine.SearchEngineConfig.SearchAddOns, ", "))
		}
	}

	if len(engine.DataStoreIds) > 0 {
		fmt.Printf("\nData Stores (%d):\n", len(engine.DataStoreIds))
		for _, dsID := range engine.DataStoreIds {
			fmt.Printf("  - %s\n", dsID)
		}
	}

	if engine.Features != nil {
		featuresOn := make([]string, 0)
		for k, v := range engine.Features {
			if strings.Contains(v, "ON") {
				featuresOn = append(featuresOn, k)
			}
		}
		fmt.Printf("\nFeatures (%d/%d enabled):\n", len(featuresOn), len(engine.Features))
		for _, feature := range featuresOn {
			fmt.Printf("  ✓ %s\n", feature)
		}
	}

	return nil
}

// outputAgents outputs agents in the specified format
func outputAgents(agents []*client.Agent, format string) error {
	switch format {
	case "json":
		return outputJSON(agents, format)
	case "yaml":
		return outputYAML(agents)
	default:
		return outputAgentsTable(agents)
	}
}

// outputAgentsTable outputs agents in table format
func outputAgentsTable(agents []*client.Agent) error {
	if len(agents) == 0 {
		fmt.Println("No agents registered for this engine.")
		return nil
	}

	fmt.Println("=" + strings.Repeat("=", 120))
	fmt.Printf("%-30s %-35s %-45s\n", "AGENT ID", "DISPLAY NAME", "DIALOGFLOW AGENT")
	fmt.Println("=" + strings.Repeat("=", 120))

	for _, agent := range agents {
		agentID := extractResourceID(agent.Name)
		dialogflowAgent := ""
		if agent.DialogflowAgentDefinition != nil {
			dialogflowAgent = agent.DialogflowAgentDefinition.DialogflowAgent
		}
		if dialogflowAgent == "" {
			dialogflowAgent = "N/A"
		}

		fmt.Printf("%-30s %-35s %-45s\n",
			agentID,
			truncateString(agent.DisplayName, 35),
			truncateString(dialogflowAgent, 45),
		)
	}

	fmt.Printf("\nTotal: %d agent(s)\n", len(agents))
	return nil
}

// outputAgentDetails outputs a single agent in the specified format
func outputAgentDetails(agent *client.Agent, format string) error {
	switch format {
	case "json":
		return outputJSON(agent, format)
	case "yaml":
		return outputYAML(agent)
	default:
		return outputAgentDetailsTable(agent)
	}
}

// outputAgentDetailsTable outputs detailed agent information in table format
func outputAgentDetailsTable(agent *client.Agent) error {
	fmt.Println("=" + strings.Repeat("=", 80))
	fmt.Printf("Agent: %s\n", agent.DisplayName)
	fmt.Println("=" + strings.Repeat("=", 80))
	fmt.Printf("Name: %s\n", agent.Name)
	fmt.Printf("Description: %s\n", agent.Description)
	fmt.Printf("Reasoning Engine: %s\n", agent.ReasoningEngine)

	if agent.DialogflowAgentDefinition != nil {
		fmt.Printf("Dialogflow Agent: %s\n", agent.DialogflowAgentDefinition.DialogflowAgent)
	}

	if agent.Icon != nil {
		if agent.Icon.URI != "" {
			fmt.Printf("Icon URI: %s\n", agent.Icon.URI)
		} else if agent.Icon.Content != "" {
			fmt.Println("Icon: [embedded Base64 content]")
		}
	}

	if len(agent.Capabilities) > 0 {
		fmt.Printf("\nCapabilities (%d):\n", len(agent.Capabilities))
		for _, capability := range agent.Capabilities {
			fmt.Printf("  - %s\n", capability)
		}
	}

	if len(agent.Labels) > 0 {
		fmt.Println("\nLabels:")
		for key, value := range agent.Labels {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	if len(agent.Annotations) > 0 {
		fmt.Println("\nAnnotations:")
		for key, value := range agent.Annotations {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	if agent.CreateTime != "" {
		fmt.Printf("\nCreated: %s\n", agent.CreateTime)
	}
	if agent.UpdateTime != "" {
		fmt.Printf("Updated: %s\n", agent.UpdateTime)
	}

	return nil
}

// outputEngineFeatures outputs engine features in the specified format.
func outputEngineFeatures(engine *client.Engine, format string) error {
	if engine == nil {
		return fmt.Errorf("engine is nil")
	}

	switch format {
	case "json":
		return outputJSON(engine.Features, format)
	case "yaml":
		return outputYAML(engine.Features)
	default:
		return outputEngineFeaturesTable(engine)
	}
}

func outputEngineFeaturesTable(engine *client.Engine) error {
	features := engine.Features
	if len(features) == 0 {
		fmt.Println("No feature configuration found for this engine.")
		return nil
	}

	keys := sortedFeatureKeys(features)
	enabled := 0

	fmt.Println("=" + strings.Repeat("=", 80))
	fmt.Printf("Features for engine: %s\n", engine.DisplayName)
	fmt.Println("=" + strings.Repeat("=", 80))
	fmt.Printf("%-40s %-20s\n", "FEATURE", "STATE")
	fmt.Println("-" + strings.Repeat("-", 80))

	for _, key := range keys {
		state := features[key]
		if strings.Contains(state, "ON") {
			enabled++
		}
		fmt.Printf("%-40s %-20s\n", key, renderFeatureState(state))
	}

	fmt.Printf("\nEnabled: %d/%d\n", enabled, len(features))
	return nil
}

// outputDataStores outputs data stores in the specified format
func outputDataStores(dataStores []*client.DataStore, format string) error {
	switch format {
	case "json":
		return outputJSON(dataStores, format)
	case "yaml":
		return outputYAML(dataStores)
	default:
		return outputDataStoresTable(dataStores)
	}
}

// outputDataStoresTable outputs data stores in table format
func outputDataStoresTable(dataStores []*client.DataStore) error {
	if len(dataStores) == 0 {
		fmt.Println("No data stores found.")
		return nil
	}

	fmt.Println("=" + strings.Repeat("=", 100))
	fmt.Printf("%-50s %-30s %-20s\n", "NAME", "DISPLAY NAME", "CONTENT CONFIG")
	fmt.Println("=" + strings.Repeat("=", 100))

	for _, ds := range dataStores {
		name := extractResourceID(ds.Name)
		displayName := ds.DisplayName
		contentConfig := ds.ContentConfig

		fmt.Printf("%-50s %-30s %-20s\n", name, displayName, contentConfig)
	}

	fmt.Printf("\nTotal: %d data store(s)\n", len(dataStores))
	return nil
}

// outputDataStoreDetails outputs data store details in the specified format
func outputDataStoreDetails(dataStore *client.DataStore, format string) error {
	switch format {
	case "json":
		return outputJSON(dataStore, format)
	case "yaml":
		return outputYAML(dataStore)
	default:
		return outputDataStoreDetailsTable(dataStore)
	}
}

// outputDataStoreDetailsTable outputs data store details in table format
func outputDataStoreDetailsTable(dataStore *client.DataStore) error {
	fmt.Println("=" + strings.Repeat("=", 80))
	fmt.Printf("Data Store: %s\n", dataStore.DisplayName)
	fmt.Println("=" + strings.Repeat("=", 80))
	fmt.Printf("Name: %s\n", dataStore.Name)
	fmt.Printf("Industry Vertical: %s\n", dataStore.IndustryVertical)
	fmt.Printf("Content Config: %s\n", dataStore.ContentConfig)
	fmt.Printf("Created: %s\n", dataStore.CreateTime)

	if len(dataStore.SolutionTypes) > 0 {
		fmt.Printf("Solution Types: %s\n", strings.Join(dataStore.SolutionTypes, ", "))
	}

	if dataStore.AclEnabled {
		fmt.Printf("ACL Enabled: %t\n", dataStore.AclEnabled)
	}

	if dataStore.BillingEstimation != nil {
		size := float64(dataStore.BillingEstimation.UnstructuredDataSize) / (1024 * 1024)
		fmt.Printf("\nBilling Estimation:\n")
		fmt.Printf("  Size: %.2f MB\n", size)
		fmt.Printf("  Updated: %s\n", dataStore.BillingEstimation.UnstructuredDataUpdateTime)
	}

	if dataStore.DocumentProcessingConfig != nil {
		fmt.Println("\nDocument Processing:")
		if chunkingConfig, ok := dataStore.DocumentProcessingConfig["chunkingConfig"].(map[string]interface{}); ok {
			if layoutConfig, ok := chunkingConfig["layoutBasedChunkingConfig"].(map[string]interface{}); ok {
				if chunkSize, ok := layoutConfig["chunkSize"].(string); ok {
					fmt.Printf("  Chunk Size: %s\n", chunkSize)
				}
			}
		}

		if defaultParsingConfig, ok := dataStore.DocumentProcessingConfig["defaultParsingConfig"].(map[string]interface{}); ok {
			if layoutParsingConfig, ok := defaultParsingConfig["layoutParsingConfig"].(map[string]interface{}); ok {
				if tableAnnotation, ok := layoutParsingConfig["enableTableAnnotation"].(bool); ok && tableAnnotation {
					fmt.Println("  ✓ Table annotation enabled")
				}
				if imageAnnotation, ok := layoutParsingConfig["enableImageAnnotation"].(bool); ok && imageAnnotation {
					fmt.Println("  ✓ Image annotation enabled")
				}
			}
		}
	}

	if dataStore.Schema != nil {
		if schemaName, ok := dataStore.Schema["name"].(string); ok {
			fmt.Printf("\nSchema: %s\n", schemaName)
		}
	}

	return nil
}

// outputDocuments outputs documents in the specified format
func outputDocuments(documents []*client.Document, dataStoreID, branch, format string) error {
	switch format {
	case "json":
		return outputJSON(documents, format)
	case "yaml":
		return outputYAML(documents)
	default:
		return outputDocumentsTable(documents, dataStoreID, branch)
	}
}

// outputDocumentsTable outputs documents in table format
func outputDocumentsTable(documents []*client.Document, dataStoreID, branch string) error {
	if len(documents) == 0 {
		fmt.Println("No documents found in this data store.")
		return nil
	}

	fmt.Println("=" + strings.Repeat("=", 100))
	fmt.Printf("Documents in Data Store: %s\n", dataStoreID)
	fmt.Printf("Branch: %s\n", branch)
	fmt.Println("=" + strings.Repeat("=", 100))
	fmt.Printf("%-40s %-50s %-25s\n", "ID", "URI", "Index Time")
	fmt.Println("-" + strings.Repeat("-", 100))

	for _, doc := range documents {
		docID := doc.ID
		if len(docID) > 40 {
			docID = docID[:40]
		}

		uri := "N/A"
		if doc.Content != nil {
			if uriValue, ok := doc.Content["uri"].(string); ok {
				uri = uriValue
				if len(uri) > 50 {
					uri = uri[:47] + "..."
				}
			}
		}

		indexTime := doc.IndexTime
		if indexTime == "" {
			indexTime = "N/A"
		}

		fmt.Printf("%-40s %-50s %-25s\n", docID, uri, indexTime)
	}

	fmt.Printf("\nTotal: %d document(s)\n", len(documents))
	return nil
}

// outputCreateResult outputs create operation result
func outputCreateResult(result *client.CreateResult, format string) error {
	switch format {
	case "json":
		return outputJSON(result, format)
	case "yaml":
		return outputYAML(result)
	default:
		return outputCreateResultTable(result)
	}
}

// outputCreateResultTable outputs create result in table format
func outputCreateResultTable(result *client.CreateResult) error {
	if result.Error != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", result.Error)
		return fmt.Errorf("create operation failed")
	}

	if result.EngineName != "" {
		fmt.Printf("✅ Successfully created engine: %s\n", result.EngineName)
	}

	if result.DataStoreName != "" {
		fmt.Printf("✅ Successfully created data store: %s\n", result.DataStoreName)
		if result.ImportOperation != nil {
			if operationName, ok := result.ImportOperation["name"].(string); ok {
				fmt.Printf("⚙️  Import Operation: %s\n", operationName)
			}
		}
	}

	return nil
}

// outputCreateDataStoreResult outputs data store create result
func outputCreateDataStoreResult(result *client.CreateResult, format string) error {
	switch format {
	case "json":
		return outputJSON(result, format)
	case "yaml":
		return outputYAML(result)
	default:
		return outputCreateDataStoreResultTable(result)
	}
}

// outputCreateDataStoreResultTable outputs data store create result in table format
func outputCreateDataStoreResultTable(result *client.CreateResult) error {
	if result.Error != "" {
		fmt.Fprintf(os.Stderr, "Error: %s\n", result.Error)
		return fmt.Errorf("create operation failed")
	}

	fmt.Printf("✅ Successfully created data store: %s\n", result.DataStoreName)
	if result.ImportOperation != nil {
		if operationName, ok := result.ImportOperation["name"].(string); ok {
			fmt.Printf("⚙️  Import Operation: %s\n", operationName)
		}
	}

	return nil
}

// outputDeleteResult outputs delete operation result
func outputDeleteResult(result *client.DeleteResult, format string) error {
	switch format {
	case "json":
		return outputJSON(result, format)
	case "yaml":
		return outputYAML(result)
	default:
		return outputDeleteResultTable(result)
	}
}

// outputDeleteResultTable outputs delete result in table format
func outputDeleteResultTable(result *client.DeleteResult) error {
	if result.Status == "success" {
		fmt.Printf("✅ %s\n", result.Message)
	} else {
		fmt.Fprintf(os.Stderr, "❌ %s\n", result.Message)
		return fmt.Errorf("delete operation failed")
	}

	return nil
}

// outputJSON outputs data in JSON format
func outputJSON(data interface{}, format string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonData))
	return nil
}

// outputYAML outputs data in YAML format
func outputYAML(data interface{}) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	fmt.Println(string(yamlData))
	return nil
}

// extractResourceID extracts the resource ID from a full resource name
func extractResourceID(resourceName string) string {
	parts := strings.Split(resourceName, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return resourceName
}

func truncateString(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func sortedFeatureKeys(features map[string]string) []string {
	keys := make([]string, 0, len(features))
	for key := range features {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func renderFeatureState(state string) string {
	switch state {
	case client.EngineFeatureStateOn:
		return "ON"
	case client.EngineFeatureStateOff:
		return "OFF"
	default:
		upper := strings.ToUpper(state)
		switch upper {
		case "ON", "OFF":
			return upper
		default:
			return state
		}
	}
}
