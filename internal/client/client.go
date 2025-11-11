package client

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/discoveryengine/v1"
	"google.golang.org/api/option"
)

// Config holds the configuration for the Gemini client
type Config struct {
	ProjectID         string
	Location          string
	Collection        string
	UseServiceAccount bool
	Format            string
}

// GeminiClient handles interactions with the Gemini Enterprise API
type GeminiClient struct {
	service    *discoveryengine.Service
	config     *Config
	httpClient *http.Client
	basePath   string
}

// Engine represents a Gemini Enterprise engine
type Engine struct {
	Name               string                 `json:"name"`
	DisplayName        string                 `json:"displayName"`
	SolutionType       string                 `json:"solutionType"`
	IndustryVertical   string                 `json:"industryVertical"`
	AppType            string                 `json:"appType"`
	CreateTime         string                 `json:"createTime"`
	DataStoreIds       []string               `json:"dataStoreIds,omitempty"`
	SearchEngineConfig *SearchEngineConfig    `json:"searchEngineConfig,omitempty"`
	CommonConfig       map[string]interface{} `json:"commonConfig,omitempty"`
	Features           map[string]string      `json:"features,omitempty"`
}

// SearchEngineConfig represents search engine configuration
type SearchEngineConfig struct {
	SearchTier   string   `json:"searchTier"`
	SearchAddOns []string `json:"searchAddOns,omitempty"`
}

// DataStore represents a Gemini Enterprise data store
type DataStore struct {
	Name                     string                 `json:"name"`
	DisplayName              string                 `json:"displayName"`
	IndustryVertical         string                 `json:"industryVertical"`
	ContentConfig            string                 `json:"contentConfig"`
	CreateTime               string                 `json:"createTime"`
	SolutionTypes            []string               `json:"solutionTypes,omitempty"`
	AclEnabled               bool                   `json:"aclEnabled,omitempty"`
	BillingEstimation        *BillingEstimation     `json:"billingEstimation,omitempty"`
	DocumentProcessingConfig map[string]interface{} `json:"documentProcessingConfig,omitempty"`
	Schema                   map[string]interface{} `json:"schema,omitempty"`
}

// BillingEstimation represents billing information
type BillingEstimation struct {
	UnstructuredDataSize       int64  `json:"unstructuredDataSize"`
	UnstructuredDataUpdateTime string `json:"unstructuredDataUpdateTime"`
}

// Document represents a document in a data store
type Document struct {
	ID        string                 `json:"id"`
	Content   map[string]interface{} `json:"content"`
	IndexTime string                 `json:"indexTime"`
}

// CreateResult represents the result of a create operation
type CreateResult struct {
	EngineName      string                 `json:"engine_name,omitempty"`
	DataStoreName   string                 `json:"data_store_name,omitempty"`
	ImportOperation map[string]interface{} `json:"import_operation,omitempty"`
	Status          string                 `json:"status"`
	Error           string                 `json:"error,omitempty"`
}

// DeleteResult represents the result of a delete operation
type DeleteResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(config *Config) (*GeminiClient, error) {
	// Set defaults
	if config.Location == "" {
		config.Location = getDefaultLocation()
	}
	if config.ProjectID == "" {
		projectID, err := getDefaultProject()
		if err != nil {
			return nil, fmt.Errorf("project ID is required: %w", err)
		}
		config.ProjectID = projectID
	}

	ctx := context.Background()
	var service *discoveryengine.Service
	var err error

	// Determine the correct API endpoint based on location
	var baseURL string
	if config.Location == "global" {
		baseURL = "https://discoveryengine.googleapis.com/"
	} else {
		// For regional locations, use the regional endpoint
		regionPrefix := config.Location
		if strings.Contains(config.Location, "-") {
			regionPrefix = strings.Split(config.Location, "-")[0]
		}
		baseURL = fmt.Sprintf("https://%s-discoveryengine.googleapis.com/", regionPrefix)
	}

	requiredScopes := []string{
		discoveryengine.CloudPlatformScope,
		discoveryengine.CloudSearchQueryScope,
		discoveryengine.DiscoveryengineAssistReadwriteScope,
		discoveryengine.DiscoveryengineReadwriteScope,
	}

	opts := []option.ClientOption{
		option.WithEndpoint(baseURL),
	}

	var httpClient *http.Client

	if config.UseServiceAccount {
		// Use Application Default Credentials
		creds, err := google.FindDefaultCredentials(ctx, requiredScopes...)
		if err != nil {
			return nil, fmt.Errorf("failed to find default credentials: %w", err)
		}

		if creds.ProjectID != "" && config.ProjectID == "" {
			config.ProjectID = creds.ProjectID
		}

		httpClient = oauth2.NewClient(ctx, creds.TokenSource)
		opts = append(opts, option.WithTokenSource(creds.TokenSource))
	} else {
		// Use user credentials via gcloud auth print-access-token
		tokenSource, err := getUserTokenSource()
		if err != nil {
			return nil, fmt.Errorf("failed to get user token source: %w", err)
		}

		httpClient = oauth2.NewClient(ctx, tokenSource)
		opts = append(opts, option.WithTokenSource(tokenSource))

		if config.ProjectID != "" {
			opts = append(opts, option.WithQuotaProject(config.ProjectID))
		}
	}

	service, err = discoveryengine.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create discoveryengine service: %w", err)
	}

	return &GeminiClient{
		service:    service,
		config:     config,
		httpClient: httpClient,
	}, nil
}

// getUserTokenSource creates a token source using gcloud auth print-access-token
func getUserTokenSource() (oauth2.TokenSource, error) {
	return oauth2.ReuseTokenSource(nil, &gcloudTokenSource{}), nil
}

// gcloudTokenSource implements oauth2.TokenSource using gcloud auth print-access-token
type gcloudTokenSource struct{}

func (g *gcloudTokenSource) Token() (*oauth2.Token, error) {
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token from gcloud: %w", err)
	}

	token := strings.TrimSpace(string(output))
	return &oauth2.Token{
		AccessToken: token,
		Expiry:      time.Now().Add(50 * time.Minute), // Tokens typically last 1 hour
	}, nil
}

// getDefaultProject gets the default project from environment or gcloud config
func getDefaultProject() (string, error) {
	// Try environment variables first
	if project := os.Getenv("GOOGLE_CLOUD_PROJECT"); project != "" {
		return project, nil
	}
	if project := os.Getenv("GCLOUD_PROJECT"); project != "" {
		return project, nil
	}

	// Try gcloud config
	cmd := exec.Command("gcloud", "config", "get-value", "project")
	output, err := cmd.Output()
	if err == nil {
		project := strings.TrimSpace(string(output))
		if project != "" {
			return project, nil
		}
	}

	// Try from credentials
	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx, discoveryengine.CloudPlatformScope)
	if err == nil && creds.ProjectID != "" {
		return creds.ProjectID, nil
	}

	return "", fmt.Errorf("no project ID found in environment variables, gcloud config, or credentials")
}

// getDefaultLocation gets the default location from environment or uses 'us'
func getDefaultLocation() string {
	if location := os.Getenv("AGENTSPACE_LOCATION"); location != "" {
		return location
	}
	if location := os.Getenv("GCLOUD_LOCATION"); location != "" {
		return location
	}
	return "us"
}
