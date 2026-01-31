# MCP Gateway Operator

A Kubernetes operator that automatically creates AWS Bedrock AgentCore gateway targets when MCP (Model Context Protocol) servers are defined as Kubernetes custom resources.

## Overview

The AgentCore Operator watches for `MCPServer` custom resources in your Kubernetes cluster and automatically registers them as gateway targets in AgentCore. This enables seamless integration between your MCP servers running in Kubernetes and AgentCore agents.

### Key Features

- **Automatic Gateway Target Management**: Creates, updates, and deletes AgentCore gateway targets based on Kubernetes resources
- **OAuth2 Authentication**: Secure authentication using OAuth2 credential providers (required for MCP servers)
- **Metadata Propagation**: Configure which HTTP headers and query parameters are forwarded to MCP servers
- **IRSA Integration**: Uses IAM Roles for Service Accounts for secure AWS authentication
- **Declarative Configuration**: Define MCP servers using familiar Kubernetes YAML manifests
- **Status Tracking**: Monitor gateway target status directly in Kubernetes

## Prerequisites

- Kubernetes 1.19+ (EKS recommended for IRSA support)
- AgentCore Gateway
- IAM role with permissions to create, get, list, update, and delete gateway targets
- Helm 3.0+ (for installation)

## Quick Start

### 1. Install the Operator

```bash
helm install mcp-gateway-operator ./helm/mcp-gateway-operator \
  --namespace mcp-gateway-operator-system \
  --create-namespace \
  --set aws.gatewayId=<YOUR_GATEWAY_ID> \
  --set aws.region=us-east-1 \
  --set serviceAccount.annotations."eks\.amazonaws\.com/role-arn"=<YOUR_IAM_ROLE_ARN>
```

### 2. Create an MCPServer Resource

**Important**: MCP server targets only support OAuth2 authentication. You must create an OAuth2 credential provider in AgentCore before creating an MCPServer resource.

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
  oauthProviderArn: arn:aws:bedrock-agentcore:us-east-1:123456789012:token-vault/default/oauth2credentialprovider/my-provider
  oauthScopes:
    - read
    - write
  description: "My MCP server"
```

```bash
kubectl apply -f mcpserver.yaml
```

### 3. Check Status

```bash
kubectl get mcpservers
kubectl describe mcpserver my-mcp-server
```

## Installation

### AWS IAM Setup

The operator requires an IAM role with the following permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "bedrock-agentcore:CreateGatewayTarget",
        "bedrock-agentcore:GetGatewayTarget",
        "bedrock-agentcore:UpdateGatewayTarget",
        "bedrock-agentcore:DeleteGatewayTarget",
        "bedrock-agentcore:ListGatewayTargets"
      ],
      "Resource": [
        "arn:aws:bedrock-agentcore:*:*:gateway/*",
        "arn:aws:bedrock-agentcore:*:*:gateway-target/*"
      ]
    }
  ]
}
```

For detailed IRSA setup instructions, see the [Helm chart README](helm/mcp-gateway-operator/README.md).

### Helm Installation

See the [Helm chart documentation](helm/mcp-gateway-operator/README.md) for detailed installation instructions and configuration options.

## Usage

### MCPServer Resource Specification

```yaml
apiVersion: mcpgateway.bedrock.aws/v1alpha1
kind: MCPServer
metadata:
  name: example-server
spec:
  # Required: HTTPS endpoint of the MCP server
  endpoint: https://mcp-server.example.com
  
  # Required: Server capabilities (must include "tools")
  capabilities:
    - tools
  
  # Authentication type: OAuth2 (Remote MCP Servers only support OAuth2)
  authType: OAuth2
  
  # Optional: Custom target name (defaults to resource name)
  targetName: my-custom-target
  
  # Optional: Description
  description: "Example MCP server"
  
  # Optional: Gateway ID (defaults to GATEWAY_ID env var)
  gatewayId: gateway-abc123
```

### Authentication Methods

#### OAuth2

Uses OAuth2 for authentication:

```yaml
spec:
  authType: OAuth2
  oauthProviderArn: arn:aws:bedrock-agentcore:us-east-1:123456789012:oauth-credential-provider/my-provider
  oauthScopes:
    - read
    - write
```

### Metadata Propagation

Configure which HTTP headers and query parameters are forwarded:

```yaml
spec:
  allowedRequestHeaders:
    - X-Custom-Header
    - Authorization
  allowedQueryParameters:
    - filter
    - page
  allowedResponseHeaders:
    - X-Response-ID
```

### Examples

See the [config/samples](config/samples/) directory for complete examples:

- [OAuth2 example](config/samples/mcpgateway_v1alpha1_mcpserver_oauth2.yaml)
- [Metadata propagation example](config/samples/mcpgateway_v1alpha1_mcpserver_metadata.yaml)

## Monitoring

### Check MCPServer Status

```bash
# List all MCP servers
kubectl get mcpservers

# Get detailed status
kubectl describe mcpserver <name>

# Check conditions
kubectl get mcpserver <name> -o jsonpath='{.status.conditions}' | jq
```

### View Operator Logs

```bash
kubectl logs -n mcp-gateway-operator-system deployment/mcp-gateway-operator -f
```

## Troubleshooting

### MCPServer stuck in "CREATING" status

Check the operator logs for errors:

```bash
kubectl logs -n mcp-gateway-operator-system deployment/mcp-gateway-operator
```

Common causes:
- Invalid endpoint URL (must be HTTPS)
- Missing or invalid gateway ID
- AWS IAM permission issues

### Validation errors

Check the MCPServer status conditions:

```bash
kubectl describe mcpserver <name>
```

The operator validates:
- Endpoint must start with `https://`
- Capabilities must include `tools`
- OAuth2 requires `oauthProviderArn`

### AWS permission errors

Verify the IAM role has the correct permissions and trust relationship. See the [Helm chart README](helm/mcp-gateway-operator/README.md#1-create-iam-role-for-irsa) for details.

## Development

### Prerequisites

- Go 1.22+
- Docker
- kubectl
- Kubebuilder 3.x

### Building

```bash
# Build the operator binary
make build

# Build and push Docker image
make docker-build docker-push IMG=<registry>/<image>:<tag>
```

### Running Locally

```bash
# Install CRDs
make install

# Run operator locally (uses current kubeconfig context)
export GATEWAY_ID=<your-gateway-id>
export AWS_REGION=<your-region>
make run
```

### Testing

```bash
# Run unit tests
make test

# Run linter
make lint
```

## Architecture

The operator consists of:

- **MCPServer CRD**: Defines the desired state of MCP server gateway targets
- **Controller**: Reconciles MCPServer resources with AWS Bedrock gateway targets
- **Config Parser**: Validates and parses MCPServer specifications
- **Bedrock Client**: Wraps AWS SDK calls with retry logic
- **Status Manager**: Updates MCPServer status and conditions

For detailed architecture documentation, see [docs/architecture.md](docs/architecture.md).

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

For issues and questions:
- Open an issue on [GitHub](https://github.com/aws/mcp-gateway-operator/issues)
- Check the [troubleshooting guide](#troubleshooting)
- Review the [examples](config/samples/)

## Related Projects

- [AWS Bedrock AgentCore](https://docs.aws.amazon.com/bedrock/)
- [Model Context Protocol (MCP)](https://modelcontextprotocol.io/)
- [Kubebuilder](https://book.kubebuilder.io/)
