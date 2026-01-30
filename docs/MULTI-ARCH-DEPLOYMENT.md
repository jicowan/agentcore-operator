# Multi-Architecture Image Deployment

## Overview

The MCP Gateway Operator now supports multi-architecture container images, enabling deployment on both AMD64 and ARM64 nodes in Kubernetes clusters.

## Supported Architectures

- `linux/amd64` - Intel/AMD 64-bit processors
- `linux/arm64` - ARM 64-bit processors (AWS Graviton, Apple Silicon, etc.)

## Building Multi-Arch Images

### Prerequisites

1. Docker Buildx with multi-platform support
2. A buildx builder configured for multi-platform builds

### Create Buildx Builder

```bash
docker buildx create --name multiarch --driver docker-container --use
```

### Build and Push Multi-Arch Image

Use the `docker-buildx` make target to build and push multi-architecture images:

```bash
make docker-buildx IMG=<registry>/<image>:<tag> PLATFORMS=linux/amd64,linux/arm64
```

Example:
```bash
make docker-buildx \
  IMG=820537372947.dkr.ecr.us-west-2.amazonaws.com/mcp-gateway-operator:v0.1.0-multiarch \
  PLATFORMS=linux/amd64,linux/arm64
```

### Verify Multi-Arch Image

Inspect the image manifest to verify it supports multiple architectures:

```bash
docker buildx imagetools inspect <registry>/<image>:<tag>
```

Example output:
```
Name:      820537372947.dkr.ecr.us-west-2.amazonaws.com/mcp-gateway-operator:v0.1.0-multiarch
MediaType: application/vnd.oci.image.index.v1+json
Digest:    sha256:24cf7bcabc1ec0b584ab7be7077b7f466fac5e4376adae0f879167b9f3fc6cf7

Manifests:
  Name:      ...@sha256:0fe9be68b1dfbfbc4041cda164cdc3625c2fb6a03674816db1bd2cb1e3924de2
  Platform:  linux/amd64

  Name:      ...@sha256:28b9459c5930e727337db7c267a014d500c7458b4e40b645f0bde662cf004adf
  Platform:  linux/arm64
```

## Deploying with Helm

### Upgrade Existing Deployment

To upgrade an existing Helm deployment to use the multi-arch image:

```bash
helm upgrade mcp-gateway-operator ./helm/mcp-gateway-operator \
  -n mcp-gateway-operator-system \
  --set image.repository=<registry>/<image> \
  --set image.tag=<multi-arch-tag> \
  --set aws.gatewayId=<gateway-id> \
  --set aws.region=<region> \
  --set serviceAccount.annotations."eks\.amazonaws\.com/role-arn"=<iam-role-arn>
```

### Fresh Installation

For a new installation:

```bash
helm install mcp-gateway-operator ./helm/mcp-gateway-operator \
  -n mcp-gateway-operator-system \
  --create-namespace \
  --set image.repository=<registry>/<image> \
  --set image.tag=<multi-arch-tag> \
  --set aws.gatewayId=<gateway-id> \
  --set aws.region=<region> \
  --set serviceAccount.annotations."eks\.amazonaws\.com/role-arn"=<iam-role-arn>
```

## Verification

### Check Pod Image

Verify the pod is using the multi-arch image:

```bash
kubectl describe pod -n mcp-gateway-operator-system \
  -l app.kubernetes.io/name=mcp-gateway-operator | grep -A 2 "Image:"
```

### Check Node Architecture

Verify which architecture the pod is running on:

```bash
POD_NAME=$(kubectl get pod -n mcp-gateway-operator-system \
  -l app.kubernetes.io/name=mcp-gateway-operator \
  -o jsonpath='{.items[0].metadata.name}')

NODE_NAME=$(kubectl get pod -n mcp-gateway-operator-system $POD_NAME \
  -o jsonpath='{.spec.nodeName}')

kubectl get node $NODE_NAME -o jsonpath='{.status.nodeInfo.architecture}'
```

## Benefits

### Cost Optimization

- **AWS Graviton Instances**: ARM64-based Graviton instances offer up to 40% better price-performance compared to x86-based instances
- **Flexible Node Selection**: Deploy on the most cost-effective node types available in your cluster

### Performance

- **Native Execution**: No emulation overhead - the correct architecture binary runs natively on each node type
- **Optimized Builds**: Each architecture is compiled with platform-specific optimizations

### Flexibility

- **Mixed Clusters**: Support heterogeneous clusters with both AMD64 and ARM64 nodes
- **Future-Proof**: Ready for new ARM-based instance types and architectures

## Makefile Configuration

The Makefile includes the following configuration for multi-arch builds:

```makefile
# Supported platforms (can be overridden)
PLATFORMS ?= linux/arm64,linux/amd64

# Buildx builder name (can be overridden)
BUILDX_BUILDER ?= multiarch

.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for cross-platform support
	# Creates Dockerfile.cross with platform-specific build args
	sed -e '1 s/\(^FROM\)/FROM --platform=\$\{BUILDPLATFORM\}/; t' \
	    -e ' 1,// s//FROM --platform=\$\{BUILDPLATFORM\}/' \
	    Dockerfile > Dockerfile.cross
	# Create builder if it doesn't exist
	- $(CONTAINER_TOOL) buildx create --name $(BUILDX_BUILDER) \
	    --driver docker-container --use 2>/dev/null || true
	$(CONTAINER_TOOL) buildx use $(BUILDX_BUILDER)
	# Build and push multi-arch image
	- $(CONTAINER_TOOL) buildx build --push \
	    --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	rm Dockerfile.cross
```

## Dockerfile Configuration

The Dockerfile uses build arguments to support multi-platform builds:

```dockerfile
FROM golang:1.25 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN GOPROXY=direct go mod download

COPY . .

# Build for the target platform
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    GOPROXY=direct go build -a -o manager cmd/main.go

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
```

## Troubleshooting

### Builder Not Found

If you get an error about the builder not existing:

```bash
docker buildx create --name multiarch --driver docker-container --use
```

### Platform Not Supported

If a platform is not supported by your builder:

```bash
docker buildx inspect multiarch --bootstrap
```

This will show the available platforms.

### ECR Authentication

For AWS ECR, authenticate before pushing:

```bash
aws ecr get-login-password --region <region> | \
  docker login --username AWS --password-stdin <account-id>.dkr.ecr.<region>.amazonaws.com
```

## Best Practices

1. **Tag Strategy**: Use descriptive tags that indicate multi-arch support (e.g., `v1.0.0-multiarch`)
2. **Testing**: Test on both AMD64 and ARM64 nodes before production deployment
3. **CI/CD**: Integrate multi-arch builds into your CI/CD pipeline
4. **Registry Support**: Ensure your container registry supports multi-arch manifests (ECR, Docker Hub, etc.)
5. **Builder Persistence**: Keep the buildx builder persistent to avoid recreation overhead

## References

- [Docker Buildx Documentation](https://docs.docker.com/build/buildx/)
- [Multi-platform Images](https://docs.docker.com/build/building/multi-platform/)
- [AWS Graviton](https://aws.amazon.com/ec2/graviton/)
- [Kubernetes Multi-Architecture Support](https://kubernetes.io/docs/concepts/cluster-administration/manage-deployment/#using-multiple-architectures)
