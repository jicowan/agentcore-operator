# MCP Server Authentication Requirements

## Overview

This document describes the authentication requirements for **MCP Server gateway targets** in AWS Bedrock AgentCore.

## Protocol Version Configuration

**Important**: The MCP protocol version is configured at the **gateway level**, not at the target level. When you create a gateway in AWS Bedrock AgentCore, you specify which MCP protocol versions the gateway supports (e.g., `2025-06-18`, `2025-03-26`). Individual gateway targets (like MCPServer resources) do not specify protocol versions - they inherit the protocol version from the gateway they're registered with.

The MCPServer CRD does not include a `protocolVersion` field because:
- Protocol version is a gateway-wide setting, not a per-target setting
- All targets registered with a gateway use the same protocol version(s) supported by that gateway
- The gateway handles protocol negotiation with MCP servers

## Important Distinction

AWS Bedrock AgentCore supports different types of MCP targets:

1. **Lambda MCP Targets** (`mcp.lambda`): Lambda functions that implement MCP protocol
   - Support **both** NoAuth (Gateway IAM Role) and OAuth2 authentication
   
2. **MCP Server Targets** (`mcp.mcpServer`): External MCP servers accessed via HTTPS endpoints
   - Support **only** OAuth2 authentication

This operator manages **MCP Server targets** (external servers with HTTPS endpoints), not Lambda MCP targets.

## Key Finding

**MCP Server targets only support OAuth2 authentication.** NoAuth (using the gateway's IAM role) is not supported for external MCP servers.

## Testing Results

When attempting to create an MCP server gateway target with NoAuth authentication, the AWS API returns:

```
ValidationException: MCP server target only supports OAUTH credential provider type
```

This was confirmed through testing on January 30, 2026, using the AWS Bedrock AgentCore Control API.

## Why OAuth2 Only for MCP Servers?

External MCP servers are accessed over HTTPS and require proper authentication to the external service. The gateway's IAM role cannot be used to authenticate to external services - it only works for AWS resources like Lambda functions. Therefore, OAuth2 (or other credential providers like API keys) must be used to authenticate to external MCP servers.

## Required Fields

For MCPServer custom resources, the following fields are **required**:

1. **endpoint**: HTTPS endpoint of the external MCP server (e.g., `https://mcp-server.example.com`)
2. **authType**: Must be set to `"OAuth2"` (this is now the default and only valid value)
3. **oauthProviderArn**: ARN of an OAuth2 credential provider created in Bedrock AgentCore
   - Format: `arn:aws:bedrock-agentcore:<region>:<account>:token-vault/default/oauth2credentialprovider/<provider-name>`
4. **oauthScopes**: At least one OAuth scope must be specified (e.g., `["read"]`)

## CRD Changes

The MCPServer CRD has been updated to reflect these requirements:

- `authType` pattern validation changed from `^(NoAuth|OAuth2)$` to `^(OAuth2)$`
- `authType` default changed from `"NoAuth"` to `"OAuth2"`
- `oauthProviderArn` is now marked as required (removed `omitempty`)
- `oauthScopes` is now marked as required with minimum 1 item

## Example Configuration

```yaml
apiVersion: mcpgateway.bedrock.aws/v1alpha1
kind: MCPServer
metadata:
  name: my-mcp-server
spec:
  endpoint: https://mcp-server.example.com
  capabilities:
    - tools
  authType: OAuth2
  oauthProviderArn: arn:aws:bedrock-agentcore:us-west-2:123456789012:token-vault/default/oauth2credentialprovider/my-provider
  oauthScopes:
    - read
    - write
  description: "My MCP server"
```

## Prerequisites

Before creating an MCPServer resource, you must:

1. Create an OAuth2 credential provider in AWS Bedrock AgentCore
2. Configure the credential provider with appropriate OAuth client credentials for your external MCP server
3. Ensure the IAM role used by the operator has permissions to access the credential provider:
   - `bedrock-agentcore:GetWorkloadAccessToken`
   - `bedrock-agentcore:GetResourceOauth2Token`

## Documentation Updates

The following documentation has been updated to reflect these requirements:

- `api/v1alpha1/mcpserver_types.go` - CRD field definitions and validation
- `config/samples/mcpgateway_v1alpha1_mcpserver_oauth2.yaml` - OAuth2 example
- `config/samples/mcpgateway_v1alpha1_mcpserver_metadata.yaml` - Metadata example with OAuth2
- `config/samples/mcpgateway_v1alpha1_mcpserver_noauth.yaml` - **Removed** (not supported for MCP servers)
- `README.md` - Quick start and usage examples
- `helm/mcp-gateway-operator/README.md` - Helm chart usage examples

## Related AWS Documentation

For more information on MCP targets and authentication in Bedrock AgentCore, refer to:
- [MCP Server Targets Documentation](https://docs.aws.amazon.com/bedrock-agentcore/latest/devguide/gateway-target-MCPservers.html)
- [Pulumi AWS Bedrock Gateway Target Guide](https://www.pulumi.com/guides/how-to/aws-bedrock-agentcore-gateway-target/) - Shows Lambda targets support both NoAuth and OAuth2
- Bedrock AgentCore Starter Toolkit: https://aws.github.io/bedrock-agentcore-starter-toolkit/

## Note on Lambda MCP Targets

If you need to use NoAuth (Gateway IAM Role) authentication, consider using a Lambda function as an MCP server instead of an external HTTPS endpoint. Lambda MCP targets support both NoAuth and OAuth2 authentication. This operator currently only supports external MCP server targets.
