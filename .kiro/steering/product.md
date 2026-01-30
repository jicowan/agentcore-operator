# Product Overview

## What is This Project?
This project is a Kubernetes operator that automatically creates AWS Bedrock AgentCore gateway targets when KRO ResourceGraphDefinitions (RGDs) are created. It bridges KRO's resource orchestration capabilities with AWS Bedrock's agent infrastructure.

## Problem Statement
When using KRO to define custom Kubernetes APIs, teams often need to integrate with AWS Bedrock agents through gateway targets. Manually creating and managing these gateway targets is error-prone and doesn't scale. This operator automates the process by:
- Watching for ResourceGraphDefinition creation events
- Automatically provisioning corresponding Bedrock gateway targets
- Managing the lifecycle of gateway targets (create, update, delete)
- Synchronizing status between Kubernetes and AWS Bedrock

## Target Users
- **Platform Teams**: Deploy KRO-based applications that need Bedrock agent integration
- **DevOps Engineers**: Automate infrastructure provisioning for AI-powered applications
- **Application Developers**: Use KRO to define applications without worrying about Bedrock gateway setup

## Key Technologies
- **KRO (Kube Resource Orchestrator)**: Creates custom Kubernetes APIs from ResourceGraphDefinitions
- **Kubebuilder**: Framework for building the Kubernetes operator
- **AWS Bedrock AgentCore**: Provides gateway infrastructure for agent communication
- **AWS Go SDK v2**: Manages Bedrock gateway targets via API calls

## Architecture
```
┌─────────────────────┐
│  KRO Controller     │
│  (watches RGDs)     │
└──────────┬──────────┘
           │
           │ RGD Created
           ▼
┌─────────────────────┐
│  Custom Operator    │◄──── This Project
│  (RGD Reconciler)   │
└──────────┬──────────┘
           │
           │ CreateGatewayTarget
           ▼
┌─────────────────────┐
│  AWS Bedrock        │
│  AgentCore          │
└─────────────────────┘
```

## Workflow
1. User creates a ResourceGraphDefinition in Kubernetes
2. KRO controller processes the RGD and creates a CRD
3. Custom operator watches RGD creation events
4. Operator extracts configuration from RGD metadata/annotations
5. Operator calls AWS Bedrock API to create gateway target
6. Operator updates RGD status with gateway target information
7. On RGD deletion, operator cleans up the gateway target
