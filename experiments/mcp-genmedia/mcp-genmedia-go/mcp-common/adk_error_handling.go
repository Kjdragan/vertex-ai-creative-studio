// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ADKErrorHandler provides comprehensive error handling with ADK integration
type ADKErrorHandler struct {
	logger ADKLogger
}

// ADKError represents an enhanced error with ADK context
type ADKError struct {
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Cause       error                  `json:"cause,omitempty"`
	SessionID   string                 `json:"session_id"`
	UserID      string                 `json:"user_id"`
	Timestamp   time.Time              `json:"timestamp"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
	Component   string                 `json:"component"`
	Operation   string                 `json:"operation"`
	Recoverable bool                   `json:"recoverable"`
	RetryCount  int                    `json:"retry_count"`
}

// Error implements the error interface
func (e *ADKError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error
func (e *ADKError) Unwrap() error {
	return e.Cause
}

// NewADKErrorHandler creates a new ADK error handler
func NewADKErrorHandler(logger ADKLogger) *ADKErrorHandler {
	return &ADKErrorHandler{
		logger: logger,
	}
}

// HandleError processes and logs errors with ADK context
func (h *ADKErrorHandler) HandleError(ctx context.Context, adkCtx ADKInvocationContext, component, operation string, err error) *ADKError {
	if err == nil {
		return nil
	}

	// Check if it's already an ADKError
	if adkErr, ok := err.(*ADKError); ok {
		adkErr.RetryCount++
		h.logError(adkErr)
		return adkErr
	}

	// Create new ADKError
	adkErr := &ADKError{
		Code:        h.generateErrorCode(component, operation, err),
		Message:     err.Error(),
		Details:     make(map[string]interface{}),
		Cause:       err,
		SessionID:   adkCtx.GetSessionID(),
		UserID:      adkCtx.GetUserID(),
		Timestamp:   time.Now(),
		StackTrace:  h.getStackTrace(),
		Component:   component,
		Operation:   operation,
		Recoverable: h.isRecoverable(err),
		RetryCount:  0,
	}

	// Add context details
	h.enrichErrorDetails(adkErr, ctx, adkCtx)

	// Log the error
	h.logError(adkErr)

	// Record error in session state
	RecordError(adkCtx, fmt.Sprintf("%s_%s", component, operation), adkErr)

	return adkErr
}

// HandleErrorWithDetails processes errors with additional details
func (h *ADKErrorHandler) HandleErrorWithDetails(ctx context.Context, adkCtx ADKInvocationContext, component, operation string, err error, details map[string]interface{}) *ADKError {
	adkErr := h.HandleError(ctx, adkCtx, component, operation, err)
	if adkErr != nil {
		// Merge additional details
		for key, value := range details {
			adkErr.Details[key] = value
		}
	}
	return adkErr
}

// CreateError creates a new ADK error from scratch
func (h *ADKErrorHandler) CreateError(ctx context.Context, adkCtx ADKInvocationContext, component, operation, code, message string, details map[string]interface{}) *ADKError {
	adkErr := &ADKError{
		Code:        code,
		Message:     message,
		Details:     details,
		SessionID:   adkCtx.GetSessionID(),
		UserID:      adkCtx.GetUserID(),
		Timestamp:   time.Now(),
		StackTrace:  h.getStackTrace(),
		Component:   component,
		Operation:   operation,
		Recoverable: true, // Default to recoverable for created errors
		RetryCount:  0,
	}

	if adkErr.Details == nil {
		adkErr.Details = make(map[string]interface{})
	}

	// Add context details
	h.enrichErrorDetails(adkErr, ctx, adkCtx)

	// Log the error
	h.logError(adkErr)

	// Record error in session state
	RecordError(adkCtx, fmt.Sprintf("%s_%s", component, operation), adkErr)

	return adkErr
}

// WrapError wraps an existing error with ADK context
func (h *ADKErrorHandler) WrapError(ctx context.Context, adkCtx ADKInvocationContext, component, operation string, err error, message string) *ADKError {
	if err == nil {
		return nil
	}

	adkErr := &ADKError{
		Code:        h.generateErrorCode(component, operation, err),
		Message:     message,
		Details:     make(map[string]interface{}),
		Cause:       err,
		SessionID:   adkCtx.GetSessionID(),
		UserID:      adkCtx.GetUserID(),
		Timestamp:   time.Now(),
		StackTrace:  h.getStackTrace(),
		Component:   component,
		Operation:   operation,
		Recoverable: h.isRecoverable(err),
		RetryCount:  0,
	}

	// Add context details
	h.enrichErrorDetails(adkErr, ctx, adkCtx)

	// Log the error
	h.logError(adkErr)

	// Record error in session state
	RecordError(adkCtx, fmt.Sprintf("%s_%s", component, operation), adkErr)

	return adkErr
}

// generateErrorCode generates a structured error code
func (h *ADKErrorHandler) generateErrorCode(component, operation string, err error) string {
	errType := "UNKNOWN"
	errMsg := strings.ToUpper(err.Error())

	// Categorize common error types
	switch {
	case strings.Contains(errMsg, "NOT FOUND"):
		errType = "NOT_FOUND"
	case strings.Contains(errMsg, "PERMISSION") || strings.Contains(errMsg, "FORBIDDEN"):
		errType = "PERMISSION_DENIED"
	case strings.Contains(errMsg, "TIMEOUT"):
		errType = "TIMEOUT"
	case strings.Contains(errMsg, "CONNECTION") || strings.Contains(errMsg, "NETWORK"):
		errType = "NETWORK_ERROR"
	case strings.Contains(errMsg, "INVALID") || strings.Contains(errMsg, "BAD"):
		errType = "INVALID_INPUT"
	case strings.Contains(errMsg, "QUOTA") || strings.Contains(errMsg, "LIMIT"):
		errType = "QUOTA_EXCEEDED"
	case strings.Contains(errMsg, "AUTHENTICATION") || strings.Contains(errMsg, "UNAUTHORIZED"):
		errType = "AUTH_ERROR"
	case strings.Contains(errMsg, "STORAGE") || strings.Contains(errMsg, "BUCKET"):
		errType = "STORAGE_ERROR"
	case strings.Contains(errMsg, "GENERATION") || strings.Contains(errMsg, "MODEL"):
		errType = "GENERATION_ERROR"
	}

	return fmt.Sprintf("%s_%s_%s", strings.ToUpper(component), strings.ToUpper(operation), errType)
}

// isRecoverable determines if an error is recoverable
func (h *ADKErrorHandler) isRecoverable(err error) bool {
	errMsg := strings.ToUpper(err.Error())

	// Non-recoverable errors
	nonRecoverable := []string{
		"PERMISSION",
		"FORBIDDEN",
		"UNAUTHORIZED",
		"NOT FOUND",
		"INVALID",
		"BAD REQUEST",
		"MALFORMED",
	}

	for _, pattern := range nonRecoverable {
		if strings.Contains(errMsg, pattern) {
			return false
		}
	}

	// Recoverable errors (typically transient)
	recoverable := []string{
		"TIMEOUT",
		"CONNECTION",
		"NETWORK",
		"QUOTA",
		"LIMIT",
		"BUSY",
		"UNAVAILABLE",
		"INTERNAL",
	}

	for _, pattern := range recoverable {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	// Default to recoverable for unknown errors
	return true
}

// enrichErrorDetails adds contextual information to error details
func (h *ADKErrorHandler) enrichErrorDetails(adkErr *ADKError, ctx context.Context, adkCtx ADKInvocationContext) {
	// Add session statistics
	if stats := GetSessionStatistics(adkCtx); stats != nil {
		adkErr.Details["session_stats"] = stats
	}

	// Add current bucket information
	if bucket := GetCurrentBucket(adkCtx); bucket != "" {
		adkErr.Details["current_bucket"] = bucket
	}

	// Add processing job information
	if jobs := GlobalSessionStateManager.GetProcessingJobs(adkCtx); len(jobs) > 0 {
		adkErr.Details["active_jobs"] = len(jobs)
		adkErr.Details["job_ids"] = func() []string {
			var ids []string
			for id := range jobs {
				ids = append(ids, id)
			}
			return ids
		}()
	}

	// Add error history count
	if errorHistory := GetErrorHistory(adkCtx); len(errorHistory) > 0 {
		adkErr.Details["recent_errors"] = len(errorHistory)
	}

	// Add context deadline information
	if deadline, ok := ctx.Deadline(); ok {
		adkErr.Details["context_deadline"] = deadline
		adkErr.Details["time_remaining"] = time.Until(deadline)
	}

	// Add runtime information
	adkErr.Details["go_version"] = runtime.Version()
	adkErr.Details["num_goroutines"] = runtime.NumGoroutine()
}

// getStackTrace captures the current stack trace
func (h *ADKErrorHandler) getStackTrace() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// logError logs the error with appropriate level and context
func (h *ADKErrorHandler) logError(adkErr *ADKError) {
	logFields := map[string]interface{}{
		"error_code":   adkErr.Code,
		"component":    adkErr.Component,
		"operation":    adkErr.Operation,
		"session_id":   adkErr.SessionID,
		"user_id":      adkErr.UserID,
		"recoverable":  adkErr.Recoverable,
		"retry_count":  adkErr.RetryCount,
		"timestamp":    adkErr.Timestamp,
	}

	// Add selected details to log
	if len(adkErr.Details) > 0 {
		logFields["details"] = adkErr.Details
	}

	if adkErr.Recoverable {
		h.logger.Warn("ADK error occurred (recoverable)",
			"message", adkErr.Message,
			"fields", logFields,
		)
	} else {
		h.logger.Error("ADK error occurred (non-recoverable)",
			"message", adkErr.Message,
			"fields", logFields,
		)
	}

	// Log stack trace for debugging if error is not recoverable
	if !adkErr.Recoverable && adkErr.StackTrace != "" {
		h.logger.Debug("ADK error stack trace",
			"error_code", adkErr.Code,
			"stack_trace", adkErr.StackTrace,
		)
	}
}

// RecoveryHandler handles panic recovery with ADK context
func (h *ADKErrorHandler) RecoveryHandler(ctx context.Context, adkCtx ADKInvocationContext, component, operation string) {
	if r := recover(); r != nil {
		var err error
		if e, ok := r.(error); ok {
			err = e
		} else {
			err = fmt.Errorf("panic: %v", r)
		}

		adkErr := h.HandleErrorWithDetails(ctx, adkCtx, component, operation, err, map[string]interface{}{
			"panic_value": r,
			"recovered":   true,
		})

		h.logger.Error("Panic recovered in ADK operation",
			"component", component,
			"operation", operation,
			"error_code", adkErr.Code,
			"session_id", adkCtx.GetSessionID(),
			"panic_value", r,
			"stack_trace", adkErr.StackTrace,
		)
	}
}

// WithRecovery wraps a function with panic recovery
func (h *ADKErrorHandler) WithRecovery(ctx context.Context, adkCtx ADKInvocationContext, component, operation string, fn func() error) error {
	defer h.RecoveryHandler(ctx, adkCtx, component, operation)
	return fn()
}

// Global ADK error handler instance
var GlobalADKErrorHandler *ADKErrorHandler

// InitializeGlobalADKErrorHandler initializes the global ADK error handler
func InitializeGlobalADKErrorHandler(logger ADKLogger) {
	GlobalADKErrorHandler = NewADKErrorHandler(logger)
}

// Convenience functions for global error handling

// HandleErrorGlobal handles errors using the global error handler
func HandleErrorGlobal(ctx context.Context, adkCtx ADKInvocationContext, component, operation string, err error) *ADKError {
	if GlobalADKErrorHandler == nil {
		// Fallback error handling if global handler not initialized
		adkErr := &ADKError{
			Code:        fmt.Sprintf("%s_%s_ERROR", strings.ToUpper(component), strings.ToUpper(operation)),
			Message:     err.Error(),
			Cause:       err,
			SessionID:   adkCtx.GetSessionID(),
			UserID:      adkCtx.GetUserID(),
			Timestamp:   time.Now(),
			Component:   component,
			Operation:   operation,
			Recoverable: true,
			RetryCount:  0,
		}
		RecordError(adkCtx, fmt.Sprintf("%s_%s", component, operation), adkErr)
		return adkErr
	}
	return GlobalADKErrorHandler.HandleError(ctx, adkCtx, component, operation, err)
}

// CreateErrorGlobal creates errors using the global error handler
func CreateErrorGlobal(ctx context.Context, adkCtx ADKInvocationContext, component, operation, code, message string, details map[string]interface{}) *ADKError {
	if GlobalADKErrorHandler == nil {
		adkErr := &ADKError{
			Code:        code,
			Message:     message,
			Details:     details,
			SessionID:   adkCtx.GetSessionID(),
			UserID:      adkCtx.GetUserID(),
			Timestamp:   time.Now(),
			Component:   component,
			Operation:   operation,
			Recoverable: true,
			RetryCount:  0,
		}
		RecordError(adkCtx, fmt.Sprintf("%s_%s", component, operation), adkErr)
		return adkErr
	}
	return GlobalADKErrorHandler.CreateError(ctx, adkCtx, component, operation, code, message, details)
}

// WithRecoveryGlobal wraps functions with panic recovery using global handler
func WithRecoveryGlobal(ctx context.Context, adkCtx ADKInvocationContext, component, operation string, fn func() error) error {
	if GlobalADKErrorHandler == nil {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("panic: %v", r)
				RecordError(adkCtx, fmt.Sprintf("%s_%s", component, operation), err)
			}
		}()
		return fn()
	}
	return GlobalADKErrorHandler.WithRecovery(ctx, adkCtx, component, operation, fn)
}
