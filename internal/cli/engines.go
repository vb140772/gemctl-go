package cli

import (
	"github.com/spf13/cobra"
)

// NewEnginesCommand creates the engines command group
func NewEnginesCommand() *cobra.Command {
	enginesCmd := &cobra.Command{
		Use:   "engines",
		Short: "Manage Gemini Enterprise engines (AI apps)",
		Long:  "Manage Gemini Enterprise engines (AI apps) including search engines and conversational AI applications.",
	}

	enginesCmd.AddCommand(NewEnginesListCommand())
	enginesCmd.AddCommand(NewEnginesDescribeCommand())
	enginesCmd.AddCommand(NewEnginesCreateCommand())
	enginesCmd.AddCommand(NewEnginesDeleteCommand())

	return enginesCmd
}
