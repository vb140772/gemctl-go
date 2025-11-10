package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vb140772/gemctl-go/internal/client"
)

// NewEnginesWorkforceCommand creates the workforce identity command group.
func NewEnginesWorkforceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workforce",
		Short: "Manage workforce identity configuration for the current project/location",
		Long: `Manage workforce identity pool linkage used for access control.

Workforce identity is configured at the project/location level and applies to engines created afterwards.`,
	}

	cmd.AddCommand(NewEnginesWorkforceShowCommand())
	cmd.AddCommand(NewEnginesWorkforceSetCommand())

	return cmd
}

// NewEnginesWorkforceShowCommand creates the workforce show subcommand.
func NewEnginesWorkforceShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the current workforce identity configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := getConfigFromFlags(cmd)
			if err != nil {
				return err
			}

			geminiClient, err := client.NewGeminiClient(config)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			workforceConfig, err := geminiClient.GetWorkforceIdentityConfig()
			if err != nil {
				return err
			}

			return outputWorkforceIdentity(workforceConfig, config.Format)
		},
	}

	return cmd
}

// NewEnginesWorkforceSetCommand creates the workforce set subcommand.
func NewEnginesWorkforceSetCommand() *cobra.Command {
	var resource string
	var clear bool
	var workforceID string
	var workforceProvider string
	var workforceLocation string

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Configure the workforce identity pool",
		Long: `Configure the workforce identity pool for the current project/location.

Provide the workforce pool resource (e.g. locations/global/workforcePools/pool-id/providers/provider-id),
or specify the pool ID / provider ID via flags. Use --clear to disable workforce identity.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if resource == "" && workforceID == "" && !clear {
				return fmt.Errorf("provide --resource, --workforce-id or use --clear")
			}
			if clear {
				if resource != "" || workforceID != "" || workforceProvider != "" {
					return fmt.Errorf("--clear cannot be combined with other flags")
				}
			}

			config, err := getConfigFromFlags(cmd)
			if err != nil {
				return err
			}

			geminiClient, err := client.NewGeminiClient(config)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			resourceValue := ""
			if !clear {
				resourceValue, err = buildWorkforceResource(resource, workforceLocation, workforceID, workforceProvider)
				if err != nil {
					return err
				}
			}

			updated, err := geminiClient.SetWorkforceIdentityConfig(resourceValue)
			if err != nil {
				return err
			}

			return outputWorkforceIdentity(updated, config.Format)
		},
	}

	cmd.Flags().StringVar(&resource, "resource", "", "Full workforce pool resource (locations/.../workforcePools/POOL[/providers/PROVIDER])")
	cmd.Flags().StringVar(&resource, "pool", "", "Alias for --resource (deprecated)")
	cmd.Flags().StringVar(&workforceID, "workforce-id", "", "Workforce pool ID component")
	cmd.Flags().StringVar(&workforceProvider, "provider-id", "", "Workforce provider ID component")
	cmd.Flags().StringVar(&workforceLocation, "workforce-location", "locations/global", "Workforce pool location (default locations/global)")
	cmd.Flags().BoolVar(&clear, "clear", false, "Disable workforce identity (clear existing configuration)")

	return cmd
}

func buildWorkforceResource(resource, location, poolID, providerID string) (string, error) {
	if strings.TrimSpace(resource) != "" {
		return resource, nil
	}
	if strings.TrimSpace(poolID) == "" {
		return "", fmt.Errorf("workforce pool ID is required when using component flags")
	}

	loc := strings.TrimSpace(location)
	if loc == "" {
		loc = "locations/global"
	}
	if !strings.HasPrefix(loc, "locations/") {
		loc = "locations/" + loc
	}

	resourceValue := fmt.Sprintf("%s/workforcePools/%s", loc, poolID)
	if strings.TrimSpace(providerID) != "" {
		resourceValue = fmt.Sprintf("%s/providers/%s", resourceValue, providerID)
	}
	return resourceValue, nil
}
