package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root command for the gemctl CLI
func NewRootCommand(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "gemctl",
		Short: "Gemini Enterprise CLI - Manage Google Cloud Gemini Enterprise (Discovery Engine) resources",
		Long: `Gemini Enterprise CLI - Manage Google Cloud Gemini Enterprise (Discovery Engine) resources.

This CLI provides gcloud-style commands for managing Gemini Enterprise resources including:
- Engines (AI apps)
- Data stores
- Documents

Authentication: Uses gcloud auth by default, or --use-service-account for ADC.
Project: Set via --project, GOOGLE_CLOUD_PROJECT env var, or gcloud config.`,
		Version: version,
	}

	// Add global flags
	rootCmd.PersistentFlags().StringP("project", "p", "", "Google Cloud project ID (can also be set via GOOGLE_CLOUD_PROJECT env var)")
	rootCmd.PersistentFlags().StringP("location", "l", "", "Location for resources (e.g., us, us-central1, global) (can also be set via AGENTSPACE_LOCATION env var)")
	rootCmd.PersistentFlags().StringP("format", "f", "table", "Output format (table, json, yaml)")
	rootCmd.PersistentFlags().StringP("collection", "c", "default_collection", "Collection ID")
	rootCmd.PersistentFlags().Bool("use-service-account", false, "Use Application Default Credentials (service account) instead of user credentials")

	// Add subcommands
	rootCmd.AddCommand(NewEnginesCommand())
	rootCmd.AddCommand(NewDataStoresCommand())

	return rootCmd
}
