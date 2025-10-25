package cli

import (
	"fmt"
	"strings"

	"github.com/bcbsma/gemctl-go/internal/client"
	"github.com/spf13/cobra"
)

// NewEnginesListCommand creates the engines list command
func NewEnginesListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all engines in a project",
		Long: `List all engines (AI apps) in a project.

Examples:
  gemctl engines list
  gemctl engines list --project-id=my-project --location=us
  gemctl engines list --use-service-account`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := getConfigFromFlags(cmd)
			if err != nil {
				return err
			}

			geminiClient, err := client.NewGeminiClient(config)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			engines, err := geminiClient.ListEngines(config.Collection)
			if err != nil {
				return fmt.Errorf("failed to list engines: %w", err)
			}

			return outputEngines(engines, config.Format)
		},
	}

	return cmd
}

// NewEnginesDescribeCommand creates the engines describe command
func NewEnginesDescribeCommand() *cobra.Command {
	var full bool

	cmd := &cobra.Command{
		Use:   "describe ENGINE_ID",
		Short: "Describe a specific engine",
		Long: `Describe a specific engine.

ENGINE_ID can be just the engine ID or the full resource name.

Examples:
  gemctl engines describe my-engine
  gemctl engines describe my-engine --full
  gemctl engines describe my-engine --format=json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := getConfigFromFlags(cmd)
			if err != nil {
				return err
			}

			geminiClient, err := client.NewGeminiClient(config)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			engineID := args[0]
			engineName := constructEngineName(engineID, config)

			if full {
				config, err := geminiClient.GetEngineFullConfig(engineName)
				if err != nil {
					return fmt.Errorf("failed to get engine full config: %w", err)
				}
				return outputJSON(config, "json")
			}

			engine, err := geminiClient.GetEngineDetails(engineName)
			if err != nil {
				return fmt.Errorf("failed to get engine details: %w", err)
			}

			if engine == nil {
				return fmt.Errorf("engine not found: %s", engineID)
			}

			return outputEngineDetails(engine, config.Format)
		},
	}

	cmd.Flags().BoolVar(&full, "full", false, "Include all data store configurations")

	return cmd
}

// NewEnginesCreateCommand creates the engines create command
func NewEnginesCreateCommand() *cobra.Command {
	var searchTier string

	cmd := &cobra.Command{
		Use:   "create ENGINE_ID DISPLAY_NAME [DATA_STORE_IDS...]",
		Short: "Create a search engine connected to data stores",
		Long: `Create a search engine connected to data stores.

ENGINE_ID: Unique ID for the engine
DISPLAY_NAME: Display name for the engine
DATA_STORE_IDS: One or more data store IDs to connect (optional)

Examples:
  gemctl engines create my-engine "My Search Engine" datastore1 datastore2
  gemctl engines create my-engine "My Search Engine" datastore1 --search-tier=SEARCH_TIER_ENTERPRISE`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := getConfigFromFlags(cmd)
			if err != nil {
				return err
			}

			geminiClient, err := client.NewGeminiClient(config)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			engineID := args[0]
			displayName := args[1]
			dataStoreIDs := args[2:]

			result, err := geminiClient.CreateSearchEngine(engineID, displayName, dataStoreIDs, searchTier)
			if err != nil {
				return fmt.Errorf("failed to create engine: %w", err)
			}

			return outputCreateResult(result, config.Format)
		},
	}

	cmd.Flags().StringVar(&searchTier, "search-tier", "SEARCH_TIER_STANDARD", "Search tier (SEARCH_TIER_STANDARD, SEARCH_TIER_ENTERPRISE)")

	return cmd
}

// NewEnginesDeleteCommand creates the engines delete command
func NewEnginesDeleteCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete ENGINE_ID",
		Short: "Delete a search engine",
		Long: `Delete a search engine.

ENGINE_ID can be just the engine ID or the full resource name.

Examples:
  gemctl engines delete my-engine
  gemctl engines delete my-engine --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := getConfigFromFlags(cmd)
			if err != nil {
				return err
			}

			geminiClient, err := client.NewGeminiClient(config)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			engineID := args[0]
			engineName := constructEngineName(engineID, config)

			// Get engine details for confirmation
			engine, err := geminiClient.GetEngineDetails(engineName)
			if err != nil {
				return fmt.Errorf("failed to get engine details: %w", err)
			}

			if engine == nil {
				return fmt.Errorf("engine not found: %s", engineID)
			}

			// Confirmation prompt unless --force is used
			if !force {
				fmt.Printf("Engine: %s\n", engine.DisplayName)
				fmt.Printf("Name: %s\n", engine.Name)
				fmt.Printf("Solution Type: %s\n", engine.SolutionType)
				fmt.Print("\nAre you sure you want to delete this engine? (y/N): ")
				
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Deletion cancelled.")
					return nil
				}
			}

			result, err := geminiClient.DeleteEngine(engineName)
			if err != nil {
				return fmt.Errorf("failed to delete engine: %w", err)
			}

			return outputDeleteResult(result, config.Format)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

// constructEngineName constructs the full engine resource name
func constructEngineName(engineID string, config *client.Config) string {
	if contains(engineID, "/") {
		return engineID
	}
	return fmt.Sprintf("projects/%s/locations/%s/collections/%s/engines/%s",
		config.ProjectID, config.Location, config.Collection, engineID)
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
