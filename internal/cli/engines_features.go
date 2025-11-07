package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vb140772/gemctl-go/internal/client"
)

var knownFeatures = map[string]struct{}{
	"*":                                    {},
	"agent-gallery":                        {},
	"no-code-agent-builder":                {},
	"prompt-gallery":                       {},
	"model-selector":                       {},
	"notebook-lm":                          {},
	"people-search":                        {},
	"people-search-org-chart":              {},
	"bi-directional-audio":                 {},
	"feedback":                             {},
	"session-sharing":                      {},
	"personalization-memory":               {},
	"disable-agent-sharing":                {},
	"disable-image-generation":             {},
	"disable-video-generation":             {},
	"disable-onedrive-upload":              {},
	"disable-talk-to-content":              {},
	"disable-google-drive-upload":          {},
	"agent-sharing-without-admin-approval": {},
}

// NewEnginesFeaturesCommand creates the engines features command group
func NewEnginesFeaturesCommand() *cobra.Command {
	featuresCmd := &cobra.Command{
		Use:   "features",
		Short: "Manage engine feature flags",
		Long: `Manage Gemini Enterprise engine feature flags such as agent-gallery or prompt-gallery.

Use subcommands to list feature states or toggle features on and off.`,
	}

	featuresCmd.AddCommand(NewEnginesFeaturesListCommand())
	featuresCmd.AddCommand(NewEnginesFeaturesEnableCommand())
	featuresCmd.AddCommand(NewEnginesFeaturesDisableCommand())

	return featuresCmd
}

// NewEnginesFeaturesListCommand lists features and their states
func NewEnginesFeaturesListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list ENGINE_ID",
		Short: "List feature states for an engine",
		Long: `List all known feature flags and their current state for the specified engine.

Examples:
  gemctl engines features list my-engine
  gemctl engines features list my-engine --format=json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := getConfigFromFlags(cmd)
			if err != nil {
				return err
			}

			engineID := args[0]

			geminiClient, err := client.NewGeminiClient(config)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			engineName := constructEngineName(engineID, config)
			engine, err := geminiClient.GetEngineDetails(engineName)
			if err != nil {
				return fmt.Errorf("failed to get engine details: %w", err)
			}

			if engine == nil {
				return fmt.Errorf("engine not found: %s", engineID)
			}

			return outputEngineFeatures(engine, config.Format)
		},
	}

	return cmd
}

// NewEnginesFeaturesEnableCommand enables one or more features
func NewEnginesFeaturesEnableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable ENGINE_ID FEATURE [FEATURE...]",
		Short: "Enable features for an engine",
		Long: `Enable one or more feature flags for an engine.

Examples:
  gemctl engines features enable my-engine agent-gallery prompt-gallery
  gemctl engines features enable agent-gallery my-engine`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := getConfigFromFlags(cmd)
			if err != nil {
				return err
			}

			engineID, features, err := parseEngineFeatureArgs(args)
			if err != nil {
				return err
			}

			updates := make(map[string]string, len(features))
			for _, feature := range features {
				featureKey := normalizeFeatureKey(feature)
				updates[featureKey] = client.EngineFeatureStateOn
			}

			return applyFeatureUpdates(config, engineID, updates)
		},
	}

	return cmd
}

// NewEnginesFeaturesDisableCommand disables one or more features
func NewEnginesFeaturesDisableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable ENGINE_ID FEATURE [FEATURE...]",
		Short: "Disable features for an engine",
		Long: `Disable one or more feature flags for an engine.

Examples:
  gemctl engines features disable my-engine agent-gallery
  gemctl engines features disable agent-gallery my-engine`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := getConfigFromFlags(cmd)
			if err != nil {
				return err
			}

			engineID, features, err := parseEngineFeatureArgs(args)
			if err != nil {
				return err
			}

			updates := make(map[string]string, len(features))
			for _, feature := range features {
				featureKey := normalizeFeatureKey(feature)
				updates[featureKey] = client.EngineFeatureStateOff
			}

			return applyFeatureUpdates(config, engineID, updates)
		},
	}

	return cmd
}

func applyFeatureUpdates(config *client.Config, engineID string, updates map[string]string) error {
	geminiClient, err := client.NewGeminiClient(config)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	engineName := constructEngineName(engineID, config)
	updatedEngine, err := geminiClient.UpdateEngineFeatures(engineName, updates)
	if err != nil {
		return err
	}

	return outputEngineFeatures(updatedEngine, config.Format)
}

func normalizeFeatureKey(feature string) string {
	return strings.TrimSpace(strings.ToLower(feature))
}

func parseEngineFeatureArgs(args []string) (string, []string, error) {
	if len(args) < 2 {
		return "", nil, fmt.Errorf("engine ID and at least one feature are required")
	}

	engineID := args[0]
	featureArgs := args[1:]

	lastArg := args[len(args)-1]
	if looksLikeFeature(engineID) && !looksLikeFeature(lastArg) {
		engineID = lastArg
		featureArgs = args[:len(args)-1]
	}

	if len(featureArgs) == 0 {
		return "", nil, fmt.Errorf("no features specified for update")
	}

	return engineID, featureArgs, nil
}

func looksLikeFeature(value string) bool {
	key := strings.ToLower(strings.TrimSpace(value))
	if key == "" {
		return false
	}

	_, ok := knownFeatures[key]
	return ok
}
