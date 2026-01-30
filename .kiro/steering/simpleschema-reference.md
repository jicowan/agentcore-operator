# SimpleSchema Reference for KRO

## Overview
SimpleSchema is KRO's concise syntax for defining API schemas. It's more readable than OpenAPI and automatically converts to valid CRD schemas.

## Basic Types

### Primitive Types
```yaml
spec:
  # String
  name: string
  description: string
  
  # Numbers
  replicas: integer
  cpu: number
  memory: integer
  
  # Boolean
  enabled: boolean
  
  # Any type (unstructured)
  metadata: object
```

## Type Markers

### Default Values
```yaml
spec:
  replicas: integer | default=3
  env: string | default=dev
  enabled: boolean | default=true
  cpu: number | default=0.5
```

### Required Fields
```yaml
spec:
  name: string | required
  image: string | required
  namespace: string | required
```

### Validation Markers
```yaml
spec:
  # Numeric constraints
  replicas: integer | min=1 | max=10
  cpu: number | min=0.1 | max=4.0
  
  # String constraints
  name: string | minLength=3 | maxLength=63
  version: string | pattern=^v[0-9]+\.[0-9]+\.[0-9]+$
  
  # Array constraints
  ports: array | minItems=1 | maxItems=10
  tags: array | uniqueItems=true
```

### Combining Markers
```yaml
spec:
  replicas: integer | required | min=1 | max=100 | default=3
  name: string | required | minLength=3 | maxLength=63 | pattern=^[a-z0-9-]+$
```

## Complex Types

### Nested Objects
```yaml
spec:
  database:
    host: string | required
    port: integer | default=5432
    name: string | required
    credentials:
      username: string | required
      password: string | required
```

### Arrays
```yaml
spec:
  # Array of primitives
  tags: "[]string"
  ports: "[]integer"
  
  # Array of objects
  containers:
    - name: string | required
      image: string | required
      ports: "[]integer"
```

### Maps
```yaml
spec:
  # Map with string values
  labels: "map[string]string"
  annotations: "map[string]string"
  
  # Map with integer values
  resourceLimits: "map[string]integer"
  
  # Map with object values
  services: "map[string]object"
```

### Nested Arrays and Maps
```yaml
spec:
  # Array of arrays
  matrix: "[][]integer"
  
  # Map of arrays
  tagsByEnv: "map[string][]string"
  
  # Array of maps
  configs: "[]map[string]string"
```

## Custom Types

### Defining Custom Types
```yaml
spec:
  schema:
    types:
      # Define reusable types
      Server:
        host: string | required
        port: integer | default=8080
        tls: boolean | default=false
      
      Database:
        type: string | required | pattern=^(postgres|mysql|mongodb)$
        connection: Server
        credentials:
          username: string | required
          password: string | required
    
    spec:
      # Use custom types
      primaryDB: Database
      cacheServer: Server
```

### Recursive Types
```yaml
types:
  Address:
    street: string
    city: string
    country: string
  
  Company:
    name: string
    address: Address
  
  Employee:
    name: string
    company: Company
    homeAddress: Address
```

### Type References
Custom types are expanded inline in the generated CRD. KRO resolves dependencies using topological sorting.

## String Validation

### Pattern Matching
```yaml
spec:
  # DNS-compatible names
  name: string | pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?$
  
  # Semantic versioning
  version: string | pattern=^v[0-9]+\.[0-9]+\.[0-9]+$
  
  # Email format
  email: string | pattern=^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$
  
  # IP address
  ipAddress: string | pattern=^([0-9]{1,3}\.){3}[0-9]{1,3}$
```

### Length Constraints
```yaml
spec:
  shortName: string | minLength=1 | maxLength=10
  description: string | maxLength=500
  password: string | minLength=8
```

### Enum Values
```yaml
spec:
  # Use pattern for enum-like validation
  environment: string | pattern=^(dev|staging|prod)$
  logLevel: string | pattern=^(debug|info|warn|error)$
```

## Array Validation

### Size Constraints
```yaml
spec:
  ports: "[]integer" | minItems=1 | maxItems=10
  tags: "[]string" | minItems=0 | maxItems=20
```

### Unique Items
```yaml
spec:
  uniquePorts: "[]integer" | uniqueItems=true
  uniqueTags: "[]string" | uniqueItems=true
```

### Combined Constraints
```yaml
spec:
  ports: "[]integer" | required | minItems=1 | maxItems=5 | uniqueItems=true
```

## Status Schema

### CEL Expressions in Status
Status fields use CEL expressions to project values from resources:

```yaml
status:
  # Simple projection
  endpoint: ${service.status.loadBalancer.ingress[0].hostname}
  
  # String templating
  connectionString: "postgres://${database.status.endpoint}:5432/${schema.spec.dbName}"
  
  # Numeric aggregation
  totalReplicas: ${deployment.status.replicas + worker.status.replicas}
  
  # Boolean status
  ready: ${deployment.status.availableReplicas >= schema.spec.replicas}
```

### Structured Status
```yaml
status:
  # Nested status object
  database:
    endpoint: ${database.status.endpoint}
    ready: ${database.status.conditions.exists(c, c.type == "Ready")}
  
  # Array status
  endpoints: ${service.status.loadBalancer.ingress.map(i, i.hostname)}
```

### Built-in Status Fields
KRO automatically adds these to every instance:
```yaml
status:
  conditions: []object  # Array of condition objects
  state: string         # High-level state (ACTIVE, PENDING, etc.)
```

## Additional Printer Columns

Define custom columns for `kubectl get` output:
```yaml
schema:
  additionalPrinterColumns:
    - name: Replicas
      type: integer
      jsonPath: .spec.replicas
      description: Number of replicas
    
    - name: Status
      type: string
      jsonPath: .status.state
      description: Current state
    
    - name: Endpoint
      type: string
      jsonPath: .status.endpoint
      description: Service endpoint
    
    - name: Age
      type: date
      jsonPath: .metadata.creationTimestamp
```

## CRD Metadata

### Labels and Annotations
Apply custom metadata to generated CRDs:
```yaml
schema:
  metadata:
    labels:
      category: database
      team: platform
    annotations:
      documentation: https://docs.example.com/database-stack
      support: platform-team@example.com
```

## Complete Example

```yaml
apiVersion: kro.run/v1alpha1
kind: ResourceGraphDefinition
metadata:
  name: web-application
spec:
  schema:
    # API version and kind
    apiVersion: v1alpha1
    kind: WebApplication
    group: mycompany.io
    
    # Custom types
    types:
      Container:
        name: string | required
        image: string | required
        port: integer | default=8080
        env: "map[string]string"
      
      IngressConfig:
        enabled: boolean | default=false
        host: string
        tls: boolean | default=false
    
    # Instance spec schema
    spec:
      name: string | required | minLength=3 | maxLength=63
      replicas: integer | default=3 | min=1 | max=10
      containers: "[]Container" | required | minItems=1
      ingress: IngressConfig
      labels: "map[string]string"
    
    # Instance status schema
    status:
      endpoint: ${service.status.loadBalancer.ingress[0].hostname}
      availableReplicas: ${deployment.status.availableReplicas}
      ready: ${deployment.status.availableReplicas >= schema.spec.replicas}
    
    # Custom kubectl columns
    additionalPrinterColumns:
      - name: Replicas
        type: integer
        jsonPath: .spec.replicas
      - name: Available
        type: integer
        jsonPath: .status.availableReplicas
      - name: Endpoint
        type: string
        jsonPath: .status.endpoint
```

## Best Practices

### Schema Design
- Keep schemas simple and focused
- Provide sensible defaults
- Use validation markers to catch errors early
- Document complex fields with descriptions

### Type Organization
- Define custom types for reusable structures
- Use nested objects for related fields
- Prefer custom types over deeply nested inline objects

### Validation
- Use `required` for mandatory fields
- Add `min`/`max` for numeric bounds
- Use `pattern` for string format validation
- Set `minItems`/`maxItems` for array size limits

### Status Fields
- Only expose essential information
- Use CEL to compute derived values
- Keep status structure flat when possible
- Provide meaningful field names

## Common Patterns

### Configuration Object
```yaml
spec:
  config:
    logLevel: string | default=info | pattern=^(debug|info|warn|error)$
    timeout: integer | default=30 | min=1 | max=300
    retries: integer | default=3 | min=0 | max=10
```

### Resource Requirements
```yaml
spec:
  resources:
    cpu: number | default=0.5 | min=0.1 | max=4.0
    memory: integer | default=512 | min=128 | max=8192
```

### Multi-Environment Config
```yaml
spec:
  environment: string | required | pattern=^(dev|staging|prod)$
  replicas: integer | default=1 | min=1 | max=100
  resources:
    cpu: number | default=0.5
    memory: integer | default=512
```

### Feature Flags
```yaml
spec:
  features:
    monitoring: boolean | default=true
    backup: boolean | default=false
    tls: boolean | default=false
```
