package cli

import (
	"github.com/spf13/cobra"
)

// NewDataStoresCommand creates the data-stores command group
func NewDataStoresCommand() *cobra.Command {
	dataStoresCmd := &cobra.Command{
		Use:   "data-stores",
		Short: "Manage Gemini Enterprise data stores",
		Long:  "Manage Gemini Enterprise data stores including document storage and processing.",
	}

	dataStoresCmd.AddCommand(NewDataStoresListCommand())
	dataStoresCmd.AddCommand(NewDataStoresDescribeCommand())
	dataStoresCmd.AddCommand(NewDataStoresCreateFromGCSCommand())
	dataStoresCmd.AddCommand(NewDataStoresListDocumentsCommand())
	dataStoresCmd.AddCommand(NewDataStoresDeleteCommand())

	return dataStoresCmd
}
