# MCP Gateway Operator Architecture

This document describes the architecture and design of the MCP Gateway Operator.

## Overview

The MCP Gateway Operator is a Kubernetes operator built with Kubebuilder that automatically manages AWS Bedrock AgentCore gateway targets for MCP (Model Context Protocol) servers defined as Kubernetes custom resources.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        Kubernetes Cluster                        │
│                                                                   │
│  ┌────────────────┐                                              │
│  │  MCPServer CR  │                                              │
│  │  (User Input)  │                                              │
│  └────────┬───────┘                                              │
│           │                                                       │
│           │ Watch                                                 │
│           ▼                                                       │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │              MCPServer Controller                           │ │
│  │                                                              │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │ │
│  │  │   Config     │  │   Target     │  │   Status     │    │ │
│  │  │   Parser     │  │   Config     │  │   Manager    │    │ │
│  │  │              │  │   Builder    │  │              │    │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘    │ │
│  │                                                              │ │
│  │  ┌──────────────────────────────────────────────────────┐  │ │
│  │  │         Bedrock Client Wrapper                       │  │ │
│  │  │         (with retry logic)                           │  │ │
│  │  └──────────────────────────────────────────────────────┘  │ │
│  └────────────────────────────┬─────────────────────────────┘ │
│                                │                                 │
│                                │ AWS SDK v2                      │
└────────────────────────────────┼─────────────────────────────────┘
                                 │
                                 │ IRSA (IAM Role)
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                         AWS Cloud                                │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │           AWS Bedrock AgentCore                             │ │
│  │                                                              │ │
│  │  ┌──────────────┐         ┌──────────────┐                │ │
│  │  │   Gateway    │────────▶│   Gateway    │                │ │
│  │  │              │         │   Target     │                │ │
│  │  └──────────────┘         └──────────────┘                │ │
│  │                                  │                          │ │
│  └──────────────────────────────────┼──────────────────────────┘ │
│                                     │                             │
│                                     │ HTTPS                       │
│                                     ▼                             │
│                            ┌──────────────┐                       │
│                            │  MCP Server  │                       │
│                            └──────────────┘                       │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### 1. MCPServer Custom Resource

The `MCPServer` CRD defines the desired state of an MCP server gateway target.

**Key Fields:**
- `spec.endpoint`: HTTPS endpoint of the MCP server
- `spec.protocolVersion`: MCP protocol version
- `spec.capabilities`: Server capabilities (must include "tools")
- `spec.authType`: Authentication method (NoAuth or OAuth2)
- `spec.gatewayId`: Bedrock gateway identifier
- `status.targetId`: AWS gateway target ID
- `status.targetStatus`: Current status (CREATING, READY, FAILED, etc.)
- `status.conditions`: Kubernetes-style status conditions

### 2. MCPServer Controller

The controller implements the reconciliation loop that keeps the actual state in sync with the desired state.

**Reconciliation Flow:**

```
1. Fetch MCPServer resource
2. Check for deletion (handle finalizer)
3. Validate spec
4. Add finalizer if not present
5. Check if target exists:
   - If no: Create gateway target
   - If yes: Check for config changes
     - If changed: Update gateway target
     - If unchanged: Sync status
6. Update MCPServer status
7. Requeue if needed
```

**Key Methods:**
- `Reconcile()`: Main reconciliation loop
- `validateSpec()`: Validates MCPServer spec
- `createGatewayTarget()`: Creates new gateway target
- `updateGatewayTarget()`: Updates existing gateway target
- `deleteGatewayTarget()`: Deletes gateway target
- `syncGatewayTargetStatus()`: Syncs status from AWS
- `detectConfigChanges()`: Detects spec changes
- `handleDeletion()`: Handles resource deletion with finalizer

### 3. Config Parser

Validates and parses MCPServer spec fields.

**Responsibilities:**
- Endpoint validation (HTTPS pattern)
- Protocol version validation
- Capabilities validation (must include "tools")
- OAuth configuration validation
- Gateway ID resolution (spec or default)

**Key Methods:**
- `ParseEndpoint()`: Validates HTTPS endpoint
- `ParseProtocolVersion()`: Validates and defaults protocol version
- `ParseCapabilities()`: Validates capabilities array
- `ParseAuthConfig()`: Validates OAuth configuration
- `GetGatewayID()`: Returns gateway ID with fallback

### 4. Target Config Builder

Builds AWS Bedrock gateway target configurations from MCPServer specs.

**Responsibilities:**
- Build MCP server target configuration
- Build credential configuration (NoAuth/OAuth2)
- Build metadata propagation configuration

**Key Methods:**
- `Build()`: Creates TargetConfiguration
- `BuildCredentialConfig()`: Creates credential provider configuration
- `BuildMetadataConfig()`: Creates metadata configuration

### 5. Bedrock Client Wrapper

Wraps AWS SDK calls with retry logic and error handling.

**Responsibilities:**
- Create gateway targets
- Get gateway target status
- Update gateway targets
- Delete gateway targets
- Retry on throttling and server errors
- Idempotent deletion (treats ResourceNotFoundException as success)

**Retry Strategy:**
- Exponential backoff: 1s → 2s → 4s → 8s (max 30s)
- Max 3 retries
- Retryable errors: ThrottlingException, TooManyRequestsException, InternalServerException
- Non-retryable errors: ValidationException, InvalidParameterException

**Key Methods:**
- `CreateGatewayTarget()`: Creates target with client token for idempotency
- `GetGatewayTarget()`: Retrieves target status
- `UpdateGatewayTarget()`: Updates target configuration
- `DeleteGatewayTarget()`: Deletes target (idempotent)

### 6. Status Manager

Manages MCPServer status updates.

**Responsibilities:**
- Update status after target creation
- Update status after target sync
- Update status conditions
- Set Ready condition
- Set Error condition

**Key Methods:**
- `UpdateTargetCreated()`: Sets TargetID, GatewayArn, TargetStatus
- `UpdateTargetStatus()`: Updates TargetStatus and StatusReasons
- `UpdateCondition()`: Adds/updates condition
- `SetReady()`: Sets Ready condition to True
- `SetError()`: Sets Ready condition to False

## Data Flow

### Create Flow

```
1. User creates MCPServer resource
   ↓
2. Controller receives reconcile request
   ↓
3. Controller validates spec (ConfigParser)
   ↓
4. Controller adds finalizer
   ↓
5. Controller builds target configuration (TargetConfigBuilder)
   ↓
6. Controller calls AWS CreateGatewayTarget (BedrockClient)
   ↓
7. Controller updates status with TargetID (StatusManager)
   ↓
8. Controller requeues to check status
   ↓
9. Controller polls AWS GetGatewayTarget
   ↓
10. When status is READY, controller sets Ready condition
```

### Update Flow

```
1. User updates MCPServer spec
   ↓
2. Kubernetes increments Generation
   ↓
3. Controller receives reconcile request
   ↓
4. Controller detects Generation != ObservedGeneration
   ↓
5. Controller builds updated configuration
   ↓
6. Controller calls AWS UpdateGatewayTarget
   ↓
7. Controller updates status with new ObservedGeneration
   ↓
8. Controller requeues to check status
```

### Delete Flow

```
1. User deletes MCPServer resource
   ↓
2. Kubernetes sets DeletionTimestamp
   ↓
3. Controller receives reconcile request
   ↓
4. Controller checks for finalizer
   ↓
5. Controller calls AWS DeleteGatewayTarget
   ↓
6. Controller removes finalizer
   ↓
7. Kubernetes deletes resource
```

## Authentication

### IRSA (IAM Roles for Service Accounts)

The operator uses IRSA for AWS authentication:

1. ServiceAccount has annotation: `eks.amazonaws.com/role-arn`
2. EKS injects AWS credentials as environment variables
3. AWS SDK automatically uses these credentials
4. IAM role has trust relationship with OIDC provider
5. IAM role has permissions to manage gateway targets

**Required IAM Permissions:**
- `bedrock-agentcore-control:CreateGatewayTarget`
- `bedrock-agentcore-control:GetGatewayTarget`
- `bedrock-agentcore-control:UpdateGatewayTarget`
- `bedrock-agentcore-control:DeleteGatewayTarget`
- `bedrock-agentcore-control:ListGatewayTargets`

## Error Handling

### Validation Errors

- Detected during spec validation
- Status condition set to False with ValidationError reason
- No AWS API calls made
- No requeue (user must fix spec)

### AWS API Errors

- Throttling errors: Retry with exponential backoff
- Server errors: Retry with exponential backoff
- Validation errors: Set error condition, no retry
- ResourceNotFoundException on delete: Treated as success

### Reconciliation Errors

- Transient errors: Requeue with backoff
- Permanent errors: Set error condition, no requeue
- Status update errors: Requeue immediately

## Idempotency

The operator ensures idempotent operations:

1. **Create**: Uses client token to prevent duplicate creates
2. **Update**: Only updates when Generation changes
3. **Delete**: Treats ResourceNotFoundException as success
4. **Status Sync**: Skips AWS calls when status is READY and no changes

## Status Conditions

The operator uses Kubernetes-style status conditions:

**Ready Condition:**
- `True`: Gateway target is ready
- `False`: Error occurred (validation, AWS API, etc.)
- `Unknown`: Status not yet determined

**Condition Reasons:**
- `GatewayTargetReady`: Target is ready
- `ValidationError`: Spec validation failed
- `ConfigurationError`: Configuration building failed
- `CreationError`: AWS CreateGatewayTarget failed
- `UpdateError`: AWS UpdateGatewayTarget failed

## Observability

### Metrics

The operator exposes Prometheus metrics:
- Controller reconciliation metrics
- API call latency
- Error rates

### Logging

Structured logging with levels:
- Info: Normal operations
- Error: Errors and failures
- Debug (V=1): Detailed reconciliation steps

### Events

Kubernetes events for:
- Target creation
- Target updates
- Target deletion
- Errors

## Scalability

### Controller Scalability

- Single replica by default
- Leader election for high availability
- Concurrent reconciliation (controller-runtime default)

### Resource Limits

- CPU: 10m request, 500m limit
- Memory: 64Mi request, 128Mi limit

### Rate Limiting

- AWS SDK built-in rate limiting
- Exponential backoff on throttling
- Requeue delays to avoid tight loops

## Security

### Least Privilege

- RBAC limited to MCPServer resources
- IAM role limited to gateway target operations
- No cluster-admin permissions required

### Pod Security

- Non-root user (65532)
- Read-only root filesystem
- No privilege escalation
- Capabilities dropped

### Secrets Management

- OAuth credentials stored in AWS (not Kubernetes)
- No secrets in MCPServer spec
- IRSA credentials injected by EKS

## Future Enhancements

Potential future improvements:

1. **Webhook Validation**: Validate MCPServer on admission
2. **Conversion Webhook**: Support multiple API versions
3. **Status Subresource**: Optimize status updates
4. **Metrics Dashboard**: Grafana dashboard for monitoring
5. **E2E Tests**: Comprehensive end-to-end test suite
6. **Multi-Gateway Support**: Manage multiple gateways
7. **Batch Operations**: Optimize bulk creates/updates
8. **Custom Metrics**: Expose operator-specific metrics

## References

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [AWS Bedrock AgentCore](https://docs.aws.amazon.com/bedrock/)
- [Model Context Protocol](https://modelcontextprotocol.io/)
- [IRSA Documentation](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html)
