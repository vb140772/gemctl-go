package cli

import (
	"fmt"
	"strings"

	"github.com/bcbsma/gemctl-go/internal/client"
	"github.com/spf13/cobra"
)

// NewDataStoresListCommand creates the data-stores list command
func NewDataStoresListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all data stores in a project",
		Long: `List all data stores in a project.

Examples:
  gemctl data-stores list
  gemctl data-stores list --location=us
  gemctl data-stores list --format=json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := getConfigFromFlags(cmd)
			if err != nil {
				return err
			}

			geminiClient, err := client.NewGeminiClient(config)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			dataStores, err := geminiClient.ListDataStores()
			if err != nil {
				return fmt.Errorf("failed to list data stores: %w", err)
			}

			return outputDataStores(dataStores, config.Format)
		},
	}

	return cmd
}

// NewDataStoresDescribeCommand creates the data-stores describe command
func NewDataStoresDescribeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe DATA_STORE_ID",
		Short: "Describe a specific data store",
		Long: `Describe a specific data store.

DATA_STORE_ID can be just the ID or the full resource name.

Examples:
  gemctl data-stores describe my-datastore
  gemctl data-stores describe my-datastore --format=json`,
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

			dataStoreID := args[0]
			dataStoreName := constructDataStoreName(dataStoreID, config)

			dataStore, err := geminiClient.GetDataStoreDetails(dataStoreName)
			if err != nil {
				return fmt.Errorf("failed to get data store details: %w", err)
			}

			if dataStore == nil {
				return fmt.Errorf("data store not found: %s", dataStoreID)
			}

			// Try to get schema
			schema, err := geminiClient.GetDataStoreSchema(dataStoreName)
			if err == nil && schema != nil {
				dataStore.Schema = schema
			}

			return outputDataStoreDetails(dataStore, config.Format)
		},
	}

	return cmd
}

// NewDataStoresCreateFromGCSCommand creates the data-stores create-from-gcs command
func NewDataStoresCreateFromGCSCommand() *cobra.Command {
	var dataSchema, reconciliationMode string

	cmd := &cobra.Command{
		Use:   "create-from-gcs DATA_STORE_ID DISPLAY_NAME GCS_URI",
		Short: "Create a data store and import data from GCS bucket",
		Long: `Create a data store and import data from GCS bucket.

DATA_STORE_ID: Unique ID for the data store
DISPLAY_NAME: Display name for the data store  
GCS_URI: GCS URI (e.g., gs://bucket-name/path/*)

Examples:
  gemctl data-stores create-from-gcs my-store "My Store" gs://my-bucket/docs/*
  gemctl data-stores create-from-gcs my-store "My Store" gs://my-bucket/data.csv --data-schema=csv`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := getConfigFromFlags(cmd)
			if err != nil {
				return err
			}

			geminiClient, err := client.NewGeminiClient(config)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			dataStoreID := args[0]
			displayName := args[1]
			gcsURI := args[2]

			result, err := geminiClient.CreateDataStoreFromGCS(dataStoreID, displayName, gcsURI, dataSchema, reconciliationMode)
			if err != nil {
				return fmt.Errorf("failed to create data store from GCS: %w", err)
			}

			return outputCreateDataStoreResult(result, config.Format)
		},
	}

	cmd.Flags().StringVar(&dataSchema, "data-schema", "content", "Data schema type (content, custom, csv, document)")
	cmd.Flags().StringVar(&reconciliationMode, "reconciliation-mode", "INCREMENTAL", "Import mode (INCREMENTAL, FULL)")

	return cmd
}

// NewDataStoresListDocumentsCommand creates the data-stores list-documents command
func NewDataStoresListDocumentsCommand() *cobra.Command {
	var branch string

	cmd := &cobra.Command{
		Use:   "list-documents DATA_STORE_ID",
		Short: "List documents in a data store",
		Long: `List documents in a data store.

DATA_STORE_ID can be just the ID or the full resource name.

Examples:
  gemctl data-stores list-documents my-datastore
  gemctl data-stores list-documents my-datastore --format=json
  gemctl data-stores list-documents my-datastore --branch=my-branch`,
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

			dataStoreID := args[0]
			dataStoreName := constructDataStoreName(dataStoreID, config)

			documents, err := geminiClient.ListDocuments(dataStoreName, branch)
			if err != nil {
				return fmt.Errorf("failed to list documents: %w", err)
			}

			return outputDocuments(documents, dataStoreID, branch, config.Format)
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "default_branch", "Branch name")

	return cmd
}

// NewDataStoresDeleteCommand creates the data-stores delete command
func NewDataStoresDeleteCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete DATA_STORE_ID",
		Short: "Delete a data store",
		Long: `Delete a data store.

DATA_STORE_ID can be just the ID or the full resource name.

Examples:
  gemctl data-stores delete my-datastore
  gemctl data-stores delete my-datastore --force`,
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

			dataStoreID := args[0]
			dataStoreName := constructDataStoreName(dataStoreID, config)

			// Get data store details for confirmation
			dataStore, err := geminiClient.GetDataStoreDetails(dataStoreName)
			if err != nil {
				return fmt.Errorf("failed to get data store details: %w", err)
			}

			if dataStore == nil {
				return fmt.Errorf("data store not found: %s", dataStoreID)
			}

			// Confirmation prompt unless --force is used
			if !force {
				fmt.Printf("Data Store: %s\n", dataStore.DisplayName)
				fmt.Printf("Name: %s\n", dataStore.Name)
				fmt.Printf("Content Config: %s\n", dataStore.ContentConfig)
				fmt.Printf("Created: %s\n", dataStore.CreateTime)
				fmt.Print("\nAre you sure you want to delete this data store? (y/N): ")
				
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Deletion cancelled.")
					return nil
				}
			}

			result, err := geminiClient.DeleteDataStore(dataStoreName)
			if err != nil {
				return fmt.Errorf("failed to delete data store: %w", err)
			}

			return outputDeleteResult(result, config.Format)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

// constructDataStoreName constructs the full data store resource name
func constructDataStoreName(dataStoreID string, config *client.Config) string {
	if strings.Contains(dataStoreID, "/") {
		return dataStoreID
	}
	return fmt.Sprintf("projects/%s/locations/%s/collections/%s/dataStores/%s",
		config.ProjectID, config.Location, config.Collection, dataStoreID)
}
