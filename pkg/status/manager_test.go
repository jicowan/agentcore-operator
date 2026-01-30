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
	"testing"

	mcpgatewayv1alpha1 "github.com/aws/mcp-gateway-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewManager(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, mcpgatewayv1alpha1.AddToScheme(scheme))

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	manager := NewManager(fakeClient)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.client)
}

func TestUpdateTargetCreated(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, mcpgatewayv1alpha1.AddToScheme(scheme))

	mcpServer := &mcpgatewayv1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-server",
			Namespace: "default",
		},
		Spec: mcpgatewayv1alpha1.MCPServerSpec{
			Endpoint:     "https://example.com",
			Capabilities: []string{"tools"},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(mcpServer).
		WithStatusSubresource(mcpServer).
		Build()

	manager := NewManager(fakeClient)
	ctx := context.Background()

	err := manager.UpdateTargetCreated(ctx, mcpServer, "target-123", "arn:aws:bedrock:us-east-1:123456789012:gateway/gw-123", "CREATING")
	require.NoError(t, err)

	// Verify the status was updated
	updated := &mcpgatewayv1alpha1.MCPServer{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-server", Namespace: "default"}, updated)
	require.NoError(t, err)

	assert.Equal(t, "target-123", updated.Status.TargetID)
	assert.Equal(t, "arn:aws:bedrock:us-east-1:123456789012:gateway/gw-123", updated.Status.GatewayArn)
	assert.Equal(t, "CREATING", updated.Status.TargetStatus)
	assert.NotNil(t, updated.Status.LastSynchronized)
}

func TestUpdateTargetStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, mcpgatewayv1alpha1.AddToScheme(scheme))

	mcpServer := &mcpgatewayv1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-server",
			Namespace: "default",
		},
		Spec: mcpgatewayv1alpha1.MCPServerSpec{
			Endpoint:     "https://example.com",
			Capabilities: []string{"tools"},
		},
		Status: mcpgatewayv1alpha1.MCPServerStatus{
			TargetID:   "target-123",
			GatewayArn: "arn:aws:bedrock:us-east-1:123456789012:gateway/gw-123",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(mcpServer).
		WithStatusSubresource(mcpServer).
		Build()

	manager := NewManager(fakeClient)
	ctx := context.Background()

	statusReasons := []string{"Waiting for DNS propagation"}
	err := manager.UpdateTargetStatus(ctx, mcpServer, "READY", statusReasons)
	require.NoError(t, err)

	// Verify the status was updated
	updated := &mcpgatewayv1alpha1.MCPServer{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-server", Namespace: "default"}, updated)
	require.NoError(t, err)

	assert.Equal(t, "READY", updated.Status.TargetStatus)
	assert.Equal(t, statusReasons, updated.Status.StatusReasons)
	assert.NotNil(t, updated.Status.LastSynchronized)
}

func TestUpdateCondition(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, mcpgatewayv1alpha1.AddToScheme(scheme))

	mcpServer := &mcpgatewayv1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-server",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: mcpgatewayv1alpha1.MCPServerSpec{
			Endpoint:     "https://example.com",
			Capabilities: []string{"tools"},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(mcpServer).
		WithStatusSubresource(mcpServer).
		Build()

	manager := NewManager(fakeClient)
	ctx := context.Background()

	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "TestReason",
		Message:            "Test message",
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: 1,
	}

	err := manager.UpdateCondition(ctx, mcpServer, condition)
	require.NoError(t, err)

	// Verify the condition was added
	updated := &mcpgatewayv1alpha1.MCPServer{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-server", Namespace: "default"}, updated)
	require.NoError(t, err)

	require.Len(t, updated.Status.Conditions, 1)
	assert.Equal(t, "Ready", updated.Status.Conditions[0].Type)
	assert.Equal(t, metav1.ConditionTrue, updated.Status.Conditions[0].Status)
	assert.Equal(t, "TestReason", updated.Status.Conditions[0].Reason)
	assert.Equal(t, "Test message", updated.Status.Conditions[0].Message)
}

func TestUpdateCondition_UpdatesExisting(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, mcpgatewayv1alpha1.AddToScheme(scheme))

	mcpServer := &mcpgatewayv1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-server",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: mcpgatewayv1alpha1.MCPServerSpec{
			Endpoint:     "https://example.com",
			Capabilities: []string{"tools"},
		},
		Status: mcpgatewayv1alpha1.MCPServerStatus{
			Conditions: []metav1.Condition{
				{
					Type:               "Ready",
					Status:             metav1.ConditionFalse,
					Reason:             "OldReason",
					Message:            "Old message",
					LastTransitionTime: metav1.Now(),
					ObservedGeneration: 1,
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(mcpServer).
		WithStatusSubresource(mcpServer).
		Build()

	manager := NewManager(fakeClient)
	ctx := context.Background()

	newCondition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "NewReason",
		Message:            "New message",
		LastTransitionTime: metav1.Now(),
		ObservedGeneration: 1,
	}

	err := manager.UpdateCondition(ctx, mcpServer, newCondition)
	require.NoError(t, err)

	// Verify the condition was updated
	updated := &mcpgatewayv1alpha1.MCPServer{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-server", Namespace: "default"}, updated)
	require.NoError(t, err)

	require.Len(t, updated.Status.Conditions, 1)
	assert.Equal(t, "Ready", updated.Status.Conditions[0].Type)
	assert.Equal(t, metav1.ConditionTrue, updated.Status.Conditions[0].Status)
	assert.Equal(t, "NewReason", updated.Status.Conditions[0].Reason)
	assert.Equal(t, "New message", updated.Status.Conditions[0].Message)
}

func TestSetReady(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, mcpgatewayv1alpha1.AddToScheme(scheme))

	mcpServer := &mcpgatewayv1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-server",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: mcpgatewayv1alpha1.MCPServerSpec{
			Endpoint:     "https://example.com",
			Capabilities: []string{"tools"},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(mcpServer).
		WithStatusSubresource(mcpServer).
		Build()

	manager := NewManager(fakeClient)
	ctx := context.Background()

	err := manager.SetReady(ctx, mcpServer)
	require.NoError(t, err)

	// Verify the Ready condition was set
	updated := &mcpgatewayv1alpha1.MCPServer{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-server", Namespace: "default"}, updated)
	require.NoError(t, err)

	require.Len(t, updated.Status.Conditions, 1)
	assert.Equal(t, "Ready", updated.Status.Conditions[0].Type)
	assert.Equal(t, metav1.ConditionTrue, updated.Status.Conditions[0].Status)
	assert.Equal(t, "GatewayTargetReady", updated.Status.Conditions[0].Reason)
	assert.Equal(t, "Gateway target is ready and accepting requests", updated.Status.Conditions[0].Message)
	assert.Equal(t, int64(1), updated.Status.Conditions[0].ObservedGeneration)
}

func TestSetError(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, mcpgatewayv1alpha1.AddToScheme(scheme))

	mcpServer := &mcpgatewayv1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-server",
			Namespace:  "default",
			Generation: 2,
		},
		Spec: mcpgatewayv1alpha1.MCPServerSpec{
			Endpoint:     "https://example.com",
			Capabilities: []string{"tools"},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(mcpServer).
		WithStatusSubresource(mcpServer).
		Build()

	manager := NewManager(fakeClient)
	ctx := context.Background()

	err := manager.SetError(ctx, mcpServer, "ValidationError", "Endpoint must match pattern https://.*")
	require.NoError(t, err)

	// Verify the error condition was set
	updated := &mcpgatewayv1alpha1.MCPServer{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-server", Namespace: "default"}, updated)
	require.NoError(t, err)

	require.Len(t, updated.Status.Conditions, 1)
	assert.Equal(t, "Ready", updated.Status.Conditions[0].Type)
	assert.Equal(t, metav1.ConditionFalse, updated.Status.Conditions[0].Status)
	assert.Equal(t, "ValidationError", updated.Status.Conditions[0].Reason)
	assert.Equal(t, "Endpoint must match pattern https://.*", updated.Status.Conditions[0].Message)
	assert.Equal(t, int64(2), updated.Status.Conditions[0].ObservedGeneration)
}

func TestSetError_MultipleReasons(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, mcpgatewayv1alpha1.AddToScheme(scheme))

	mcpServer := &mcpgatewayv1alpha1.MCPServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-server",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: mcpgatewayv1alpha1.MCPServerSpec{
			Endpoint:     "https://example.com",
			Capabilities: []string{"tools"},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(mcpServer).
		WithStatusSubresource(mcpServer).
		Build()

	manager := NewManager(fakeClient)
	ctx := context.Background()

	// Set first error
	err := manager.SetError(ctx, mcpServer, "ValidationError", "Invalid endpoint")
	require.NoError(t, err)

	// Fetch updated resource
	updated := &mcpgatewayv1alpha1.MCPServer{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-server", Namespace: "default"}, updated)
	require.NoError(t, err)

	// Set second error (should update the existing condition)
	err = manager.SetError(ctx, updated, "AWSError", "Failed to create gateway target")
	require.NoError(t, err)

	// Verify the condition was updated
	final := &mcpgatewayv1alpha1.MCPServer{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "test-server", Namespace: "default"}, final)
	require.NoError(t, err)

	require.Len(t, final.Status.Conditions, 1)
	assert.Equal(t, "Ready", final.Status.Conditions[0].Type)
	assert.Equal(t, metav1.ConditionFalse, final.Status.Conditions[0].Status)
	assert.Equal(t, "AWSError", final.Status.Conditions[0].Reason)
	assert.Equal(t, "Failed to create gateway target", final.Status.Conditions[0].Message)
}
