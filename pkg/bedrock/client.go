/*
Copyright 2025.

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

package bedrock

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentcorecontrol/types"
	"github.com/aws/smithy-go"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
)

const (
	maxRetries        = 3
	initialBackoff    = 1 * time.Second
	maxBackoff        = 30 * time.Second
	backoffMultiplier = 2.0
)

// BedrockClientWrapper wraps the AWS Bedrock AgentCore client with retry logic and error handling
type BedrockClientWrapper struct {
	client *bedrockagentcorecontrol.Client
	logger logr.Logger
}

// NewBedrockClientWrapper creates a new BedrockClientWrapper
func NewBedrockClientWrapper(client *bedrockagentcorecontrol.Client, logger logr.Logger) *BedrockClientWrapper {
	return &BedrockClientWrapper{
		client: client,
		logger: logger,
	}
}

// CreateGatewayTarget creates a new gateway target in AWS Bedrock AgentCore
// It includes retry logic for transient errors and idempotency via client tokens
func (w *BedrockClientWrapper) CreateGatewayTarget(
	ctx context.Context,
	input *bedrockagentcorecontrol.CreateGatewayTargetInput,
) (*bedrockagentcorecontrol.CreateGatewayTargetOutput, error) {
	// Generate unique client token for idempotency if not provided
	if input.ClientToken == nil {
		clientToken := uuid.New().String()
		input.ClientToken = aws.String(clientToken)
		w.logger.V(1).Info("Generated client token for idempotency", "clientToken", clientToken)
	}

	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			w.logger.Info("Retrying CreateGatewayTarget", "attempt", attempt, "backoff", backoff)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff = time.Duration(math.Min(float64(backoff)*backoffMultiplier, float64(maxBackoff)))
		}

		output, err := w.client.CreateGatewayTarget(ctx, input)
		if err == nil {
			w.logger.Info("Successfully created gateway target",
				"targetId", aws.ToString(output.TargetId),
				"status", output.Status)
			return output, nil
		}

		lastErr = err

		// Check if error is retryable
		if !w.isRetryableError(err) {
			w.logger.Error(err, "Non-retryable error creating gateway target")
			return nil, err
		}

		w.logger.Info("Retryable error creating gateway target", "error", err, "attempt", attempt)
	}

	return nil, fmt.Errorf("failed to create gateway target after %d attempts: %w", maxRetries+1, lastErr)
}

// GetGatewayTarget retrieves information about a gateway target
func (w *BedrockClientWrapper) GetGatewayTarget(
	ctx context.Context,
	gatewayID string,
	targetID string,
) (*bedrockagentcorecontrol.GetGatewayTargetOutput, error) {
	input := &bedrockagentcorecontrol.GetGatewayTargetInput{
		GatewayIdentifier: aws.String(gatewayID),
		TargetId:          aws.String(targetID),
	}

	output, err := w.client.GetGatewayTarget(ctx, input)
	if err != nil {
		w.logger.Error(err, "Failed to get gateway target",
			"gatewayId", gatewayID,
			"targetId", targetID)
		return nil, err
	}

	w.logger.V(1).Info("Successfully retrieved gateway target",
		"targetId", targetID,
		"status", output.Status)
	return output, nil
}

// UpdateGatewayTarget updates an existing gateway target
func (w *BedrockClientWrapper) UpdateGatewayTarget(
	ctx context.Context,
	input *bedrockagentcorecontrol.UpdateGatewayTargetInput,
) (*bedrockagentcorecontrol.UpdateGatewayTargetOutput, error) {
	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			w.logger.Info("Retrying UpdateGatewayTarget", "attempt", attempt, "backoff", backoff)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
			backoff = time.Duration(math.Min(float64(backoff)*backoffMultiplier, float64(maxBackoff)))
		}

		output, err := w.client.UpdateGatewayTarget(ctx, input)
		if err == nil {
			w.logger.Info("Successfully updated gateway target",
				"targetId", aws.ToString(input.TargetId),
				"status", output.Status)
			return output, nil
		}

		lastErr = err

		// Check if error is retryable
		if !w.isRetryableError(err) {
			w.logger.Error(err, "Non-retryable error updating gateway target")
			return nil, err
		}

		w.logger.Info("Retryable error updating gateway target", "error", err, "attempt", attempt)
	}

	return nil, fmt.Errorf("failed to update gateway target after %d attempts: %w", maxRetries+1, lastErr)
}

// DeleteGatewayTarget deletes a gateway target
// ResourceNotFoundException is treated as success (idempotent deletion)
func (w *BedrockClientWrapper) DeleteGatewayTarget(
	ctx context.Context,
	gatewayID string,
	targetID string,
) error {
	input := &bedrockagentcorecontrol.DeleteGatewayTargetInput{
		GatewayIdentifier: aws.String(gatewayID),
		TargetId:          aws.String(targetID),
	}

	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			w.logger.Info("Retrying DeleteGatewayTarget", "attempt", attempt, "backoff", backoff)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			backoff = time.Duration(math.Min(float64(backoff)*backoffMultiplier, float64(maxBackoff)))
		}

		_, err := w.client.DeleteGatewayTarget(ctx, input)
		if err == nil {
			w.logger.Info("Successfully deleted gateway target",
				"gatewayId", gatewayID,
				"targetId", targetID)
			return nil
		}

		// ResourceNotFoundException means the target is already deleted - treat as success
		if w.isResourceNotFoundError(err) {
			w.logger.Info("Gateway target not found, treating as successful deletion",
				"gatewayId", gatewayID,
				"targetId", targetID)
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !w.isRetryableError(err) {
			w.logger.Error(err, "Non-retryable error deleting gateway target")
			return err
		}

		w.logger.Info("Retryable error deleting gateway target", "error", err, "attempt", attempt)
	}

	return fmt.Errorf("failed to delete gateway target after %d attempts: %w", maxRetries+1, lastErr)
}

// isRetryableError determines if an error should be retried
func (w *BedrockClientWrapper) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for throttling errors
	if w.isThrottlingError(err) {
		return true
	}

	// Check for internal server errors
	if w.isInternalServerError(err) {
		return true
	}

	// Check for network/timeout errors
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false // Don't retry context errors
	}

	// Check for validation errors (not retryable)
	if w.isValidationError(err) {
		return false
	}

	// Default to not retrying unknown errors
	return false
}

// isThrottlingError checks if the error is a throttling error
func (w *BedrockClientWrapper) isThrottlingError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == "ThrottlingException" ||
			code == "TooManyRequestsException" ||
			code == "RequestLimitExceeded"
	}
	return false
}

// isInternalServerError checks if the error is an internal server error
func (w *BedrockClientWrapper) isInternalServerError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == "InternalServerException" ||
			code == "ServiceUnavailableException" ||
			code == "InternalFailure"
	}
	return false
}

// isValidationError checks if the error is a validation error
func (w *BedrockClientWrapper) isValidationError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == "ValidationException" ||
			code == "InvalidParameterException" ||
			code == "InvalidRequestException"
	}
	return false
}

// isResourceNotFoundError checks if the error is a ResourceNotFoundException
func (w *BedrockClientWrapper) isResourceNotFoundError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "ResourceNotFoundException"
	}

	// Also check for the typed error
	var notFoundErr *types.ResourceNotFoundException
	return errors.As(err, &notFoundErr)
}
