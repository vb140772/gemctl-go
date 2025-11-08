package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vb140772/gemctl-go/internal/client"
)

// NewEnginesSnapshotCommand creates the engines snapshot command group.
func NewEnginesSnapshotCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Manage engine snapshots (export, diff, restore)",
		Long: `Manage Gemini Enterprise engine snapshots for backup, cloning, or rollback.

Snapshots capture engine configuration, feature flags, and registered agents.
They can be compared (diff) or restored to create/update engines across projects.`,
	}

	cmd.AddCommand(NewEnginesSnapshotCreateCommand())
	cmd.AddCommand(NewEnginesSnapshotDiffCommand())
	cmd.AddCommand(NewEnginesSnapshotRestoreCommand())

	return cmd
}

// NewEnginesSnapshotCreateCommand creates the snapshot create command.
func NewEnginesSnapshotCreateCommand() *cobra.Command {
	var outputPath string
	var notes string
	var description string

	cmd := &cobra.Command{
		Use:   "create ENGINE_ID",
		Short: "Create a JSON snapshot for an engine",
		Long: `Create a JSON snapshot for an engine including configuration, features, and agents.

Examples:
  gemctl engines snapshot create my-engine --output=engine-snapshot.json
  gemctl engines snapshot create my-engine --notes="Pre-deployment backup"`,
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

			engineName := constructEngineName(args[0], config)
			snapshot, err := geminiClient.CreateEngineSnapshot(engineName)
			if err != nil {
				return fmt.Errorf("failed to create snapshot: %w", err)
			}

			if notes != "" {
				snapshot.Metadata.Notes = notes
			}
			if description != "" {
				snapshot.Metadata.Description = description
			}

			data, err := snapshot.MarshalJSONBytes()
			if err != nil {
				return fmt.Errorf("failed to marshal snapshot: %w", err)
			}

			if outputPath == "" {
				fmt.Println(string(data))
				return nil
			}

			if err := os.WriteFile(outputPath, data, 0o644); err != nil {
				return fmt.Errorf("failed to write snapshot file: %w", err)
			}
			fmt.Printf("Snapshot written to %s\n", outputPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Path to write snapshot JSON (default stdout)")
	cmd.Flags().StringVar(&notes, "notes", "", "Optional notes stored in snapshot metadata")
	cmd.Flags().StringVar(&description, "description", "", "Optional description stored in snapshot metadata")

	return cmd
}

// NewEnginesSnapshotDiffCommand creates the snapshot diff command.
func NewEnginesSnapshotDiffCommand() *cobra.Command {
	var engineID string

	cmd := &cobra.Command{
		Use:   "diff SNAPSHOT_A [SNAPSHOT_B]",
		Short: "Diff two snapshots or a snapshot against the current engine state",
		Long: `Compare two snapshot files or compare a snapshot to the current engine state.

Examples:
  gemctl engines snapshot diff snapshot-old.json snapshot-new.json
  gemctl engines snapshot diff snapshot.json --engine my-engine`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := getConfigFromFlags(cmd)
			if err != nil {
				return err
			}

			snapshotA, err := loadSnapshotFromFile(args[0])
			if err != nil {
				return err
			}

			var diff client.SnapshotDiff

			if len(args) == 2 {
				snapshotB, err := loadSnapshotFromFile(args[1])
				if err != nil {
					return err
				}
				diff = client.DiffSnapshots(snapshotA, snapshotB)
			} else {
				if engineID == "" {
					return fmt.Errorf("engine ID is required when diffing snapshot against live state")
				}
				geminiClient, err := client.NewGeminiClient(config)
				if err != nil {
					return fmt.Errorf("failed to create client: %w", err)
				}
				engineName := constructEngineName(engineID, config)
				diff, err = geminiClient.DiffSnapshotWithEngine(snapshotA, engineName)
				if err != nil {
					return err
				}
			}

			return outputSnapshotDiff(diff, config.Format)
		},
	}

	cmd.Flags().StringVar(&engineID, "engine", "", "Engine ID for diffing snapshot against live configuration")

	return cmd
}

// NewEnginesSnapshotRestoreCommand creates the snapshot restore command.
func NewEnginesSnapshotRestoreCommand() *cobra.Command {
	var targetEngineID string
	var newEngineID string
	var allowCreate bool
	var dryRun bool
	var force bool
	var notes string
	var updateExisting bool

	cmd := &cobra.Command{
		Use:   "restore SNAPSHOT_PATH [ENGINE_ID]",
		Short: "Restore an engine snapshot",
		Long: `Restore an engine snapshot to create a new engine or rollback an existing one.

Examples:
  # Dry-run (diff only)
  gemctl engines snapshot restore snapshot.json my-engine --dry-run

  # Restore to existing engine with confirmation
  gemctl engines snapshot restore snapshot.json my-engine

  # Clone snapshot into new engine
  gemctl engines snapshot restore snapshot.json --new-engine-id cloned-engine --allow-create`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := getConfigFromFlags(cmd)
			if err != nil {
				return err
			}

			snapshot, err := loadSnapshotFromFile(args[0])
			if err != nil {
				return err
			}
			if notes != "" {
				snapshot.Metadata.Notes = notes
			}

			if len(args) == 2 && targetEngineID == "" {
				targetEngineID = args[1]
			}
			if newEngineID != "" {
				targetEngineID = newEngineID
				allowCreate = true
				updateExisting = false
			}

			if targetEngineID == "" {
				targetEngineID = snapshot.Metadata.OriginalEngineID
				if targetEngineID == "" {
					return fmt.Errorf("target engine ID is required")
				}
			}

			geminiClient, err := client.NewGeminiClient(config)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			targetEngineName := constructEngineName(targetEngineID, config)

			previewOpts := client.SnapshotRestoreOptions{
				TargetEngineName: targetEngineName,
				CreateIfMissing:  allowCreate,
				UpdateExisting:   updateExisting,
				DryRun:           true,
			}

			_, diff, err := geminiClient.RestoreEngineSnapshot(snapshot, previewOpts)
			if err != nil {
				return err
			}

			fmt.Println("=== Restore Preview ===")
			if err := outputSnapshotDiff(diff, config.Format); err != nil {
				return err
			}

			if dryRun {
				fmt.Println("Dry run complete. No changes applied.")
				return nil
			}

			if diff.IsEmpty() {
				fmt.Println("No changes detected; restore skipped.")
				return nil
			}

			if !force {
				if proceed, err := promptForConfirmation("Apply these changes? (y/N): "); err != nil || !proceed {
					if err != nil {
						return err
					}
					fmt.Println("Restore cancelled.")
					return nil
				}
			}

			applyOpts := client.SnapshotRestoreOptions{
				TargetEngineName: targetEngineName,
				CreateIfMissing:  allowCreate,
				UpdateExisting:   updateExisting,
				DryRun:           false,
			}

			result, _, err := geminiClient.RestoreEngineSnapshot(snapshot, applyOpts)
			if err != nil {
				return err
			}

			return outputSnapshotRestoreResult(result)
		},
	}

	cmd.Flags().StringVar(&targetEngineID, "engine-id", "", "Target engine ID (defaults to snapshot metadata)")
	cmd.Flags().StringVar(&newEngineID, "new-engine-id", "", "Create a new engine with the given ID")
	cmd.Flags().BoolVar(&allowCreate, "allow-create", false, "Allow creating the engine if it does not exist")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Only preview changes without applying them")
	cmd.Flags().BoolVar(&force, "force", false, "Apply changes without confirmation")
	cmd.Flags().BoolVar(&updateExisting, "update-existing", true, "Update existing engine fields when restoring")
	cmd.Flags().StringVar(&notes, "notes", "", "Override snapshot notes before restore")

	return cmd
}

func loadSnapshotFromFile(path string) (*client.EngineSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot file %s: %w", path, err)
	}
	snapshot, err := client.LoadEngineSnapshot(data)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func promptForConfirmation(question string) (bool, error) {
	fmt.Print(question)
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return false, err
	}
	switch response {
	case "y", "Y", "yes", "YES":
		return true, nil
	default:
		return false, nil
	}
}

func outputSnapshotRestoreResult(result *client.SnapshotRestoreResult) error {
	if result == nil {
		fmt.Println("No changes applied.")
		return nil
	}

	fmt.Printf("Restore applied at %s\n", time.Now().Format(time.RFC3339))
	fmt.Printf("Target Engine: %s\n", result.EngineName)
	if result.Created {
		fmt.Println("Engine created.")
	}
	if result.EnginePatched {
		fmt.Println("Engine configuration updated.")
	}
	if len(result.FeatureChanges) > 0 {
		fmt.Println("\nFeature changes:")
		for _, change := range result.FeatureChanges {
			fmt.Printf("  %s: %s -> %s\n", change.Feature, change.Old, change.New)
		}
	}
	if len(result.AgentChanges) > 0 {
		fmt.Println("\nAgent changes:")
		for _, change := range result.AgentChanges {
			fmt.Printf("  %s: %s\n", change.Key, change.ChangeType)
		}
	}
	return nil
}
