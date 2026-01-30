# Implementation Plan: MCP Gateway Operator

## Overview

This implementation plan breaks down the MCP Gateway Operator into discrete coding tasks. The operator is built with Kubebuilder and watches MCPServer custom resources, automatically registering them as gateway targets in AWS Bedrock AgentCore. Tasks are organized to build incrementally, with testing integrated throughout.

## Tasks

- [x] 1. Initialize Kubebuilder project and set up project structure
  - Initialize Kubebuilder project with domain and repository
  - Add AWS SDK v2 dependencies (bedrockagentcorecontrol, config)
  - Set up project directory structure (internal/controller, pkg/bedrock, pkg/config)
  - Create main.go entry point with AWS client initialization
  - Configure environment variable loading (GATEWAY_ID, AWS_REGION)
  - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5_

- [x] 2. Create MCPServer API types
  - [x] 2.1 Create API using Kubebuilder
    - Run `kubebuilder create api --group mcpgateway.bedrock.aws --version v1alpha1 --kind MCPServer`
    - Generate CRD scaffolding
    - _Requirements: 1.1_
  
  - [x] 2.2 Define MCPServerSpec struct
    - Add required fields: Endpoint, ProtocolVersion, Capabilities
    - Add optional fields: GatewayID, TargetName, Description
    - Add auth fields: AuthType, OauthProviderArn, OauthScopes
    - Add metadata fields: AllowedRequestHeaders, AllowedQueryParameters, AllowedResponseHeaders
    - Add JSON tags and validation markers
    - _Requirements: 3.1, 5.1, 5.2, 6.1, 6.2, 6.3_
  
  - [x] 2.3 Define MCPServerStatus struct
    - Add TargetID, GatewayArn, TargetStatus fields
    - Add StatusReasons, LastSynchronized fields
    - Add Conditions array (using metav1.Condition)
    - Add JSON tags
    - _Requirements: 7.1, 7.2, 7.3, 7.5, 7.6_
  
  - [x] 2.4 Add Kubebuilder markers to MCPServer type
    - Add +kubebuilder:object:root=true
    - Add +kubebuilder:subresource:status
    - Add +kubebuilder:resource markers
    - Add validation markers for required fields
    - Add default value markers
    - _Requirements: 1.1, 2.1, 3.2, 19.4, 19.5, 20.2_
  
  - [x] 2.5 Generate manifests and DeepCopy methods
    - Run `make manifests generate`
    - Verify CRD generation in config/crd/bases
    - _Requirements: 1.1_

- [x] 3. Implement configuration parser and validator
  - [x] 3.1 Create pkg/config package and ConfigParser struct
    - Create pkg/config/parser.go
    - Define ConfigParser struct with defaultGatewayID field
    - _Requirements: 18.1, 18.2_
  
  - [x] 3.2 Implement validation methods in ConfigParser
    - Implement ParseEndpoint with HTTPS pattern validation
    - Implement ParseProtocolVersion with supported version check
    - Implement ParseCapabilities with "tools" requirement check
    - Implement ParseAuthConfig for NoAuth and OAuth2
    - Implement ParseMetadataConfig for header/parameter propagation
    - Implement GetGatewayID with default fallback
    - _Requirements: 3.2, 3.3, 19.2, 19.3, 19.4, 20.2, 20.3, 18.1, 18.2_
  
  - [ ]* 3.3 Write property test for endpoint validation
    - **Property 1: Endpoint Validation**
    - **Validates: Requirements 3.2, 3.3**
  
  - [ ]* 3.4 Write property test for protocol version validation
    - **Property 2: Protocol Version Validation**
    - **Validates: Requirements 19.4**
  
  - [ ]* 3.5 Write property test for tool capability validation
    - **Property 3: Tool Capability Validation**
    - **Validates: Requirements 20.2, 20.3**
  
  - [ ]* 3.6 Write property test for required field validation
    - **Property 4: Required Field Validation**
    - **Validates: Requirements 2.1, 2.2, 2.4, 3.8**
  
  - [ ]* 3.7 Write property test for OAuth configuration validation
    - **Property 5: OAuth Configuration Validation**
    - **Validates: Requirements 5.4**

- [x] 4. Implement AWS Bedrock client wrapper
  - [x] 4.1 Create pkg/bedrock package and BedrockClientWrapper
    - Create pkg/bedrock/client.go
    - Define BedrockClientWrapper struct wrapping bedrockagentcorecontrol.Client
    - Add logger field
    - Implement retry logic with exponential backoff
    - _Requirements: 14.1, 14.3_
  
  - [x] 4.2 Implement CreateGatewayTarget method
    - Call AWS CreateGatewayTarget API
    - Include client token for idempotency
    - Handle throttling and server errors with retry
    - Handle validation errors without retry
    - _Requirements: 4.1, 4.5, 14.1, 14.2, 14.3_
  
  - [x] 4.3 Implement GetGatewayTarget method
    - Call AWS GetGatewayTarget API
    - Handle errors appropriately
    - _Requirements: 8.1_
  
  - [x] 4.4 Implement UpdateGatewayTarget method
    - Call AWS UpdateGatewayTarget API
    - Handle errors appropriately
    - _Requirements: 9.2, 9.3, 9.4_
  
  - [x] 4.5 Implement DeleteGatewayTarget method
    - Call AWS DeleteGatewayTarget API
    - Treat ResourceNotFoundException as success
    - Handle other errors with retry
    - _Requirements: 11.1, 11.3, 11.4_
  
  - [ ]* 4.6 Write unit tests for error handling
    - Test throttling error retry
    - Test validation error no-retry
    - Test server error retry
    - Test ResourceNotFoundException handling
    - _Requirements: 14.1, 14.2, 14.3, 11.3_

- [x] 5. Implement target configuration builder
  - [x] 5.1 Create pkg/bedrock/config_builder.go with TargetConfigBuilder
    - Create TargetConfigBuilder struct
    - Implement Build method to create MCP server configuration
    - Return TargetConfiguration with McpTargetConfiguration
    - _Requirements: 4.2_
  
  - [x] 5.2 Implement BuildCredentialConfig method
    - Handle NoAuth case (GatewayIamRole)
    - Handle OAuth2 case with provider ARN and scopes
    - Return appropriate CredentialProviderConfiguration
    - _Requirements: 5.1, 5.2, 5.3, 5.5_
  
  - [x] 5.3 Implement BuildMetadataConfig method
    - Check if any metadata fields are present
    - Return nil if no metadata fields
    - Return MetadataConfiguration with allowed headers/parameters
    - _Requirements: 6.1, 6.2, 6.3, 6.4_
  
  - [ ]* 5.4 Write property test for NoAuth configuration
    - **Property 10: NoAuth Configuration**
    - **Validates: Requirements 5.1, 5.3**
  
  - [ ]* 5.5 Write property test for OAuth2 configuration
    - **Property 11: OAuth2 Configuration**
    - **Validates: Requirements 5.2, 5.5**
  
  - [ ]* 5.6 Write property test for metadata propagation
    - **Property 12: Metadata Propagation Configuration**
    - **Validates: Requirements 6.1, 6.2, 6.3**
  
  - [ ]* 5.7 Write property test for no metadata when absent
    - **Property 13: No Metadata When Absent**
    - **Validates: Requirements 6.4**

- [x] 6. Implement status manager
  - [x] 6.1 Create pkg/status package and StatusManager struct
    - Create pkg/status/manager.go
    - Define StatusManager struct with Kubernetes client field
    - _Requirements: 7.1_
  
  - [x] 6.2 Implement UpdateTargetCreated method
    - Update TargetID, GatewayArn, TargetStatus fields
    - Update LastSynchronized timestamp
    - Call Kubernetes API to update status
    - _Requirements: 7.1, 7.2, 7.3, 7.5_
  
  - [x] 6.3 Implement UpdateTargetStatus method
    - Update TargetStatus and StatusReasons fields
    - Update LastSynchronized timestamp
    - Call Kubernetes API to update status
    - _Requirements: 7.3, 7.5_
  
  - [x] 6.4 Implement UpdateCondition method
    - Add or update condition in Conditions array
    - Use metav1.Condition type
    - Call Kubernetes API to update status
    - _Requirements: 7.6_
  
  - [x] 6.5 Implement SetReady method
    - Set Ready condition with status True
    - Call UpdateCondition
    - _Requirements: 7.6_
  
  - [x] 6.6 Implement SetError method
    - Set Ready condition with status False
    - Include reason and message
    - Call UpdateCondition
    - _Requirements: 2.2, 3.3, 5.4_
  
  - [ ]* 6.7 Write property test for status update after creation
    - **Property 7: Status Update After Creation**
    - **Validates: Requirements 4.3, 7.1, 7.2, 7.3**
  
  - [ ]* 6.8 Write property test for ready condition
    - **Property 18: Ready Condition**
    - **Validates: Requirements 7.6**

- [x] 7. Implement MCPServerReconciler controller
  - [x] 7.1 Create controller using Kubebuilder
    - Run `kubebuilder create api --group mcpgateway.bedrock.aws --version v1alpha1 --kind MCPServer --controller=true --resource=false`
    - This creates internal/controller/mcpserver_controller.go
    - _Requirements: 1.1_
  
  - [x] 7.2 Add fields to MCPServerReconciler struct
    - Add BedrockClient field (*bedrockagentcorecontrol.Client)
    - Add DefaultGatewayID field (string)
    - Add ConfigParser field (*config.ConfigParser)
    - Add TargetConfigBuilder field (*bedrock.TargetConfigBuilder)
    - Add StatusManager field (*status.StatusManager)
    - _Requirements: 1.1_
  
  - [x] 7.3 Implement Reconcile method skeleton
    - Fetch MCPServer resource
    - Handle not found (resource deleted)
    - Check for deletion timestamp (finalizer logic)
    - Return appropriate Result
    - _Requirements: 1.2, 1.3, 1.4, 14.4_
  
  - [x] 7.4 Implement validateSpec method
    - Call ConfigParser methods to validate all fields
    - Return validation errors
    - _Requirements: 2.1, 3.2, 19.4, 20.2_
  
  - [x] 7.5 Implement validation error handling in Reconcile
    - Call validateSpec
    - If validation fails, call StatusManager.SetError
    - Return without requeue
    - _Requirements: 2.2, 14.2_
  
  - [ ]* 7.6 Write property test for validation error handling
    - **Property 24: No AWS Calls for Invalid Resources**
    - **Validates: Requirements 2.4, 16.3**

- [-] 8. Implement finalizer management
  - [x] 8.1 Add finalizer constant in controller
    - Define "bedrock.aws/gateway-target-finalizer"
    - _Requirements: 10.1_
  
  - [x] 8.2 Implement finalizer addition logic in Reconcile
    - Check if finalizer is present
    - If not, add finalizer and update resource
    - _Requirements: 10.1, 10.5_
  
  - [x] 8.3 Implement handleDeletion method
    - Check if finalizer is present
    - If present, call deleteGatewayTarget
    - Remove finalizer after successful deletion
    - Update resource
    - _Requirements: 10.2, 10.3, 10.5_
  
  - [ ]* 8.4 Write property test for finalizer addition
    - **Property 14: Finalizer Addition**
    - **Validates: Requirements 10.1, 10.5**
  
  - [ ]* 8.5 Write unit test for deletion workflow
    - Test finalizer triggers cleanup
    - Test finalizer removal after cleanup
    - _Requirements: 10.2, 10.3_

- [x] 9. Implement gateway target creation logic
  - [x] 9.1 Implement createGatewayTarget method in controller
    - Extract configuration using ConfigParser
    - Build target configuration using TargetConfigBuilder
    - Build credential configuration
    - Build metadata configuration
    - Call BedrockClient.CreateGatewayTarget
    - Update status with target ID and ARN
    - _Requirements: 4.1, 4.2, 4.3_
  
  - [x] 9.2 Add creation logic to Reconcile method
    - Check if TargetID is empty in status
    - If empty, call createGatewayTarget
    - Handle errors appropriately
    - _Requirements: 4.1, 17.1_
  
  - [ ]* 9.3 Write property test for gateway target creation
    - **Property 6: Gateway Target Creation**
    - **Validates: Requirements 4.1, 4.2**
  
  - [ ]* 9.4 Write property test for configuration extraction
    - **Property 9: Configuration Extraction**
    - **Validates: Requirements 3.1, 3.4, 3.5, 3.6, 3.7, 18.1, 18.2, 19.1**
  
  - [ ]* 9.5 Write property test for client token idempotency
    - **Property 23: Client Token Idempotency**
    - **Validates: Requirements 4.5**
  
  - [ ]* 9.6 Write property test for URL passthrough
    - **Property 21: URL Passthrough**
    - **Validates: Requirements 21.1, 21.5**
  
  - [ ]* 9.7 Write property test for target name defaulting
    - **Property 25: Target Name Defaulting**
    - **Validates: Requirements 3.6**
  
  - [ ]* 9.8 Write property test for protocol version default
    - **Property 22: Protocol Version Default**
    - **Validates: Requirements 19.5**

- [-] 10. Implement gateway target status synchronization
  - [x] 10.1 Implement syncGatewayTargetStatus method in controller
    - Call BedrockClient.GetGatewayTarget
    - Update status with current AWS status
    - If status is READY, call StatusManager.SetReady
    - If status is not READY, requeue after 10 seconds
    - _Requirements: 8.1, 8.2, 8.3_
  
  - [x] 10.2 Add status sync logic to Reconcile method
    - After creation, call syncGatewayTargetStatus
    - Handle requeue for non-ready status
    - _Requirements: 8.1, 8.2, 8.3_
  
  - [ ]* 10.3 Write property test for status synchronization
    - **Property 17: Status Synchronization**
    - **Validates: Requirements 8.1, 8.3**
  
  - [ ]* 10.4 Write unit test for requeue behavior
    - Test requeue when status is not READY
    - Test completion when status is READY
    - _Requirements: 8.2, 8.3_

- [ ] 11. Implement gateway target update logic
  - [x] 11.1 Implement detectConfigChanges method in controller
    - Compare current spec with previous spec (from annotation or status)
    - Return true if endpoint, auth, or metadata changed
    - _Requirements: 9.1_
  
  - [x] 11.2 Implement updateGatewayTarget method in controller
    - Build updated target configuration
    - Call BedrockClient.UpdateGatewayTarget
    - Update status
    - _Requirements: 9.2, 9.3, 9.4, 9.5_
  
  - [x] 11.3 Add update logic to Reconcile method
    - If TargetID exists, check for config changes
    - If changes detected, call updateGatewayTarget
    - _Requirements: 9.1, 9.2_
  
  - [ ]* 11.4 Write property test for configuration change detection
    - **Property 19: Configuration Change Detection**
    - **Validates: Requirements 9.1, 9.2, 9.3, 9.4**

- [ ] 12. Implement gateway target deletion logic
  - [x] 12.1 Implement deleteGatewayTarget method in controller
    - Extract gateway ID and target ID
    - Call BedrockClient.DeleteGatewayTarget
    - Handle ResourceNotFoundException as success
    - Handle other errors with requeue
    - _Requirements: 11.1, 11.3, 11.4_
  
  - [x] 12.2 Integrate deletion into handleDeletion method
    - Call deleteGatewayTarget
    - Only remove finalizer after successful deletion
    - _Requirements: 11.1, 11.5_
  
  - [ ]* 12.3 Write property test for gateway target deletion
    - **Property 15: Gateway Target Deletion**
    - **Validates: Requirements 11.1, 11.5**
  
  - [ ]* 12.4 Write property test for idempotent deletion
    - **Property 16: Idempotent Deletion**
    - **Validates: Requirements 11.3**
  
  - [ ]* 12.5 Write unit test for deletion error handling
    - Test ResourceNotFoundException handling
    - Test other error requeue
    - _Requirements: 11.3, 11.4_

- [ ] 13. Implement idempotency checks
  - [x] 13.1 Add idempotency check in Reconcile method
    - If TargetID exists and no config changes, skip AWS calls
    - Return success without making AWS API calls
    - _Requirements: 17.2_
  
  - [ ]* 13.2 Write property test for idempotent reconciliation
    - **Property 8: Idempotent Reconciliation**
    - **Validates: Requirements 17.1, 17.3**

- [ ] 14. Implement RBAC and Kubebuilder markers
  - [x] 14.1 Add RBAC markers to MCPServerReconciler
    - Add markers for get, list, watch MCPServer
    - Add markers for update MCPServer
    - Add markers for update MCPServer/status
    - Add markers for update MCPServer/finalizers
    - _Requirements: 13.1, 13.2, 13.3, 13.5_
  
  - [x] 14.2 Generate RBAC manifests
    - Run `make manifests` to generate RBAC YAML
    - Verify generated permissions in config/rbac/role.yaml
    - _Requirements: 13.4_

- [ ] 15. Implement SetupWithManager
  - [x] 15.1 Implement SetupWithManager method in controller
    - Register controller with manager
    - Set up watch for MCPServer resources
    - Configure reconciler options
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [ ] 16. Update main.go with controller setup
  - [x] 16.1 Add AWS SDK imports to main.go
    - Import github.com/aws/aws-sdk-go-v2/config
    - Import github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol
    - Import MCPServer API types
    - _Requirements: 12.1_
  
  - [x] 16.2 Initialize AWS Bedrock client in main.go
    - Load AWS configuration using config.LoadDefaultConfig
    - Create BedrockClient from configuration
    - Handle initialization errors
    - _Requirements: 12.1, 12.4, 12.5_
  
  - [x] 16.3 Add environment variable configuration
    - Read GATEWAY_ID environment variable
    - Read AWS_REGION environment variable
    - Validate required environment variables
    - _Requirements: 12.2, 12.3_
  
  - [x] 16.4 Initialize helper components
    - Create ConfigParser with default gateway ID
    - Create TargetConfigBuilder
    - Create StatusManager with Kubernetes client
    - _Requirements: 1.1_
  
  - [x] 16.5 Create and register MCPServerReconciler
    - Initialize reconciler with all dependencies
    - Call SetupWithManager
    - Handle setup errors
    - _Requirements: 1.1_
  
  - [x] 16.6 Register MCPServer scheme
    - Add MCPServer types to scheme in init()
    - Ensure scheme registration before manager creation
    - _Requirements: 1.1_

- [ ] 17. Create Helm chart
  - [x] 17.1 Initialize Helm chart structure
    - Create helm/ directory
    - Create Chart.yaml with metadata
    - Create values.yaml with defaults
    - Create templates/ directory
    - _Requirements: 22.1_
  
  - [x] 17.2 Create ServiceAccount template
    - Add ServiceAccount resource in templates/serviceaccount.yaml
    - Support IAM role annotation for IRSA
    - Make annotation configurable via values
    - _Requirements: 22.2, 22.3_
  
  - [x] 17.3 Create RBAC templates
    - Add Role and RoleBinding templates
    - Add ClusterRole and ClusterRoleBinding templates
    - Use generated RBAC manifests as base
    - _Requirements: 22.6_
  
  - [x] 17.4 Create Deployment template
    - Add Deployment resource in templates/deployment.yaml
    - Configure replicas via values
    - Configure resource requests/limits via values
    - Add environment variables for GATEWAY_ID and AWS_REGION
    - Mount ServiceAccount token for IRSA
    - _Requirements: 22.4, 22.5, 22.7, 22.8_
  
  - [x] 17.5 Document required IAM permissions
    - Add IAM policy document to helm/README.md
    - Document IRSA setup steps
    - Document required permissions for gateway target operations
    - _Requirements: 22.9_
  
  - [x] 17.6 Create values.yaml with documentation
    - Add all configurable values
    - Add comments explaining each value
    - Add sensible defaults
    - _Requirements: 22.10_

- [ ] 18. Create example MCPServer resources
  - [x] 18.1 Create example with NoAuth
    - Create config/samples/mcpgateway_v1alpha1_mcpserver_noauth.yaml
    - Example MCPServer with NoAuth authentication
    - Include all required fields
    - Add comments explaining fields
    - _Requirements: 5.1_
  
  - [x] 18.2 Create example with OAuth2
    - Create config/samples/mcpgateway_v1alpha1_mcpserver_oauth2.yaml
    - Example MCPServer with OAuth2 authentication
    - Include OAuth provider ARN and scopes
    - Add comments explaining OAuth setup
    - _Requirements: 5.2_
  
  - [x] 18.3 Create example with metadata propagation
    - Create config/samples/mcpgateway_v1alpha1_mcpserver_metadata.yaml
    - Example MCPServer with allowed headers/parameters
    - Add comments explaining metadata propagation
    - _Requirements: 6.1, 6.2, 6.3_

- [ ] 19. Write integration tests
  - [ ]* 19.1 Set up envtest environment
    - Configure envtest with MCPServer CRD
    - Set up test Kubernetes API server
    - _Requirements: 1.1_
  
  - [ ]* 19.2 Write integration test for full lifecycle
    - Create MCPServer resource
    - Verify gateway target creation
    - Update MCPServer resource
    - Verify gateway target update
    - Delete MCPServer resource
    - Verify gateway target deletion
    - _Requirements: 4.1, 9.1, 11.1_
  
  - [ ]* 19.3 Write integration test for validation errors
    - Create invalid MCPServer resources
    - Verify error conditions are set
    - Verify no AWS API calls are made
    - _Requirements: 2.2, 2.4_

- [ ] 20. Write documentation
  - [x] 20.1 Create README.md
    - Add project overview
    - Add prerequisites (AWS credentials, Kubernetes cluster)
    - Add installation instructions (Helm chart)
    - Add configuration guide (environment variables, IRSA)
    - Add example MCPServer resources
    - Add troubleshooting section
    - _Requirements: All_
  
  - [x] 20.2 Create CONTRIBUTING.md
    - Add development setup instructions
    - Add testing instructions (unit, integration, e2e)
    - Add code style guidelines
    - Add PR process
    - _Requirements: All_
  
  - [x] 20.3 Create architecture documentation
    - Create docs/architecture.md
    - Document component interactions
    - Document reconciliation flow
    - Add architecture diagrams
    - _Requirements: All_

- [x] 21. Checkpoint - Ensure all tests pass
  - Run `make test` to verify unit tests
  - Run `make test-e2e` to verify e2e tests
  - Ensure all tests pass, ask the user if questions arise

## Notes

- Tasks marked with `*` are optional property-based tests and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
- Integration tests validate end-to-end workflows
