# GitHub OAuth Setup Guide

This guide walks you through setting up GitHub OAuth2 authentication for MCP servers with the MCP Gateway Operator.

## Overview

The MCP Gateway Operator integrates with AWS Bedrock AgentCore to manage gateway targets for MCP servers. When using OAuth2 authentication, Bedrock AgentCore handles the OAuth flow and token management, while your MCP server validates the Bearer tokens.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  GitHub OAuth                                               │
│  - Authorization endpoint                                   │
│  - Token endpoint                                           │
└─────────────────────────────────────────────────────────────┘
                           ▲
                           │ OAuth flow
                           │
┌─────────────────────────────────────────────────────────────┐
│  AWS Bedrock AgentCore                                      │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  OAuth2 Credential Provider (GitHub)                 │  │
│  │  - Client ID/Secret stored in Secrets Manager       │  │
│  │  - Manages OAuth token lifecycle                    │  │
│  └──────────────────────────────────────────────────────┘  │
│                           │                                  │
│                           │ Bearer token                     │
│                           ▼                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Gateway Target (MCP Server)                         │  │
│  │  - Endpoint: https://your-mcp-server.com             │  │
│  │  - Auth: OAuth2 (GitHub)                             │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           │
                           │ API calls with Bearer token
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  Your MCP Server                                            │
│  - Validates Bearer token                                   │
│  - Returns tools/resources                                  │
└─────────────────────────────────────────────────────────────┘
```

## Prerequisites

1. **AWS Account** with Bedrock AgentCore access
2. **EKS Cluster** with the MCP Gateway Operator installed
3. **GitHub Account** to create OAuth Apps
4. **MCP Server** with HTTPS endpoint and OAuth2 support

## Step 1: Create GitHub OAuth App

1. Go to https://github.com/settings/developers
2. Click **"New OAuth App"**
3. Fill in the application details:
   - **Application name**: `MCP Gateway - <Your App Name>`
   - **Homepage URL**: `https://your-mcp-server.com` (or your company website)
   - **Authorization callback URL**: Leave blank for now (we'll update this later)
   - **Application description**: Optional description
4. Click **"Register application"**
5. On the next page, click **"Generate a new client secret"**
6. **Save both the Client ID and Client Secret** - you'll need them in the next step

## Step 2: Create OAuth2 Credential Provider in Bedrock AgentCore

Create a GitHub OAuth2 credential provider in AWS Bedrock AgentCore:

```bash
aws bedrock-agentcore-control create-oauth2-credential-provider \
  --name my-github-oauth \
  --credential-provider-vendor GithubOauth2 \
  --oauth2-provider-config-input '{
    "githubOauth2ProviderConfig": {
      "clientId": "YOUR_GITHUB_CLIENT_ID",
      "clientSecret": "YOUR_GITHUB_CLIENT_SECRET"
    }
  }' \
  --region us-west-2 \
  --output json
```

**Important**: Save the response, especially:
- `credentialProviderArn` - You'll use this in your MCPServer resource
- `callbackUrl` - You need to add this to your GitHub OAuth App

Example response:
```json
{
  "credentialProviderArn": "arn:aws:bedrock-agentcore:us-west-2:123456789012:token-vault/default/oauth2credentialprovider/my-github-oauth",
  "callbackUrl": "https://bedrock-agentcore.us-west-2.amazonaws.com/identities/oauth2/callback/abc123...",
  "clientSecretArn": {
    "secretArn": "arn:aws:secretsmanager:us-west-2:123456789012:secret:bedrock-agentcore-identity!default/oauth2/my-github-oauth-AbCdEf"
  },
  ...
}
```

## Step 3: Update GitHub OAuth App Callback URL

1. Go back to your GitHub OAuth App settings
2. Update the **Authorization callback URL** with the `callbackUrl` from the previous step
3. Click **"Update application"**

## Step 4: Verify IAM Permissions

Ensure the operator's IAM role has the required permissions:

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

**Critical**: The Secrets Manager permissions are required because AWS Bedrock AgentCore assumes the operator's IAM role when retrieving OAuth client secrets.

## Step 5: Create MCPServer Resource

Create a Kubernetes MCPServer resource:

```yaml
apiVersion: mcpgateway.bedrock.aws/v1alpha1
kind: MCPServer
metadata:
  name: my-mcp-server
  namespace: default
spec:
  # Your MCP server's HTTPS endpoint
  endpoint: https://your-mcp-server.com
  
  # Required capabilities
  capabilities:
    - tools
  
  # OAuth2 authentication (required for MCP servers)
  authType: OAuth2
  
  # GitHub OAuth provider ARN from Step 2
  oauthProviderArn: arn:aws:bedrock-agentcore:us-west-2:123456789012:token-vault/default/oauth2credentialprovider/my-github-oauth
  
  # OAuth scopes (optional)
  oauthScopes:
    - read
    - write
  
  # Description
  description: "My MCP server with GitHub OAuth2"
```

Apply the resource:

```bash
kubectl apply -f mcpserver.yaml
```

## Step 6: Monitor Status

Check the MCPServer status:

```bash
# Get status
kubectl get mcpserver my-mcp-server -o yaml | grep -A 20 "status:"

# Watch for READY status
kubectl get mcpserver my-mcp-server -w

# Check operator logs
kubectl logs -n mcp-gateway-operator-system -l app=mcp-gateway-operator -f
```

Expected status progression:
1. `CREATING` - Gateway target is being created
2. `READY` - Gateway target is ready and MCP server is accessible

## Troubleshooting

### Error: "You are not authorized to perform: secretsmanager:GetSecretValue"

**Cause**: The operator's IAM role lacks Secrets Manager permissions.

**Solution**: Add the Secrets Manager permissions to the operator's IAM role (see Step 4).

**Verification**:
```bash
# Check CloudTrail for the failing principal
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=EventName,AttributeValue=GetSecretValue \
  --max-results 10 \
  --query 'Events[?contains(CloudTrailEvent, `AccessDenied`)]'
```

### Error: "Could not access Provider Token Endpoint"

**Cause**: AWS Bedrock AgentCore cannot connect to the OAuth token endpoint (usually due to TLS certificate issues).

**Solutions**:
1. Ensure your MCP server has a valid TLS certificate (not self-signed)
2. Use AWS Certificate Manager (ACM) for your load balancer
3. Use API Gateway with ACM certificate in front of your MCP server

### Error: "Error parsing ClientCredentials response"

**Cause**: The OAuth token endpoint is returning an unexpected response format.

**Solution**: Ensure your OAuth token endpoint returns a valid OAuth2 token response:
```json
{
  "access_token": "gho_...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read write"
}
```

### MCPServer stuck in CREATING status

**Causes**:
1. MCP server endpoint is not accessible
2. OAuth configuration is incorrect
3. Network connectivity issues

**Debug steps**:
```bash
# Check operator logs
kubectl logs -n mcp-gateway-operator-system -l app=mcp-gateway-operator --tail=50

# Check MCPServer events
kubectl describe mcpserver my-mcp-server

# Test MCP server endpoint manually
curl -k https://your-mcp-server.com/health
```

## Testing with Mock MCP Server

For testing purposes, you can use the included mock MCP server:

1. Deploy the mock server:
```bash
# Create TLS certificate
openssl req -x509 -newkey rsa:2048 -keyout /tmp/tls.key -out /tmp/tls.crt \
  -days 365 -nodes -subj "/CN=mock-mcp-server"

kubectl create secret tls mock-mcp-tls -n mock-mcp-server \
  --cert=/tmp/tls.crt --key=/tmp/tls.key

# Deploy
kubectl apply -f test/mock-mcp-server/deployment.yaml
```

2. Get the LoadBalancer endpoint:
```bash
kubectl get svc -n mock-mcp-server mock-mcp-server
```

3. Create MCPServer resource pointing to the mock server (see `config/samples/mcpgateway_v1alpha1_mcpserver_github.yaml`)

**Note**: The mock server uses a self-signed certificate, which may cause validation issues with AWS Bedrock AgentCore. For production testing, use a proper TLS certificate.

## Best Practices

1. **Use separate OAuth Apps** for different environments (dev, staging, prod)
2. **Rotate client secrets** regularly
3. **Use minimal OAuth scopes** - only request what your MCP server needs
4. **Monitor OAuth token usage** in CloudWatch
5. **Use proper TLS certificates** - avoid self-signed certificates in production
6. **Test OAuth flow** before deploying to production
7. **Document callback URLs** for your team

## Security Considerations

1. **Client Secret Storage**: GitHub client secrets are stored in AWS Secrets Manager by Bedrock AgentCore
2. **IAM Permissions**: The operator's IAM role needs Secrets Manager access to retrieve OAuth credentials
3. **Token Lifecycle**: Bedrock AgentCore manages OAuth token refresh automatically
4. **TLS Requirements**: MCP servers must use HTTPS with valid certificates
5. **Scope Limitation**: Use minimal OAuth scopes to limit access

## Additional Resources

- [GitHub OAuth Apps Documentation](https://docs.github.com/en/developers/apps/building-oauth-apps)
- [AWS Bedrock AgentCore Documentation](https://docs.aws.amazon.com/bedrock/)
- [MCP Protocol Specification](https://modelcontextprotocol.io/)
- [Operator Helm Chart README](../helm/mcp-gateway-operator/README.md)

## Support

For issues and questions:
- Check the [Troubleshooting Guide](./TROUBLESHOOTING.md)
- Review operator logs: `kubectl logs -n mcp-gateway-operator-system -l app=mcp-gateway-operator`
- Open an issue on GitHub
