# Requirements Document

## Introduction

The MCP Gateway Operator is a Kubernetes operator that automates the integration between MCPServer custom resources (created via a KRO ResourceGraphDefinition) and AWS Bedrock AgentCore Gateway. When a user creates an MCPServer resource instance, the operator automatically provisions a corresponding gateway target in AWS Bedrock AgentCore, enabling seamless integration between Bedrock agents and external MCP servers.

The workflow is:
1. Platform engineer creates an RGD that defines the MCPServer Kind
2. KRO processes the RGD and creates the corresponding MCPServer CRD
3. User creates an MCPServer instance (e.g., `kubectl apply -f my-mcp-server.yaml`)
4. The operator watches for MCPServer resources and registers them as gateway targets in AWS Bedrock AgentCore

## Glossary

- **Operator**: The Kubernetes controller built with Kubebuilder that watches and reconciles MCPServer resources
- **MCPServer**: A custom Kubernetes resource representing a Model Context Protocol server configuration
- **RGD**: ResourceGraphDefinition, a custom resource from KRO (kro.run/v1alpha1) that defines the MCPServer CRD
- **MCP_Server_Endpoint**: The external Model Context Protocol server endpoint that provides tools and context to AI agents
- **Gateway_Target**: An AWS Bedrock AgentCore resource that defines an endpoint for gateway connections
- **BedrockClient**: AWS Go SDK v2 client for the bedrockagentcorecontrol service
- **Reconciler**: The controller component that processes MCPServer events and manages gateway target lifecycle
- **Finalizer**: A Kubernetes mechanism to ensure cleanup logic runs before resource deletion

## Requirements

### Requirement 1: Watch MCPServer Resources

**User Story:** As a platform engineer, I want the operator to watch for MCPServer resource events, so that it can automatically respond to MCPServer lifecycle changes.

#### Acceptance Criteria

1. WHEN the Operator starts, THE Operator SHALL register a watch for MCPServer resources
2. WHEN an MCPServer resource is created, THE Operator SHALL receive a reconciliation event
3. WHEN an MCPServer resource is updated, THE Operator SHALL receive a reconciliation event
4. WHEN an MCPServer resource is deleted, THE Operator SHALL receive a reconciliation event
5. THE Operator SHALL use Kubebuilder's controller-runtime to manage the watch lifecycle

### Requirement 2: Validate MCPServer Resources

**User Story:** As a platform engineer, I want the operator to validate MCPServer resources, so that only valid configurations are processed.

#### Acceptance Criteria

1. WHEN an MCPServer resource is reconciled, THE Operator SHALL verify it has the required spec fields
2. WHEN an MCPServer resource is missing required fields, THE Operator SHALL update the status with a validation error condition
3. THE Operator SHALL log validation decisions for debugging purposes
4. THE Operator SHALL skip creating gateway targets for invalid MCPServer resources

### Requirement 3: Extract MCP Server Configuration

**User Story:** As a platform engineer, I want the operator to extract MCP server configuration from MCPServer spec fields, so that gateway targets can be created with the correct settings.

#### Acceptance Criteria

1. WHEN processing an MCPServer resource, THE Operator SHALL extract the MCP endpoint from the spec field "endpoint"
2. WHEN the endpoint field is present, THE Operator SHALL validate it matches the pattern "https://.*"
3. WHEN the endpoint field does not match the required pattern, THE Operator SHALL update the MCPServer status with a validation error condition
4. WHEN the spec field "gatewayId" is present, THE Operator SHALL use it as the gateway identifier
5. WHEN the spec field "targetName" is present, THE Operator SHALL use it as the gateway target name
6. WHEN the spec field "targetName" is absent, THE Operator SHALL use the MCPServer resource name as the target name
7. WHEN the spec field "description" is present, THE Operator SHALL use it as the gateway target description
8. WHEN required configuration is missing, THE Operator SHALL update the MCPServer status with an error condition and not create a gateway target

### Requirement 4: Create Gateway Targets

**User Story:** As a platform engineer, I want the operator to create gateway targets in AWS Bedrock AgentCore, so that MCP servers can be integrated with Bedrock agents.

#### Acceptance Criteria

1. WHEN an MCPServer resource is identified and has valid configuration, THE Operator SHALL call the BedrockClient CreateGatewayTarget API
2. WHEN creating a gateway target, THE Operator SHALL use the MCP server configuration type
3. WHEN the CreateGatewayTarget API succeeds, THE Operator SHALL store the target ID in the MCPServer status field "targetId"
4. WHEN the CreateGatewayTarget API fails, THE Operator SHALL log the error and requeue the reconciliation
5. THE Operator SHALL include a client token for idempotency when creating gateway targets

### Requirement 5: Configure Credential Providers

**User Story:** As a platform engineer, I want to configure credential providers for gateway targets, so that MCP servers can authenticate with different methods.

#### Acceptance Criteria

1. WHEN the spec field "authType" is set to "NoAuth", THE Operator SHALL configure the gateway target without authentication
2. WHEN the spec field "authType" is set to "OAuth2", THE Operator SHALL configure OAuth2 credentials using the provider ARN from spec field "oauthProviderArn"
3. WHEN the spec field "authType" is absent, THE Operator SHALL default to "NoAuth"
4. WHEN OAuth2 authentication is configured and the "oauthProviderArn" field is missing, THE Operator SHALL update the MCPServer status with a validation error condition
5. WHEN OAuth2 credentials are configured and the spec field "oauthScopes" is present, THE Operator SHALL parse and include the scopes
6. THE Operator SHALL validate that the OAuth provider ARN is in the same AWS account and region as the gateway

### Requirement 6: Configure Metadata Propagation

**User Story:** As a platform engineer, I want to configure which HTTP headers and query parameters are propagated, so that I can control metadata flow between agents and MCP servers.

#### Acceptance Criteria

1. WHEN the spec field "allowedRequestHeaders" is present, THE Operator SHALL parse the list and configure allowed request headers
2. WHEN the spec field "allowedQueryParameters" is present, THE Operator SHALL parse the list and configure allowed query parameters
3. WHEN the spec field "allowedResponseHeaders" is present, THE Operator SHALL parse the list and configure allowed response headers
4. WHEN metadata propagation fields are absent, THE Operator SHALL not configure metadata propagation

### Requirement 7: Update MCPServer Status

**User Story:** As a platform engineer, I want the operator to update MCPServer status with gateway target information, so that I can monitor the integration state.

#### Acceptance Criteria

1. WHEN a gateway target is successfully created, THE Operator SHALL update the MCPServer status field "targetId" with the target ID
2. WHEN a gateway target is successfully created, THE Operator SHALL update the MCPServer status field "gatewayArn" with the gateway ARN
3. WHEN a gateway target status changes, THE Operator SHALL update the MCPServer status field "targetStatus" with the current status
4. WHEN updating MCPServer status fails, THE Operator SHALL log the error and requeue the reconciliation
5. THE Operator SHALL update the MCPServer status field "lastSynchronized" with the timestamp of the last successful synchronization
6. THE Operator SHALL add a status condition "Ready" with type "True" when the gateway target reaches READY status

### Requirement 8: Synchronize Gateway Target Status

**User Story:** As a platform engineer, I want the operator to periodically check gateway target status, so that MCPServer status reflects the current state.

#### Acceptance Criteria

1. WHEN an MCPServer resource has a gateway target ID in its status, THE Operator SHALL call the BedrockClient GetGatewayTarget API to retrieve current status
2. WHEN the gateway target status is not "READY", THE Operator SHALL requeue the reconciliation after 10 seconds
3. WHEN the gateway target status is "READY", THE Operator SHALL update the MCPServer status and complete reconciliation
4. WHEN the GetGatewayTarget API fails, THE Operator SHALL log the error and requeue the reconciliation
5. THE Operator SHALL include status reasons in log messages when the target is not ready

### Requirement 9: Update Gateway Targets

**User Story:** As a platform engineer, I want the operator to update gateway targets when MCPServer configuration changes, so that changes are reflected in AWS Bedrock.

#### Acceptance Criteria

1. WHEN an MCPServer resource with an existing gateway target is updated, THE Operator SHALL detect configuration changes
2. WHEN MCP endpoint configuration changes, THE Operator SHALL call the BedrockClient UpdateGatewayTarget API
3. WHEN credential configuration changes, THE Operator SHALL call the BedrockClient UpdateGatewayTarget API
4. WHEN metadata propagation configuration changes, THE Operator SHALL call the BedrockClient UpdateGatewayTarget API
5. WHEN the UpdateGatewayTarget API succeeds, THE Operator SHALL update MCPServer status with the new status

### Requirement 10: Manage Finalizers

**User Story:** As a platform engineer, I want the operator to use finalizers, so that gateway targets are properly cleaned up when MCPServer resources are deleted.

#### Acceptance Criteria

1. WHEN processing an MCPServer resource without a finalizer, THE Operator SHALL add the finalizer "bedrock.aws/gateway-target-finalizer"
2. WHEN an MCPServer resource with the finalizer is being deleted, THE Operator SHALL execute cleanup logic before allowing deletion
3. WHEN cleanup is complete, THE Operator SHALL remove the finalizer from the MCPServer resource
4. WHEN the finalizer is removed, THE Operator SHALL allow Kubernetes to complete the MCPServer resource deletion
5. THE Operator SHALL update the MCPServer resource after adding or removing finalizers

### Requirement 11: Delete Gateway Targets

**User Story:** As a platform engineer, I want the operator to delete gateway targets when MCPServer resources are deleted, so that AWS resources are properly cleaned up.

#### Acceptance Criteria

1. WHEN an MCPServer resource with a gateway target ID is being deleted, THE Operator SHALL call the BedrockClient DeleteGatewayTarget API
2. WHEN the DeleteGatewayTarget API succeeds, THE Operator SHALL log the deletion and remove the finalizer
3. WHEN the DeleteGatewayTarget API fails with ResourceNotFoundException, THE Operator SHALL treat it as success and remove the finalizer
4. WHEN the DeleteGatewayTarget API fails with other errors, THE Operator SHALL log the error and requeue the reconciliation
5. THE Operator SHALL not remove the finalizer until the gateway target is successfully deleted

### Requirement 12: Handle AWS Client Initialization

**User Story:** As a platform engineer, I want the operator to initialize the AWS Bedrock client properly, so that it can communicate with AWS services.

#### Acceptance Criteria

1. WHEN the Operator starts, THE Operator SHALL load AWS configuration using the default credential chain
2. WHEN the environment variable "GATEWAY_ID" is set, THE Operator SHALL use it as the default gateway identifier
3. WHEN the environment variable "AWS_REGION" is set, THE Operator SHALL use it to configure the AWS client
4. WHEN AWS configuration loading fails, THE Operator SHALL log the error and exit
5. THE Operator SHALL create a BedrockClient instance from the loaded configuration

### Requirement 13: Implement RBAC Permissions

**User Story:** As a platform engineer, I want the operator to have appropriate RBAC permissions, so that it can watch and update MCPServer resources.

#### Acceptance Criteria

1. THE Operator SHALL have permission to get, list, and watch MCPServer resources
2. THE Operator SHALL have permission to update MCPServer resources
3. THE Operator SHALL have permission to update the status subresource of MCPServer resources
4. THE Operator SHALL generate RBAC manifests using Kubebuilder markers
5. THE Operator SHALL include permissions for finalizer management on MCPServer resources

### Requirement 14: Implement Error Handling

**User Story:** As a platform engineer, I want the operator to handle errors gracefully, so that transient failures don't cause permanent issues.

#### Acceptance Criteria

1. WHEN the BedrockClient returns a throttling error, THE Operator SHALL requeue the reconciliation with exponential backoff
2. WHEN the BedrockClient returns a validation error, THE Operator SHALL log the error and not requeue
3. WHEN the BedrockClient returns an internal server error, THE Operator SHALL requeue the reconciliation
4. WHEN an RGD is not found during reconciliation, THE Operator SHALL treat it as deleted and return without error
5. THE Operator SHALL include error details in log messages for debugging

### Requirement 15: Implement Logging

**User Story:** As a platform engineer, I want the operator to log important events, so that I can monitor and debug the system.

#### Acceptance Criteria

1. WHEN the Operator starts, THE Operator SHALL log the startup message with configuration details
2. WHEN a gateway target is created, THE Operator SHALL log the target ID and status
3. WHEN a gateway target is updated, THE Operator SHALL log the update operation
4. WHEN a gateway target is deleted, THE Operator SHALL log the deletion operation
5. WHEN errors occur, THE Operator SHALL log error messages with context

### Requirement 16: Support Configuration via Spec Fields

**User Story:** As a platform engineer, I want to configure gateway targets via MCPServer spec fields, so that I have a declarative configuration mechanism.

#### Acceptance Criteria

1. THE Operator SHALL read all configuration from MCPServer spec fields
2. WHEN a spec field is malformed, THE Operator SHALL update the MCPServer status with a validation error condition
3. WHEN required spec fields are missing, THE Operator SHALL update the MCPServer status with an error condition and skip processing
4. THE Operator SHALL document all supported spec fields in the MCPServer CRD schema
5. THE Operator SHALL validate spec field values before using them

### Requirement 17: Implement Idempotent Reconciliation

**User Story:** As a platform engineer, I want reconciliation to be idempotent, so that repeated reconciliations don't cause issues.

#### Acceptance Criteria

1. WHEN reconciling an MCPServer resource that already has a gateway target, THE Operator SHALL not create a duplicate target
2. WHEN reconciling an MCPServer resource with no changes, THE Operator SHALL complete without making AWS API calls
3. WHEN reconciling an MCPServer resource multiple times, THE Operator SHALL produce the same result
4. THE Operator SHALL use the target ID in the MCPServer status to determine if a gateway target exists
5. THE Operator SHALL verify gateway target existence before attempting updates

### Requirement 18: Support Multiple Gateway Identifiers

**User Story:** As a platform engineer, I want to support multiple gateway identifiers, so that different MCPServer resources can target different gateways.

#### Acceptance Criteria

1. WHEN an MCPServer resource has the spec field "gatewayId", THE Operator SHALL use it as the gateway identifier
2. WHEN an MCPServer resource lacks the gateway ID spec field, THE Operator SHALL use the default gateway ID from environment variables
3. WHEN no gateway ID is available, THE Operator SHALL update the MCPServer status with an error condition and skip processing
4. THE Operator SHALL validate that the gateway ID is not empty
5. THE Operator SHALL support both gateway IDs and gateway ARNs as identifiers


### Requirement 19: Validate MCP Protocol Version

**User Story:** As a platform engineer, I want the operator to validate MCP protocol versions, so that only supported versions are registered with the gateway.

#### Acceptance Criteria

1. WHEN processing an MCPServer resource, THE Operator SHALL extract the protocol version from the spec field "protocolVersion"
2. WHEN the protocol version is "2025-06-18", THE Operator SHALL accept it as valid
3. WHEN the protocol version is "2025-03-26", THE Operator SHALL accept it as valid
4. WHEN the protocol version is not one of the supported versions, THE Operator SHALL update the MCPServer status with a validation error condition
5. WHEN the protocol version field is absent, THE Operator SHALL default to "2025-06-18"

### Requirement 20: Validate Tool Capabilities

**User Story:** As a platform engineer, I want the operator to ensure MCP servers have tool capabilities, so that they can provide tools to Bedrock agents.

#### Acceptance Criteria

1. WHEN processing an MCPServer resource, THE Operator SHALL verify the server has tool capabilities
2. WHEN the spec field "capabilities" includes "tools", THE Operator SHALL accept the configuration
3. WHEN the spec field "capabilities" does not include "tools", THE Operator SHALL update the MCPServer status with a validation error condition
4. THE Operator SHALL log a warning if additional capabilities beyond "tools" are specified

### Requirement 21: Encode Endpoint URLs

**User Story:** As a platform engineer, I want the operator to properly encode endpoint URLs, so that the gateway can invoke the MCP server correctly.

#### Acceptance Criteria

1. WHEN creating a gateway target, THE Operator SHALL use the endpoint URL exactly as provided in the MCPServer spec
2. WHEN the endpoint URL contains special characters, THE Operator SHALL ensure they are properly URL-encoded
3. THE Operator SHALL validate that the endpoint URL is a valid URL before creating the gateway target
4. THE Operator SHALL document that users must provide properly encoded URLs in the MCPServer spec
5. THE Operator SHALL pass the encoded URL to the BedrockClient CreateGatewayTarget API without modification


### Requirement 22: Provide Helm Chart for Deployment

**User Story:** As a platform engineer, I want a Helm chart to deploy the operator, so that I can easily install it in my Kubernetes cluster.

#### Acceptance Criteria

1. THE Operator SHALL provide a Helm chart for deployment to Kubernetes clusters
2. THE Helm chart SHALL create a ServiceAccount for the operator
3. THE Helm chart SHALL support configuring an IAM role ARN annotation on the ServiceAccount for IRSA (IAM Roles for Service Accounts)
4. THE Helm chart SHALL allow configuration of the default gateway ID via values
5. THE Helm chart SHALL allow configuration of AWS region via values
6. THE Helm chart SHALL create the necessary RBAC resources (Role, RoleBinding, ClusterRole, ClusterRoleBinding)
7. THE Helm chart SHALL create a Deployment for the operator with configurable replicas
8. THE Helm chart SHALL support configuring resource requests and limits for the operator pods
9. THE Helm chart SHALL document the required IAM permissions for the operator's IAM role
10. THE Helm chart SHALL include a values.yaml file with sensible defaults and documentation
