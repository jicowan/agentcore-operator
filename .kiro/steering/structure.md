# Project Structure

## Operator Project Layout
```
.
├── api/                           # API definitions
│   └── v1alpha1/                 # API version
│       └── *_types.go            # Custom resource types (if any)
├── internal/
│   └── controller/               # Controller implementations
│       └── rgd_controller.go     # RGD reconciler
├── config/                       # Kubernetes manifests
│   ├── crd/                      # CRD definitions (if any)
│   ├── rbac/                     # RBAC permissions
│   ├── manager/                  # Manager deployment
│   └── samples/                  # Sample resources
├── cmd/
│   └── main.go                   # Entry point
├── pkg/                          # Shared packages
│   ├── bedrock/                  # Bedrock client wrapper
│   └── config/                   # Configuration parsing
├── examples/                     # Example RGDs
│   ├── lambda-target.yaml        # RGD with Lambda target config
│   ├── openapi-target.yaml       # RGD with OpenAPI target config
│   └── mcp-server-target.yaml    # RGD with MCP server config
├── docs/                         # Documentation
│   ├── architecture.md           # Architecture decisions
│   ├── configuration.md          # Configuration guide
│   └── runbooks/                 # Operational guides
├── Dockerfile                    # Container image
├── Makefile                      # Build automation
└── go.mod                        # Go dependencies
```

## File Naming Conventions

### Go Files
- Use snake_case for file names: `rgd_controller.go`, `gateway_target.go`
- Test files: `*_test.go`
- One controller per file

### Kubernetes Manifests
- Use kebab-case: `deployment.yaml`, `service-account.yaml`
- Group by purpose in config/ subdirectories

### Example RGDs
- Use kebab-case with descriptive names: `lambda-target.yaml`
- Include comments explaining the gateway target configuration
- One example per target type

## Controller Structure

### RGD Controller
The main controller watches ResourceGraphDefinitions and manages gateway targets:

```go
type RGDReconciler struct {
    client.Client
    Scheme         *runtime.Scheme
    BedrockClient  *bedrockagentcorecontrol.Client
    GatewayID      string  // Configured gateway identifier
}
```

### Reconciliation Logic
1. Fetch ResourceGraphDefinition
2. Check for deletion (handle finalizer)
3. Extract gateway target configuration from RGD annotations/labels
4. Create/Update gateway target in AWS Bedrock
5. Update RGD status with target information
6. Requeue if target not ready

### Configuration Sources
Gateway target configuration can come from:
- **RGD Annotations**: `bedrock.aws/gateway-target-config`
- **RGD Labels**: `bedrock.aws/target-type`, `bedrock.aws/lambda-arn`
- **ConfigMap**: Cluster-wide default configurations
- **Environment Variables**: Operator-level settings

## RGD Annotation Conventions

### Gateway Target Configuration
```yaml
apiVersion: kro.run/v1alpha1
kind: ResourceGraphDefinition
metadata:
  name: my-application
  annotations:
    # Gateway target configuration
    bedrock.aws/gateway-id: "gateway-abc123"
    bedrock.aws/target-type: "lambda"
    bedrock.aws/target-name: "my-app-target"
    bedrock.aws/target-description: "Gateway target for my application"
    
    # Lambda-specific configuration
    bedrock.aws/lambda-arn: "arn:aws:lambda:us-east-1:123456789012:function:my-function"
    bedrock.aws/lambda-tool-schema: "s3://my-bucket/schema.json"
    
    # Credential configuration
    bedrock.aws/credential-type: "IAM_ROLE"
    
    # Optional: Metadata propagation
    bedrock.aws/allowed-request-headers: "X-Custom-Header,Authorization"
    bedrock.aws/allowed-query-parameters: "filter,page"
spec:
  # ... RGD spec
```

### Status Updates
The operator updates RGD status with gateway target information:
```yaml
status:
  # Operator-managed fields
  gatewayTarget:
    targetId: "abc123xyz"
    status: "READY"
    gatewayArn: "arn:aws:bedrock-agentcore:..."
    createdAt: "2024-01-15T10:30:00Z"
    lastSynchronized: "2024-01-15T10:35:00Z"
```

## Package Organization

### internal/controller
- `rgd_controller.go`: Main reconciliation logic
- `rgd_controller_test.go`: Controller tests
- `predicates.go`: Event filtering predicates

### pkg/bedrock
- `client.go`: Bedrock client wrapper with retry logic
- `gateway_target.go`: Gateway target CRUD operations
- `config_builder.go`: Builds target configurations from RGD metadata

### pkg/config
- `parser.go`: Parses RGD annotations into target configurations
- `validator.go`: Validates target configurations
- `defaults.go`: Default configuration values

## Documentation Standards

### Code Comments
```go
// RGDReconciler reconciles ResourceGraphDefinition objects and creates
// corresponding AWS Bedrock gateway targets. It watches for RGD creation,
// update, and deletion events.
type RGDReconciler struct {
    // ...
}

// Reconcile handles RGD events and manages gateway target lifecycle.
// It extracts configuration from RGD annotations, creates or updates
// the gateway target in AWS Bedrock, and updates the RGD status.
func (r *RGDReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // ...
}
```

### README Sections
- Overview and purpose
- Prerequisites (KRO, AWS credentials)
- Installation instructions
- Configuration guide
- Example RGDs
- Troubleshooting

## Version Control
- Commit operator code and manifests together
- Tag releases with semantic versioning
- Keep examples/ directory up to date with latest annotation format
- Document breaking changes in CHANGELOG.md
