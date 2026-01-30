# MCP Gateway Operator Helm Chart

This Helm chart deploys the MCP Gateway Operator, which automatically creates AWS Bedrock AgentCore gateway targets for MCP servers defined as Kubernetes custom resources.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- AWS EKS cluster with IRSA (IAM Roles for Service Accounts) configured
- AWS Bedrock AgentCore gateway created

## Installation

### 1. Create IAM Role for IRSA

The operator requires AWS IAM permissions to manage Bedrock AgentCore gateway targets. Create an IAM role with the following policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "BedrockAgentCoreAccess",
      "Effect": "Allow",
      "Action": [
        "bedrock-agentcore:CreateGatewayTarget",
        "bedrock-agentcore:GetGatewayTarget",
        "bedrock-agentcore:UpdateGatewayTarget",
        "bedrock-agentcore:DeleteGatewayTarget",
        "bedrock-agentcore:ListGatewayTargets",
        "bedrock-agentcore:GetWorkloadAccessToken",
        "bedrock-agentcore:GetResourceOauth2Token"
      ],
      "Resource": [
        "arn:aws:bedrock-agentcore:*:*:gateway/*",
        "arn:aws:bedrock-agentcore:*:*:gateway-target/*",
        "arn:aws:bedrock-agentcore:*:*:workload-identity-directory/*",
        "arn:aws:bedrock-agentcore:*:*:token-vault/*"
      ]
    },
    {
      "Sid": "SecretsManagerAccess",
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret"
      ],
      "Resource": "arn:aws:secretsmanager:*:*:secret:bedrock-agentcore-identity!default/oauth2/*"
    }
  ]
}
```

**Important Notes:**

- **Bedrock AgentCore Permissions**: Required for managing gateway targets and accessing OAuth2 credential providers
- **Secrets Manager Permissions**: **Required for OAuth2 authentication**. When you create an OAuth2 credential provider in Bedrock AgentCore, it stores the client secret in AWS Secrets Manager. The operator's IAM role must have permission to read these secrets because AWS Bedrock AgentCore assumes the operator's role when retrieving OAuth credentials during gateway target registration
- The Secrets Manager resource pattern `bedrock-agentcore-identity!default/oauth2/*` matches all OAuth2 credential provider secrets created by Bedrock AgentCore. Note that Secrets Manager appends a 6-character random suffix to secret names (e.g., `-Hj3Bj2`)

**Without Secrets Manager permissions**, you will encounter errors like:
```
You are not authorized to perform: secretsmanager:GetSecretValue
(Service: AgentCredentialProvider, Status Code: 403)
```

### 2. Configure Trust Relationship

Add a trust relationship to allow the Kubernetes service account to assume the role:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::<AWS_ACCOUNT_ID>:oidc-provider/<OIDC_PROVIDER>"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "<OIDC_PROVIDER>:sub": "system:serviceaccount:<NAMESPACE>:mcp-gateway-operator",
          "<OIDC_PROVIDER>:aud": "sts.amazonaws.com"
        }
      }
    }
  ]
}
```

Replace:
- `<AWS_ACCOUNT_ID>` with your AWS account ID
- `<OIDC_PROVIDER>` with your EKS cluster's OIDC provider (e.g., `oidc.eks.us-east-1.amazonaws.com/id/EXAMPLED539D4633E53DE1B71EXAMPLE`)
- `<NAMESPACE>` with the Kubernetes namespace where you'll install the operator

### 3. Install the Chart

```bash
helm install mcp-gateway-operator ./helm/mcp-gateway-operator \
  --namespace mcp-gateway-operator-system \
  --create-namespace \
  --set aws.gatewayId=<YOUR_GATEWAY_ID> \
  --set aws.region=<YOUR_AWS_REGION> \
  --set serviceAccount.annotations."eks\.amazonaws\.com/role-arn"=<YOUR_IAM_ROLE_ARN>
```

Example:

```bash
helm install mcp-gateway-operator ./helm/mcp-gateway-operator \
  --namespace mcp-gateway-operator-system \
  --create-namespace \
  --set aws.gatewayId=gateway-abc123 \
  --set aws.region=us-east-1 \
  --set serviceAccount.annotations."eks\.amazonaws\.com/role-arn"=arn:aws:iam::123456789012:role/mcp-gateway-operator-role
```

## Configuration

The following table lists the configurable parameters of the chart and their default values.

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of operator replicas | `1` |
| `image.repository` | Container image repository | `mcp-gateway-operator` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `image.tag` | Image tag (defaults to chart appVersion) | `""` |
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.annotations` | Service account annotations (for IRSA) | `{}` |
| `serviceAccount.name` | Service account name | `""` |
| `aws.gatewayId` | AWS Bedrock gateway identifier (required) | `""` |
| `aws.region` | AWS region | `""` |
| `operator.leaderElection` | Enable leader election | `false` |
| `operator.metrics.secure` | Enable secure metrics endpoint | `true` |
| `operator.metrics.bindAddress` | Metrics bind address | `"0"` |
| `operator.healthProbeBindAddress` | Health probe bind address | `":8081"` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `128Mi` |
| `resources.requests.cpu` | CPU request | `10m` |
| `resources.requests.memory` | Memory request | `64Mi` |
| `rbac.create` | Create RBAC resources | `true` |

## Usage

After installing the operator, create MCPServer custom resources to register MCP servers as gateway targets.

**Important**: MCP server targets only support OAuth2 authentication. You must create an OAuth2 credential provider in Bedrock AgentCore before creating an MCPServer resource.

```yaml
apiVersion: mcpgateway.bedrock.aws/v1alpha1
kind: MCPServer
metadata:
  name: my-mcp-server
spec:
  endpoint: https://mcp-server.example.com
  protocolVersion: "2025-06-18"
  capabilities:
    - tools
  authType: OAuth2
  oauthProviderArn: arn:aws:bedrock-agentcore:us-east-1:123456789012:token-vault/default/oauth2credentialprovider/my-provider
  oauthScopes:
    - read
    - write
  description: "My MCP server"
```

See the [examples](../../config/samples/) directory for more examples.

## Uninstallation

```bash
helm uninstall mcp-gateway-operator --namespace mcp-gateway-operator-system
```

## Troubleshooting

### Operator fails to start with "gateway-id is required" error

Ensure you've set the `aws.gatewayId` value during installation:

```bash
helm upgrade mcp-gateway-operator ./helm/mcp-gateway-operator \
  --namespace mcp-gateway-operator-system \
  --set aws.gatewayId=<YOUR_GATEWAY_ID>
```

### Operator cannot create gateway targets (AWS permission errors)

1. Verify the IAM role has the correct permissions (see IAM policy above)
2. Verify the trust relationship allows the service account to assume the role
3. Check the service account has the correct annotation:

```bash
kubectl get serviceaccount mcp-gateway-operator -n mcp-gateway-operator-system -o yaml
```

The annotation should be:

```yaml
annotations:
  eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/mcp-gateway-operator-role
```

### OAuth2 authentication fails with Secrets Manager access denied

If you see errors like:
```
You are not authorized to perform: secretsmanager:GetSecretValue
(Service: AgentCredentialProvider, Status Code: 403)
```

This means the operator's IAM role lacks Secrets Manager permissions. AWS Bedrock AgentCore assumes the operator's IAM role when retrieving OAuth2 client secrets during gateway target registration.

**Solution**: Add the Secrets Manager permissions to the operator's IAM role (see IAM policy in the Installation section above).

**To verify the issue using CloudTrail**:
```bash
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=EventName,AttributeValue=GetSecretValue \
  --max-results 10 \
  --query 'Events[?contains(CloudTrailEvent, `AccessDenied`)]'
```

Look for events where the `userIdentity.arn` matches your operator's role and `errorCode` is `AccessDenied`.

### Check operator logs

```bash
kubectl logs -n mcp-gateway-operator-system deployment/mcp-gateway-operator -f
```

## Support

For issues and questions, please open an issue on the [GitHub repository](https://github.com/aws/mcp-gateway-operator).
