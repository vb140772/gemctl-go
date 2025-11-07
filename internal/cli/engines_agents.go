package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vb140772/gemctl-go/internal/client"
)

// NewEnginesAgentsCommand creates the engines agents command group
func NewEnginesAgentsCommand() *cobra.Command {
	agentsCmd := &cobra.Command{
		Use:   "agents",
		Short: "Manage Dialogflow agents connected to an engine assistant",
		Long: `Manage Dialogflow agents connected to a Gemini Enterprise engine's default assistant.

Use these commands to list, describe, register, update, and delete agent registrations
using the Discovery Engine v1alpha assistant APIs.`,
	}

	agentsCmd.AddCommand(NewEnginesAgentsListCommand())
	agentsCmd.AddCommand(NewEnginesAgentsDescribeCommand())
	agentsCmd.AddCommand(NewEnginesAgentsCreateCommand())
	agentsCmd.AddCommand(NewEnginesAgentsUpdateCommand())
	agentsCmd.AddCommand(NewEnginesAgentsDeleteCommand())

	return agentsCmd
}

// NewEnginesAgentsListCommand creates the engines agents list command
func NewEnginesAgentsListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list ENGINE_ID",
		Short: "List agents registered to an engine",
		Long: `List all Dialogflow agents that are registered with an engine's default assistant.

Examples:
  gemctl engines agents list my-engine
  gemctl engines agents list my-engine --format=json`,
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

			agents, err := geminiClient.ListAgents(engineName)
			if err != nil {
				return fmt.Errorf("failed to list agents: %w", err)
			}

			return outputAgents(agents, config.Format)
		},
	}

	return cmd
}

// NewEnginesAgentsDescribeCommand creates the engines agents describe command
func NewEnginesAgentsDescribeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe ENGINE_ID AGENT_ID",
		Short: "Describe an agent registration",
		Long: `Describe a specific Dialogflow agent registration for an engine assistant.

Examples:
  gemctl engines agents describe my-engine 12345678901234567890
  gemctl engines agents describe my-engine projects/my-project/locations/global/collections/default_collection/engines/my-engine/assistants/default_assistant/agents/12345678901234567890 --format=json`,
		Args: cobra.ExactArgs(2),
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
			agentName := client.ConstructAgentName(engineName, args[1])

			agent, err := geminiClient.GetAgent(agentName)
			if err != nil {
				return fmt.Errorf("failed to get agent: %w", err)
			}

			if agent == nil {
				return fmt.Errorf("agent not found: %s", args[1])
			}

			return outputAgentDetails(agent, config.Format)
		},
	}

	return cmd
}

// NewEnginesAgentsCreateCommand creates the engines agents create command
func NewEnginesAgentsCreateCommand() *cobra.Command {
	var displayName string
	var description string
	var iconURI string
	var iconContent string
	var reasoningEngine string
	var dialogflowAgent string
	var dialogflowProject string
	var dialogflowLocation string
	var dialogflowAgentID string

	cmd := &cobra.Command{
		Use:   "create ENGINE_ID",
		Short: "Register a Dialogflow agent with an engine assistant",
		Long: `Register a Dialogflow agent so it can be invoked by the engine's default assistant.

You must provide a display name, description, reasoning engine, and the Dialogflow agent to link.

Examples:
  gemctl engines agents create my-engine \
    --display-name="Invoice Helper" \
    --description="Extract key data from invoices" \
    --reasoning-engine=projects/my-project/locations/us/collections/default_collection/engines/my-engine \
    --dialogflow-project-id=my-dialogflow-project \
    --dialogflow-location=us-central1 \
    --dialogflow-agent-id=abcd1234-ef56-7890

  gemctl engines agents create my-engine \
    --display-name="Support Bot" \
    --description="Handles support FAQs" \
    --reasoning-engine=projects/my-project/locations/global/collections/default_collection/engines/my-engine \
    --dialogflow-agent=projects/my-dialogflow-project/locations/global/agents/1234567890 \
    --icon-uri=https://example.com/icon.png`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := getConfigFromFlags(cmd)
			if err != nil {
				return err
			}

			if strings.TrimSpace(displayName) == "" {
				return fmt.Errorf("--display-name is required")
			}

			if strings.TrimSpace(description) == "" {
				return fmt.Errorf("--description is required")
			}

			if strings.TrimSpace(reasoningEngine) == "" {
				return fmt.Errorf("--reasoning-engine is required")
			}

			dialogflowResource, err := resolveDialogflowAgentResource(dialogflowAgent, dialogflowProject, dialogflowLocation, dialogflowAgentID)
			if err != nil {
				return err
			}
			if dialogflowResource == "" {
				return fmt.Errorf("dialogflow agent information is required via --dialogflow-agent or the project/location/agent ID flags")
			}

			geminiClient, err := client.NewGeminiClient(config)
			if err != nil {
				return fmt.Errorf("failed to create client: %w", err)
			}

			engineName := constructEngineName(args[0], config)

			createInput := &client.AgentCreateInput{
				DisplayName:     displayName,
				Description:     description,
				ReasoningEngine: reasoningEngine,
				DialogflowAgentDefinition: &client.DialogflowAgentDefinition{
					DialogflowAgent: dialogflowResource,
				},
			}

			if iconURI != "" || iconContent != "" {
				createInput.Icon = &client.AgentIcon{
					URI:     iconURI,
					Content: iconContent,
				}
			}

			agent, err := geminiClient.CreateAgent(engineName, createInput)
			if err != nil {
				return fmt.Errorf("failed to create agent: %w", err)
			}

			return outputAgentDetails(agent, config.Format)
		},
	}

	cmd.Flags().StringVar(&displayName, "display-name", "", "Display name for the agent (required)")
	cmd.Flags().StringVar(&description, "description", "", "Description of the agent's purpose (required)")
	cmd.Flags().StringVar(&reasoningEngine, "reasoning-engine", "", "Fully qualified reasoning engine resource (required)")
	cmd.Flags().StringVar(&iconURI, "icon-uri", "", "Public URI for the agent icon")
	cmd.Flags().StringVar(&iconContent, "icon-content", "", "Base64-encoded image content for the agent icon")
	cmd.Flags().StringVar(&dialogflowAgent, "dialogflow-agent", "", "Fully qualified Dialogflow agent resource name")
	cmd.Flags().StringVar(&dialogflowProject, "dialogflow-project-id", "", "Dialogflow agent project ID")
	cmd.Flags().StringVar(&dialogflowLocation, "dialogflow-location", "", "Dialogflow agent location (e.g., global, us-central1)")
	cmd.Flags().StringVar(&dialogflowAgentID, "dialogflow-agent-id", "", "Dialogflow agent ID")

	return cmd
}

// NewEnginesAgentsUpdateCommand creates the engines agents update command
func NewEnginesAgentsUpdateCommand() *cobra.Command {
	var displayName string
	var description string
	var iconURI string
	var iconContent string
	var clearIcon bool
	var reasoningEngine string
	var dialogflowAgent string
	var dialogflowProject string
	var dialogflowLocation string
	var dialogflowAgentID string

	cmd := &cobra.Command{
		Use:   "update ENGINE_ID AGENT_ID",
		Short: "Update an existing agent registration",
		Long: `Update metadata or Dialogflow linkage for a registered agent.

Use the flags to specify which fields should be updated. At least one field must be changed.

Examples:
  gemctl engines agents update my-engine 1234567890 --display-name="New Name" --description="Updated description"

  gemctl engines agents update my-engine 1234567890 \
    --dialogflow-project-id=my-dialogflow-project \
    --dialogflow-location=us-central1 \
    --dialogflow-agent-id=abcd1234 \
    --reasoning-engine=projects/my-project/locations/us/collections/default_collection/engines/my-engine`,
		Args: cobra.ExactArgs(2),
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
			agentName := client.ConstructAgentName(engineName, args[1])

			updateInput := &client.AgentUpdateInput{}
			updateMask := make([]string, 0)

			if cmd.Flags().Changed("display-name") {
				updateInput.DisplayName = displayName
				updateMask = append(updateMask, "displayName")
			}

			if cmd.Flags().Changed("description") {
				updateInput.Description = description
				updateMask = append(updateMask, "description")
			}

			if cmd.Flags().Changed("reasoning-engine") {
				if strings.TrimSpace(reasoningEngine) == "" {
					return fmt.Errorf("--reasoning-engine cannot be empty when specified")
				}
				updateInput.ReasoningEngine = reasoningEngine
				updateMask = append(updateMask, "reasoningEngine")
			}

			if cmd.Flags().Changed("icon-uri") || cmd.Flags().Changed("icon-content") || clearIcon {
				if clearIcon {
					updateInput.Icon = &client.AgentIcon{}
				} else {
					updateInput.Icon = &client.AgentIcon{
						URI:     iconURI,
						Content: iconContent,
					}
				}
				updateMask = append(updateMask, "icon")
			}

			if cmd.Flags().Changed("dialogflow-agent") ||
				cmd.Flags().Changed("dialogflow-project-id") ||
				cmd.Flags().Changed("dialogflow-location") ||
				cmd.Flags().Changed("dialogflow-agent-id") {
				dialogflowResource, err := resolveDialogflowAgentResource(dialogflowAgent, dialogflowProject, dialogflowLocation, dialogflowAgentID)
				if err != nil {
					return err
				}
				if dialogflowResource == "" {
					return fmt.Errorf("dialogflow agent information is required when updating the Dialogflow linkage")
				}
				updateInput.DialogflowAgentDefinition = &client.DialogflowAgentDefinition{
					DialogflowAgent: dialogflowResource,
				}
				updateMask = append(updateMask, "dialogflowAgentDefinition.dialogflowAgent")
			}

			if len(updateMask) == 0 {
				return fmt.Errorf("no fields specified for update")
			}

			agent, err := geminiClient.UpdateAgent(agentName, updateInput, updateMask)
			if err != nil {
				return fmt.Errorf("failed to update agent: %w", err)
			}

			return outputAgentDetails(agent, config.Format)
		},
	}

	cmd.Flags().StringVar(&displayName, "display-name", "", "Updated display name for the agent")
	cmd.Flags().StringVar(&description, "description", "", "Updated description for the agent")
	cmd.Flags().StringVar(&reasoningEngine, "reasoning-engine", "", "Updated reasoning engine resource")
	cmd.Flags().StringVar(&iconURI, "icon-uri", "", "Updated icon URI")
	cmd.Flags().StringVar(&iconContent, "icon-content", "", "Updated icon content (Base64)")
	cmd.Flags().BoolVar(&clearIcon, "clear-icon", false, "Clear the agent icon")
	cmd.Flags().StringVar(&dialogflowAgent, "dialogflow-agent", "", "Updated Dialogflow agent resource name")
	cmd.Flags().StringVar(&dialogflowProject, "dialogflow-project-id", "", "Dialogflow agent project ID")
	cmd.Flags().StringVar(&dialogflowLocation, "dialogflow-location", "", "Dialogflow agent location (e.g., global, us-central1)")
	cmd.Flags().StringVar(&dialogflowAgentID, "dialogflow-agent-id", "", "Dialogflow agent ID")

	return cmd
}

// NewEnginesAgentsDeleteCommand creates the engines agents delete command
func NewEnginesAgentsDeleteCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete ENGINE_ID AGENT_ID",
		Short: "Delete an agent registration",
		Long: `Delete a Dialogflow agent registration from an engine assistant.

Examples:
  gemctl engines agents delete my-engine 1234567890
  gemctl engines agents delete my-engine 1234567890 --force`,
		Args: cobra.ExactArgs(2),
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
			agentName := client.ConstructAgentName(engineName, args[1])

			agent, err := geminiClient.GetAgent(agentName)
			if err != nil {
				return fmt.Errorf("failed to get agent: %w", err)
			}

			if agent == nil {
				return fmt.Errorf("agent not found: %s", args[1])
			}

			if !force {
				fmt.Printf("Agent: %s\n", agent.DisplayName)
				fmt.Printf("Name: %s\n", agent.Name)
				fmt.Printf("Reasoning Engine: %s\n", agent.ReasoningEngine)
				fmt.Print("\nAre you sure you want to delete this agent? (y/N): ")

				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Deletion cancelled.")
					return nil
				}
			}

			result, err := geminiClient.DeleteAgent(agentName)
			if err != nil {
				return fmt.Errorf("failed to delete agent: %w", err)
			}

			return outputDeleteResult(result, config.Format)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

func resolveDialogflowAgentResource(fullResource, projectID, location, agentID string) (string, error) {
	fullResource = strings.TrimSpace(fullResource)
	projectID = strings.TrimSpace(projectID)
	location = strings.TrimSpace(location)
	agentID = strings.TrimSpace(agentID)

	if fullResource != "" {
		// Basic validation to ensure resource path resembles Dialogflow agent resource.
		if !strings.HasPrefix(fullResource, "projects/") || !strings.Contains(fullResource, "/agents/") {
			return "", fmt.Errorf("dialogflow agent must be in the form projects/PROJECT/locations/LOCATION/agents/AGENT_ID")
		}
		return fullResource, nil
	}

	if projectID == "" && location == "" && agentID == "" {
		return "", nil
	}

	if projectID == "" || location == "" || agentID == "" {
		return "", fmt.Errorf("dialogflow agent requires --dialogflow-project-id, --dialogflow-location, and --dialogflow-agent-id")
	}

	return fmt.Sprintf("projects/%s/locations/%s/agents/%s", projectID, location, agentID), nil
}
