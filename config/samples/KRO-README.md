# KRO Integration for MCP Gateway Operator

This directory contains a KRO (Kube Resource Orchestrator) ResourceGraphDefinition that simplifies MCPServer creation by providing a developer-friendly API.

## Overview

The `mcpserver` RGD wraps the `MCPServer` custom resource and provides a simplified interface where developers only need to specify:
- **metadata.name**: The name of the MCP server (via Kubernetes metadata)
- **metadata.namespace**: The namespace to deploy to (via Kubernetes metadata)
- **spec.description**: A description of the MCP server
- **spec.endpoint**: The HTTPS endpoint of the MCP server

All other configuration (OAuth provider, scopes, capabilities) is statically defined by the platform team.

## Prerequisites

1. **KRO installed** in your cluster:
   ```bash
   helm install kro oci://registry.k8s.io/kro/charts/kro \
     --namespace kro-system \
     --create-namespace
   ```

2. **MCP Gateway Operator** installed and running

3. **OAuth2 Credential Provider** created in AWS Bedrock AgentCore

## Installation

### 1. Apply the ResourceGraphDefinition

```bash
kubectl apply -f config/samples/kro_mcpserver_rgd.yaml
```

This creates:
- A new CRD: `MCPServerApp` in the `apps.example.com` API group
- The RGD that manages the lifecycle of MCPServer resources

### 2. Verify the RGD

```bash
# Check RGD status
kubectl get rgd mcpserver -o yaml

# Verify the CRD was created
kubectl get crd mcpserverapps.apps.example.com
```

## Usage

### Create an MCPServer Instance

```yaml
apiVersion: apps.example.com/v1alpha1
kind: MCPServerApp
metadata:
  name: my-mcp-server
  namespace: default
spec:
  name: my-mcp-server
  namespace: default
  description: My custom MCP server for testing
  endpoint: https://mcp-test.jicomusic.com
```

Apply it:
```bash
kubectl apply -f config/samples/kro_mcpserver_instance.yaml
```

### Check Status

```bash
# List all MCPServerApp instances
kubectl get mcpserverapps

# Get detailed status
kubectl get mcpserverapp my-mcp-server -o yaml

# Check the underlying MCPServer resource
kubectl get mcpserver my-mcp-server
```

### Monitor Progress

```bash
# Watch the instance status
kubectl get mcpserverapp my-mcp-server -w

# Check conditions
kubectl get mcpserverapp my-mcp-server -o jsonpath='{.status}' | jq
```

## API Reference

### MCPServerApp Spec

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `description` | string | Yes | - | Description of the MCP server (10-200 chars) |
| `endpoint` | string | Yes | - | HTTPS endpoint of the MCP server (must start with `https://`) |

**Note**: Name and namespace are specified in `metadata`, not `spec`.

### MCPServerApp Status

| Field | Type | Description |
|-------|------|-------------|
| `targetId` | string | AWS Bedrock gateway target ID |
| `targetStatus` | string | Gateway target status (PENDING, CREATING, READY, FAILED) |
| `gatewayArn` | string | ARN of the AWS Bedrock gateway |
| `ready` | boolean | Whether the gateway target is ready |
| `message` | string | Status message from the gateway target |

## Static Configuration

The following values are statically configured in the RGD and cannot be changed by developers:

```yaml
authType: OAuth2
oauthProviderArn: arn:aws:bedrock-agentcore:us-west-2:820537372947:token-vault/default/oauth2credentialprovider/test-mcp-oauth-https
oauthScopes:
  - read
  - write
capabilities:
  - tools
```

### Customizing Static Values

Platform teams can customize these values by editing the RGD:

```bash
kubectl edit rgd mcpserver
```

Update the `resources[0].template.spec` section with your desired values:
- Change `oauthProviderArn` to your OAuth provider
- Modify `oauthScopes` as needed
- Add or remove `capabilities`

After editing, all new instances will use the updated configuration. Existing instances will be updated on the next reconciliation.

## Examples

### Example 1: Basic MCP Server

```yaml
apiVersion: apps.example.com/v1alpha1
kind: MCPServerApp
metadata:
  name: weather-service
  namespace: production
spec:
  description: Weather data MCP server for AI agents
  endpoint: https://weather.example.com
```

### Example 2: Development MCP Server

```yaml
apiVersion: apps.example.com/v1alpha1
kind: MCPServerApp
metadata:
  name: dev-mcp-server
  namespace: development
spec:
  description: Development MCP server for testing new tools
  endpoint: https://dev-mcp.example.com
```

### Example 3: Multiple Environments

```yaml
---
apiVersion: apps.example.com/v1alpha1
kind: MCPServerApp
metadata:
  name: api-server-dev
  namespace: dev
spec:
  description: API MCP server for development environment
  endpoint: https://api-dev.example.com
---
apiVersion: apps.example.com/v1alpha1
kind: MCPServerApp
metadata:
  name: api-server-staging
  namespace: staging
spec:
  description: API MCP server for staging environment
  endpoint: https://api-staging.example.com
---
apiVersion: apps.example.com/v1alpha1
kind: MCPServerApp
metadata:
  name: api-server-prod
  namespace: production
spec:
  description: API MCP server for production environment
  endpoint: https://api.example.com
```

## Validation

The RGD includes validation rules to ensure correct configuration:

### Metadata Validation
Name and namespace follow standard Kubernetes naming conventions (enforced by Kubernetes itself):
- **Pattern**: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
- **Length**: 1-253 characters for namespace, 1-63 for name
- **Example valid**: `my-server`, `api-v1`, `mcp-123`
- **Example invalid**: `My-Server` (uppercase), `-server` (starts with hyphen)

### Description Validation
- **Length**: 10-200 characters
- **Required**: Yes
- **Example valid**: `Weather data MCP server for AI agents`
- **Example invalid**: `Test` (too short)

### Endpoint Validation
- **Pattern**: `^https://.*`
- **Required**: Yes
- **Must start with**: `https://`
- **Example valid**: `https://mcp.example.com`, `https://api.example.com:8443`
- **Example invalid**: `http://mcp.example.com` (not HTTPS), `mcp.example.com` (missing protocol)

## Troubleshooting

### RGD Not Creating CRD

Check RGD status:
```bash
kubectl get rgd mcpserver -o yaml
```

Look for errors in the status conditions. Common issues:
- Invalid CEL expressions
- Circular dependencies
- Invalid schema definitions

### Instance Stuck in PENDING

Check the instance status:
```bash
kubectl get mcpserverapp my-mcp-server -o jsonpath='{.status}' | jq
```

Check the underlying MCPServer:
```bash
kubectl get mcpserver my-mcp-server -o yaml
```

Check operator logs:
```bash
kubectl logs -n mcp-gateway-operator-system deployment/mcp-gateway-operator -c manager -f
```

### Gateway Target FAILED

Common causes:
1. **Invalid endpoint**: Ensure the endpoint is accessible over HTTPS
2. **OAuth configuration**: Verify the OAuth provider ARN is correct
3. **MCP protocol**: Ensure the server implements proper MCP JSON-RPC 2.0 protocol
4. **TLS certificate**: Use valid certificates (ACM recommended)

Check the status message:
```bash
kubectl get mcpserverapp my-mcp-server -o jsonpath='{.status.message}'
```

### Validation Errors

If you get validation errors when creating an instance:

```bash
# Check the error message
kubectl apply -f instance.yaml

# Common fixes:
# - Ensure name is lowercase with hyphens only
# - Ensure description is 10-200 characters
# - Ensure endpoint starts with https://
# - Ensure namespace exists
```

## Advanced Usage

### Watching for Changes

```bash
# Watch all MCPServerApp instances
kubectl get mcpserverapps -A -w

# Watch specific instance
kubectl get mcpserverapp my-mcp-server -w
```

### Filtering by Status

```bash
# Get all ready instances
kubectl get mcpserverapps -A -o json | \
  jq '.items[] | select(.status.ready == true) | .metadata.name'

# Get all failed instances
kubectl get mcpserverapps -A -o json | \
  jq '.items[] | select(.status.targetStatus == "FAILED") | .metadata.name'
```

### Bulk Operations

```bash
# Create multiple instances
kubectl apply -f instances/

# Delete all instances in a namespace
kubectl delete mcpserverapps -n development --all

# Update all instances with a label
kubectl label mcpserverapps -n production environment=prod --all
```

## Benefits of Using KRO

1. **Simplified API**: Developers only need to know 4 fields instead of 7
2. **Consistent Configuration**: Platform team controls OAuth and capability settings
3. **Validation**: Built-in validation ensures correct configuration
4. **Status Projection**: Simplified status view with ready condition
5. **Custom Columns**: `kubectl get` shows relevant information at a glance
6. **Declarative**: Full GitOps support with ArgoCD, Flux, etc.

## Integration with GitOps

### ArgoCD

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: mcp-servers
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/example/mcp-servers
    targetRevision: main
    path: manifests
  destination:
    server: https://kubernetes.default.svc
    namespace: default
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

### Flux

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: mcp-servers
  namespace: flux-system
spec:
  interval: 5m
  path: ./manifests
  prune: true
  sourceRef:
    kind: GitRepository
    name: mcp-servers
```

## Cleanup

### Delete an Instance

```bash
kubectl delete mcpserverapp my-mcp-server
```

This will automatically delete the underlying MCPServer resource and clean up the AWS Bedrock gateway target.

### Delete the RGD

```bash
kubectl delete rgd mcpserver
```

**Warning**: This will delete the CRD and all instances. Ensure you have backups if needed.

## References

- [KRO Documentation](https://kro.run/docs)
- [MCP Gateway Operator](../../README.md)
- [MCP Protocol Specification](https://modelcontextprotocol.io/)
- [AWS Bedrock AgentCore](https://docs.aws.amazon.com/bedrock/)
