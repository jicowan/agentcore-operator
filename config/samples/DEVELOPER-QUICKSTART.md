# Developer Quick Start - MCPServerApp

Quick guide for developers to create MCP servers using the simplified KRO API.

## TL;DR

```bash
# 1. Create your MCP server
cat <<EOF | kubectl apply -f -
apiVersion: apps.example.com/v1alpha1
kind: MCPServerApp
metadata:
  name: my-mcp-server
  namespace: default
spec:
  description: My custom MCP server for AI agents
  endpoint: https://mcp.example.com
EOF

# 2. Check status
kubectl get mcpserverapp my-mcp-server

# 3. Wait for READY
kubectl wait --for=condition=Ready mcpserverapp/my-mcp-server --timeout=5m
```

## What You Need to Provide

Only 2 fields in `spec`:

| Field | Description | Example |
|-------|-------------|---------|
| `description` | What your server does (10-200 chars) | `Weather data MCP server` |
| `endpoint` | HTTPS URL of your server | `https://mcp.example.com` |

Plus standard Kubernetes metadata:

| Field | Description | Example |
|-------|-------------|---------|
| `metadata.name` | Server name (lowercase, hyphens) | `my-mcp-server` |
| `metadata.namespace` | Kubernetes namespace | `default` |

## What's Configured Automatically

The platform team has pre-configured:
- ✅ OAuth2 authentication
- ✅ OAuth provider and credentials
- ✅ OAuth scopes (read, write)
- ✅ Capabilities (tools)

You don't need to worry about these!

## Examples

### Basic Example

```yaml
apiVersion: apps.example.com/v1alpha1
kind: MCPServerApp
metadata:
  name: weather-api
  namespace: default
spec:
  description: Weather data provider for AI agents
  endpoint: https://weather.example.com
```

### Production Example

```yaml
apiVersion: apps.example.com/v1alpha1
kind: MCPServerApp
metadata:
  name: customer-api
  namespace: production
spec:
  description: Customer data API for AI-powered support agents
  endpoint: https://api.example.com
```

### Development Example

```yaml
apiVersion: apps.example.com/v1alpha1
kind: MCPServerApp
metadata:
  name: test-server
  namespace: dev
spec:
  description: Development MCP server for testing new features
  endpoint: https://dev-mcp.example.com
```

## Checking Status

### Quick Status Check

```bash
kubectl get mcpserverapps
```

Output:
```
NAME            ENDPOINT                      STATUS    READY   AGE
my-mcp-server   https://mcp.example.com      READY     true    5m
```

### Detailed Status

```bash
kubectl get mcpserverapp my-mcp-server -o yaml
```

Look for:
```yaml
status:
  targetId: ABC123XYZ
  targetStatus: READY
  ready: true
  message: Gateway target is ready and accepting requests
```

### Status Values

| Status | Meaning |
|--------|---------|
| `PENDING` | Waiting to be created |
| `CREATING` | Being created in AWS |
| `READY` | ✅ Ready to use! |
| `FAILED` | ❌ Something went wrong |

## Common Issues

### ❌ Validation Error: Invalid Name

```
Error: metadata.name must match pattern ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
```

**Fix**: Use lowercase letters, numbers, and hyphens only in `metadata.name`:
- ✅ `my-server`, `api-v1`, `mcp-123`
- ❌ `My-Server`, `server_1`, `-server`

### ❌ Validation Error: Description Too Short

```
Error: description must be at least 10 characters
```

**Fix**: Provide a meaningful description (10-200 characters):
- ✅ `Weather data MCP server for AI agents`
- ❌ `Test` (too short)

### ❌ Validation Error: Invalid Endpoint

```
Error: endpoint must match pattern ^https://.*
```

**Fix**: Use HTTPS (not HTTP):
- ✅ `https://mcp.example.com`
- ❌ `http://mcp.example.com`
- ❌ `mcp.example.com`

### ❌ Status: FAILED

Check the status message:
```bash
kubectl get mcpserverapp my-mcp-server -o jsonpath='{.status.message}'
```

Common causes:
1. **Endpoint not accessible**: Ensure your server is reachable over HTTPS
2. **Invalid TLS certificate**: Use a valid certificate (Let's Encrypt, ACM, etc.)
3. **MCP protocol not implemented**: Your server must implement MCP JSON-RPC 2.0

## Updating Your Server

### Change the Endpoint

```bash
kubectl edit mcpserverapp my-mcp-server
```

Update the `spec.endpoint` field and save. The gateway target will be updated automatically.

### Change the Description

```bash
kubectl patch mcpserverapp my-mcp-server --type=merge -p '{"spec":{"description":"New description here"}}'
```

## Deleting Your Server

```bash
kubectl delete mcpserverapp my-mcp-server
```

This will:
1. Delete the Kubernetes resource
2. Clean up the AWS Bedrock gateway target
3. Remove all associated resources

## Getting Help

### Check Logs

```bash
# Operator logs
kubectl logs -n mcp-gateway-operator-system deployment/mcp-gateway-operator -c manager -f

# Your server logs (if running in Kubernetes)
kubectl logs -n <namespace> deployment/<your-server> -f
```

### Check Events

```bash
kubectl get events --sort-by='.lastTimestamp' | grep my-mcp-server
```

### Ask for Help

Include this information when asking for help:

```bash
# Get full status
kubectl get mcpserverapp my-mcp-server -o yaml > my-server-status.yaml

# Get operator logs
kubectl logs -n mcp-gateway-operator-system deployment/mcp-gateway-operator -c manager --tail=100 > operator-logs.txt

# Get events
kubectl get events --sort-by='.lastTimestamp' | grep my-mcp-server > events.txt
```

## Best Practices

### 1. Use Descriptive Names

✅ Good:
- `customer-support-api`
- `weather-data-service`
- `inventory-mcp-server`

❌ Bad:
- `server1`
- `test`
- `api`

### 2. Use Namespaces for Environments

```yaml
# Development
namespace: dev

# Staging
namespace: staging

# Production
namespace: production
```

### 3. Add Labels for Organization

```yaml
metadata:
  name: my-mcp-server
  labels:
    team: platform
    environment: production
    app: customer-api
```

### 4. Use Meaningful Descriptions

✅ Good:
- `Customer data API for AI-powered support agents`
- `Weather forecast provider with 7-day predictions`
- `Inventory management tools for warehouse automation`

❌ Bad:
- `Test server`
- `API endpoint`
- `MCP server`

### 5. Monitor Your Servers

```bash
# Set up a watch
kubectl get mcpserverapps -A -w

# Check regularly
kubectl get mcpserverapps -A -o wide
```

## Next Steps

1. **Deploy your MCP server** - Ensure it implements the MCP JSON-RPC 2.0 protocol
2. **Create the MCPServerApp** - Use the examples above
3. **Wait for READY status** - Usually takes 1-2 minutes
4. **Test with AWS Bedrock** - Your server is now available to AI agents!

## Resources

- [Full Documentation](./KRO-README.md)
- [MCP Protocol Specification](https://modelcontextprotocol.io/)
- [Mock MCP Server Example](../../test/mock-mcp-server/)
- [Troubleshooting Guide](./KRO-README.md#troubleshooting)
