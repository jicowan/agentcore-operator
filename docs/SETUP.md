# MCP Gateway Operator - Setup Guide

## Project Structure

The MCP Gateway Operator is built with Kubebuilder and follows the standard operator project layout:

```
.
├── cmd/
│   └── main.go                    # Entry point with AWS client initialization
├── internal/
│   └── controller/                # Controller implementations
│       └── mcpserver_controller.go (to be created)
├── pkg/
│   ├── bedrock/                   # AWS Bedrock client wrapper
│   ├── config/                    # Configuration parsing and validation
│   └── status/                    # Status management utilities
├── api/
│   └── v1alpha1/                  # MCPServer CRD types (to be created)
├── config/                        # Kubernetes manifests
│   ├── crd/                       # CRD definitions
│   ├── rbac/                      # RBAC permissions
│   ├── manager/                   # Manager deployment
│   └── samples/                   # Example MCPServer resources
├── helm/                          # Helm chart (to be created)
└── test/                          # Tests
```

## Prerequisites

- Go 1.25.3 or later
- Kubebuilder 4.11.1 or later
- kubectl configured for your cluster
- AWS credentials configured (for local development)
- Docker (for building images)

## Project Initialization

The project has been initialized with:

```bash
# Domain: bedrock.aws
# Repository: github.com/aws/mcp-gateway-operator
```

## AWS SDK Dependencies

The following AWS SDK v2 packages are included:

- `github.com/aws/aws-sdk-go-v2/config` - AWS configuration loading
- `github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol` - Bedrock AgentCore API client

## Environment Variables

The operator supports the following environment variables:

- `GATEWAY_ID` - Default AWS Bedrock gateway identifier (required)
- `AWS_REGION` - AWS region for Bedrock API calls (optional, uses default credential chain if not set)

These can also be set via command-line flags:
- `--gateway-id` - AWS Bedrock gateway identifier
- `--aws-region` - AWS region

## AWS Authentication

The operator uses the AWS SDK default credential chain, which supports:

1. **Environment variables** - `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`
2. **Shared credentials file** - `~/.aws/credentials`
3. **IAM roles for EC2 instances** - When running on EC2
4. **IAM Roles for Service Accounts (IRSA)** - When running on EKS (recommended for production)

### IRSA Setup for EKS

For production deployments on EKS, use IRSA:

1. Create an IAM role with the required permissions
2. Annotate the ServiceAccount with the role ARN
3. The operator pods will automatically assume the role

Required IAM permissions:
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

## Development Workflow

### Building

```bash
# Build the operator binary
make build

# Build Docker image
make docker-build IMG=<registry>/<image>:<tag>
```

### Running Locally

```bash
# Install CRDs
make install

# Run operator locally (uses current kubeconfig context)
export GATEWAY_ID=<your-gateway-id>
export AWS_REGION=us-east-1
make run
```

### Testing

```bash
# Run unit tests
make test

# Run end-to-end tests
make test-e2e
```

### Deployment

```bash
# Build and push image
export IMG=<registry>/<image>:<tag>
make docker-build docker-push IMG=$IMG

# Deploy to cluster
make deploy IMG=$IMG

# Configure gateway ID
kubectl set env deployment/mcp-gateway-operator-controller-manager \
  -n mcp-gateway-operator-system \
  GATEWAY_ID=<your-gateway-id>
```

## Next Steps

1. Create MCPServer API types (Task 2)
2. Implement configuration parser (Task 3)
3. Implement AWS Bedrock client wrapper (Task 4)
4. Implement MCPServer controller (Task 7)
5. Create Helm chart (Task 17)

## References

- [Kubebuilder Book](https://book.kubebuilder.io)
- [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/)
- [Bedrock AgentCore Documentation](https://docs.aws.amazon.com/bedrock-agentcore/)
