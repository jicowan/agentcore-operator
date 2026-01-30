# KRO Troubleshooting Guide

## RGD Validation Failures

### Symptom: RGD fails to apply with validation errors

#### CEL Syntax Errors
```
Error: CEL expression syntax error at position X
```
**Solution:**
- Check for unmatched braces `${}` 
- Verify proper escaping in strings
- Use multiline syntax with `|-` for complex expressions
- Test expressions incrementally

#### Type Mismatch Errors
```
Error: expression type 'string' cannot be assigned to field type 'integer'
```
**Solution:**
- Use type conversion functions: `int()`, `string()`, `double()`
- Verify field types in resource schemas
- Check CEL expression output type matches target field

#### Field Not Found
```
Error: undefined field 'fieldName'
```
**Solution:**
- Verify field exists in resource schema
- Use `?` operator for optional fields: `${resource.field.?optionalField}`
- Check for typos in field paths
- Ensure resource type and apiVersion are correct

#### Circular Dependency
```
Error: circular dependency detected: resourceA → resourceB → resourceA
```
**Solution:**
- Review CEL expressions to identify the cycle
- Break the cycle by using `schema.spec` instead of resource references
- Restructure resources to eliminate circular references
- Check dependency graph: `kubectl get rgd <name> -o jsonpath='{.status.topologicalOrder}'`

## Instance Creation Issues

### Symptom: Instance stuck in PENDING state

#### Check Instance Conditions
```bash
kubectl get <kind> <name> -o jsonpath='{.status.conditions}' | jq
```

**Common causes:**
- Dependency resources not ready
- `readyWhen` conditions not satisfied
- External references not found
- CEL expression evaluation errors

#### Check Resource Creation
```bash
# List resources managed by instance
kubectl get all -l kro.run/owned=true,kro.run/resource-graph-definition-name=<rgd-name>

# Check events
kubectl get events --sort-by='.lastTimestamp' | grep <instance-name>
```

### Symptom: Resources not created

#### Verify includeWhen Conditions
```bash
# Check RGD for conditional resources
kubectl get rgd <name> -o yaml | grep -A 5 includeWhen
```

**Solution:**
- Verify `includeWhen` expressions evaluate to true
- Check instance spec provides required values
- Test conditions manually with expected values

#### Check Resource Dependencies
```bash
# View topological order
kubectl get rgd <name> -o jsonpath='{.status.topologicalOrder}'
```

**Solution:**
- Ensure dependent resources are created first
- Check for missing resource references
- Verify CEL expressions resolve correctly

## Status Issues

### Symptom: Status fields not updating

#### Check Status CEL Expressions
```bash
kubectl describe <kind> <name>
```

**Common causes:**
- Referenced resource doesn't exist yet
- Field path incorrect in CEL expression
- Resource not ready (check `readyWhen`)
- Optional field missing (use `?` operator)

**Solution:**
- Verify resources exist: `kubectl get <resource-type> <resource-name>`
- Check resource status has expected fields
- Use `?` for optional fields: `${resource.status.?field}`
- Add `readyWhen` to ensure resource is ready before status projection

### Symptom: Status shows null or empty values

**Solution:**
- Use `.orValue()` for defaults: `${field.?value.orValue("default")}`
- Check if resource has populated the status field
- Verify CEL expression syntax
- Ensure resource is in ready state

## External Reference Issues

### Symptom: External resource not found

```
Error: external resource not found: ConfigMap "config-name" in namespace "default"
```

**Solution:**
- Verify external resource exists: `kubectl get <kind> <name> -n <namespace>`
- Check namespace (defaults to instance namespace if not specified)
- Ensure resource name is correct
- Verify RBAC permissions for kro to read the resource

### Symptom: External reference data not accessible

**Solution:**
- Always use `?` operator for unstructured data:
  ```yaml
  value: ${config.data.?key}
  ```
- Provide defaults with `.orValue()`:
  ```yaml
  value: ${config.data.?key.orValue("default")}
  ```
- Document expected structure of external resources

## Performance Issues

### Symptom: Slow reconciliation

#### Check Controller Logs
```bash
kubectl logs -n kro-system deployment/kro -f
```

#### Monitor Resource Usage
```bash
kubectl top pods -n kro-system
```

**Solutions:**
- Reduce complexity of CEL expressions
- Minimize number of resources in RGD
- Check for resource contention in cluster
- Consider splitting large RGDs into smaller ones
- Review controller tuning options

### Symptom: High memory usage

**Solutions:**
- Reduce number of instances
- Simplify resource templates
- Check for resource leaks (orphaned resources)
- Monitor controller metrics

## Debugging Techniques

### Enable Verbose Logging
```bash
# Check kro controller logs
kubectl logs -n kro-system deployment/kro --tail=100 -f

# Filter for specific RGD or instance
kubectl logs -n kro-system deployment/kro | grep <rgd-name>
```

### Inspect RGD Status
```bash
# Full RGD status
kubectl get rgd <name> -o yaml

# Specific status fields
kubectl get rgd <name> -o jsonpath='{.status.state}'
kubectl get rgd <name> -o jsonpath='{.status.conditions}'
kubectl get rgd <name> -o jsonpath='{.status.topologicalOrder}'
```

### Inspect Instance Status
```bash
# Full instance status
kubectl get <kind> <name> -o yaml

# Conditions hierarchy
kubectl get <kind> <name> -o jsonpath='{.status.conditions}' | jq

# Specific status fields
kubectl get <kind> <name> -o jsonpath='{.status.state}'
```

### Check Managed Resources
```bash
# List all resources managed by kro
kubectl get all -l kro.run/owned=true

# List resources for specific RGD
kubectl get all -l kro.run/resource-graph-definition-name=<rgd-name>

# List resources for specific instance
kubectl get all -l kro.run/resource-graph-definition-name=<rgd-name> -n <namespace>
```

### Validate CEL Expressions
Test CEL expressions incrementally:
1. Start with simple field access: `${schema.spec.name}`
2. Add complexity gradually: `${schema.spec.name + "-suffix"}`
3. Test conditionals separately: `${schema.spec.enabled ? "yes" : "no"}`
4. Verify type compatibility at each step

### Check Events
```bash
# All events sorted by time
kubectl get events --sort-by='.lastTimestamp'

# Events for specific resource
kubectl get events --field-selector involvedObject.name=<resource-name>

# Events in specific namespace
kubectl get events -n <namespace> --sort-by='.lastTimestamp'
```

## Common Error Messages

### "field not found"
- Field doesn't exist in resource schema
- Typo in field path
- Use `?` operator for optional fields

### "type mismatch"
- CEL expression output type doesn't match target field type
- Use type conversion functions
- Check resource schema for expected types

### "circular dependency"
- Resources reference each other creating a cycle
- Break cycle by using `schema.spec` references
- Restructure resource relationships

### "resource not ready"
- Dependent resource hasn't satisfied `readyWhen` conditions
- Check resource status and conditions
- Verify `readyWhen` expressions are correct

### "expression evaluation failed"
- CEL expression has runtime error
- Check for null references
- Use `?` operator for optional fields
- Verify all referenced resources exist

## Best Practices for Debugging

### Start Simple
- Begin with minimal RGD
- Add resources incrementally
- Test each addition before proceeding

### Use Descriptive Names
- Clear resource IDs help identify issues
- Meaningful field names in status
- Descriptive error messages in conditions

### Validate Early
- Apply RGD before creating instances
- Check RGD status after applying
- Verify CRD was created successfully

### Test Incrementally
- Create test instance with minimal config
- Add optional features one at a time
- Verify each configuration works

### Monitor Continuously
- Watch instance status during creation
- Check events for warnings
- Review controller logs for errors

### Document Assumptions
- Document expected external resources
- Note required cluster features
- List dependencies and prerequisites

## Getting Help

### Check Documentation
- KRO documentation: https://kro.run/docs
- CEL specification: https://github.com/google/cel-spec
- Kubernetes API reference: https://kubernetes.io/docs/reference/

### Community Resources
- GitHub issues: https://github.com/kro-run/kro/issues
- Community discussions: Check KRO repository for community links

### Gather Information
When reporting issues, include:
- RGD definition (sanitized)
- Instance definition (sanitized)
- RGD status: `kubectl get rgd <name> -o yaml`
- Instance status: `kubectl get <kind> <name> -o yaml`
- Controller logs: `kubectl logs -n kro-system deployment/kro`
- Events: `kubectl get events --sort-by='.lastTimestamp'`
- Kubernetes version: `kubectl version`
- KRO version: `helm list -n kro-system`
