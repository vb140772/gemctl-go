package client

import (
	"fmt"
	"strings"

	"google.golang.org/api/discoveryengine/v1"
	"google.golang.org/api/googleapi"
)

const (
	idpTypeThirdParty        = "THIRD_PARTY"
	idpTypeUnspecified       = "IDP_TYPE_UNSPECIFIED"
	defaultWorkforceLocation = "locations/global"
)

// WorkforceIdentityConfig represents the workforce identity pool configuration for a project/location.
type WorkforceIdentityConfig struct {
	IdpType           string `json:"idpType"`
	WorkforcePoolName string `json:"workforcePoolName,omitempty"`
	WorkforceLocation string `json:"workforceLocation,omitempty"`
	WorkforcePoolID   string `json:"workforcePoolId,omitempty"`
	WorkforceProvider string `json:"workforceProviderId,omitempty"`
}

// GetWorkforceIdentityConfig retrieves the workforce identity configuration for the current project/location.
func (c *GeminiClient) GetWorkforceIdentityConfig() (*WorkforceIdentityConfig, error) {
	if c.config == nil {
		return nil, fmt.Errorf("client configuration is missing")
	}

	name := fmt.Sprintf("projects/%s/locations/%s/aclConfig", c.config.ProjectID, c.config.Location)
	cfg, err := c.service.Projects.Locations.GetAclConfig(name).Do()
	if err != nil {
		if apiErr, ok := err.(*googleapi.Error); ok && apiErr.Code == 404 {
			return &WorkforceIdentityConfig{}, nil
		}
		return nil, fmt.Errorf("failed to get workforce identity configuration: %w", err)
	}

	result := &WorkforceIdentityConfig{}
	if cfg.IdpConfig != nil {
		result.IdpType = cfg.IdpConfig.IdpType
		if cfg.IdpConfig.ExternalIdpConfig != nil {
			result.WorkforcePoolName = cfg.IdpConfig.ExternalIdpConfig.WorkforcePoolName
		}
	}
	populateDerivedWorkforceFields(result)
	return result, nil
}

// SetWorkforceIdentityConfig updates the workforce identity pool configuration.
// The resource string should follow locations/{location}/workforcePools/{pool}[/providers/{provider}].
// Passing an empty string disables workforce identity.
func (c *GeminiClient) SetWorkforceIdentityConfig(workforceResource string) (*WorkforceIdentityConfig, error) {
	if c.config == nil {
		return nil, fmt.Errorf("client configuration is missing")
	}

	name := fmt.Sprintf("projects/%s/locations/%s/aclConfig", c.config.ProjectID, c.config.Location)
	request := &discoveryengine.GoogleCloudDiscoveryengineV1AclConfig{
		Name: name,
	}

	if strings.TrimSpace(workforceResource) == "" {
		request.IdpConfig = &discoveryengine.GoogleCloudDiscoveryengineV1IdpConfig{
			IdpType: idpTypeUnspecified,
		}
	} else {
		request.IdpConfig = &discoveryengine.GoogleCloudDiscoveryengineV1IdpConfig{
			IdpType: idpTypeThirdParty,
			ExternalIdpConfig: &discoveryengine.GoogleCloudDiscoveryengineV1IdpConfigExternalIdpConfig{
				WorkforcePoolName: workforceResource,
			},
		}
	}

	resp, err := c.service.Projects.Locations.UpdateAclConfig(name, request).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to update workforce identity configuration: %w", err)
	}

	result := &WorkforceIdentityConfig{}
	if resp.IdpConfig != nil {
		result.IdpType = resp.IdpConfig.IdpType
		if resp.IdpConfig.ExternalIdpConfig != nil {
			result.WorkforcePoolName = resp.IdpConfig.ExternalIdpConfig.WorkforcePoolName
		}
	}
	populateDerivedWorkforceFields(result)
	return result, nil
}

func populateDerivedWorkforceFields(cfg *WorkforceIdentityConfig) {
	if cfg == nil {
		return
	}
	resource := strings.TrimSpace(cfg.WorkforcePoolName)
	if resource == "" {
		return
	}

	segments := strings.Split(resource, "/")
	for i := 0; i < len(segments); i++ {
		switch segments[i] {
		case "locations":
			if i+1 < len(segments) {
				cfg.WorkforceLocation = fmt.Sprintf("locations/%s", segments[i+1])
			}
		case "workforcePools":
			if i+1 < len(segments) {
				cfg.WorkforcePoolID = segments[i+1]
			}
		case "providers":
			if i+1 < len(segments) {
				cfg.WorkforceProvider = segments[i+1]
			}
		}
	}
}
