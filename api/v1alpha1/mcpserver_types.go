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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MCPServerSpec defines the desired state of MCPServer
type MCPServerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// Endpoint is the HTTPS endpoint of the MCP server
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^https://.*`
	Endpoint string `json:"endpoint"`

	// Capabilities are the server capabilities (must include "tools")
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Capabilities []string `json:"capabilities"`

	// GatewayID is the gateway identifier (defaults to env var if not specified)
	// +optional
	GatewayID string `json:"gatewayId,omitempty"`

	// TargetName is the custom target name (defaults to resource name if not specified)
	// +optional
	TargetName string `json:"targetName,omitempty"`

	// Description is the target description
	// +optional
	Description string `json:"description,omitempty"`

	// AuthType is the authentication type
	// Note: MCP server targets only support OAuth2 authentication.
	// NoAuth (using gateway IAM role) is not supported for MCP servers.
	// +kubebuilder:validation:Pattern=`^(OAuth2)$`
	// +kubebuilder:default="OAuth2"
	// +optional
	AuthType string `json:"authType,omitempty"`

	// OauthProviderArn is the OAuth provider ARN
	// Required for MCP server targets (AuthType must be OAuth2)
	// Example: arn:aws:bedrock-agentcore:us-west-2:123456789012:token-vault/default/oauth2credentialprovider/my-provider
	// +kubebuilder:validation:Required
	OauthProviderArn string `json:"oauthProviderArn"`

	// OauthScopes are the OAuth scopes to request
	// At least one scope is required for OAuth2 authentication
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	OauthScopes []string `json:"oauthScopes"`

	// AllowedRequestHeaders are the allowed request headers for metadata propagation
	// +optional
	AllowedRequestHeaders []string `json:"allowedRequestHeaders,omitempty"`

	// AllowedQueryParameters are the allowed query parameters for metadata propagation
	// +optional
	AllowedQueryParameters []string `json:"allowedQueryParameters,omitempty"`

	// AllowedResponseHeaders are the allowed response headers for metadata propagation
	// +optional
	AllowedResponseHeaders []string `json:"allowedResponseHeaders,omitempty"`
}

// MCPServerStatus defines the observed state of MCPServer.
type MCPServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// ObservedGeneration is the generation observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// TargetID is the gateway target ID from AWS
	// +optional
	TargetID string `json:"targetId,omitempty"`

	// GatewayArn is the gateway ARN
	// +optional
	GatewayArn string `json:"gatewayArn,omitempty"`

	// TargetStatus is the current target status (CREATING, READY, FAILED, etc.)
	// +optional
	TargetStatus string `json:"targetStatus,omitempty"`

	// StatusReasons are the status reasons from AWS
	// +optional
	StatusReasons []string `json:"statusReasons,omitempty"`

	// LastSynchronized is the last synchronization timestamp
	// +optional
	LastSynchronized *metav1.Time `json:"lastSynchronized,omitempty"`

	// conditions represent the current state of the MCPServer resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=mcps
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.spec.endpoint`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.targetStatus`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// MCPServer is the Schema for the mcpservers API
type MCPServer struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of MCPServer
	// +required
	Spec MCPServerSpec `json:"spec"`

	// status defines the observed state of MCPServer
	// +optional
	Status MCPServerStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// MCPServerList contains a list of MCPServer
type MCPServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []MCPServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MCPServer{}, &MCPServerList{})
}
