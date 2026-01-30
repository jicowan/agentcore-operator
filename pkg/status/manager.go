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

package status

import (
	"context"

	mcpgatewayv1alpha1 "github.com/aws/mcp-gateway-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Manager manages MCPServer status updates.
type Manager struct {
	client client.Client
}

// NewManager creates a new StatusManager.
func NewManager(client client.Client) *Manager {
	return &Manager{
		client: client,
	}
}

// UpdateTargetCreated updates the MCPServer status after a gateway target is created.
// It sets the TargetID, GatewayArn, TargetStatus fields and updates the LastSynchronized timestamp.
func (m *Manager) UpdateTargetCreated(ctx context.Context, mcpServer *mcpgatewayv1alpha1.MCPServer, targetID, gatewayArn, targetStatus string) error {
	mcpServer.Status.ObservedGeneration = mcpServer.Generation
	mcpServer.Status.TargetID = targetID
	mcpServer.Status.GatewayArn = gatewayArn
	mcpServer.Status.TargetStatus = targetStatus
	now := metav1.Now()
	mcpServer.Status.LastSynchronized = &now

	return m.client.Status().Update(ctx, mcpServer)
}

// UpdateTargetStatus updates the MCPServer status with the current gateway target status.
// It sets the TargetStatus and StatusReasons fields and updates the LastSynchronized timestamp.
func (m *Manager) UpdateTargetStatus(ctx context.Context, mcpServer *mcpgatewayv1alpha1.MCPServer, targetStatus string, statusReasons []string) error {
	mcpServer.Status.ObservedGeneration = mcpServer.Generation
	mcpServer.Status.TargetStatus = targetStatus
	mcpServer.Status.StatusReasons = statusReasons
	now := metav1.Now()
	mcpServer.Status.LastSynchronized = &now

	return m.client.Status().Update(ctx, mcpServer)
}

// UpdateCondition adds or updates a condition in the MCPServer status.
// It uses meta.SetStatusCondition to handle the condition update logic.
func (m *Manager) UpdateCondition(ctx context.Context, mcpServer *mcpgatewayv1alpha1.MCPServer, condition metav1.Condition) error {
	meta.SetStatusCondition(&mcpServer.Status.Conditions, condition)
	return m.client.Status().Update(ctx, mcpServer)
}

// SetReady sets the Ready condition to True, indicating the gateway target is ready.
func (m *Manager) SetReady(ctx context.Context, mcpServer *mcpgatewayv1alpha1.MCPServer) error {
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "GatewayTargetReady",
		Message:            "Gateway target is ready and accepting requests",
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: mcpServer.Generation,
	}
	return m.UpdateCondition(ctx, mcpServer, condition)
}

// SetError sets the Ready condition to False with the provided reason and message.
// This is used to indicate validation errors, AWS API errors, or other failures.
func (m *Manager) SetError(ctx context.Context, mcpServer *mcpgatewayv1alpha1.MCPServer, reason, message string) error {
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: mcpServer.Generation,
	}
	return m.UpdateCondition(ctx, mcpServer, condition)
}
