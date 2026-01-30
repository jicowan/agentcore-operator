/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	mcpgatewayv1alpha1 "github.com/aws/mcp-gateway-operator/api/v1alpha1"
)

// ConfigParser validates and parses MCPServer spec fields
type ConfigParser struct {
	defaultGatewayID string
}

// NewConfigParser creates a new ConfigParser with the specified default gateway ID
func NewConfigParser(defaultGatewayID string) *ConfigParser {
	return &ConfigParser{
		defaultGatewayID: defaultGatewayID,
	}
}

// AuthConfig represents parsed authentication configuration
type AuthConfig struct {
	Type             string
	OauthProviderArn string
	OauthScopes      []string
}

// MetadataConfig represents parsed metadata propagation configuration
type MetadataConfig struct {
	AllowedRequestHeaders  []string
	AllowedQueryParameters []string
	AllowedResponseHeaders []string
}

// ParseEndpoint validates that the endpoint matches the HTTPS pattern
// Returns the endpoint if valid, or an error if invalid
func (p *ConfigParser) ParseEndpoint(endpoint string) (string, error) {
	if endpoint == "" {
		return "", fmt.Errorf("endpoint is required")
	}

	// Validate HTTPS pattern
	httpsPattern := regexp.MustCompile(`^https://.*`)
	if !httpsPattern.MatchString(endpoint) {
		return "", fmt.Errorf("endpoint must match pattern ^https://.* (got: %s)", endpoint)
	}

	return endpoint, nil
}

// ParseCapabilities validates that the capabilities include "tools"
// Returns an error if "tools" is not present
func (p *ConfigParser) ParseCapabilities(capabilities []string) error {
	if len(capabilities) == 0 {
		return fmt.Errorf("capabilities are required and must include 'tools'")
	}

	// Check if "tools" is present
	hasTools := slices.Contains(capabilities, "tools")

	if !hasTools {
		return fmt.Errorf("capabilities must include 'tools' (got: %v)", capabilities)
	}

	return nil
}

// ParseAuthConfig parses and validates authentication configuration
// Returns AuthConfig if valid, or an error if invalid
func (p *ConfigParser) ParseAuthConfig(mcpServer *mcpgatewayv1alpha1.MCPServer) (*AuthConfig, error) {
	authType := mcpServer.Spec.AuthType
	if authType == "" {
		// Default to NoAuth
		authType = "NoAuth"
	}

	config := &AuthConfig{
		Type: authType,
	}

	switch authType {
	case "NoAuth":
		// No additional validation needed
		return config, nil

	case "OAuth2":
		// OAuth2 requires OauthProviderArn
		if mcpServer.Spec.OauthProviderArn == "" {
			return nil, fmt.Errorf("oauthProviderArn is required when authType is OAuth2")
		}
		config.OauthProviderArn = mcpServer.Spec.OauthProviderArn
		config.OauthScopes = mcpServer.Spec.OauthScopes
		return config, nil

	default:
		return nil, fmt.Errorf("unsupported authType: %s (must be NoAuth or OAuth2)", authType)
	}
}

// ParseMetadataConfig parses metadata propagation configuration
// Returns MetadataConfig with the configured headers and parameters
func (p *ConfigParser) ParseMetadataConfig(mcpServer *mcpgatewayv1alpha1.MCPServer) *MetadataConfig {
	config := &MetadataConfig{
		AllowedRequestHeaders:  mcpServer.Spec.AllowedRequestHeaders,
		AllowedQueryParameters: mcpServer.Spec.AllowedQueryParameters,
		AllowedResponseHeaders: mcpServer.Spec.AllowedResponseHeaders,
	}

	return config
}

// GetGatewayID returns the gateway ID from the spec or the default gateway ID
// Returns an error if no gateway ID is available
func (p *ConfigParser) GetGatewayID(mcpServer *mcpgatewayv1alpha1.MCPServer) (string, error) {
	// Use spec.GatewayID if present
	if mcpServer.Spec.GatewayID != "" {
		gatewayID := strings.TrimSpace(mcpServer.Spec.GatewayID)
		if gatewayID == "" {
			return "", fmt.Errorf("gatewayId cannot be empty")
		}
		return gatewayID, nil
	}

	// Fall back to default gateway ID
	if p.defaultGatewayID == "" {
		return "", fmt.Errorf("no gatewayId specified in spec and no default gateway ID configured")
	}

	return p.defaultGatewayID, nil
}
