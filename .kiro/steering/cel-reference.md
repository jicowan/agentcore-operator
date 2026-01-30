# CEL Expression Reference for KRO

## Overview
Common Expression Language (CEL) is used throughout KRO for dynamic values, dependencies, and conditions. CEL is type-safe, non-Turing-complete, and validated at RGD creation time.

## Expression Syntax

### Delimiters
```yaml
# Standalone expression (entire field value)
replicas: ${schema.spec.replicas}

# String template (embedded in string)
name: "${schema.spec.name}-deployment"

# Multiple expressions in string
url: "https://${service.status.loadBalancer.ingress[0].hostname}:${schema.spec.port}"
```

### Multiline Expressions
Use YAML block scalars with chomp indicator to avoid trailing newlines:
```yaml
readyWhen:
  - |-
    ${deployment.status.conditions.exists(c, 
      c.type == "Available" && c.status == "True")}
```

## Variable References

### Schema Variable
Access user-provided instance values:
```yaml
${schema.spec.replicas}           # User-specified replicas
${schema.spec.name}               # Instance name
${schema.metadata.namespace}      # Instance namespace
${schema.metadata.labels.team}    # Instance labels
```

### Resource Variables
Reference other resources by their ID:
```yaml
${deployment.spec.replicas}                    # Field from deployment
${service.status.loadBalancer.ingress[0].ip}  # Status from service
${configmap.data.DATABASE_URL}                # ConfigMap data
```

### Special Variables
```yaml
${each}  # In forEach loops, references current iteration item
```

## Field Access

### Dot Notation
```yaml
${deployment.spec.template.spec.containers[0].image}
${service.metadata.name}
${database.status.endpoint}
```

### Array Indexing
```yaml
${schema.spec.images[0]}                      # First element
${service.status.loadBalancer.ingress[0]}     # First ingress
${deployment.spec.template.spec.containers[1].name}
```

### Optional Operator (?)
Use when field might not exist:
```yaml
${config.data.?LOG_LEVEL}                     # Returns null if missing
${service.status.?loadBalancer.?ingress[0]}   # Chain optional access
${secret.data.?password.orValue("default")}   # With default value
```

## Operators

### Comparison
```yaml
${schema.spec.replicas > 3}
${schema.spec.env == "prod"}
${deployment.status.availableReplicas >= schema.spec.replicas}
${schema.spec.version != "latest"}
```

### Logical
```yaml
${schema.spec.enabled && schema.spec.replicas > 0}
${schema.spec.env == "prod" || schema.spec.env == "staging"}
${!schema.spec.disabled}
```

### Arithmetic
```yaml
${schema.spec.replicas * 2}
${schema.spec.memory + 512}
${schema.spec.cpu / 1000}
```

### String Concatenation
```yaml
${schema.spec.name + "-deployment"}
${"app-" + schema.spec.env + "-" + schema.spec.version}
```

### Ternary Operator
```yaml
${schema.spec.env == "prod" ? 3 : 1}
${schema.spec.tls.enabled ? "https" : "http"}
${schema.spec.replicas > 0 ? schema.spec.replicas : 1}
```

## Built-in Functions

### String Functions
```yaml
${schema.spec.name.size()}                    # String length
${schema.spec.name.startsWith("app-")}        # Prefix check
${schema.spec.name.endsWith("-prod")}         # Suffix check
${schema.spec.name.contains("staging")}       # Substring check
${schema.spec.name.matches("^[a-z]+$")}       # Regex match
${schema.spec.name.lowerAscii()}              # Convert to lowercase
${schema.spec.name.upperAscii()}              # Convert to uppercase
```

### List Functions
```yaml
${schema.spec.tags.size()}                    # List length
${schema.spec.ports.exists(p, p > 8000)}      # Any element matches
${schema.spec.ports.all(p, p > 0)}            # All elements match
${schema.spec.names.map(n, n + "-suffix")}    # Transform elements
${schema.spec.items.filter(i, i.enabled)}     # Filter elements
```

### Type Conversions
```yaml
${string(schema.spec.port)}                   # Convert to string
${int(schema.spec.replicas)}                  # Convert to int
${double(schema.spec.cpu)}                    # Convert to double
```

### Utility Functions
```yaml
${has(schema.spec.optional)}                  # Check if field exists
${schema.spec.value.orValue("default")}       # Provide default
```

## Common Patterns

### Conditional Resource Creation
```yaml
includeWhen:
  - ${schema.spec.ingress.enabled}
  - ${schema.spec.env == "prod"}
```

### Resource Readiness
```yaml
readyWhen:
  - ${deployment.status.availableReplicas > 0}
  - ${deployment.status.conditions.exists(c, c.type == "Available" && c.status == "True")}
```

### Building Connection Strings
```yaml
status:
  connectionString: "postgres://${database.status.endpoint}:5432/${schema.spec.dbName}"
  redisUrl: "redis://${redis.status.host}:${redis.status.port}"
```

### Label Propagation
```yaml
metadata:
  labels:
    app: ${schema.spec.name}
    env: ${schema.spec.env}
    team: ${schema.metadata.labels.?team.orValue("default")}
```

### Port Mapping
```yaml
ports:
  - name: http
    port: ${schema.spec.port}
    targetPort: ${schema.spec.port}
```

### Replica Calculation
```yaml
replicas: ${schema.spec.env == "prod" ? 3 : 1}
replicas: ${schema.spec.replicas.orValue(2)}
```

### Resource Naming
```yaml
name: ${schema.spec.name + "-" + schema.spec.component}
name: ${schema.metadata.namespace + "-" + schema.spec.name}
```

### Aggregating Status
```yaml
status:
  totalReplicas: ${frontend.status.replicas + backend.status.replicas}
  allReady: ${frontend.status.ready && backend.status.ready}
```

## Type Checking

### Type Compatibility
CEL expressions are type-checked at RGD creation:
- Expression output type must match target field type
- Structural compatibility (duck typing) is supported
- Map ↔ Struct conversions are allowed

### Common Type Errors
```yaml
# ✗ Wrong: string assigned to integer field
replicas: ${schema.spec.name}

# ✓ Correct: convert to integer
replicas: ${int(schema.spec.replicaCount)}

# ✗ Wrong: integer assigned to string field
name: ${schema.spec.port}

# ✓ Correct: convert to string
name: ${string(schema.spec.port)}
```

## Advanced Patterns

### List Comprehension
```yaml
# Map over list
env: ${schema.spec.envVars.map(e, {"name": e.key, "value": e.value})}

# Filter list
ports: ${schema.spec.allPorts.filter(p, p.enabled)}
```

### Nested Conditionals
```yaml
image: ${
  schema.spec.env == "prod" ? "myapp:stable" :
  schema.spec.env == "staging" ? "myapp:latest" :
  "myapp:dev"
}
```

### Complex Readiness
```yaml
readyWhen:
  - ${deployment.status.?availableReplicas.orValue(0) >= schema.spec.replicas}
  - ${service.status.?loadBalancer.?ingress.size().orValue(0) > 0}
  - ${deployment.status.conditions.exists(c, 
      c.type == "Available" && 
      c.status == "True" && 
      c.observedGeneration == deployment.metadata.generation
    )}
```

### Dynamic Labels
```yaml
labels: ${
  schema.spec.labels.map(l, {l.key: l.value})
}
```

## Debugging Tips

### Validation Errors
- Check field paths exist in resource schemas
- Verify expression output types match target types
- Ensure boolean expressions in conditions
- Use `?` for optional/unknown fields

### Runtime Issues
- Check instance status conditions for CEL evaluation errors
- Verify referenced resources exist and are ready
- Use `kubectl describe` to see detailed error messages
- Test expressions incrementally

## CEL Libraries Available

KRO includes these CEL extension libraries:
- **Strings**: Advanced string manipulation
- **Encoders**: Base64, URL encoding
- **Lists**: List operations and comprehensions
- **URLs**: URL parsing and manipulation
- **Regex**: Regular expression matching
- **Kubernetes**: K8s-specific functions

Refer to CEL documentation for complete function reference.
