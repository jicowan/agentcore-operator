# KRO Best Practices

## ResourceGraphDefinition Design

### Schema Design
- **Keep it simple**: Only expose fields users need to configure
- **Provide defaults**: Use `| default=value` for sensible defaults
- **Validate inputs**: Use markers like `min`, `max`, `pattern`, `required`
- **Use custom types**: Define reusable types in the `types` section for complex structures

### Resource Templates
- **Use descriptive IDs**: Resource IDs become CEL variables, make them clear
- **Leverage CEL**: Reference other resources to create dependencies automatically
- **Avoid hardcoding**: Use `schema.spec` references instead of hardcoded values
- **Handle optionals**: Use `?` operator for fields that might not exist

### Dependency Management
- **Let kro infer**: Don't manually specify order, let CEL references create the DAG
- **Avoid cycles**: Circular dependencies will cause validation failures
- **Use readyWhen**: Define when resources are truly ready, not just created
- **Check topological order**: Verify with `kubectl get rgd <name> -o jsonpath='{.status.topologicalOrder}'`

## CEL Expression Guidelines

### Type Safety
- CEL expressions are type-checked at RGD creation time
- Ensure expression output types match target field types
- Use CEL functions for type conversions when needed

### Common Patterns
```yaml
# String concatenation
name: ${schema.spec.name + "-deployment"}

# Conditional values
replicas: ${schema.spec.env == "prod" ? 3 : 1}

# Array access
image: ${schema.spec.images[0]}

# Optional fields with defaults
value: ${config.data.?LOG_LEVEL.orValue("info")}

# Checking conditions
readyWhen:
  - ${deployment.status.conditions.exists(c, c.type == "Available" && c.status == "True")}
```

### Performance
- CEL expressions are compiled once and evaluated many times
- Keep expressions simple and focused
- Avoid complex computations in hot paths

## Conditional Resources

### includeWhen
- Use for optional features (monitoring, backups, ingress)
- Currently limited to `schema.spec` references
- All conditions must be true (AND logic)
- Dependent resources are automatically skipped if parent is excluded

### readyWhen
- Define when a resource is truly ready for dependents
- Can only reference the resource itself (use `${resourceId}`)
- Must return boolean values
- Use for resources with async initialization (databases, load balancers)

## External References

### When to Use
- Referencing shared configuration (ConfigMaps, Secrets)
- Reading pre-provisioned infrastructure
- Accessing cluster-wide resources

### Best Practices
- Use `externalRef` instead of `template` for existing resources
- Always use `?` operator for unstructured data (ConfigMap.data, Secret.data)
- Provide defaults with `.orValue()` for missing fields
- Document expected structure of external resources

## Status Design

### What to Expose
- Connection strings and endpoints
- Resource identifiers (ARNs, IDs)
- Readiness indicators
- Aggregated metrics from multiple resources

### What to Avoid
- Duplicating all underlying resource status
- Exposing internal implementation details
- Complex nested structures

### CEL in Status
```yaml
status:
  # Simple projection
  endpoint: ${service.status.loadBalancer.ingress[0].hostname}
  
  # String templating
  connectionString: "postgres://${database.status.endpoint}:5432/${schema.spec.dbName}"
  
  # Aggregation
  totalReplicas: ${deployment.status.replicas + worker.status.replicas}
  
  # Conditional status
  publicUrl: ${schema.spec.ingress.enabled ? ingress.status.loadBalancer.ingress[0].hostname : ""}
```

## Validation and Testing

### Static Analysis
- kro validates RGDs before accepting them
- Check for CEL syntax errors, type mismatches, circular dependencies
- Review validation errors carefully - they prevent runtime issues

### Testing RGDs
1. Apply the RGD and check status: `kubectl get rgd <name>`
2. Verify CRD was created: `kubectl get crd`
3. Create a test instance with minimal configuration
4. Check instance status and conditions
5. Verify all resources were created in correct order
6. Test with different configurations (optional features on/off)

### Debugging
```bash
# Check RGD status
kubectl describe rgd <name>

# View instance conditions
kubectl get <kind> <name> -o jsonpath='{.status.conditions}' | jq

# Check managed resources
kubectl get all -l kro.run/owned=true

# View events
kubectl get events --sort-by='.lastTimestamp'
```

## Security Considerations

### Least Privilege
- Only expose necessary configuration options
- Use RBAC to control who can create RGDs vs instances
- Validate and sanitize user inputs with markers

### Secrets Management
- Never hardcode secrets in RGDs
- Use external references to existing Secrets
- Consider using external secret operators

### Multi-Tenancy
- Use namespaces for isolation
- Apply appropriate labels for resource tracking
- Consider using resource quotas and limit ranges

## ArgoCD Integration

### Resource Tracking
Add this annotation to all templated resources for ArgoCD compatibility:
```yaml
metadata:
  annotations:
    argocd.argoproj.io/tracking-id: >-
      ${schema.metadata.namespace + "_" + schema.metadata.name + "_" + schema.apiVersion + "_" + schema.kind}
```

### Sync Policies
- Use ArgoCD's automated sync for instances
- Consider sync waves for complex dependencies
- Monitor sync status in ArgoCD UI

## Performance and Scale

### Controller Tuning
- kro uses a dynamic controller architecture
- Adjust worker count for high-volume environments
- Monitor controller metrics and logs

### Resource Limits
- Set appropriate resource requests/limits in RGDs
- Consider cluster capacity when designing RGDs
- Test with realistic workloads

## Migration and Updates

### Schema Evolution
- Additive changes are safe (new optional fields)
- Breaking changes require careful migration
- Use API versioning (v1alpha1 → v1beta1 → v1)
- Test schema updates in non-production first

### Instance Updates
- kro continuously reconciles instances
- Changes to RGDs affect new instances, not existing ones
- Update instances by modifying their spec
- kro handles resource updates automatically

## Common Pitfalls

### Avoid
- Circular dependencies between resources
- Referencing non-existent fields without `?`
- Hardcoding values that should be configurable
- Creating overly complex RGDs (split into multiple if needed)
- Forgetting to set `readyWhen` for async resources

### Do
- Start simple and iterate
- Use static analysis feedback to fix issues early
- Document complex CEL expressions
- Test with various configurations
- Monitor instance status and conditions
