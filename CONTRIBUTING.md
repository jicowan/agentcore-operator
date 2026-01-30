# Contributing to MCP Gateway Operator

Thank you for your interest in contributing to the MCP Gateway Operator! This document provides guidelines and instructions for contributing.

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code. Please be respectful and constructive in all interactions.

## Getting Started

### Prerequisites

- Go 1.22 or later
- Docker
- kubectl
- Kubebuilder 3.x
- AWS account with Bedrock AgentCore access
- EKS cluster (for testing with IRSA)

### Development Setup

1. **Clone the repository**

```bash
git clone https://github.com/aws/mcp-gateway-operator.git
cd mcp-gateway-operator
```

2. **Install dependencies**

```bash
go mod download
```

3. **Install CRDs**

```bash
make install
```

4. **Run locally**

```bash
export GATEWAY_ID=<your-gateway-id>
export AWS_REGION=<your-region>
make run
```

## Development Workflow

### Making Changes

1. **Create a branch**

```bash
git checkout -b feature/my-feature
```

2. **Make your changes**

Follow the coding standards and best practices outlined below.

3. **Test your changes**

```bash
# Run unit tests
make test

# Run linter
make lint

# Build to verify compilation
make build
```

4. **Commit your changes**

```bash
git add .
git commit -m "feat: add new feature"
```

Use conventional commit messages:
- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation changes
- `test:` for test changes
- `refactor:` for code refactoring
- `chore:` for maintenance tasks

5. **Push and create a pull request**

```bash
git push origin feature/my-feature
```

Then create a pull request on GitHub.

## Coding Standards

### Go Code Style

- Follow standard Go formatting (`gofmt`, `goimports`)
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions focused and small
- Handle errors explicitly

### Kubebuilder Conventions

- Use Kubebuilder markers for CRD generation
- Follow Kubernetes API conventions
- Use `metav1.Condition` for status conditions
- Implement proper finalizer handling
- Make reconciliation idempotent

### Testing

- Write unit tests for all new functionality
- Aim for high test coverage
- Use table-driven tests where appropriate
- Mock external dependencies (AWS SDK calls)
- Test error cases and edge conditions

Example test structure:

```go
func TestConfigParser_ParseEndpoint(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:    "valid https endpoint",
            input:   "https://example.com",
            want:    "https://example.com",
            wantErr: false,
        },
        {
            name:    "invalid http endpoint",
            input:   "http://example.com",
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parser := NewConfigParser("")
            got, err := parser.ParseEndpoint(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseEndpoint() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("ParseEndpoint() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Project Structure

```
.
├── api/v1alpha1/              # API types (CRD schemas)
├── cmd/main.go                # Entry point
├── config/                    # Kubernetes manifests
│   ├── crd/                   # Generated CRDs
│   ├── rbac/                  # Generated RBAC
│   └── samples/               # Example resources
├── docs/                      # Documentation
├── helm/                      # Helm chart
├── internal/controller/       # Controller logic
├── pkg/                       # Shared packages
│   ├── bedrock/              # AWS Bedrock client
│   ├── config/               # Configuration parser
│   └── status/               # Status manager
└── test/                      # Tests
```

## Adding New Features

### Adding a New Field to MCPServer

1. Update `api/v1alpha1/mcpserver_types.go`
2. Add validation markers
3. Run `make manifests generate`
4. Update config parser if needed
5. Update controller logic
6. Add tests
7. Update documentation and examples

### Adding a New Controller

1. Use Kubebuilder to scaffold:
   ```bash
   kubebuilder create api --group <group> --version <version> --kind <Kind>
   ```
2. Implement reconciliation logic
3. Add RBAC markers
4. Run `make manifests`
5. Register controller in `cmd/main.go`
6. Add tests
7. Update documentation

## Testing

### Unit Tests

```bash
# Run all tests
make test

# Run specific package tests
go test ./pkg/config/...

# Run with coverage
go test -cover ./...
```

### Integration Tests

Integration tests require:
- Running Kubernetes cluster
- AWS credentials configured
- Bedrock gateway created

```bash
# Set up test environment
export GATEWAY_ID=<test-gateway-id>
export AWS_REGION=<test-region>

# Run integration tests
make test-integration
```

### End-to-End Tests

E2E tests validate the complete workflow:

```bash
# Deploy to test cluster
make deploy IMG=<test-image>

# Run E2E tests
make test-e2e

# Clean up
make undeploy
```

## Documentation

### Code Documentation

- Add godoc comments for all exported types and functions
- Include examples in comments where helpful
- Document complex logic with inline comments

### User Documentation

- Update README.md for user-facing changes
- Add examples to config/samples/
- Update Helm chart documentation
- Create architecture diagrams for significant changes

## Pull Request Process

1. **Ensure all tests pass**

```bash
make test lint
```

2. **Update documentation**

- Update README.md if needed
- Add/update examples
- Update CHANGELOG.md

3. **Create pull request**

- Provide clear description of changes
- Reference related issues
- Include test results
- Add screenshots for UI changes (if applicable)

4. **Address review feedback**

- Respond to comments
- Make requested changes
- Re-request review when ready

5. **Merge**

- Squash commits if requested
- Ensure CI passes
- Maintainer will merge when approved

## Release Process

Releases are managed by maintainers:

1. Update version in Chart.yaml and PROJECT
2. Update CHANGELOG.md
3. Create git tag
4. Build and push Docker image
5. Create GitHub release
6. Update Helm chart repository

## Getting Help

- Open an issue for bugs or feature requests
- Ask questions in discussions
- Review existing issues and PRs
- Check documentation and examples

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
