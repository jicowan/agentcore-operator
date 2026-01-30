/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	mcpgatewayv1alpha1 "github.com/aws/mcp-gateway-operator/api/v1alpha1"
	"github.com/aws/mcp-gateway-operator/pkg/bedrock"
	"github.com/aws/mcp-gateway-operator/pkg/config"
	"github.com/aws/mcp-gateway-operator/pkg/status"
)

const gatewayTargetFinalizer = "bedrock.aws/gateway-target-finalizer"

// MCPServerReconciler reconciles a MCPServer object
type MCPServerReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	BedrockClient       *bedrockagentcorecontrol.Client
	DefaultGatewayID    string
	ConfigParser        *config.ConfigParser
	TargetConfigBuilder *bedrock.TargetConfigBuilder
	StatusManager       *status.Manager
}

// +kubebuilder:rbac:groups=mcpgateway.bedrock.aws,resources=mcpservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mcpgateway.bedrock.aws,resources=mcpservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mcpgateway.bedrock.aws,resources=mcpservers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *MCPServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the MCPServer resource
	mcpServer := &mcpgatewayv1alpha1.MCPServer{}
	if err := r.Get(ctx, req.NamespacedName, mcpServer); err != nil {
		if apierrors.IsNotFound(err) {
			// Resource not found, likely deleted
			log.Info("MCPServer resource not found, likely deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get MCPServer resource")
		return ctrl.Result{}, err
	}

	// Check if the resource is being deleted
	if !mcpServer.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, mcpServer, log)
	}

	// Validate the spec
	if err := r.validateSpec(mcpServer); err != nil {
		log.Error(err, "Spec validation failed")
		if statusErr := r.StatusManager.SetError(ctx, mcpServer, "ValidationError", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status with validation error")
			return ctrl.Result{}, statusErr
		}
		// Don't requeue for validation errors
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(mcpServer, gatewayTargetFinalizer) {
		controllerutil.AddFinalizer(mcpServer, gatewayTargetFinalizer)
		if err := r.Update(ctx, mcpServer); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		log.Info("Added finalizer to MCPServer")
	}

	// Check if gateway target already exists
	if mcpServer.Status.TargetID == "" {
		// Create gateway target
		return r.createGatewayTarget(ctx, mcpServer, log)
	}

	// Check for configuration changes
	if r.detectConfigChanges(ctx, mcpServer, log) {
		// Update gateway target
		return r.updateGatewayTarget(ctx, mcpServer, log)
	}

	// Idempotency check: if target is already READY and no changes, skip AWS calls
	if mcpServer.Status.TargetStatus == "READY" && mcpServer.Generation == mcpServer.Status.ObservedGeneration {
		log.V(1).Info("Gateway target is ready and no changes detected, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	// Sync gateway target status
	return r.syncGatewayTargetStatus(ctx, mcpServer, log)
}

// validateSpec validates all required fields in the MCPServer spec
func (r *MCPServerReconciler) validateSpec(mcpServer *mcpgatewayv1alpha1.MCPServer) error {
	// Validate endpoint
	if _, err := r.ConfigParser.ParseEndpoint(mcpServer.Spec.Endpoint); err != nil {
		return fmt.Errorf("invalid endpoint: %w", err)
	}

	// Validate capabilities
	if err := r.ConfigParser.ParseCapabilities(mcpServer.Spec.Capabilities); err != nil {
		return fmt.Errorf("invalid capabilities: %w", err)
	}

	// Validate auth configuration
	if mcpServer.Spec.AuthType == "OAuth2" {
		if mcpServer.Spec.OauthProviderArn == "" {
			return fmt.Errorf("oauthProviderArn is required when authType is OAuth2")
		}
	}

	// Validate gateway ID is available
	if _, err := r.ConfigParser.GetGatewayID(mcpServer); err != nil {
		return fmt.Errorf("gateway ID not available: %w", err)
	}

	return nil
}

// handleDeletion handles the deletion of an MCPServer resource
func (r *MCPServerReconciler) handleDeletion(ctx context.Context, mcpServer *mcpgatewayv1alpha1.MCPServer, log logr.Logger) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(mcpServer, gatewayTargetFinalizer) {
		// Delete gateway target from AWS
		if err := r.deleteGatewayTarget(ctx, mcpServer, log); err != nil {
			log.Error(err, "Failed to delete gateway target")
			return ctrl.Result{}, err
		}

		// Remove finalizer after successful deletion
		controllerutil.RemoveFinalizer(mcpServer, gatewayTargetFinalizer)
		if err := r.Update(ctx, mcpServer); err != nil {
			log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, err
		}
		log.Info("Removed finalizer from MCPServer after successful deletion")
	}
	return ctrl.Result{}, nil
}

// deleteGatewayTarget deletes the gateway target from AWS Bedrock AgentCore
func (r *MCPServerReconciler) deleteGatewayTarget(ctx context.Context, mcpServer *mcpgatewayv1alpha1.MCPServer, log logr.Logger) error {
	// Skip deletion if no target ID (target was never created)
	if mcpServer.Status.TargetID == "" {
		log.Info("No target ID found, skipping deletion")
		return nil
	}

	// Extract gateway ID
	gatewayID, err := r.ConfigParser.GetGatewayID(mcpServer)
	if err != nil {
		log.Error(err, "Failed to get gateway ID")
		return err
	}

	// Create Bedrock client wrapper
	bedrockWrapper := bedrock.NewBedrockClientWrapper(r.BedrockClient, log)

	// Delete gateway target
	log.Info("Deleting gateway target", "gatewayId", gatewayID, "targetId", mcpServer.Status.TargetID)
	if err := bedrockWrapper.DeleteGatewayTarget(ctx, gatewayID, mcpServer.Status.TargetID); err != nil {
		log.Error(err, "Failed to delete gateway target")
		return err
	}

	log.Info("Gateway target deleted successfully", "targetId", mcpServer.Status.TargetID)
	return nil
}

// createGatewayTarget creates a new gateway target in AWS Bedrock AgentCore
func (r *MCPServerReconciler) createGatewayTarget(ctx context.Context, mcpServer *mcpgatewayv1alpha1.MCPServer, log logr.Logger) (ctrl.Result, error) {
	// Extract gateway ID
	gatewayID, err := r.ConfigParser.GetGatewayID(mcpServer)
	if err != nil {
		log.Error(err, "Failed to get gateway ID")
		return ctrl.Result{}, err
	}

	// Determine target name (use spec.TargetName or default to resource name)
	targetName := mcpServer.Spec.TargetName
	if targetName == "" {
		targetName = mcpServer.Name
	}

	// Build target configuration
	targetConfig, err := r.TargetConfigBuilder.Build(mcpServer)
	if err != nil {
		log.Error(err, "Failed to build target configuration")
		if statusErr := r.StatusManager.SetError(ctx, mcpServer, "ConfigurationError", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status with configuration error")
		}
		return ctrl.Result{}, err
	}

	// Build credential configuration
	credentialConfig, err := r.TargetConfigBuilder.BuildCredentialConfig(mcpServer)
	if err != nil {
		log.Error(err, "Failed to build credential configuration")
		if statusErr := r.StatusManager.SetError(ctx, mcpServer, "ConfigurationError", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status with configuration error")
		}
		return ctrl.Result{}, err
	}

	// Build metadata configuration
	metadataConfig := r.TargetConfigBuilder.BuildMetadataConfig(mcpServer)

	// Build CreateGatewayTargetInput
	input := &bedrockagentcorecontrol.CreateGatewayTargetInput{
		GatewayIdentifier:                aws.String(gatewayID),
		Name:                             aws.String(targetName),
		TargetConfiguration:              targetConfig,
		CredentialProviderConfigurations: credentialConfig,
	}

	// Add description if provided
	if mcpServer.Spec.Description != "" {
		input.Description = aws.String(mcpServer.Spec.Description)
	}

	// Add metadata configuration if present
	if metadataConfig != nil {
		input.MetadataConfiguration = metadataConfig
	}

	// Create Bedrock client wrapper
	bedrockWrapper := bedrock.NewBedrockClientWrapper(r.BedrockClient, log)

	// Create gateway target
	log.Info("Creating gateway target", "gatewayId", gatewayID, "targetName", targetName)
	output, err := bedrockWrapper.CreateGatewayTarget(ctx, input)
	if err != nil {
		log.Error(err, "Failed to create gateway target")
		if statusErr := r.StatusManager.SetError(ctx, mcpServer, "CreationError", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status with creation error")
		}
		return ctrl.Result{}, err
	}

	// Re-fetch the resource to get the latest version before updating status
	latestMCPServer := &mcpgatewayv1alpha1.MCPServer{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(mcpServer), latestMCPServer); err != nil {
		log.Error(err, "Failed to re-fetch MCPServer before status update")
		return ctrl.Result{}, err
	}

	// Update status with target information
	if err := r.StatusManager.UpdateTargetCreated(ctx, latestMCPServer, *output.TargetId, *output.GatewayArn, string(output.Status)); err != nil {
		log.Error(err, "Failed to update status after creation")
		// If it's a conflict error, requeue to retry
		if apierrors.IsConflict(err) {
			log.V(1).Info("Conflict updating status after creation, will retry")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	log.Info("Gateway target created successfully", "targetId", *output.TargetId, "status", output.Status)

	// Requeue to check status
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MCPServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mcpgatewayv1alpha1.MCPServer{}).
		Named("mcpserver").
		Complete(r)
}

// detectConfigChanges checks if the MCPServer spec has changed compared to what's in AWS
func (r *MCPServerReconciler) detectConfigChanges(ctx context.Context, mcpServer *mcpgatewayv1alpha1.MCPServer, log logr.Logger) bool {
	// For now, we'll use annotations to track the last applied configuration
	// In a production system, you might want to fetch the current AWS configuration and compare

	// Check if this is the first reconciliation after creation (no generation tracking yet)
	if mcpServer.Status.TargetStatus == "" || mcpServer.Status.TargetStatus == "CREATING" {
		// Don't update while creating
		return false
	}

	// Check if the resource generation has changed (indicates spec update)
	// The generation is incremented by Kubernetes whenever the spec changes
	if mcpServer.Generation != mcpServer.Status.ObservedGeneration {
		log.Info("Configuration change detected", "generation", mcpServer.Generation, "observedGeneration", mcpServer.Status.ObservedGeneration)
		return true
	}

	return false
}

// updateGatewayTarget updates an existing gateway target in AWS Bedrock AgentCore
func (r *MCPServerReconciler) updateGatewayTarget(ctx context.Context, mcpServer *mcpgatewayv1alpha1.MCPServer, log logr.Logger) (ctrl.Result, error) {
	// Extract gateway ID
	gatewayID, err := r.ConfigParser.GetGatewayID(mcpServer)
	if err != nil {
		log.Error(err, "Failed to get gateway ID")
		return ctrl.Result{}, err
	}

	// Determine target name (use spec.TargetName or default to resource name)
	targetName := mcpServer.Spec.TargetName
	if targetName == "" {
		targetName = mcpServer.Name
	}

	// Build target configuration
	targetConfig, err := r.TargetConfigBuilder.Build(mcpServer)
	if err != nil {
		log.Error(err, "Failed to build target configuration")
		if statusErr := r.StatusManager.SetError(ctx, mcpServer, "ConfigurationError", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status with configuration error")
		}
		return ctrl.Result{}, err
	}

	// Build credential configuration
	credentialConfig, err := r.TargetConfigBuilder.BuildCredentialConfig(mcpServer)
	if err != nil {
		log.Error(err, "Failed to build credential configuration")
		if statusErr := r.StatusManager.SetError(ctx, mcpServer, "ConfigurationError", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status with configuration error")
		}
		return ctrl.Result{}, err
	}

	// Build metadata configuration
	metadataConfig := r.TargetConfigBuilder.BuildMetadataConfig(mcpServer)

	// Build UpdateGatewayTargetInput
	input := &bedrockagentcorecontrol.UpdateGatewayTargetInput{
		GatewayIdentifier:                aws.String(gatewayID),
		TargetId:                         aws.String(mcpServer.Status.TargetID),
		Name:                             aws.String(targetName),
		TargetConfiguration:              targetConfig,
		CredentialProviderConfigurations: credentialConfig,
	}

	// Add description if provided
	if mcpServer.Spec.Description != "" {
		input.Description = aws.String(mcpServer.Spec.Description)
	}

	// Add metadata configuration if present
	if metadataConfig != nil {
		input.MetadataConfiguration = metadataConfig
	}

	// Create Bedrock client wrapper
	bedrockWrapper := bedrock.NewBedrockClientWrapper(r.BedrockClient, log)

	// Update gateway target
	log.Info("Updating gateway target", "gatewayId", gatewayID, "targetId", mcpServer.Status.TargetID, "targetName", targetName)
	output, err := bedrockWrapper.UpdateGatewayTarget(ctx, input)
	if err != nil {
		log.Error(err, "Failed to update gateway target")
		if statusErr := r.StatusManager.SetError(ctx, mcpServer, "UpdateError", err.Error()); statusErr != nil {
			log.Error(statusErr, "Failed to update status with update error")
		}
		return ctrl.Result{}, err
	}

	// Re-fetch the resource to get the latest version before updating status
	latestMCPServer := &mcpgatewayv1alpha1.MCPServer{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(mcpServer), latestMCPServer); err != nil {
		log.Error(err, "Failed to re-fetch MCPServer before status update")
		return ctrl.Result{}, err
	}

	// Update status with new information
	if err := r.StatusManager.UpdateTargetStatus(ctx, latestMCPServer, string(output.Status), output.StatusReasons); err != nil {
		log.Error(err, "Failed to update status after update")
		// If it's a conflict error, requeue to retry
		if apierrors.IsConflict(err) {
			log.V(1).Info("Conflict updating status after update, will retry")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	log.Info("Gateway target updated successfully", "targetId", *output.TargetId, "status", output.Status)

	// Requeue to check status
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// syncGatewayTargetStatus synchronizes the gateway target status from AWS
func (r *MCPServerReconciler) syncGatewayTargetStatus(ctx context.Context, mcpServer *mcpgatewayv1alpha1.MCPServer, log logr.Logger) (ctrl.Result, error) {
	// Extract gateway ID
	gatewayID, err := r.ConfigParser.GetGatewayID(mcpServer)
	if err != nil {
		log.Error(err, "Failed to get gateway ID")
		return ctrl.Result{}, err
	}

	// Create Bedrock client wrapper
	bedrockWrapper := bedrock.NewBedrockClientWrapper(r.BedrockClient, log)

	// Get gateway target status
	log.V(1).Info("Syncing gateway target status", "targetId", mcpServer.Status.TargetID)
	output, err := bedrockWrapper.GetGatewayTarget(ctx, gatewayID, mcpServer.Status.TargetID)
	if err != nil {
		log.Error(err, "Failed to get gateway target status")
		return ctrl.Result{}, err
	}

	// Extract status reasons
	var statusReasons []string
	if output.StatusReasons != nil {
		statusReasons = output.StatusReasons
	}

	// Re-fetch the resource to get the latest version before updating status
	// This prevents conflicts when multiple reconciliation loops run concurrently
	latestMCPServer := &mcpgatewayv1alpha1.MCPServer{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(mcpServer), latestMCPServer); err != nil {
		log.Error(err, "Failed to re-fetch MCPServer before status update")
		return ctrl.Result{}, err
	}

	// Update status with current AWS status
	if err := r.StatusManager.UpdateTargetStatus(ctx, latestMCPServer, string(output.Status), statusReasons); err != nil {
		log.Error(err, "Failed to update target status")
		// If it's a conflict error, requeue to retry
		if apierrors.IsConflict(err) {
			log.V(1).Info("Conflict updating status, will retry")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	// Check if target is ready
	if output.Status == "READY" {
		log.Info("Gateway target is ready", "targetId", latestMCPServer.Status.TargetID)

		// Re-fetch again before setting ready condition
		if err := r.Get(ctx, client.ObjectKeyFromObject(mcpServer), latestMCPServer); err != nil {
			log.Error(err, "Failed to re-fetch MCPServer before setting ready condition")
			return ctrl.Result{}, err
		}

		if err := r.StatusManager.SetReady(ctx, latestMCPServer); err != nil {
			log.Error(err, "Failed to set ready condition")
			// If it's a conflict error, requeue to retry
			if apierrors.IsConflict(err) {
				log.V(1).Info("Conflict setting ready condition, will retry")
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// If not ready, log status and requeue
	log.Info("Gateway target not ready yet", "targetId", latestMCPServer.Status.TargetID, "status", output.Status, "reasons", statusReasons)
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}
