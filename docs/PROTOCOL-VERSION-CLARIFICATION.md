# Protocol Version Clarification

## Summary

The `protocolVersion` field has been **removed** from the MCPServer CRD because protocol version is a **gateway-level configuration**, not a target-level configuration.

## Investigation Results

Through investigation of AWS documentation and SDK references, we confirmed:

1. **Gateway-Level Configuration**: The MCP protocol version is configured when creating a gateway using the `MCPGatewayConfiguration.supportedVersions` field
2. **Not Target-Level**: Gateway targets (including MCP server targets) do not have a protocol version field
3. **AWS SDK Evidence**:
   - AWS CDK `McpServerTargetConfiguration` only has an `endpoint` field
   - AWS CLI `create-gateway-target` for `mcpServer` only accepts `{"endpoint": "string"}`
   - AWS Go SDK `McpServerTargetConfiguration` type only includes endpoint

## Changes Made

### 1. CRD Changes
- **Removed**: `protocolVersion` field from `MCPServerSpec` in `api/v1alpha1/mcpserver_types.go`
- **Regenerated**: CRD manifests with `make manifests generate`

### 2. Code Changes
- **Removed**: `ParseProtocolVersion` method from `pkg/config/parser.go`
- **Removed**: Protocol version validation from `internal/controller/mcpserver_controller.go`
- **Removed**: `TestParseProtocolVersion` test from `pkg/config/parser_test.go`

### 3. Example Updates
- **Updated**: `config/samples/mcpgateway_v1alpha1_mcpserver_oauth2.yaml` - removed protocolVersion
- **Updated**: `config/samples/mcpgateway_v1alpha1_mcpserver_metadata.yaml` - removed protocolVersion

### 4. Test Updates
- **Updated**: All test files to remove `ProtocolVersion` field from test MCPServer specs
- **Updated**: Controller test to use OAuth2 authentication (required for MCP servers)

### 5. Documentation Updates
- **Updated**: `docs/AUTHENTICATION-REQUIREMENTS.md` with protocol version clarification
- **Added**: This clarification document

## How Protocol Version Works

When you create an AWS Bedrock AgentCore gateway, you specify which MCP protocol versions it supports:

```python
# Example using AWS CDK
gateway = agentcore.Gateway(self, "MyGateway",
    gateway_name="my-gateway",
    protocol_configuration=agentcore.McpProtocolConfiguration(
        supported_versions=[
            agentcore.MCPProtocolVersion.MCP_2025_06_18,
            agentcore.MCPProtocolVersion.MCP_2025_03_26
        ]
    )
)
```

All gateway targets registered with this gateway will use the protocol versions supported by the gateway. The gateway handles protocol negotiation with MCP servers.

## Impact on Users

**No action required** for existing deployments:
- The `protocolVersion` field was never actually used when creating gateway targets
- Removing it from the CRD aligns the operator with the actual AWS API behavior
- Users should configure protocol version at the gateway level when creating the gateway

## References

- [AWS CDK MCPGatewayConfiguration](https://docs.aws.amazon.com/cdk/api/v2/python/aws_cdk.aws_bedrock_agentcore_alpha/MCPGatewayConfiguration.html)
- [AWS CDK McpServerTargetConfiguration](https://docs.aws.amazon.com/cdk/api/v2/python/aws_cdk.aws_bedrock_agentcore_alpha/McpServerTargetConfiguration.html)
- [AWS CDK MCPProtocolVersion](https://docs.aws.amazon.com/cdk/api/v2/python/aws_cdk.aws_bedrock_agentcore_alpha/MCPProtocolVersion.html)
- [AWS CLI create-gateway-target](https://docs.aws.amazon.com/cli/latest/reference/bedrock-agentcore-control/create-gateway-target.html)

## Date

January 30, 2026
