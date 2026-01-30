# Platform Team Guide - KRO Integration

Guide for platform teams managing the MCP Gateway Operator with KRO.

## Overview

The KRO ResourceGraphDefinition provides a simplified, opinionated API for developers while giving platform teams centralized control over security and configuration.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Developer                                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  MCPServerApp (Simplified API)                       │  │
│  │  - name                                              │  │
│  │  - namespace                                         │  │
│  │  - description                                       │  │
│  │  - endpoint                                          │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           │
                           │ KRO transforms
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  Platform Team                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  MCPServer (Full API)                                │  │
│  │  - endpoint (from user)                              │  │
│  │  - description (from user)                           │  │
│  │  - authType: OAuth2 (static)                         │  │
│  │  - oauthProviderArn (static)                         │  │
│  │  - oauthScopes (static)                              │  │
│  │  - capabilities (static)                             │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           │
                           │ Operator reconciles
                           ▼
┌─────────────────────────────────────────────────────────────┐
│  AWS Bedrock AgentCore                                       │
│  - Gateway Target created                                    │
│  - OAuth configured                                          │
│  - Tools available to AI agents                              │
└─────────────────────────────────────────────────────────────┘
```

## Benefits

### For Platform Teams

1. **Centralized Security**: OAuth configuration managed in one place
2. **Consistent Standards**: All MCP servers use the same authentication
3. **Simplified Onboarding**: Developers don't need AWS/OAuth expertise
4. **Audit Trail**: All changes tracked through Kubernetes events
5. **Policy Enforcement**: Validation rules ensure compliance

### For Developers

1. **Simple API**: Only 4 fields to configure
2. **Fast Onboarding**: No OAuth setup required
3. **Self-Service**: Create servers without platform team involvement
4. **Clear Validation**: Immediate feedback on configuration errors
5. **GitOps Ready**: Declarative configuration for CI/CD

## Configuration Management

### Static Configuration

The RGD defines static values that apply to all MCPServer instances:

```yaml
# In kro_mcpserver_rgd.yaml
resources:
  - id: mcpserver
    template:
      spec:
        # Static values (platform team controls)
        authType: OAuth2
        oauthProviderArn: arn:aws:bedrock-agentcore:us-west-2:820537372947:token-vault/default/oauth2credentialprovider/test-mcp-oauth-https
        oauthScopes:
          - read
          - write
        capabilities:
          - tools
```

### Updating Static Configuration

To change the OAuth provider or other static values:

```bash
# Edit the RGD
kubectl edit rgd mcpserver

# Or apply an updated file
kubectl apply -f config/samples/kro_mcpserver_rgd.yaml
```

**Important**: Changes to the RGD affect:
- ✅ New instances created after the change
- ✅ Existing instances on next reconciliation
- ❌ Instances that are not reconciled (manual intervention needed)

### Multiple OAuth Providers

To support multiple OAuth providers, create separate RGDs:

```yaml
# kro_mcpserver_rgd_github.yaml
metadata:
  name: mcpserver-github
spec:
  schema:
    kind: MCPServerAppGitHub
  resources:
    - template:
        spec:
          oauthProviderArn: arn:aws:bedrock-agentcore:...:github-oauth

# kro_mcpserver_rgd_custom.yaml
metadata:
  name: mcpserver-custom
spec:
  schema:
    kind: MCPServerAppCustom
  resources:
    - template:
        spec:
          oauthProviderArn: arn:aws:bedrock-agentcore:...:custom-oauth
```

Developers then choose which type to use:
```yaml
kind: MCPServerAppGitHub  # Uses GitHub OAuth
# or
kind: MCPServerAppCustom  # Uses custom OAuth
```

## Validation Rules

### Current Validation

```yaml
spec:
  name: string | required | minLength=3 | maxLength=63 | pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
  namespace: string | default=default | pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
  description: string | required | minLength=10 | maxLength=200
  endpoint: string | required | pattern=^https://.*
```

### Adding Custom Validation

To add additional validation rules:

```yaml
# Example: Restrict endpoints to specific domains
endpoint: string | required | pattern=^https://(.*\.example\.com|.*\.internal\.com)$

# Example: Require specific naming convention
name: string | required | pattern=^(dev|staging|prod)-[a-z0-9-]+$

# Example: Limit description length
description: string | required | minLength=20 | maxLength=100
```

## Monitoring and Observability

### Metrics to Track

1. **Instance Count**: Number of MCPServerApp instances
2. **Success Rate**: Percentage of instances in READY state
3. **Creation Time**: Time from creation to READY
4. **Failure Rate**: Percentage of instances in FAILED state

### Prometheus Queries

```promql
# Total instances
count(kube_customresource_mcpserverapp_info)

# Ready instances
count(kube_customresource_mcpserverapp_status{status="READY"})

# Failed instances
count(kube_customresource_mcpserverapp_status{status="FAILED"})

# Success rate
count(kube_customresource_mcpserverapp_status{status="READY"}) / count(kube_customresource_mcpserverapp_info)
```

### Alerting Rules

```yaml
# Alert on high failure rate
- alert: MCPServerHighFailureRate
  expr: |
    (count(kube_customresource_mcpserverapp_status{status="FAILED"}) / 
     count(kube_customresource_mcpserverapp_info)) > 0.2
  for: 10m
  annotations:
    summary: "High MCP server failure rate"
    description: "More than 20% of MCP servers are in FAILED state"

# Alert on stuck instances
- alert: MCPServerStuckCreating
  expr: |
    kube_customresource_mcpserverapp_status{status="CREATING"} > 0
  for: 15m
  annotations:
    summary: "MCP server stuck in CREATING state"
    description: "Instance {{ $labels.name }} has been CREATING for >15 minutes"
```

## RBAC Configuration

### Developer Role

Allow developers to create and manage their own MCP servers:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: mcpserver-developer
  namespace: default
rules:
  # Allow managing MCPServerApp instances
  - apiGroups: ["apps.example.com"]
    resources: ["mcpserverapps"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  
  # Allow viewing status
  - apiGroups: ["apps.example.com"]
    resources: ["mcpserverapps/status"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: mcpserver-developer
  namespace: default
subjects:
  - kind: Group
    name: developers
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: mcpserver-developer
  apiGroup: rbac.authorization.k8s.io
```

### Platform Team Role

Allow platform team to manage RGDs and view all instances:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: mcpserver-platform-admin
rules:
  # Manage RGDs
  - apiGroups: ["kro.run"]
    resources: ["resourcegraphdefinitions"]
    verbs: ["*"]
  
  # View all MCPServerApp instances
  - apiGroups: ["apps.example.com"]
    resources: ["mcpserverapps"]
    verbs: ["get", "list", "watch"]
  
  # View underlying MCPServer resources
  - apiGroups: ["mcpgateway.bedrock.aws"]
    resources: ["mcpservers"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: mcpserver-platform-admin
subjects:
  - kind: Group
    name: platform-team
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: mcpserver-platform-admin
  apiGroup: rbac.authorization.k8s.io
```

## Multi-Tenancy

### Namespace Isolation

Use namespaces to isolate teams:

```yaml
# Team A namespace
apiVersion: v1
kind: Namespace
metadata:
  name: team-a
---
# Team A can only create in their namespace
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: mcpserver-developer
  namespace: team-a
subjects:
  - kind: Group
    name: team-a
    apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: mcpserver-developer
  apiGroup: rbac.authorization.k8s.io
```

### Resource Quotas

Limit the number of MCP servers per namespace:

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: mcpserver-quota
  namespace: team-a
spec:
  hard:
    count/mcpserverapps.apps.example.com: "10"
```

### Network Policies

Restrict network access to MCP servers:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: mcpserver-access
  namespace: team-a
spec:
  podSelector:
    matchLabels:
      app: mcp-server
  policyTypes:
    - Ingress
  ingress:
    # Only allow from AWS Bedrock IP ranges
    - from:
        - ipBlock:
            cidr: 52.94.0.0/16  # Example AWS IP range
```

## Cost Management

### Tracking Costs

Add labels to track costs by team/project:

```yaml
# Update RGD to add labels
resources:
  - id: mcpserver
    template:
      metadata:
        labels:
          team: ${schema.metadata.labels.?team.orValue("unknown")}
          project: ${schema.metadata.labels.?project.orValue("unknown")}
          cost-center: ${schema.metadata.labels.?cost-center.orValue("unknown")}
```

Developers then add labels:
```yaml
metadata:
  name: my-mcp-server
  labels:
    team: platform
    project: ai-agents
    cost-center: engineering
```

### Cost Allocation

Query AWS Cost Explorer with tags:
```bash
aws ce get-cost-and-usage \
  --time-period Start=2024-01-01,End=2024-01-31 \
  --granularity MONTHLY \
  --metrics BlendedCost \
  --group-by Type=TAG,Key=team
```

## Disaster Recovery

### Backup Strategy

1. **Backup RGDs**:
   ```bash
   kubectl get rgd -o yaml > rgd-backup.yaml
   ```

2. **Backup Instances**:
   ```bash
   kubectl get mcpserverapps -A -o yaml > instances-backup.yaml
   ```

3. **Backup OAuth Providers**:
   ```bash
   aws bedrock-agentcore-control list-oauth2-credential-providers \
     --region us-west-2 > oauth-providers-backup.json
   ```

### Recovery Procedure

1. **Restore RGDs**:
   ```bash
   kubectl apply -f rgd-backup.yaml
   ```

2. **Wait for CRDs**:
   ```bash
   kubectl wait --for=condition=Ready rgd/mcpserver --timeout=5m
   ```

3. **Restore Instances**:
   ```bash
   kubectl apply -f instances-backup.yaml
   ```

## Troubleshooting

### Common Issues

#### RGD Not Creating CRD

```bash
# Check RGD status
kubectl get rgd mcpserver -o yaml

# Look for validation errors
kubectl describe rgd mcpserver

# Check KRO controller logs
kubectl logs -n kro-system deployment/kro -f
```

#### Instances Not Creating MCPServer

```bash
# Check instance status
kubectl get mcpserverapp my-server -o yaml

# Check for events
kubectl get events --sort-by='.lastTimestamp' | grep my-server

# Verify RGD is ready
kubectl get rgd mcpserver -o jsonpath='{.status.state}'
```

#### High Failure Rate

```bash
# List all failed instances
kubectl get mcpserverapps -A -o json | \
  jq -r '.items[] | select(.status.targetStatus == "FAILED") | "\(.metadata.namespace)/\(.metadata.name): \(.status.message)"'

# Common causes:
# 1. Invalid endpoints (not HTTPS, not accessible)
# 2. Invalid OAuth provider ARN
# 3. MCP protocol not implemented correctly
# 4. TLS certificate issues
```

## Best Practices

### 1. Version Control

Store RGDs in Git:
```bash
git/
├── rgds/
│   ├── mcpserver-prod.yaml
│   ├── mcpserver-dev.yaml
│   └── mcpserver-staging.yaml
└── instances/
    ├── prod/
    ├── dev/
    └── staging/
```

### 2. Environment-Specific RGDs

Use different OAuth providers per environment:
```yaml
# prod-rgd.yaml
oauthProviderArn: arn:aws:bedrock-agentcore:...:prod-oauth

# dev-rgd.yaml
oauthProviderArn: arn:aws:bedrock-agentcore:...:dev-oauth
```

### 3. Automated Testing

Test RGD changes before deploying:
```bash
# Validate RGD
kubectl apply --dry-run=server -f rgd.yaml

# Test with sample instance
kubectl apply -f test-instance.yaml
kubectl wait --for=condition=Ready mcpserverapp/test-instance --timeout=5m
kubectl delete mcpserverapp test-instance
```

### 4. Documentation

Maintain internal documentation:
- OAuth provider setup
- Endpoint requirements
- Validation rules
- Troubleshooting guides

### 5. Change Management

Use a change management process:
1. Propose RGD changes in PR
2. Review by platform team
3. Test in dev environment
4. Deploy to staging
5. Deploy to production

## Migration Guide

### From Direct MCPServer to KRO

1. **Deploy RGD**:
   ```bash
   kubectl apply -f kro_mcpserver_rgd.yaml
   ```

2. **Create MCPServerApp for each existing MCPServer**:
   ```bash
   # For each MCPServer
   kubectl get mcpserver my-server -o yaml | \
     # Transform to MCPServerApp format
     # Apply new resource
   ```

3. **Verify both resources exist**:
   ```bash
   kubectl get mcpserver,mcpserverapp
   ```

4. **Delete old MCPServer** (KRO will recreate):
   ```bash
   kubectl delete mcpserver my-server
   ```

5. **Verify MCPServerApp recreates it**:
   ```bash
   kubectl get mcpserver my-server
   ```

## Support

### Getting Help

1. **Check documentation**: Start with this guide and KRO-README.md
2. **Review logs**: Operator and KRO controller logs
3. **Check status**: Instance and RGD status
4. **Open issue**: Include logs, status, and configuration

### Providing Support

When helping developers:
1. Check their MCPServerApp configuration
2. Verify validation errors
3. Check underlying MCPServer status
4. Review operator logs for errors
5. Test endpoint accessibility

## Resources

- [KRO Documentation](https://kro.run/docs)
- [Developer Quick Start](./DEVELOPER-QUICKSTART.md)
- [Full KRO README](./KRO-README.md)
- [MCP Gateway Operator](../../README.md)
