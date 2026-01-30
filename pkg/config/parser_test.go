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
	"testing"

	mcpgatewayv1alpha1 "github.com/aws/mcp-gateway-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseEndpoint(t *testing.T) {
	parser := NewConfigParser("default-gateway")

	tests := []struct {
		name      string
		endpoint  string
		wantErr   bool
		errSubstr string
	}{
		{
			name:     "valid https endpoint",
			endpoint: "https://example.com/mcp",
			wantErr:  false,
		},
		{
			name:     "valid https endpoint with port",
			endpoint: "https://example.com:8080/mcp",
			wantErr:  false,
		},
		{
			name:     "valid https endpoint with path",
			endpoint: "https://api.example.com/v1/mcp/server",
			wantErr:  false,
		},
		{
			name:      "invalid http endpoint",
			endpoint:  "http://example.com/mcp",
			wantErr:   true,
			errSubstr: "must match pattern ^https://",
		},
		{
			name:      "invalid ftp endpoint",
			endpoint:  "ftp://example.com/mcp",
			wantErr:   true,
			errSubstr: "must match pattern ^https://",
		},
		{
			name:      "empty endpoint",
			endpoint:  "",
			wantErr:   true,
			errSubstr: "endpoint is required",
		},
		{
			name:      "invalid no protocol",
			endpoint:  "example.com/mcp",
			wantErr:   true,
			errSubstr: "must match pattern ^https://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseEndpoint(tt.endpoint)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseEndpoint() expected error but got none")
				} else if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("ParseEndpoint() error = %v, want substring %v", err, tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("ParseEndpoint() unexpected error = %v", err)
				}
				if result != tt.endpoint {
					t.Errorf("ParseEndpoint() = %v, want %v", result, tt.endpoint)
				}
			}
		})
	}
}

func TestParseCapabilities(t *testing.T) {
	parser := NewConfigParser("default-gateway")

	tests := []struct {
		name         string
		capabilities []string
		wantErr      bool
		errSubstr    string
	}{
		{
			name:         "valid with tools only",
			capabilities: []string{"tools"},
			wantErr:      false,
		},
		{
			name:         "valid with tools and other capabilities",
			capabilities: []string{"tools", "prompts", "resources"},
			wantErr:      false,
		},
		{
			name:         "valid with tools in middle",
			capabilities: []string{"prompts", "tools", "resources"},
			wantErr:      false,
		},
		{
			name:         "invalid without tools",
			capabilities: []string{"prompts", "resources"},
			wantErr:      true,
			errSubstr:    "must include 'tools'",
		},
		{
			name:         "invalid empty capabilities",
			capabilities: []string{},
			wantErr:      true,
			errSubstr:    "capabilities are required",
		},
		{
			name:         "invalid nil capabilities",
			capabilities: nil,
			wantErr:      true,
			errSubstr:    "capabilities are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.ParseCapabilities(tt.capabilities)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseCapabilities() expected error but got none")
				} else if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("ParseCapabilities() error = %v, want substring %v", err, tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("ParseCapabilities() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestParseAuthConfig(t *testing.T) {
	parser := NewConfigParser("default-gateway")

	tests := []struct {
		name      string
		mcpServer *mcpgatewayv1alpha1.MCPServer
		want      *AuthConfig
		wantErr   bool
		errSubstr string
	}{
		{
			name: "NoAuth explicit",
			mcpServer: &mcpgatewayv1alpha1.MCPServer{
				Spec: mcpgatewayv1alpha1.MCPServerSpec{
					AuthType: "NoAuth",
				},
			},
			want: &AuthConfig{
				Type: "NoAuth",
			},
			wantErr: false,
		},
		{
			name: "NoAuth default when empty",
			mcpServer: &mcpgatewayv1alpha1.MCPServer{
				Spec: mcpgatewayv1alpha1.MCPServerSpec{
					AuthType: "",
				},
			},
			want: &AuthConfig{
				Type: "NoAuth",
			},
			wantErr: false,
		},
		{
			name: "OAuth2 with provider ARN",
			mcpServer: &mcpgatewayv1alpha1.MCPServer{
				Spec: mcpgatewayv1alpha1.MCPServerSpec{
					AuthType:         "OAuth2",
					OauthProviderArn: "arn:aws:bedrock-agentcore:us-east-1:123456789012:oauth-provider/my-provider",
				},
			},
			want: &AuthConfig{
				Type:             "OAuth2",
				OauthProviderArn: "arn:aws:bedrock-agentcore:us-east-1:123456789012:oauth-provider/my-provider",
			},
			wantErr: false,
		},
		{
			name: "OAuth2 with provider ARN and scopes",
			mcpServer: &mcpgatewayv1alpha1.MCPServer{
				Spec: mcpgatewayv1alpha1.MCPServerSpec{
					AuthType:         "OAuth2",
					OauthProviderArn: "arn:aws:bedrock-agentcore:us-east-1:123456789012:oauth-provider/my-provider",
					OauthScopes:      []string{"read", "write"},
				},
			},
			want: &AuthConfig{
				Type:             "OAuth2",
				OauthProviderArn: "arn:aws:bedrock-agentcore:us-east-1:123456789012:oauth-provider/my-provider",
				OauthScopes:      []string{"read", "write"},
			},
			wantErr: false,
		},
		{
			name: "OAuth2 without provider ARN",
			mcpServer: &mcpgatewayv1alpha1.MCPServer{
				Spec: mcpgatewayv1alpha1.MCPServerSpec{
					AuthType: "OAuth2",
				},
			},
			wantErr:   true,
			errSubstr: "oauthProviderArn is required when authType is OAuth2",
		},
		{
			name: "invalid auth type",
			mcpServer: &mcpgatewayv1alpha1.MCPServer{
				Spec: mcpgatewayv1alpha1.MCPServerSpec{
					AuthType: "BasicAuth",
				},
			},
			wantErr:   true,
			errSubstr: "unsupported authType",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseAuthConfig(tt.mcpServer)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseAuthConfig() expected error but got none")
				} else if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("ParseAuthConfig() error = %v, want substring %v", err, tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("ParseAuthConfig() unexpected error = %v", err)
				}
				if !authConfigEqual(result, tt.want) {
					t.Errorf("ParseAuthConfig() = %+v, want %+v", result, tt.want)
				}
			}
		})
	}
}

func TestParseMetadataConfig(t *testing.T) {
	parser := NewConfigParser("default-gateway")

	tests := []struct {
		name      string
		mcpServer *mcpgatewayv1alpha1.MCPServer
		want      *MetadataConfig
	}{
		{
			name: "all metadata fields present",
			mcpServer: &mcpgatewayv1alpha1.MCPServer{
				Spec: mcpgatewayv1alpha1.MCPServerSpec{
					AllowedRequestHeaders:  []string{"X-Custom-Header", "Authorization"},
					AllowedQueryParameters: []string{"filter", "page"},
					AllowedResponseHeaders: []string{"X-Response-Id"},
				},
			},
			want: &MetadataConfig{
				AllowedRequestHeaders:  []string{"X-Custom-Header", "Authorization"},
				AllowedQueryParameters: []string{"filter", "page"},
				AllowedResponseHeaders: []string{"X-Response-Id"},
			},
		},
		{
			name: "only request headers",
			mcpServer: &mcpgatewayv1alpha1.MCPServer{
				Spec: mcpgatewayv1alpha1.MCPServerSpec{
					AllowedRequestHeaders: []string{"X-Custom-Header"},
				},
			},
			want: &MetadataConfig{
				AllowedRequestHeaders:  []string{"X-Custom-Header"},
				AllowedQueryParameters: nil,
				AllowedResponseHeaders: nil,
			},
		},
		{
			name: "no metadata fields",
			mcpServer: &mcpgatewayv1alpha1.MCPServer{
				Spec: mcpgatewayv1alpha1.MCPServerSpec{},
			},
			want: &MetadataConfig{
				AllowedRequestHeaders:  nil,
				AllowedQueryParameters: nil,
				AllowedResponseHeaders: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseMetadataConfig(tt.mcpServer)
			if !metadataConfigEqual(result, tt.want) {
				t.Errorf("ParseMetadataConfig() = %+v, want %+v", result, tt.want)
			}
		})
	}
}

func TestGetGatewayID(t *testing.T) {
	tests := []struct {
		name             string
		defaultGatewayID string
		mcpServer        *mcpgatewayv1alpha1.MCPServer
		want             string
		wantErr          bool
		errSubstr        string
	}{
		{
			name:             "use spec gateway ID",
			defaultGatewayID: "default-gateway",
			mcpServer: &mcpgatewayv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-server",
				},
				Spec: mcpgatewayv1alpha1.MCPServerSpec{
					GatewayID: "custom-gateway",
				},
			},
			want:    "custom-gateway",
			wantErr: false,
		},
		{
			name:             "use default gateway ID when spec is empty",
			defaultGatewayID: "default-gateway",
			mcpServer: &mcpgatewayv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-server",
				},
				Spec: mcpgatewayv1alpha1.MCPServerSpec{
					GatewayID: "",
				},
			},
			want:    "default-gateway",
			wantErr: false,
		},
		{
			name:             "error when no gateway ID available",
			defaultGatewayID: "",
			mcpServer: &mcpgatewayv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-server",
				},
				Spec: mcpgatewayv1alpha1.MCPServerSpec{
					GatewayID: "",
				},
			},
			wantErr:   true,
			errSubstr: "no gatewayId specified",
		},
		{
			name:             "trim whitespace from spec gateway ID",
			defaultGatewayID: "default-gateway",
			mcpServer: &mcpgatewayv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-server",
				},
				Spec: mcpgatewayv1alpha1.MCPServerSpec{
					GatewayID: "  custom-gateway  ",
				},
			},
			want:    "custom-gateway",
			wantErr: false,
		},
		{
			name:             "error when spec gateway ID is only whitespace",
			defaultGatewayID: "default-gateway",
			mcpServer: &mcpgatewayv1alpha1.MCPServer{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-server",
				},
				Spec: mcpgatewayv1alpha1.MCPServerSpec{
					GatewayID: "   ",
				},
			},
			wantErr:   true,
			errSubstr: "gatewayId cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewConfigParser(tt.defaultGatewayID)
			result, err := parser.GetGatewayID(tt.mcpServer)
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetGatewayID() expected error but got none")
				} else if tt.errSubstr != "" && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("GetGatewayID() error = %v, want substring %v", err, tt.errSubstr)
				}
			} else {
				if err != nil {
					t.Errorf("GetGatewayID() unexpected error = %v", err)
				}
				if result != tt.want {
					t.Errorf("GetGatewayID() = %v, want %v", result, tt.want)
				}
			}
		})
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func authConfigEqual(a, b *AuthConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Type != b.Type || a.OauthProviderArn != b.OauthProviderArn {
		return false
	}
	return stringSliceEqual(a.OauthScopes, b.OauthScopes)
}

func metadataConfigEqual(a, b *MetadataConfig) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return stringSliceEqual(a.AllowedRequestHeaders, b.AllowedRequestHeaders) &&
		stringSliceEqual(a.AllowedQueryParameters, b.AllowedQueryParameters) &&
		stringSliceEqual(a.AllowedResponseHeaders, b.AllowedResponseHeaders)
}

func stringSliceEqual(a, b []string) bool {
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
