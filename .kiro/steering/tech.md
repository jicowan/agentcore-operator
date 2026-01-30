# Technology Stack

## Build System
- **Language**: Go 1.22+
- **Build Tool**: Make
- **Container Runtime**: Docker
- **Deployment**: Kubernetes manifests and Kustomize

## Tech Stack
- **Platform**: Kubernetes (any cluster with kubectl access)
- **Operator Framework**: Kubebuilder 3.x
- **AWS SDK**: AWS Go SDK v2 (`github.com/aws/aws-sdk-go-v2`)
- **Controller Runtime**: `sigs.k8s.io/controller-runtime`
- **KRO**: Watches ResourceGraphDefinitions (external CRD)

## Core Technologies
- **KRO Controller**: Installed via Helm chart from `oci://registry.k8s.io/kro/charts/kro`
- **Kubebuilder**: Scaffolds operator code, generates CRDs, manages RBAC
- **AWS Bedrock AgentCore**: `bedrockagentcorecontrol` package for gateway target management
- **Controller Runtime**: Provides reconciliation loop, caching, and event handling

## Common Commands

### Operator Development
```bash
# Initialize Kubebuilder project (already done)
kubebuilder init --domain=example.com --repo=example.com/rgd-gateway-operator

# Build operator
make build

# Run operator locally (for development)
make run

# Run tests
make test

# Build and push Docker image
make docker-build docker-push IMG=<registry>/<image>:<tag>

# Deploy operator to cluster
make deploy IMG=<registry>/<image>:<tag>

# Undeploy operator
make undeploy
```

### KRO Installation
```bash
# Install KRO (prerequisite)
helm install kro oci://registry.k8s.io/kro/charts/kro \
  --namespace kro-system \
  --create-namespace

# Verify KRO installation
kubectl get pods -n kro-system
```

### Working with ResourceGraphDefinitions
```bash
# Apply an RGD (triggers operator)
kubectl apply -f resourcegraphdefinition.yaml

# List RGDs
kubectl get resourcegraphdefinitions
kubectl get rgd

# Inspect RGD status (includes gateway target info)
kubectl get rgd <name> -o yaml
kubectl describe rgd <name>

# Check operator logs
kubectl logs -n <operator-namespace> deployment/<operator-name> -f
```

### AWS Bedrock Gateway Targets
```bash
# List gateway targets (via AWS CLI)
aws bedrock-agentcore-control list-gateway-targets \
  --gateway-identifier <gateway-id>

# Get gateway target details
aws bedrock-agentcore-control get-gateway-target \
  --gateway-identifier <gateway-id> \
  --target-id <target-id>

# Delete gateway target manually (if needed)
aws bedrock-agentcore-control delete-gateway-target \
  --gateway-identifier <gateway-id> \
  --target-id <target-id>
```

### Debugging
```bash
# Check operator events
kubectl get events -n <operator-namespace> --sort-by='.lastTimestamp'

# View operator metrics
kubectl port-forward -n <operator-namespace> deployment/<operator-name> 8080:8080

# Check RBAC permissions
kubectl auth can-i get resourcegraphdefinitions \
  --as=system:serviceaccount:<namespace>:<serviceaccount>
```
