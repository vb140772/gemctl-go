package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"google.golang.org/api/discoveryengine/v1"
	"google.golang.org/api/googleapi"
)

const snapshotVersion = "v1"

// SnapshotMetadata captures contextual information for a snapshot.
type SnapshotMetadata struct {
	Version            string `json:"version"`
	OriginalEngineName string `json:"originalEngineName"`
	OriginalEngineID   string `json:"originalEngineId"`
	SourceProjectID    string `json:"sourceProjectId"`
	SourceLocation     string `json:"sourceLocation"`
	SourceCollection   string `json:"sourceCollection"`
	TakenAt            string `json:"takenAt"`
	DisplayName        string `json:"displayName,omitempty"`
	Description        string `json:"description,omitempty"`
	Notes              string `json:"notes,omitempty"`
}

// EngineConfigSnapshot represents the engine configuration captured in a snapshot.
type EngineConfigSnapshot struct {
	DisplayName      string                 `json:"displayName"`
	SolutionType     string                 `json:"solutionType"`
	IndustryVertical string                 `json:"industryVertical"`
	AppType          string                 `json:"appType"`
	DataStoreIds     []string               `json:"dataStoreIds,omitempty"`
	CommonConfig     map[string]interface{} `json:"commonConfig,omitempty"`
	Features         map[string]string      `json:"features,omitempty"`
	SearchConfig     *SearchEngineConfig    `json:"searchConfig,omitempty"`
}

// EngineSnapshot bundles metadata, engine configuration, and related agents.
type EngineSnapshot struct {
	Metadata SnapshotMetadata     `json:"metadata"`
	Engine   EngineConfigSnapshot `json:"engine"`
	Agents   []*Agent             `json:"agents,omitempty"`
}

// SnapshotDiff summarizes differences between two snapshots or between a snapshot and live state.
type SnapshotDiff struct {
	MetadataChanges []FieldDiff   `json:"metadataChanges,omitempty"`
	EngineChanges   []FieldDiff   `json:"engineChanges,omitempty"`
	FeatureChanges  []FeatureDiff `json:"featureChanges,omitempty"`
	AgentChanges    []AgentDiff   `json:"agentChanges,omitempty"`
}

// FieldDiff represents a change in a simple field.
type FieldDiff struct {
	Field string      `json:"field"`
	Old   interface{} `json:"old,omitempty"`
	New   interface{} `json:"new,omitempty"`
}

// FeatureDiff represents a change in a feature flag.
type FeatureDiff struct {
	Feature string `json:"feature"`
	Old     string `json:"old,omitempty"`
	New     string `json:"new,omitempty"`
}

// AgentDiffKind enumerates agent change types.
type AgentDiffKind string

const (
	// AgentDiffAdded indicates an agent is present in the desired snapshot but not in the current engine.
	AgentDiffAdded AgentDiffKind = "added"
	// AgentDiffRemoved indicates an agent exists in the current engine but not in the snapshot.
	AgentDiffRemoved AgentDiffKind = "removed"
	// AgentDiffUpdated indicates an agent exists in both but fields differ.
	AgentDiffUpdated AgentDiffKind = "updated"
)

// AgentDiff captures differences for a single agent.
type AgentDiff struct {
	Key        string        `json:"key"`
	ChangeType AgentDiffKind `json:"changeType"`
	Old        *Agent        `json:"old,omitempty"`
	New        *Agent        `json:"new,omitempty"`
}

// SnapshotRestoreOptions control how a snapshot is restored.
type SnapshotRestoreOptions struct {
	TargetEngineName string
	CreateIfMissing  bool
	UpdateExisting   bool
	DryRun           bool
}

// SnapshotRestoreResult reports actions taken during restore.
type SnapshotRestoreResult struct {
	EngineName     string        `json:"engineName"`
	Created        bool          `json:"created"`
	EnginePatched  bool          `json:"enginePatched"`
	FeatureChanges []FeatureDiff `json:"featureChanges,omitempty"`
	AgentChanges   []AgentDiff   `json:"agentChanges,omitempty"`
}

// CreateEngineSnapshot captures the engine configuration and registered agents.
func (c *GeminiClient) CreateEngineSnapshot(engineName string) (*EngineSnapshot, error) {
	engine, err := c.GetEngineDetails(engineName)
	if err != nil {
		return nil, fmt.Errorf("failed to get engine details: %w", err)
	}

	agents, err := c.ListAgents(engineName)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	engineID := extractResourceID(engine.Name)

	configSnapshot := EngineConfigSnapshot{
		DisplayName:      engine.DisplayName,
		SolutionType:     engine.SolutionType,
		IndustryVertical: engine.IndustryVertical,
		AppType:          engine.AppType,
		DataStoreIds:     append([]string{}, engine.DataStoreIds...),
		CommonConfig:     cloneStringInterfaceMap(engine.CommonConfig),
		Features:         cloneStringMap(engine.Features),
		SearchConfig:     engine.SearchEngineConfig,
	}

	metadata := SnapshotMetadata{
		Version:            snapshotVersion,
		OriginalEngineName: engine.Name,
		OriginalEngineID:   engineID,
		SourceProjectID:    c.config.ProjectID,
		SourceLocation:     c.config.Location,
		SourceCollection:   c.config.Collection,
		DisplayName:        engine.DisplayName,
		TakenAt:            time.Now().UTC().Format(time.RFC3339),
	}

	return &EngineSnapshot{
		Metadata: metadata,
		Engine:   configSnapshot,
		Agents:   cloneAgents(agents),
	}, nil
}

// Serialize snapshot to JSON bytes.
func (s *EngineSnapshot) MarshalJSONBytes() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// LoadEngineSnapshot parses snapshot JSON bytes.
func LoadEngineSnapshot(data []byte) (*EngineSnapshot, error) {
	var snapshot EngineSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to decode snapshot: %w", err)
	}
	return &snapshot, nil
}

// DiffSnapshots compares two snapshots.
func DiffSnapshots(a, b *EngineSnapshot) SnapshotDiff {
	diff := SnapshotDiff{}

	diff.MetadataChanges = append(diff.MetadataChanges, diffMetadata(a.Metadata, b.Metadata)...)
	diff.EngineChanges = append(diff.EngineChanges, diffEngineConfig(a.Engine, b.Engine)...)
	diff.FeatureChanges = append(diff.FeatureChanges, diffFeatures(a.Engine.Features, b.Engine.Features)...)
	diff.AgentChanges = append(diff.AgentChanges, diffAgents(a.Agents, b.Agents)...)

	return diff
}

// DiffSnapshotWithEngine compares a snapshot with current engine state.
func (c *GeminiClient) DiffSnapshotWithEngine(snapshot *EngineSnapshot, engineName string) (SnapshotDiff, error) {
	engine, err := c.GetEngineDetails(engineName)
	if err != nil {
		return SnapshotDiff{}, fmt.Errorf("failed to get engine details: %w", err)
	}

	agents, err := c.ListAgents(engineName)
	if err != nil {
		return SnapshotDiff{}, fmt.Errorf("failed to list agents: %w", err)
	}

	currentSnapshot := EngineSnapshot{
		Metadata: SnapshotMetadata{
			Version:            snapshotVersion,
			OriginalEngineName: engine.Name,
			OriginalEngineID:   extractResourceID(engine.Name),
			DisplayName:        engine.DisplayName,
			TakenAt:            time.Now().UTC().Format(time.RFC3339),
		},
		Engine: EngineConfigSnapshot{
			DisplayName:      engine.DisplayName,
			SolutionType:     engine.SolutionType,
			IndustryVertical: engine.IndustryVertical,
			AppType:          engine.AppType,
			DataStoreIds:     append([]string{}, engine.DataStoreIds...),
			CommonConfig:     cloneStringInterfaceMap(engine.CommonConfig),
			Features:         cloneStringMap(engine.Features),
			SearchConfig:     engine.SearchEngineConfig,
		},
		Agents: cloneAgents(agents),
	}

	return DiffSnapshots(snapshot, &currentSnapshot), nil
}

// RestoreEngineSnapshot restores snapshot content to a target engine.
func (c *GeminiClient) RestoreEngineSnapshot(snapshot *EngineSnapshot, opts SnapshotRestoreOptions) (*SnapshotRestoreResult, SnapshotDiff, error) {
	if snapshot == nil {
		return nil, SnapshotDiff{}, fmt.Errorf("snapshot is nil")
	}
	if opts.TargetEngineName == "" {
		return nil, SnapshotDiff{}, fmt.Errorf("target engine name is required")
	}

	var existingEngine *Engine
	var existingAgents []*Agent
	engine, err := c.GetEngineDetails(opts.TargetEngineName)
	if err != nil {
		if !isNotFound(err) {
			return nil, SnapshotDiff{}, fmt.Errorf("failed to get target engine: %w", err)
		}
		if !opts.CreateIfMissing {
			return nil, SnapshotDiff{}, fmt.Errorf("engine %s not found and create option disabled", opts.TargetEngineName)
		}
	} else {
		existingEngine = engine
		agents, err := c.ListAgents(opts.TargetEngineName)
		if err != nil {
			return nil, SnapshotDiff{}, fmt.Errorf("failed to list agents for target engine: %w", err)
		}
		existingAgents = agents
	}

	currentDiff := SnapshotDiff{}
	if existingEngine != nil {
		currentDiff = DiffSnapshots(snapshot, &EngineSnapshot{
			Metadata: SnapshotMetadata{
				Version:            snapshotVersion,
				OriginalEngineName: existingEngine.Name,
				OriginalEngineID:   extractResourceID(existingEngine.Name),
				DisplayName:        existingEngine.DisplayName,
				TakenAt:            time.Now().UTC().Format(time.RFC3339),
			},
			Engine: EngineConfigSnapshot{
				DisplayName:      existingEngine.DisplayName,
				SolutionType:     existingEngine.SolutionType,
				IndustryVertical: existingEngine.IndustryVertical,
				AppType:          existingEngine.AppType,
				DataStoreIds:     append([]string{}, existingEngine.DataStoreIds...),
				CommonConfig:     cloneStringInterfaceMap(existingEngine.CommonConfig),
				Features:         cloneStringMap(existingEngine.Features),
				SearchConfig:     existingEngine.SearchEngineConfig,
			},
			Agents: cloneAgents(existingAgents),
		})
	} else {
		currentDiff.EngineChanges = append(currentDiff.EngineChanges, FieldDiff{
			Field: "engine",
			Old:   "missing",
			New:   "will be created",
		})
		currentDiff.FeatureChanges = diffFeatures(map[string]string{}, snapshot.Engine.Features)
		currentDiff.AgentChanges = diffAgents(nil, snapshot.Agents)
	}

	if opts.DryRun {
		return nil, currentDiff, nil
	}

	result := &SnapshotRestoreResult{
		EngineName: opts.TargetEngineName,
	}

	if existingEngine == nil {
		if !opts.CreateIfMissing {
			return nil, currentDiff, fmt.Errorf("engine %s not found", opts.TargetEngineName)
		}
		if err := c.createEngineFromSnapshot(opts.TargetEngineName, snapshot.Engine); err != nil {
			return nil, currentDiff, fmt.Errorf("failed to create engine: %w", err)
		}
		result.Created = true
	} else if opts.UpdateExisting {
		if err := c.patchEngineFromSnapshot(opts.TargetEngineName, existingEngine, snapshot.Engine); err != nil {
			return nil, currentDiff, fmt.Errorf("failed to patch engine: %w", err)
		}
		result.EnginePatched = true
	}

	if err := c.applyFeatureSnapshot(opts.TargetEngineName, snapshot.Engine.Features); err != nil {
		return nil, currentDiff, fmt.Errorf("failed to apply feature snapshot: %w", err)
	}
	result.FeatureChanges = diffFeatures(existingEngineFeatures(existingEngine), snapshot.Engine.Features)

	agentChanges, err := c.syncAgentsWithSnapshot(opts.TargetEngineName, existingAgents, snapshot.Agents)
	if err != nil {
		return nil, currentDiff, fmt.Errorf("failed to sync agents: %w", err)
	}
	result.AgentChanges = agentChanges

	return result, currentDiff, nil
}

func (c *GeminiClient) createEngineFromSnapshot(engineName string, cfg EngineConfigSnapshot) error {
	engineID := extractResourceID(engineName)
	parent := fmt.Sprintf("projects/%s/locations/%s/collections/%s",
		c.config.ProjectID, c.config.Location, c.config.Collection)

	engine := &discoveryengine.GoogleCloudDiscoveryengineV1Engine{
		DisplayName:      cfg.DisplayName,
		SolutionType:     cfg.SolutionType,
		IndustryVertical: cfg.IndustryVertical,
		AppType:          cfg.AppType,
		DataStoreIds:     append([]string{}, cfg.DataStoreIds...),
		Features:         cloneStringMap(cfg.Features),
	}

	if cfg.SearchConfig != nil {
		engine.SearchEngineConfig = &discoveryengine.GoogleCloudDiscoveryengineV1EngineSearchEngineConfig{
			SearchTier:   cfg.SearchConfig.SearchTier,
			SearchAddOns: append([]string{}, cfg.SearchConfig.SearchAddOns...),
		}
	}

	if company, ok := cfg.CommonConfig["companyName"].(string); ok && company != "" {
		engine.CommonConfig = &discoveryengine.GoogleCloudDiscoveryengineV1EngineCommonConfig{
			CompanyName: company,
		}
	}

	call := c.service.Projects.Locations.Collections.Engines.Create(parent, engine)
	call.EngineId(engineID)
	if _, err := call.Do(); err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}
	return nil
}

func (c *GeminiClient) patchEngineFromSnapshot(engineName string, current *Engine, cfg EngineConfigSnapshot) error {
	if current == nil {
		return fmt.Errorf("current engine is nil")
	}

	updateMask := []string{}
	patch := &discoveryengine.GoogleCloudDiscoveryengineV1Engine{}

	if cfg.DisplayName != "" && cfg.DisplayName != current.DisplayName {
		patch.DisplayName = cfg.DisplayName
		updateMask = append(updateMask, "displayName")
	}

	if cfg.IndustryVertical != "" && cfg.IndustryVertical != current.IndustryVertical {
		patch.IndustryVertical = cfg.IndustryVertical
		updateMask = append(updateMask, "industryVertical")
	}

	if cfg.AppType != "" && cfg.AppType != current.AppType {
		patch.AppType = cfg.AppType
		updateMask = append(updateMask, "appType")
	}

	if !stringSlicesEqual(cfg.DataStoreIds, current.DataStoreIds) {
		patch.DataStoreIds = append([]string{}, cfg.DataStoreIds...)
		updateMask = append(updateMask, "dataStoreIds")
	}

	if cfg.SearchConfig != nil {
		currentTier := ""
		currentAddOns := []string{}
		if current.SearchEngineConfig != nil {
			currentTier = current.SearchEngineConfig.SearchTier
			currentAddOns = current.SearchEngineConfig.SearchAddOns
		}

		if cfg.SearchConfig.SearchTier != currentTier || !stringSlicesEqual(cfg.SearchConfig.SearchAddOns, currentAddOns) {
			patch.SearchEngineConfig = &discoveryengine.GoogleCloudDiscoveryengineV1EngineSearchEngineConfig{
				SearchTier:   cfg.SearchConfig.SearchTier,
				SearchAddOns: append([]string{}, cfg.SearchConfig.SearchAddOns...),
			}
			updateMask = append(updateMask, "searchEngineConfig")
		}
	}

	if len(updateMask) == 0 {
		return nil
	}

	call := c.service.Projects.Locations.Collections.Engines.Patch(engineName, patch)
	call.UpdateMask(strings.Join(updateMask, ","))

	if _, err := call.Do(); err != nil {
		return fmt.Errorf("failed to patch engine: %w", err)
	}

	return nil
}

func (c *GeminiClient) applyFeatureSnapshot(engineName string, desired map[string]string) error {
	if desired == nil {
		return nil
	}
	_, err := c.UpdateEngineFeatures(engineName, desired)
	return err
}

func (c *GeminiClient) syncAgentsWithSnapshot(engineName string, current []*Agent, desired []*Agent) ([]AgentDiff, error) {
	currentMap := make(map[string]*Agent)
	for _, agent := range current {
		currentMap[agentSnapshotKey(agent)] = agent
	}

	desiredMap := make(map[string]*Agent)
	for _, agent := range desired {
		desiredMap[agentSnapshotKey(agent)] = agent
	}

	changes := []AgentDiff{}

	for key, desiredAgent := range desiredMap {
		if existing, ok := currentMap[key]; ok {
			updateMask, needsUpdate := diffAgentsFields(existing, desiredAgent)
			if needsUpdate {
				if err := c.updateAgentFromSnapshot(engineName, existing, desiredAgent, updateMask); err != nil {
					return nil, err
				}
				changes = append(changes, AgentDiff{
					Key:        key,
					ChangeType: AgentDiffUpdated,
					Old:        existing,
					New:        desiredAgent,
				})
			}
		} else {
			if err := c.createAgentFromSnapshot(engineName, desiredAgent); err != nil {
				return nil, err
			}
			changes = append(changes, AgentDiff{
				Key:        key,
				ChangeType: AgentDiffAdded,
				New:        desiredAgent,
			})
		}
	}

	for key, existing := range currentMap {
		if _, ok := desiredMap[key]; !ok {
			if _, err := c.DeleteAgent(existing.Name); err != nil {
				return nil, fmt.Errorf("failed to delete agent %s: %w", existing.Name, err)
			}
			changes = append(changes, AgentDiff{
				Key:        key,
				ChangeType: AgentDiffRemoved,
				Old:        existing,
			})
		}
	}

	return changes, nil
}

func (c *GeminiClient) createAgentFromSnapshot(engineName string, agent *Agent) error {
	if agent == nil {
		return nil
	}
	input := &AgentCreateInput{
		DisplayName:     agent.DisplayName,
		Description:     agent.Description,
		ReasoningEngine: agent.ReasoningEngine,
	}
	if agent.Icon != nil {
		input.Icon = &AgentIcon{
			URI:     agent.Icon.URI,
			Content: agent.Icon.Content,
		}
	}
	if agent.DialogflowAgentDefinition != nil {
		if agent.DialogflowAgentDefinition.DialogflowAgent == "" {
			return fmt.Errorf("snapshot agent %s missing dialogflow agent resource", agent.DisplayName)
		}
		input.DialogflowAgentDefinition = &DialogflowAgentDefinition{
			DialogflowAgent: agent.DialogflowAgentDefinition.DialogflowAgent,
		}
	}
	if agent.ReasoningEngine != "" {
		input.ReasoningEngine = agent.ReasoningEngine
	} else {
		input.ReasoningEngine = engineName
	}
	_, err := c.CreateAgent(engineName, input)
	if err != nil {
		return fmt.Errorf("failed to create agent %s: %w", agent.DisplayName, err)
	}
	return nil
}

func (c *GeminiClient) updateAgentFromSnapshot(engineName string, existing, desired *Agent, mask []string) error {
	if len(mask) == 0 {
		return nil
	}
	input := &AgentUpdateInput{
		DisplayName:     desired.DisplayName,
		Description:     desired.Description,
		ReasoningEngine: desired.ReasoningEngine,
	}
	if desired.Icon != nil {
		input.Icon = &AgentIcon{
			URI:     desired.Icon.URI,
			Content: desired.Icon.Content,
		}
	}
	if desired.DialogflowAgentDefinition != nil {
		input.DialogflowAgentDefinition = &DialogflowAgentDefinition{
			DialogflowAgent: desired.DialogflowAgentDefinition.DialogflowAgent,
		}
	}
	_, err := c.UpdateAgent(existing.Name, input, mask)
	if err != nil {
		return fmt.Errorf("failed to update agent %s: %w", existing.Name, err)
	}
	return nil
}

func diffMetadata(a, b SnapshotMetadata) []FieldDiff {
	var diffs []FieldDiff
	if a.DisplayName != b.DisplayName {
		diffs = append(diffs, FieldDiff{Field: "displayName", Old: a.DisplayName, New: b.DisplayName})
	}
	if a.Description != b.Description {
		diffs = append(diffs, FieldDiff{Field: "description", Old: a.Description, New: b.Description})
	}
	if a.Notes != b.Notes {
		diffs = append(diffs, FieldDiff{Field: "notes", Old: a.Notes, New: b.Notes})
	}
	return diffs
}

func diffEngineConfig(a, b EngineConfigSnapshot) []FieldDiff {
	var diffs []FieldDiff
	if a.DisplayName != b.DisplayName {
		diffs = append(diffs, FieldDiff{Field: "displayName", Old: a.DisplayName, New: b.DisplayName})
	}
	if a.SolutionType != b.SolutionType {
		diffs = append(diffs, FieldDiff{Field: "solutionType", Old: a.SolutionType, New: b.SolutionType})
	}
	if a.IndustryVertical != b.IndustryVertical {
		diffs = append(diffs, FieldDiff{Field: "industryVertical", Old: a.IndustryVertical, New: b.IndustryVertical})
	}
	if a.AppType != b.AppType {
		diffs = append(diffs, FieldDiff{Field: "appType", Old: a.AppType, New: b.AppType})
	}
	if !stringSlicesEqual(a.DataStoreIds, b.DataStoreIds) {
		diffs = append(diffs, FieldDiff{Field: "dataStoreIds", Old: a.DataStoreIds, New: b.DataStoreIds})
	}
	if !mapsEqual(a.CommonConfig, b.CommonConfig) {
		diffs = append(diffs, FieldDiff{Field: "commonConfig", Old: a.CommonConfig, New: b.CommonConfig})
	}
	if !compareSearchConfigs(a.SearchConfig, b.SearchConfig) {
		diffs = append(diffs, FieldDiff{Field: "searchConfig", Old: a.SearchConfig, New: b.SearchConfig})
	}
	return diffs
}

func diffFeatures(a, b map[string]string) []FeatureDiff {
	diffs := []FeatureDiff{}
	allKeys := map[string]struct{}{}
	for k := range a {
		allKeys[k] = struct{}{}
	}
	for k := range b {
		allKeys[k] = struct{}{}
	}

	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		oldVal := a[key]
		newVal := b[key]
		if oldVal != newVal {
			diffs = append(diffs, FeatureDiff{
				Feature: key,
				Old:     oldVal,
				New:     newVal,
			})
		}
	}

	return diffs
}

func diffAgents(current, desired []*Agent) []AgentDiff {
	currentMap := make(map[string]*Agent)
	for _, agent := range current {
		currentMap[agentSnapshotKey(agent)] = agent
	}

	desiredMap := make(map[string]*Agent)
	for _, agent := range desired {
		desiredMap[agentSnapshotKey(agent)] = agent
	}

	changes := []AgentDiff{}

	for key, desiredAgent := range desiredMap {
		if existing, ok := currentMap[key]; ok {
			if _, needsUpdate := diffAgentsFields(existing, desiredAgent); needsUpdate {
				changes = append(changes, AgentDiff{
					Key:        key,
					ChangeType: AgentDiffUpdated,
					Old:        existing,
					New:        desiredAgent,
				})
			}
		} else {
			changes = append(changes, AgentDiff{
				Key:        key,
				ChangeType: AgentDiffAdded,
				New:        desiredAgent,
			})
		}
	}

	for key, existing := range currentMap {
		if _, ok := desiredMap[key]; !ok {
			changes = append(changes, AgentDiff{
				Key:        key,
				ChangeType: AgentDiffRemoved,
				Old:        existing,
			})
		}
	}

	return changes
}

func diffAgentsFields(current, desired *Agent) ([]string, bool) {
	if current == nil || desired == nil {
		return nil, false
	}

	updateMask := []string{}

	if current.DisplayName != desired.DisplayName {
		updateMask = append(updateMask, "displayName")
	}
	if current.Description != desired.Description {
		updateMask = append(updateMask, "description")
	}
	if current.ReasoningEngine != desired.ReasoningEngine && desired.ReasoningEngine != "" {
		updateMask = append(updateMask, "reasoningEngine")
	}

	currentDialogflow := ""
	if current.DialogflowAgentDefinition != nil {
		currentDialogflow = current.DialogflowAgentDefinition.DialogflowAgent
	}
	desiredDialogflow := ""
	if desired.DialogflowAgentDefinition != nil {
		desiredDialogflow = desired.DialogflowAgentDefinition.DialogflowAgent
	}
	if currentDialogflow != desiredDialogflow && desiredDialogflow != "" {
		updateMask = append(updateMask, "dialogflowAgentDefinition.dialogflowAgent")
	}

	currentIconURI := ""
	currentIconContent := ""
	if current.Icon != nil {
		currentIconURI = current.Icon.URI
		currentIconContent = current.Icon.Content
	}
	desiredIconURI := ""
	desiredIconContent := ""
	if desired.Icon != nil {
		desiredIconURI = desired.Icon.URI
		desiredIconContent = desired.Icon.Content
	}
	if currentIconURI != desiredIconURI || currentIconContent != desiredIconContent {
		updateMask = append(updateMask, "icon")
	}

	return updateMask, len(updateMask) > 0
}

func (d SnapshotDiff) IsEmpty() bool {
	return len(d.MetadataChanges) == 0 &&
		len(d.EngineChanges) == 0 &&
		len(d.FeatureChanges) == 0 &&
		len(d.AgentChanges) == 0
}

func agentSnapshotKey(agent *Agent) string {
	if agent == nil {
		return ""
	}
	if agent.DialogflowAgentDefinition != nil && agent.DialogflowAgentDefinition.DialogflowAgent != "" {
		return strings.ToLower(agent.DialogflowAgentDefinition.DialogflowAgent)
	}
	if agent.DisplayName != "" {
		return strings.ToLower(agent.DisplayName)
	}
	return strings.ToLower(agent.Name)
}

func cloneAgents(src []*Agent) []*Agent {
	cloned := make([]*Agent, 0, len(src))
	for _, agent := range src {
		if agent == nil {
			continue
		}
		copy := *agent
		if agent.Icon != nil {
			icon := *agent.Icon
			copy.Icon = &icon
		}
		if agent.DialogflowAgentDefinition != nil {
			def := *agent.DialogflowAgentDefinition
			copy.DialogflowAgentDefinition = &def
		}
		cloned = append(cloned, &copy)
	}
	return cloned
}

func cloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneStringInterfaceMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || fmt.Sprintf("%v", v) != fmt.Sprintf("%v", bv) {
			return false
		}
	}
	return true
}

func compareSearchConfigs(a, b *SearchEngineConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.SearchTier != b.SearchTier {
		return false
	}
	return stringSlicesEqual(a.SearchAddOns, b.SearchAddOns)
}

func existingEngineFeatures(engine *Engine) map[string]string {
	if engine == nil {
		return map[string]string{}
	}
	return cloneStringMap(engine.Features)
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == 404
	}
	return false
}

func extractResourceID(resourceName string) string {
	if resourceName == "" {
		return ""
	}
	parts := strings.Split(resourceName, "/")
	if len(parts) == 0 {
		return resourceName
	}
	return parts[len(parts)-1]
}
