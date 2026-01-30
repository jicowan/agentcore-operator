# Mock MCP Server with JSON-RPC 2.0 Protocol

A mock MCP (Model Context Protocol) server for testing the MCP Gateway Operator with proper JSON-RPC 2.0 protocol implementation and OAuth2 authentication.

## Features

- **Implements MCP JSON-RPC 2.0 protocol** with proper lifecycle management
- **Initialization handshake** with capability negotiation
- **Tools listing** via `tools/list` method
- **Tool invocation** via `tools/call` method
- Provides 3 sample tools: `get_weather`, `calculate`, `get_time`
- **OAuth2 authentication** with `client_credentials` grant type
- HTTPS support via AWS Certificate Manager (ACM) + Application Load Balancer (ALB)
- Health check endpoint at `/health`

## MCP Protocol Implementation

This server implements the [Model Context Protocol](https://modelcontextprotocol.io/) specification, which uses JSON-RPC 2.0 for all communication.

### Supported Methods

1. **initialize** - MCP lifecycle initialization
   - Request: `{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2025-11-25", "capabilities": {}, "clientInfo": {...}}}`
   - Response: Returns server capabilities and info

2. **tools/list** - List available tools
   - Request: `{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {}}`
   - Response: Returns array of tool definitions

3. **tools/call** - Invoke a tool
   - Request: `{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "get_weather", "arguments": {"location": "Seattle"}}}`
   - Response: Returns tool execution result

4. **notifications/*** - Handle lifecycle notifications
   - Example: `{"jsonrpc": "2.0", "method": "notifications/initialized"}`
   - No response required (notifications are one-way)

## Prerequisites

### Infrastructure
- **EKS Cluster** with AWS Load Balancer Controller installed
- **Route 53 Hosted Zone** for your domain
- **ACM Certificate** for your subdomain

### AWS Resources
- **IAM Role** for Load Balancer Controller with proper permissions
- **OAuth2 Credential Provider** in AWS Bedrock AgentCore

## Quick Start

### 1. Deploy to Kubernetes

```bash
# Deploy the mock server
kubectl apply -f test/mock-mcp-server/deployment.yaml

# Deploy the ALB ingress with ACM certificate
kubectl apply -f test/mock-mcp-server/ingress.yaml

# Wait for ALB to be provisioned
kubectl get ingress -n mock-mcp-server mock-mcp-server -w
```

### 2. Get the ALB Endpoint

```bash
# Get the ALB hostname
export ALB_HOSTNAME=$(kubectl get ingress -n mock-mcp-server mock-mcp-server -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')

echo "ALB endpoint: https://$ALB_HOSTNAME"
```

### 3. Create Route 53 DNS Record

```bash
# Create A record pointing to ALB
aws route53 change-resource-record-sets \
  --hosted-zone-id YOUR_HOSTED_ZONE_ID \
  --change-batch '{
    "Changes": [{
      "Action": "CREATE",
      "ResourceRecordSet": {
        "Name": "mcp-test.yourdomain.com",
        "Type": "A",
        "AliasTarget": {
          "HostedZoneId": "ALB_HOSTED_ZONE_ID",
          "DNSName": "'$ALB_HOSTNAME'",
          "EvaluateTargetHealth": false
        }
      }
    }]
  }'
```

### 4. Create OAuth2 Credential Provider

```bash
# Create OAuth provider in Bedrock AgentCore
aws bedrock-agentcore-control create-oauth2-credential-provider \
  --name test-mcp-oauth-https \
  --credential-provider-vendor CustomOauth2 \
  --oauth2-provider-config-input '{
    "customOauth2ProviderConfig": {
      "authorizationUrl": "https://mcp-test.yourdomain.com/oauth/authorize",
      "tokenUrl": "https://mcp-test.yourdomain.com/oauth/token",
      "clientId": "test-client-id",
      "clientSecret": "test-client-secret",
      "grantType": "CLIENT_CREDENTIALS",
      "scopes": ["read", "write"]
    }
  }' \
  --region us-west-2
```

### 5. Create MCPServer Resource

```bash
# Apply the MCPServer resource
kubectl apply -f config/samples/mcpgateway_v1alpha1_mcpserver_jicomusic.yaml
```

### 6. Monitor Status

```bash
# Watch the MCPServer status
kubectl get mcpserver mcp-jicomusic -o yaml | grep -A 20 "status:"

# Check operator logs
kubectl logs -n mcp-gateway-operator-system deployment/mcp-gateway-operator -c manager -f

# Check mock server logs
kubectl logs -n mock-mcp-server deployment/mock-mcp-server -f
```

## Testing the JSON-RPC 2.0 Protocol

### Test Initialize Request

```bash
curl -X POST https://mcp-test.yourdomain.com/ \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2025-11-25",
      "capabilities": {},
      "clientInfo": {
        "name": "test-client",
        "version": "1.0.0"
      }
    }
  }'
```

Expected response:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-11-25",
    "capabilities": {
      "tools": {
        "listChanged": false
      }
    },
    "serverInfo": {
      "name": "mock-mcp-server",
      "version": "1.0.0",
      "description": "Mock MCP server for testing AWS Bedrock AgentCore integration"
    }
  }
}
```

### Test Tools List Request

```bash
curl -X POST https://mcp-test.yourdomain.com/ \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/list",
    "params": {}
  }'
```

### Test Tool Invocation

```bash
curl -X POST https://mcp-test.yourdomain.com/ \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "get_weather",
      "arguments": {
        "location": "Seattle"
      }
    }
  }'
```

## Endpoints

- `GET /health` - Health check endpoint (no auth required)
- `GET /.well-known/openid-configuration` - OAuth metadata
- `POST /oauth/token` - OAuth token endpoint (mock - accepts any credentials)
- `POST /` - MCP JSON-RPC 2.0 endpoint (requires Bearer token)

## Available Tools

1. **get_weather** - Get current weather for a location
   - Input: `{"location": "string"}`
   - Output: Mock weather data

2. **calculate** - Perform basic arithmetic calculations
   - Input: `{"expression": "string"}`
   - Output: Calculation result (uses Python `eval`)

3. **get_time** - Get current time in a timezone
   - Input: `{"timezone": "string"}`
   - Output: Current timestamp

## MCP Protocol Flow

When AWS Bedrock AgentCore connects to the mock server:

1. **OAuth Discovery**: Fetches `/.well-known/openid-configuration`
2. **OAuth Token**: Requests access token from `/oauth/token` with `client_credentials` grant
3. **Initialize**: Sends `initialize` request with protocol version and capabilities
4. **Initialize Response**: Server responds with its capabilities and server info
5. **Initialized Notification**: Client sends `notifications/initialized` (optional)
6. **Tools List**: Sends `tools/list` request to discover available tools
7. **Tools Response**: Server returns list of tools with schemas
8. **Gateway Target Ready**: AWS Bedrock marks gateway target as READY

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  AWS Bedrock AgentCore                                      │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  OAuth2 Credential Provider (Custom)                 │  │
│  │  - Stores client credentials in Secrets Manager     │  │
│  │  - Uses custom OAuth endpoints                       │  │
│  └──────────────────────────────────────────────────────┘  │
│                           │                                  │
│                           │ Gets OAuth token                 │
│                           ▼                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Gateway Target (MCP Server)                         │  │
│  │  - Endpoint: https://mcp-test.yourdomain.com         │  │
│  │  - Auth: OAuth2 (client_credentials)                 │  │
│  │  - Protocol: MCP JSON-RPC 2.0                        │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           │
                           │ JSON-RPC 2.0 requests
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  Application Load Balancer (ALB)                            │
│  - TLS termination with ACM certificate                     │
│  - Routes HTTPS traffic to backend                          │
└─────────────────────────────────────────────────────────────┘
                           │
                           │ HTTP traffic
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  Mock MCP Server (Kubernetes)                               │
│  - Validates Bearer token                                   │
│  - Implements MCP JSON-RPC 2.0 protocol                     │
│  - Returns mock tools and execution results                 │
└─────────────────────────────────────────────────────────────┘
```


## Troubleshooting

### Gateway Target Status is FAILED

If the gateway target shows `FAILED` status with "Unable to connect to the MCP server":

1. **Check MCP protocol implementation**:
   - Ensure the server implements the `initialize` method
   - Verify JSON-RPC 2.0 format is used for all responses
   - Check that `tools/list` returns proper tool definitions

2. **Verify TLS certificate**:
   - Use ACM certificate with ALB (recommended)
   - Ensure certificate is valid and trusted
   - Check that HTTPS is working: `curl https://your-endpoint/health`

3. **Check OAuth configuration**:
   - Verify OAuth metadata endpoint is accessible
   - Ensure token endpoint returns valid tokens
   - Check that Bearer tokens are accepted

4. **Review logs**:
   ```bash
   # Mock server logs
   kubectl logs -n mock-mcp-server deployment/mock-mcp-server -f
   
   # Operator logs
   kubectl logs -n mcp-gateway-operator-system deployment/mcp-gateway-operator -c manager -f
   ```

### OAuth Token Endpoint Errors

If you see "Could not access Provider Token Endpoint":

- Verify the mock server is accessible over HTTPS
- Check that the TLS certificate is valid (use ACM)
- Ensure the LoadBalancer/ALB is routing traffic correctly
- Verify OAuth metadata endpoint returns correct URLs

### JSON-RPC 2.0 Format Errors

If AWS Bedrock cannot parse responses:

- Ensure all responses include `"jsonrpc": "2.0"`
- Verify `id` field matches the request
- Check that `result` or `error` field is present
- Use proper error codes for JSON-RPC errors

### ALB/Ingress Issues

If the ALB is not provisioning:

1. **Check Load Balancer Controller logs**:
   ```bash
   kubectl logs -n kube-system deployment/aws-load-balancer-controller -f
   ```

2. **Verify IAM permissions**:
   - Load Balancer Controller needs permissions for ACM, EC2, ELB, WAF
   - Check IRSA (IAM Roles for Service Accounts) is configured

3. **Check ingress annotations**:
   - Verify ACM certificate ARN is correct
   - Ensure target type is `ip` for EKS
   - Check scheme is `internet-facing`

## Notes

- This is a **mock server for testing only**
- Implements **MCP JSON-RPC 2.0 protocol** as per specification
- Uses **custom OAuth provider** with `client_credentials` grant type
- All tool responses are hardcoded mock data
- **Not suitable for production use**

## References

- [MCP Specification](https://modelcontextprotocol.io/specification/)
- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
- [AWS Certificate Manager](https://docs.aws.amazon.com/acm/)
- [AWS Load Balancer Controller](https://kubernetes-sigs.github.io/aws-load-balancer-controller/)
