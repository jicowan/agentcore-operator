# Task 6: MCP Protocol Implementation and TLS Setup

## Overview
Successfully implemented proper MCP (Model Context Protocol) JSON-RPC 2.0 protocol in the mock server and set up valid TLS using AWS Certificate Manager (ACM) with Application Load Balancer (ALB).

## Problem Statement
The mock MCP server was returning tools in a simple REST format (`{"tools": [...]}`), but AWS Bedrock AgentCore expects the MCP protocol which uses JSON-RPC 2.0 format. This caused gateway targets to fail with the error: "Failed to connect and fetch tools from the provided MCP target server."

## Root Cause
The MCP protocol specification requires:
1. **JSON-RPC 2.0 format**: All requests and responses must follow JSON-RPC 2.0 structure
2. **Initialization handshake**: Client must send `initialize` request before any other operations
3. **Proper method names**: Tools listing uses `tools/list` method, not REST endpoints
4. **Lifecycle notifications**: Client sends `notifications/initialized` after successful initialization

## Solution

### 1. Infrastructure Setup (ACM + ALB)
- **Domain**: `jicomusic.com` (Route 53 hosted zone: `Z11NQ8WVIQM93N`)
- **Subdomain**: `mcp-test.jicomusic.com`
- **ACM Certificate**: `arn:aws:acm:us-west-2:820537372947:certificate/b78b1865-a798-40c1-8b41-d62d484753aa`
- **ALB**: `k8s-mockmcps-mockmcps-af03f9ab17-1977873696.us-west-2.elb.amazonaws.com`
- **TLS Termination**: ALB handles TLS with ACM certificate, backend uses HTTP (port 8080)

### 2. MCP Protocol Implementation
Updated mock server to implement proper JSON-RPC 2.0 protocol:

#### Initialize Method
```python
if method == 'initialize':
    protocol_version = params.get('protocolVersion', '2025-11-25')
    response = {
        "jsonrpc": "2.0",
        "id": request_id,
        "result": {
            "protocolVersion": protocol_version,
            "capabilities": {
                "tools": {
                    "listChanged": False
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

#### Tools List Method
```python
elif method == 'tools/list':
    response = {
        "jsonrpc": "2.0",
        "id": request_id,
        "result": {
            "tools": MOCK_TOOLS
        }
    }
```

#### Tools Call Method
```python
elif method == 'tools/call':
    tool_name = params.get('name')
    tool_args = params.get('arguments', {})
    # Execute tool and return result
    result = {
        "content": [
            {
                "type": "text",
                "text": "Tool execution result"
            }
        ]
    }
    response = {
        "jsonrpc": "2.0",
        "id": request_id,
        "result": result
    }
```

#### Notifications Handler
```python
if method and method.startswith('notifications/'):
    # Notifications don't require responses
    self.send_response(200)
    self.send_header('Content-Type', 'application/json')
    self.end_headers()
    self.wfile.write(b'{}')
    return
```

### 3. OAuth Integration
The mock server continues to support OAuth 2.0 with `client_credentials` grant type:
- **OAuth metadata endpoint**: `/.well-known/openid-configuration`
- **Token endpoint**: `/oauth/token`
- **OAuth provider ARN**: `arn:aws:bedrock-agentcore:us-west-2:820537372947:token-vault/default/oauth2credentialprovider/test-mcp-oauth-https`

## Verification

### 1. Test Initialize Request
```bash
curl -X POST https://mcp-test.jicomusic.com/ \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token" \
  -d '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2025-11-25", "capabilities": {}, "clientInfo": {"name": "test-client", "version": "1.0.0"}}}'
```

Response:
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

### 2. Test Tools List Request
```bash
curl -X POST https://mcp-test.jicomusic.com/ \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test-token" \
  -d '{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {}}'
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "get_weather",
        "title": "Weather Information Provider",
        "description": "Get current weather for a location",
        "inputSchema": {
          "type": "object",
          "properties": {
            "location": {
              "type": "string",
              "description": "City name"
            }
          },
          "required": ["location"]
        }
      },
      {
        "name": "calculate",
        "title": "Calculator",
        "description": "Perform basic arithmetic calculations",
        "inputSchema": {
          "type": "object",
          "properties": {
            "expression": {
              "type": "string",
              "description": "Math expression"
            }
          },
          "required": ["expression"]
        }
      },
      {
        "name": "get_time",
        "title": "Time Provider",
        "description": "Get current time in a timezone",
        "inputSchema": {
          "type": "object",
          "properties": {
            "timezone": {
              "type": "string",
              "description": "Timezone name"
            }
          },
          "required": ["timezone"]
        }
      }
    ]
  }
}
```

### 3. Gateway Target Status
```bash
kubectl get mcpserver mcp-jicomusic -o jsonpath='{.status.targetStatus}'
```

Output: `READY`

### 4. Mock Server Logs
```
INFO:__main__:Returning OAuth metadata
INFO:__main__:OAuth token request received: grant_type=client_credentials&scope=read+write
INFO:__main__:JSON-RPC request: {"jsonrpc": "2.0", "method": "initialize", "id": "13ac318a-0", "params": {...}}
INFO:__main__:Handling initialize request
INFO:__main__:JSON-RPC response: {"jsonrpc": "2.0", "id": "13ac318a-0", "result": {...}}
INFO:__main__:Received notification: notifications/initialized
INFO:__main__:JSON-RPC request: {"jsonrpc": "2.0", "method": "tools/list", "id": "13ac318a-1", "params": {}}
INFO:__main__:Handling tools/list request
INFO:__main__:JSON-RPC response: {"jsonrpc": "2.0", "id": "13ac318a-1", "result": {"tools": [...]}}
```

## MCP Protocol Flow

1. **OAuth Discovery**: AWS Bedrock fetches `/.well-known/openid-configuration`
2. **OAuth Token**: AWS Bedrock requests access token from `/oauth/token`
3. **Initialize**: AWS Bedrock sends `initialize` request with protocol version and capabilities
4. **Initialize Response**: Mock server responds with its capabilities and server info
5. **Initialized Notification**: AWS Bedrock sends `notifications/initialized` (optional)
6. **Tools List**: AWS Bedrock sends `tools/list` request
7. **Tools Response**: Mock server returns list of available tools
8. **Gateway Target Ready**: AWS Bedrock marks gateway target as READY

## Key Learnings

### MCP Protocol Requirements
- **JSON-RPC 2.0 is mandatory**: All MCP communication uses JSON-RPC 2.0 format
- **Initialization is required**: The `initialize` method must be implemented
- **Protocol version negotiation**: Server should accept client's protocol version or respond with its own
- **Capabilities declaration**: Server must declare which features it supports (tools, resources, prompts, etc.)
- **Notifications are one-way**: Notifications like `notifications/initialized` don't require responses

### TLS Setup with ACM + ALB
- **ACM provides free certificates**: No need for Let's Encrypt when using AWS
- **ALB handles TLS termination**: Backend can use HTTP, ALB handles HTTPS
- **DNS validation is automatic**: Route 53 integration makes certificate validation seamless
- **Load Balancer Controller needs proper IAM**: Must have permissions for ACM, EC2, ELB, and WAF

### AWS Bedrock AgentCore Behavior
- **Strict protocol compliance**: AWS Bedrock expects exact MCP protocol implementation
- **Generic error messages**: Failures show "Unable to connect to the MCP server" without specific details
- **OAuth discovery first**: Always fetches OAuth metadata before attempting to connect
- **Multiple validation attempts**: Makes several requests during gateway target creation

## Files Modified
- `test/mock-mcp-server/deployment.yaml` - Updated to implement JSON-RPC 2.0 MCP protocol
- `test/mock-mcp-server/ingress.yaml` - Created ALB ingress with ACM certificate
- `config/samples/mcpgateway_v1alpha1_mcpserver_jicomusic.yaml` - MCPServer resource with valid endpoint

## References
- [MCP Specification - Tools](https://modelcontextprotocol.io/specification/draft/server/tools)
- [MCP Specification - Lifecycle](https://modelcontextprotocol.io/specification/draft/basic/lifecycle)
- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
- [AWS Certificate Manager Documentation](https://docs.aws.amazon.com/acm/)
- [AWS Load Balancer Controller](https://kubernetes-sigs.github.io/aws-load-balancer-controller/)

## Next Steps
1. Test tool invocation with `tools/call` method
2. Implement additional MCP features (resources, prompts)
3. Add proper error handling for tool execution failures
4. Consider implementing MCP server capabilities like `listChanged` notifications
5. Document the complete MCP protocol implementation for production use
