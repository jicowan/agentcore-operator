/*
Copyright 2025.

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

package bedrock

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol/types"

	mcpgatewayv1alpha1 "github.com/aws/mcp-gateway-operator/api/v1alpha1"
)

// TargetConfigBuilder builds AWS Bedrock gateway target configuration from MCPServer spec
type TargetConfigBuilder struct{}

// NewTargetConfigBuilder creates a new TargetConfigBuilder
func NewTargetConfigBuilder() *TargetConfigBuilder {
	return &TargetConfigBuilder{}
}

// Build creates a TargetConfiguration for an MCP server
// It builds the MCP server configuration with the endpoint from the MCPServer spec
func (b *TargetConfigBuilder) Build(mcpServer *mcpgatewayv1alpha1.MCPServer) (types.TargetConfiguration, error) {
	if mcpServer == nil {
		return nil, fmt.Errorf("mcpServer cannot be nil")
	}

	if mcpServer.Spec.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}

	return &types.TargetConfigurationMemberMcp{
		Value: &types.McpTargetConfigurationMemberMcpServer{
			Value: types.McpServerTargetConfiguration{
				Endpoint: aws.String(mcpServer.Spec.Endpoint),
			},
		},
	}, nil
}

// BuildCredentialConfig creates credential provider configuration based on the auth type
// For NoAuth: returns GatewayIamRole credential type
// For OAuth2: returns OAuth credential type with provider ARN and scopes
func (b *TargetConfigBuilder) BuildCredentialConfig(mcpServer *mcpgatewayv1alpha1.MCPServer) ([]types.CredentialProviderConfiguration, error) {
	if mcpServer == nil {
		return nil, fmt.Errorf("mcpServer cannot be nil")
	}

	authType := mcpServer.Spec.AuthType
	if authType == "" {
		authType = "NoAuth" // Default to NoAuth
	}

	switch authType {
	case "NoAuth":
		return []types.CredentialProviderConfiguration{
			{
				CredentialProviderType: types.CredentialProviderTypeGatewayIamRole,
			},
		}, nil

	case "OAuth2":
		if mcpServer.Spec.OauthProviderArn == "" {
			return nil, fmt.Errorf("oauthProviderArn is required when authType is OAuth2")
		}

		return []types.CredentialProviderConfiguration{
			{
				CredentialProviderType: types.CredentialProviderTypeOauth,
				CredentialProvider: &types.CredentialProviderMemberOauthCredentialProvider{
					Value: types.OAuthCredentialProvider{
						ProviderArn: aws.String(mcpServer.Spec.OauthProviderArn),
						Scopes:      mcpServer.Spec.OauthScopes,
						GrantType:   types.OAuthGrantTypeClientCredentials,
					},
				},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported auth type: %s", authType)
	}
}

// BuildMetadataConfig creates metadata configuration for header and parameter propagation
// Returns nil if no metadata fields are present
// Returns MetadataConfiguration with allowed headers/parameters if at least one field is present
func (b *TargetConfigBuilder) BuildMetadataConfig(mcpServer *mcpgatewayv1alpha1.MCPServer) *types.MetadataConfiguration {
	if mcpServer == nil {
		return nil
	}

	// Check if any metadata fields are present
	hasRequestHeaders := len(mcpServer.Spec.AllowedRequestHeaders) > 0
	hasQueryParameters := len(mcpServer.Spec.AllowedQueryParameters) > 0
	hasResponseHeaders := len(mcpServer.Spec.AllowedResponseHeaders) > 0

	// Return nil if no metadata fields are present
	if !hasRequestHeaders && !hasQueryParameters && !hasResponseHeaders {
		return nil
	}

	// Build metadata configuration with present fields
	return &types.MetadataConfiguration{
		AllowedRequestHeaders:  mcpServer.Spec.AllowedRequestHeaders,
		AllowedQueryParameters: mcpServer.Spec.AllowedQueryParameters,
		AllowedResponseHeaders: mcpServer.Spec.AllowedResponseHeaders,
	}
}
