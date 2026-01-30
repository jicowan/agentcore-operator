# Design Document: MCP Gateway Operator

## Overview

The MCP Gateway Operator is a Kubernetes operator built with Kubebuilder that watches MCPServer custom resources and automatically registers them as gateway targets in AWS Bedrock AgentCore. The operator bridges Kubernetes-native MCP server configuration with AWS Bedrock's agent infrastructure, enabling seamless integration between MCP servers and Bedrock agents.

### Key Design Principles

1. **Declarative Configuration**: Users declare desired MCP server state via MCPServer resources
2. **Reconciliation Loop**: Operator continuously reconciles actual state (AWS gateway targets) with desired state (MCPServer specs)
3. **Idempotency**: Repeated reconciliations produce the same result
4. **Error Handling**: Transient failures are retried, permanent failures are reported in status
5. **Clean Lifecycle Management**: Finalizers ensure proper cleanup of AWS resources

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
│                                                              │
│  ┌────────────────┐         ┌──────────────────────┐       │
│  │  MCPServer CRD │         │  MCP Gateway         │       │
│  │  (from KRO RGD)│         │  Operator            │       │
│  └────────────────┘         │                      │       │
│                              │  ┌────────────────┐ │       │
│  ┌────────────────┐         │  │  Reconciler    │ │       │
│  │  MCPServer     │────────>│  │  Controller    │ │       │
│  │  Instance      │  Watch  │  └────────────────┘ │       │
│  └────────────────┘         │         │           │       │
│                              │         │           │       │
│                              │         v           │       │
│                              │  ┌────────────────┐ │       │
│                              │  │  AWS Bedrock   │ │       │
│                              │  │  Client        │ │       │
│                              │  └────────────────┘ │       │
│                              └──────────┬───────────┘       │
└─────────────────────────────────────────┼───────────────────┘
                                          │
                                          │ AWS SDK
                                          v
                              ┌───────────────────────┐
                              │  AWS Bedrock          │
                              │  AgentCore            │
                              │                       │
                              │  ┌─────────────────┐ │
                              │  │ Gateway Target  │ │
                              │  └─────────────────┘ │
                              └───────────────────────┘
```

### Component Interaction Flow

1. **User creates MCPServer resource** → Kubernetes API Server stores it
2. **Operator watches MCPServer** → Receives reconciliation event
3. **Reconciler validates spec** → Checks required fields and patterns
4. **Reconciler calls AWS API** → Creates/updates gateway target
5. **Reconciler updates status** → Stores target ID and status in MCPServer
6. **Reconciler polls status** → Waits for gateway target to reach READY state
7. **User deletes MCPServer** → Finalizer triggers cleanup
8. **Reconciler deletes gateway target** → Removes AWS resource
9. **Reconciler removes finalizer** → Allows Kubernetes to complete deletion

## Components and Interfaces

### 1. MCPServer Custom Resource

The MCPServer CRD is defined via a KRO ResourceGraphDefinition and represents the desired state of an MCP server registration.

#### Spec Schema

```yaml
apiVersion: mcpgateway.bedrock.aws/v1alpha1
kind: MCPServer
metadata:
  name: my-mcp-server
  namespace: default
spec:
  # Required: HTTPS endpoint of the MCP server
  endpoint: string | required | pattern=^https://.*
  
  # Required: MCP protocol version
  protocolVersion: string | required | pattern=^(2025-06-18|2025-03-26)$ | default=2025-06-18
  
  # Required: Server capabilities (must include "tools")
  capabilities: []string | required | minItems=1
  
  # Optional: Gateway identifier (defaults to env var)
  gatewayId: string
  
  # Optional: Custom target name (defaults to resource name)
  targetName: string
  
  # Optional: Target description
  description: string
  
  # Optional: Authentication type
  authType: string | pattern=^(NoAuth|OAuth2)$ | default=NoAuth
  
  # Required when authType=OAuth2: OAuth provider ARN
  oauthProviderArn: string
  
  # Optional: OAuth scopes
  oauthScopes: []string
  
  # Optional: Allowed request headers
  allowedRequestHeaders: []string
  
  # Optional: Allowed query parameters
  allowedQueryParameters: []string
  
  # Optional: Allowed response headers
  allowedResponseHeaders: []string
```

#### Status Schema

```yaml
status:
  # Gateway target ID from AWS
  targetId: string
  
  # Gateway ARN
  gatewayArn: string
  
  # Current target status (CREATING, READY, FAILED, etc.)
  targetStatus: string
  
  # Status reasons from AWS
  statusReasons: []string
  
  # Last synchronization timestamp
  lastSynchronized: string
  
  # Conditions
  conditions:
    - type: string        # Ready, ValidationError, etc.
      status: string      # True, False, Unknown
      reason: string
      message: string
      lastTransitionTime: string
```

### 2. MCPServerReconciler

The main controller that reconciles MCPServer resources.

#### Structure

```go
type MCPServerReconciler struct {
    client.Client
    Scheme         *runtime.Scheme
    BedrockClient  *bedrockagentcorecontrol.Client
    DefaultGatewayID string
    Log            logr.Logger
}
```

#### Reconciliation Logic

```go
func (r *MCPServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch MCPServer resource
    // 2. Handle deletion (finalizer logic)
    // 3. Validate spec fields
    // 4. Add finalizer if not present
    // 5. Check if gateway target exists (via status.targetId)
    // 6. Create or update gateway target
    // 7. Poll gateway target status
    // 8. Update MCPServer status
    // 9. Requeue if not ready
}
```

#### Key Methods

- `validateSpec(mcpServer *MCPServer) error`: Validates all spec fields
- `createGatewayTarget(ctx, mcpServer) (string, error)`: Creates AWS gateway target
- `updateGatewayTarget(ctx, mcpServer) error`: Updates existing gateway target
- `syncGatewayTargetStatus(ctx, mcpServer) (ctrl.Result, error)`: Polls AWS for status
- `deleteGatewayTarget(ctx, mcpServer) error`: Deletes AWS gateway target
- `buildTargetConfiguration(mcpServer) (*types.TargetConfiguration, error)`: Builds AWS config

### 3. AWS Bedrock Client Wrapper

Wraps the AWS SDK client with retry logic and error handling.

#### Structure

```go
type BedrockClientWrapper struct {
    client *bedrockagentcorecontrol.Client
    logger logr.Logger
}
```

#### Methods

- `CreateGatewayTarget(ctx, input) (*CreateGatewayTargetOutput, error)`
- `GetGatewayTarget(ctx, gatewayId, targetId) (*GetGatewayTargetOutput, error)`
- `UpdateGatewayTarget(ctx, input) (*UpdateGatewayTargetOutput, error)`
- `DeleteGatewayTarget(ctx, gatewayId, targetId) error`

#### Retry Strategy

- Exponential backoff for throttling errors
- Maximum 3 retries
- Immediate return for validation errors (no retry)

### 4. Configuration Parser

Parses and validates MCPServer spec fields.

#### Structure

```go
type ConfigParser struct {
    defaultGatewayID string
}
```

#### Methods

- `ParseEndpoint(endpoint string) (string, error)`: Validates HTTPS pattern
- `ParseProtocolVersion(version string) (string, error)`: Validates supported versions
- `ParseCapabilities(caps []string) error`: Validates "tools" is present
- `ParseAuthConfig(mcpServer) (*AuthConfig, error)`: Parses auth configuration
- `ParseMetadataConfig(mcpServer) (*MetadataConfig, error)`: Parses metadata propagation
- `GetGatewayID(mcpServer) (string, error)`: Returns gateway ID (spec or default)

### 5. Target Configuration Builder

Builds AWS Bedrock gateway target configuration from MCPServer spec.

#### Structure

```go
type TargetConfigBuilder struct{}
```

#### Methods

```go
func (b *TargetConfigBuilder) Build(mcpServer *MCPServer) (*types.TargetConfiguration, error) {
    return &types.TargetConfiguration{
        Mcp: &types.McpTargetConfiguration{
            McpServer: &types.McpServerConfiguration{
                Endpoint: aws.String(mcpServer.Spec.Endpoint),
            },
        },
    }, nil
}

func (b *TargetConfigBuilder) BuildCredentialConfig(mcpServer *MCPServer) ([]types.CredentialProviderConfiguration, error) {
    switch mcpServer.Spec.AuthType {
    case "NoAuth":
        return []types.CredentialProviderConfiguration{
            {
                CredentialProviderType: types.CredentialProviderTypeGatewayIamRole,
            },
        }, nil
    case "OAuth2":
        return []types.CredentialProviderConfiguration{
            {
                CredentialProviderType: types.CredentialProviderTypeOauth,
                CredentialProvider: &types.CredentialProvider{
                    OauthCredentialProvider: &types.OauthCredentialProvider{
                        ProviderArn: aws.String(mcpServer.Spec.OauthProviderArn),
                        Scopes:      mcpServer.Spec.OauthScopes,
                        GrantType:   types.GrantTypeClientCredentials,
                    },
                },
            },
        }, nil
    default:
        return nil, fmt.Errorf("unsupported auth type: %s", mcpServer.Spec.AuthType)
    }
}

func (b *TargetConfigBuilder) BuildMetadataConfig(mcpServer *MCPServer) *types.MetadataConfiguration {
    if len(mcpServer.Spec.AllowedRequestHeaders) == 0 &&
       len(mcpServer.Spec.AllowedQueryParameters) == 0 &&
       len(mcpServer.Spec.AllowedResponseHeaders) == 0 {
        return nil
    }
    
    return &types.MetadataConfiguration{
        AllowedRequestHeaders:  mcpServer.Spec.AllowedRequestHeaders,
        AllowedQueryParameters: mcpServer.Spec.AllowedQueryParameters,
        AllowedResponseHeaders: mcpServer.Spec.AllowedResponseHeaders,
    }
}
```

### 6. Status Manager

Manages MCPServer status updates.

#### Structure

```go
type StatusManager struct {
    client client.Client
}
```

#### Methods

```go
func (m *StatusManager) UpdateTargetCreated(ctx context.Context, mcpServer *MCPServer, targetId, gatewayArn string) error
func (m *StatusManager) UpdateTargetStatus(ctx context.Context, mcpServer *MCPServer, status string, reasons []string) error
func (m *StatusManager) UpdateCondition(ctx context.Context, mcpServer *MCPServer, condition metav1.Condition) error
func (m *StatusManager) SetReady(ctx context.Context, mcpServer *MCPServer) error
func (m *StatusManager) SetError(ctx context.Context, mcpServer *MCPServer, reason, message string) error
```

## Data Models

### MCPServer Spec

```go
type MCPServerSpec struct {
    // Required fields
    Endpoint         string   `json:"endpoint"`
    ProtocolVersion  string   `json:"protocolVersion"`
    Capabilities     []string `json:"capabilities"`
    
    // Optional gateway configuration
    GatewayID        string   `json:"gatewayId,omitempty"`
    TargetName       string   `json:"targetName,omitempty"`
    Description      string   `json:"description,omitempty"`
    
    // Authentication configuration
    AuthType         string   `json:"authType,omitempty"`
    OauthProviderArn string   `json:"oauthProviderArn,omitempty"`
    OauthScopes      []string `json:"oauthScopes,omitempty"`
    
    // Metadata propagation
    AllowedRequestHeaders  []string `json:"allowedRequestHeaders,omitempty"`
    AllowedQueryParameters []string `json:"allowedQueryParameters,omitempty"`
    AllowedResponseHeaders []string `json:"allowedResponseHeaders,omitempty"`
}
```

### MCPServer Status

```go
type MCPServerStatus struct {
    // Gateway target information
    TargetID         string    `json:"targetId,omitempty"`
    GatewayArn       string    `json:"gatewayArn,omitempty"`
    TargetStatus     string    `json:"targetStatus,omitempty"`
    StatusReasons    []string  `json:"statusReasons,omitempty"`
    LastSynchronized *metav1.Time `json:"lastSynchronized,omitempty"`
    
    // Conditions
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

### Internal Configuration Models

```go
type AuthConfig struct {
    Type             string
    OauthProviderArn string
    OauthScopes      []string
}

type MetadataConfig struct {
    AllowedRequestHeaders  []string
    AllowedQueryParameters []string
    AllowedResponseHeaders []string
}

type GatewayTargetConfig struct {
    GatewayID            string
    TargetName           string
    Description          string
    Endpoint             string
    ProtocolVersion      string
    Capabilities         []string
    Auth                 *AuthConfig
    Metadata             *MetadataConfig
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*



### Property Reflection

After analyzing all acceptance criteria, I've identified several areas where properties can be consolidated:

**Validation Properties**: Multiple criteria (2.1, 2.2, 2.4, 3.8, 16.2, 16.3) test validation logic. These can be combined into comprehensive validation properties.

**Configuration Extraction**: Criteria 3.1, 3.4, 3.5, 3.7, 18.1, 19.1 all test field extraction. These can be combined into properties about configuration parsing.

**Status Update Properties**: Criteria 4.3, 7.1, 7.2, 7.3, 7.5 all test status updates. These can be combined into properties about status synchronization.

**Idempotency Properties**: Criteria 17.1, 17.2, 17.3 all test idempotent behavior and can be combined.

**Auth Configuration**: Criteria 5.1, 5.2, 5.3, 5.4, 5.5 test authentication configuration and can be combined into comprehensive auth properties.

### Correctness Properties

Property 1: Endpoint Validation
*For any* MCPServer resource, if the endpoint field does not match the pattern "https://.*", then the operator should set a validation error condition in the status and not create a gateway target
**Validates: Requirements 3.2, 3.3**

Property 2: Protocol Version Validation
*For any* MCPServer resource, if the protocolVersion field is not "2025-06-18" or "2025-03-26", then the operator should set a validation error condition in the status
**Validates: Requirements 19.4**

Property 3: Tool Capability Validation
*For any* MCPServer resource, if the capabilities field does not include "tools", then the operator should set a validation error condition in the status and not create a gateway target
**Validates: Requirements 20.2, 20.3**

Property 4: Required Field Validation
*For any* MCPServer resource, if any required field (endpoint, protocolVersion, capabilities) is missing, then the operator should set a validation error condition in the status and not create a gateway target
**Validates: Requirements 2.1, 2.2, 2.4, 3.8**

Property 5: OAuth Configuration Validation
*For any* MCPServer resource, if authType is "OAuth2" and oauthProviderArn is missing, then the operator should set a validation error condition in the status
**Validates: Requirements 5.4**

Property 6: Gateway Target Creation
*For any* valid MCPServer resource without a targetId in status, reconciliation should result in a CreateGatewayTarget API call with the correct endpoint, protocol version, and capabilities
**Validates: Requirements 4.1, 4.2**

Property 7: Status Update After Creation
*For any* MCPServer resource, if CreateGatewayTarget succeeds, then the status should be updated with targetId, gatewayArn, and targetStatus fields
**Validates: Requirements 4.3, 7.1, 7.2, 7.3**

Property 8: Idempotent Reconciliation
*For any* MCPServer resource with a targetId in status, repeated reconciliations should not create duplicate gateway targets
**Validates: Requirements 17.1, 17.3**

Property 9: Configuration Extraction
*For any* MCPServer resource, the extracted configuration should match the spec fields: endpoint, gatewayId (or default), targetName (or resource name), description, authType, and metadata propagation settings
**Validates: Requirements 3.1, 3.4, 3.5, 3.6, 3.7, 18.1, 18.2, 19.1**

Property 10: NoAuth Configuration
*For any* MCPServer resource with authType "NoAuth" or absent, the gateway target configuration should use GatewayIamRole credential type
**Validates: Requirements 5.1, 5.3**

Property 11: OAuth2 Configuration
*For any* MCPServer resource with authType "OAuth2" and valid oauthProviderArn, the gateway target configuration should use OAuth credential type with the specified provider ARN and scopes
**Validates: Requirements 5.2, 5.5**

Property 12: Metadata Propagation Configuration
*For any* MCPServer resource with allowedRequestHeaders, allowedQueryParameters, or allowedResponseHeaders specified, the gateway target configuration should include the corresponding metadata configuration
**Validates: Requirements 6.1, 6.2, 6.3**

Property 13: No Metadata When Absent
*For any* MCPServer resource without any metadata propagation fields, the gateway target configuration should not include metadata configuration
**Validates: Requirements 6.4**

Property 14: Finalizer Addition
*For any* MCPServer resource without the finalizer "bedrock.aws/gateway-target-finalizer", reconciliation should add the finalizer
**Validates: Requirements 10.1, 10.5**

Property 15: Gateway Target Deletion
*For any* MCPServer resource being deleted with a targetId in status, the operator should call DeleteGatewayTarget before removing the finalizer
**Validates: Requirements 11.1, 11.5**

Property 16: Idempotent Deletion
*For any* MCPServer resource being deleted, if DeleteGatewayTarget returns ResourceNotFoundException, the operator should remove the finalizer (treating it as successful deletion)
**Validates: Requirements 11.3**

Property 17: Status Synchronization
*For any* MCPServer resource with a targetId in status, calling GetGatewayTarget should update the targetStatus field with the current AWS status
**Validates: Requirements 8.1, 8.3**

Property 18: Ready Condition
*For any* MCPServer resource, when the targetStatus becomes "READY", the operator should set a Ready condition with status "True"
**Validates: Requirements 7.6**

Property 19: Configuration Change Detection
*For any* MCPServer resource with a targetId in status, if the spec changes (endpoint, authType, metadata), reconciliation should call UpdateGatewayTarget
**Validates: Requirements 9.1, 9.2, 9.3, 9.4**

Property 20: Gateway ID Validation
*For any* MCPServer resource, if no gatewayId is specified in spec and no default gateway ID is configured, the operator should set a validation error condition
**Validates: Requirements 18.3, 18.4**

Property 21: URL Passthrough
*For any* MCPServer resource, the endpoint URL should be passed to CreateGatewayTarget exactly as provided in the spec without modification
**Validates: Requirements 21.1, 21.5**

Property 22: Protocol Version Default
*For any* MCPServer resource without a protocolVersion field, the operator should default to "2025-06-18"
**Validates: Requirements 19.5**

Property 23: Client Token Idempotency
*For any* CreateGatewayTarget API call, the operator should include a unique client token for idempotency
**Validates: Requirements 4.5**

Property 24: No AWS Calls for Invalid Resources
*For any* MCPServer resource with validation errors, the operator should not make any AWS API calls (CreateGatewayTarget, UpdateGatewayTarget)
**Validates: Requirements 2.4, 16.3**

Property 25: Target Name Defaulting
*For any* MCPServer resource without a targetName field, the operator should use the resource name as the gateway target name
**Validates: Requirements 3.6**

## Error Handling

### Error Categories

1. **Validation Errors** (Permanent)
   - Invalid endpoint pattern
   - Unsupported protocol version
   - Missing tool capability
   - Missing required fields
   - Invalid OAuth configuration
   - Action: Set error condition, do not retry

2. **AWS API Errors** (Transient)
   - Throttling errors
   - Internal server errors
   - Network errors
   - Action: Requeue with exponential backoff

3. **AWS API Errors** (Permanent)
   - Validation errors from AWS
   - Resource not found (for updates)
   - Action: Set error condition, do not retry

4. **Kubernetes API Errors** (Transient)
   - Status update failures
   - Finalizer update failures
   - Action: Requeue

### Error Handling Strategy

```go
func (r *MCPServerReconciler) handleError(ctx context.Context, mcpServer *MCPServer, err error) (ctrl.Result, error) {
    if isValidationError(err) {
        // Permanent error - set condition and don't retry
        r.StatusManager.SetError(ctx, mcpServer, "ValidationError", err.Error())
        return ctrl.Result{}, nil
    }
    
    if isThrottlingError(err) {
        // Transient error - retry with backoff
        return ctrl.Result{RequeueAfter: calculateBackoff()}, nil
    }
    
    if isAWSValidationError(err) {
        // Permanent AWS error - set condition and don't retry
        r.StatusManager.SetError(ctx, mcpServer, "AWSValidationError", err.Error())
        return ctrl.Result{}, nil
    }
    
    // Default: transient error, requeue
    return ctrl.Result{}, err
}
```

### Status Conditions

The operator uses Kubernetes conditions to communicate state:

```go
// Ready condition - gateway target is ready
{
    Type:   "Ready",
    Status: "True",
    Reason: "GatewayTargetReady",
    Message: "Gateway target is ready and accepting requests"
}

// ValidationError condition - spec validation failed
{
    Type:   "Ready",
    Status: "False",
    Reason: "ValidationError",
    Message: "Endpoint must match pattern https://.*"
}

// AWSError condition - AWS API error
{
    Type:   "Ready",
    Status: "False",
    Reason: "AWSError",
    Message: "Failed to create gateway target: ThrottlingException"
}

// Creating condition - gateway target is being created
{
    Type:   "Ready",
    Status: "Unknown",
    Reason: "Creating",
    Message: "Gateway target is being created"
}
```

## Testing Strategy

### Dual Testing Approach

The testing strategy combines unit tests for specific examples and edge cases with property-based tests for universal correctness properties.

**Unit Tests**:
- Specific validation examples (valid/invalid endpoints, protocol versions)
- Edge cases (empty strings, nil values, boundary conditions)
- Error conditions (AWS API failures, network errors)
- Integration points (Kubernetes API, AWS SDK)

**Property-Based Tests**:
- Universal properties across all inputs (validation, configuration extraction, idempotency)
- Comprehensive input coverage through randomization
- Each property test runs minimum 100 iterations

### Property-Based Testing Configuration

The operator uses a property-based testing library (e.g., gopter for Go) to verify correctness properties. Each property test:

1. Generates random MCPServer resources with various configurations
2. Executes the reconciliation logic
3. Verifies the property holds for all generated inputs
4. Tags the test with the property number and description

Example property test structure:

```go
// Feature: mcp-gateway-operator, Property 1: Endpoint Validation
func TestProperty_EndpointValidation(t *testing.T) {
    properties := gopter.NewProperties(nil)
    
    properties.Property("Invalid endpoints should set validation error", prop.ForAll(
        func(endpoint string) bool {
            if !strings.HasPrefix(endpoint, "https://") {
                mcpServer := &MCPServer{
                    Spec: MCPServerSpec{
                        Endpoint: endpoint,
                        // ... other required fields
                    },
                }
                
                result := reconcile(mcpServer)
                
                // Verify validation error condition is set
                return hasValidationError(mcpServer.Status.Conditions)
            }
            return true
        },
        gen.AnyString(),
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

### Test Coverage

**Validation Tests**:
- Property 1: Endpoint pattern validation
- Property 2: Protocol version validation
- Property 3: Tool capability validation
- Property 4: Required field validation
- Property 5: OAuth configuration validation
- Property 20: Gateway ID validation

**Configuration Tests**:
- Property 9: Configuration extraction
- Property 10: NoAuth configuration
- Property 11: OAuth2 configuration
- Property 12: Metadata propagation configuration
- Property 13: No metadata when absent
- Property 22: Protocol version default
- Property 25: Target name defaulting

**Lifecycle Tests**:
- Property 6: Gateway target creation
- Property 7: Status update after creation
- Property 14: Finalizer addition
- Property 15: Gateway target deletion
- Property 16: Idempotent deletion

**Idempotency Tests**:
- Property 8: Idempotent reconciliation
- Property 21: URL passthrough
- Property 23: Client token idempotency
- Property 24: No AWS calls for invalid resources

**Status Tests**:
- Property 17: Status synchronization
- Property 18: Ready condition
- Property 19: Configuration change detection

### Integration Tests

Integration tests verify the operator works correctly with:
- Real Kubernetes API server (using envtest)
- Mocked AWS Bedrock client
- Complete reconciliation loops
- Finalizer cleanup

### End-to-End Tests

E2E tests verify the operator in a real environment:
- Deploy operator to test cluster
- Create MCPServer resources
- Verify gateway targets are created in AWS
- Update MCPServer resources
- Verify gateway targets are updated
- Delete MCPServer resources
- Verify gateway targets are deleted

## Deployment Architecture

### Helm Chart Structure

```
helm/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── deployment.yaml
│   ├── serviceaccount.yaml
│   ├── role.yaml
│   ├── rolebinding.yaml
│   ├── clusterrole.yaml
│   ├── clusterrolebinding.yaml
│   └── configmap.yaml
└── README.md
```

### Deployment Configuration

```yaml
# values.yaml
replicaCount: 1

image:
  repository: mcp-gateway-operator
  tag: latest
  pullPolicy: IfNotPresent

serviceAccount:
  create: true
  annotations:
    eks.amazonaws.com/role-arn: ""  # IAM role ARN for IRSA

aws:
  region: us-east-1
  defaultGatewayId: ""  # Default gateway ID

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi

nodeSelector: {}
tolerations: []
affinity: {}
```

### Required IAM Permissions

The operator's IAM role requires these permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "bedrock-agentcore-control:CreateGatewayTarget",
        "bedrock-agentcore-control:GetGatewayTarget",
        "bedrock-agentcore-control:UpdateGatewayTarget",
        "bedrock-agentcore-control:DeleteGatewayTarget"
      ],
      "Resource": [
        "arn:aws:bedrock-agentcore:*:*:gateway/*",
        "arn:aws:bedrock-agentcore:*:*:gateway-target/*"
      ]
    }
  ]
}
```

### IRSA Configuration

For EKS clusters using IAM Roles for Service Accounts (IRSA):

1. Create IAM role with trust policy for the ServiceAccount
2. Attach the required permissions policy
3. Annotate the ServiceAccount with the role ARN
4. Operator pods automatically assume the role

```yaml
# ServiceAccount with IRSA annotation
apiVersion: v1
kind: ServiceAccount
metadata:
  name: mcp-gateway-operator
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/mcp-gateway-operator-role
```

## Operational Considerations

### Monitoring

**Metrics to Monitor**:
- Reconciliation rate (reconciliations per second)
- Reconciliation duration (p50, p95, p99)
- Error rate by type (validation, AWS API, Kubernetes API)
- Gateway target creation rate
- Gateway target deletion rate
- Status synchronization lag

**Logging**:
- Structured logging with context (MCPServer name, namespace, target ID)
- Log levels: DEBUG, INFO, WARN, ERROR
- Key events: creation, update, deletion, errors, status changes

### Scaling

**Horizontal Scaling**:
- Multiple operator replicas with leader election
- Only one replica actively reconciles at a time
- Other replicas are hot standbys

**Vertical Scaling**:
- Adjust CPU/memory based on cluster size
- Monitor resource usage and adjust limits

### Troubleshooting

**Common Issues**:

1. **Gateway target not created**
   - Check MCPServer status conditions
   - Verify endpoint matches https:// pattern
   - Verify protocol version is supported
   - Verify capabilities include "tools"
   - Check operator logs for errors

2. **OAuth authentication fails**
   - Verify oauthProviderArn is correct
   - Verify OAuth provider is in same account/region
   - Check OAuth scopes are correct

3. **Status not updating**
   - Check operator has RBAC permissions for status subresource
   - Check operator logs for status update errors
   - Verify Kubernetes API server is accessible

4. **Gateway target not deleted**
   - Check finalizer is present on MCPServer
   - Check operator logs for deletion errors
   - Verify IAM permissions for DeleteGatewayTarget

**Debug Commands**:

```bash
# Check MCPServer status
kubectl get mcpserver my-server -o yaml

# Check operator logs
kubectl logs -n mcp-gateway-operator-system deployment/mcp-gateway-operator

# Check operator events
kubectl get events -n default --field-selector involvedObject.name=my-server

# Check RBAC permissions
kubectl auth can-i get mcpservers --as=system:serviceaccount:mcp-gateway-operator-system:mcp-gateway-operator
```

## Security Considerations

### Least Privilege

- Operator only has permissions for MCPServer resources
- IAM role only has permissions for gateway target operations
- No permissions for other Kubernetes resources
- No permissions for other AWS services

### Secrets Management

- OAuth provider ARNs reference existing providers
- No secrets stored in MCPServer specs
- Credentials managed by AWS Bedrock AgentCore

### Network Security

- Operator communicates with AWS via HTTPS
- Endpoint URLs must use HTTPS
- No plaintext credentials in transit

### Audit Logging

- All AWS API calls are logged in CloudTrail
- All Kubernetes API calls are logged in audit logs
- Operator logs all reconciliation events

## Future Enhancements

### Potential Features

1. **Multiple Gateway Support**: Allow MCPServer to register with multiple gateways
2. **Health Checks**: Periodic health checks of MCP server endpoints
3. **Metrics Export**: Export gateway target metrics to Prometheus
4. **Webhook Validation**: Admission webhook for MCPServer validation
5. **Status Subresource**: Use status subresource for better conflict handling
6. **Custom Conditions**: More granular conditions for different states
7. **Retry Configuration**: Configurable retry policies per MCPServer
8. **Batch Operations**: Batch create/update/delete for multiple MCPServers

### API Evolution

The MCPServer API is versioned (v1alpha1) to allow for future changes:
- v1alpha1 → v1beta1: Add new optional fields, deprecate old fields
- v1beta1 → v1: Stabilize API, remove deprecated fields
- Conversion webhooks for backward compatibility
