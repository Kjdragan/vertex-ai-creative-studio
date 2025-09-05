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
	"fmt"
	"time"
)

// SessionStateKeys defines standard keys for media handling session state
type SessionStateKeys struct {
	// Media processing keys
	BucketName       string
	UploadCounter    string
	ProcessingJobs   string
	LastUploadURI    string
	MediaHistory     string
	
	// User preference keys
	UserPreferences  string
	DefaultBucket    string
	PreferredRegion  string
	
	// Error tracking keys
	ErrorHistory     string
	RetryCount       string
	LastError        string
	
	// Resource tracking keys
	ActiveResources  string
	ResourceMetrics  string
}

// NewSessionStateKeys creates standard session state keys
func NewSessionStateKeys() *SessionStateKeys {
	return &SessionStateKeys{
		// Media processing keys
		BucketName:      "media:bucket_name",
		UploadCounter:   "media:upload_counter",
		ProcessingJobs:  "media:processing_jobs",
		LastUploadURI:   "media:last_upload_uri",
		MediaHistory:    "media:history",
		
		// User preference keys
		UserPreferences: "user:preferences",
		DefaultBucket:   "user:default_bucket",
		PreferredRegion: "user:preferred_region",
		
		// Error tracking keys
		ErrorHistory:    "error:history",
		RetryCount:      "error:retry_count",
		LastError:       "error:last_error",
		
		// Resource tracking keys
		ActiveResources: "resource:active",
		ResourceMetrics: "resource:metrics",
	}
}

// Global instance for convenience
var StateKeys = NewSessionStateKeys()

// SessionStateManager provides high-level session state operations
type SessionStateManager struct {
	keys *SessionStateKeys
}

// NewSessionStateManager creates a new session state manager
func NewSessionStateManager() *SessionStateManager {
	return &SessionStateManager{
		keys: NewSessionStateKeys(),
	}
}

// Bucket Management Functions

// StoreBucketInSession stores the resolved bucket name in session state
func (ssm *SessionStateManager) StoreBucketInSession(ctx ADKInvocationContext, bucketName string) {
	sessionState := ctx.GetSessionState()
	sessionState.Set(ssm.keys.BucketName, bucketName)
	
	// Also store in user preferences for future sessions
	preferences := ssm.GetUserPreferences(ctx)
	preferences["last_used_bucket"] = bucketName
	preferences["bucket_updated"] = time.Now()
	ssm.SetUserPreferences(ctx, preferences)
	
	ctx.GetLogger().Debug("SessionStateManager: stored bucket in session",
		"bucketName", bucketName,
		"sessionID", ctx.GetSessionID(),
	)
}

// GetBucketFromSession retrieves the bucket name from session state
func (ssm *SessionStateManager) GetBucketFromSession(ctx ADKInvocationContext) (string, bool) {
	sessionState := ctx.GetSessionState()
	value, exists := sessionState.Get(ssm.keys.BucketName)
	if !exists {
		return "", false
	}
	
	bucketName, ok := value.(string)
	if !ok {
		ctx.GetLogger().Error("SessionStateManager: invalid bucket name type in session state",
			"value", value,
			"sessionID", ctx.GetSessionID(),
		)
		return "", false
	}
	
	return bucketName, true
}

// Upload Counter Management

// IncrementUploadCounter increments and returns the upload counter
func (ssm *SessionStateManager) IncrementUploadCounter(ctx ADKInvocationContext) int {
	sessionState := ctx.GetSessionState()
	
	currentValue, exists := sessionState.Get(ssm.keys.UploadCounter)
	counter := 0
	if exists {
		if c, ok := currentValue.(int); ok {
			counter = c
		}
	}
	
	counter++
	sessionState.Set(ssm.keys.UploadCounter, counter)
	
	ctx.GetLogger().Debug("SessionStateManager: incremented upload counter",
		"counter", counter,
		"sessionID", ctx.GetSessionID(),
	)
	
	return counter
}

// GetUploadCounter returns the current upload counter
func (ssm *SessionStateManager) GetUploadCounter(ctx ADKInvocationContext) int {
	sessionState := ctx.GetSessionState()
	value, exists := sessionState.Get(ssm.keys.UploadCounter)
	if !exists {
		return 0
	}
	
	if counter, ok := value.(int); ok {
		return counter
	}
	return 0
}

// Media History Management

// AddToMediaHistory adds a media operation to the session history
func (ssm *SessionStateManager) AddToMediaHistory(ctx ADKInvocationContext, operation string, details map[string]interface{}) {
	sessionState := ctx.GetSessionState()
	
	// Get existing history
	history := ssm.getMediaHistoryList(ctx)
	
	// Create new entry
	entry := map[string]interface{}{
		"operation":  operation,
		"details":    details,
		"timestamp":  time.Now(),
		"session_id": ctx.GetSessionID(),
		"user_id":    ctx.GetUserID(),
	}
	
	// Add to history (keep last 50 entries)
	history = append(history, entry)
	if len(history) > 50 {
		history = history[len(history)-50:]
	}
	
	sessionState.Set(ssm.keys.MediaHistory, history)
	
	ctx.GetLogger().Debug("SessionStateManager: added to media history",
		"operation", operation,
		"historySize", len(history),
		"sessionID", ctx.GetSessionID(),
	)
}

// GetMediaHistory returns the media operation history
func (ssm *SessionStateManager) GetMediaHistory(ctx ADKInvocationContext) []map[string]interface{} {
	return ssm.getMediaHistoryList(ctx)
}

// getMediaHistoryList internal helper to get history as slice
func (ssm *SessionStateManager) getMediaHistoryList(ctx ADKInvocationContext) []map[string]interface{} {
	sessionState := ctx.GetSessionState()
	value, exists := sessionState.Get(ssm.keys.MediaHistory)
	if !exists {
		return []map[string]interface{}{}
	}
	
	if history, ok := value.([]map[string]interface{}); ok {
		return history
	}
	
	// Try to convert from []interface{}
	if historyInterface, ok := value.([]interface{}); ok {
		history := make([]map[string]interface{}, 0, len(historyInterface))
		for _, item := range historyInterface {
			if entry, ok := item.(map[string]interface{}); ok {
				history = append(history, entry)
			}
		}
		return history
	}
	
	return []map[string]interface{}{}
}

// User Preferences Management

// GetUserPreferences returns user preferences from session state
func (ssm *SessionStateManager) GetUserPreferences(ctx ADKInvocationContext) map[string]interface{} {
	sessionState := ctx.GetSessionState()
	value, exists := sessionState.Get(ssm.keys.UserPreferences)
	if !exists {
		return make(map[string]interface{})
	}
	
	if prefs, ok := value.(map[string]interface{}); ok {
		return prefs
	}
	
	return make(map[string]interface{})
}

// SetUserPreferences stores user preferences in session state
func (ssm *SessionStateManager) SetUserPreferences(ctx ADKInvocationContext, preferences map[string]interface{}) {
	sessionState := ctx.GetSessionState()
	sessionState.Set(ssm.keys.UserPreferences, preferences)
	
	ctx.GetLogger().Debug("SessionStateManager: updated user preferences",
		"preferencesCount", len(preferences),
		"sessionID", ctx.GetSessionID(),
	)
}

// SetUserPreference sets a single user preference
func (ssm *SessionStateManager) SetUserPreference(ctx ADKInvocationContext, key string, value interface{}) {
	preferences := ssm.GetUserPreferences(ctx)
	preferences[key] = value
	ssm.SetUserPreferences(ctx, preferences)
}

// GetUserPreference gets a single user preference
func (ssm *SessionStateManager) GetUserPreference(ctx ADKInvocationContext, key string) (interface{}, bool) {
	preferences := ssm.GetUserPreferences(ctx)
	value, exists := preferences[key]
	return value, exists
}

// Error Tracking Management

// RecordError records an error in session state
func (ssm *SessionStateManager) RecordError(ctx ADKInvocationContext, operation string, err error) {
	sessionState := ctx.GetSessionState()
	
	// Get existing error history
	history := ssm.getErrorHistoryList(ctx)
	
	// Create error entry
	errorEntry := map[string]interface{}{
		"operation":  operation,
		"error":      err.Error(),
		"timestamp":  time.Now(),
		"session_id": ctx.GetSessionID(),
		"user_id":    ctx.GetUserID(),
	}
	
	// Add to history (keep last 20 errors)
	history = append(history, errorEntry)
	if len(history) > 20 {
		history = history[len(history)-20:]
	}
	
	sessionState.Set(ssm.keys.ErrorHistory, history)
	sessionState.Set(ssm.keys.LastError, errorEntry)
	
	// Increment retry count for this operation
	retryKey := fmt.Sprintf("%s:retry_count", operation)
	currentRetries := ssm.getRetryCount(ctx, retryKey)
	sessionState.Set(retryKey, currentRetries+1)
	
	ctx.GetLogger().Error("SessionStateManager: recorded error",
		"operation", operation,
		"error", err.Error(),
		"retryCount", currentRetries+1,
		"sessionID", ctx.GetSessionID(),
	)
}

// GetErrorHistory returns the error history
func (ssm *SessionStateManager) GetErrorHistory(ctx ADKInvocationContext) []map[string]interface{} {
	return ssm.getErrorHistoryList(ctx)
}

// GetRetryCount returns the retry count for an operation
func (ssm *SessionStateManager) GetRetryCount(ctx ADKInvocationContext, operation string) int {
	retryKey := fmt.Sprintf("%s:retry_count", operation)
	return ssm.getRetryCount(ctx, retryKey)
}

// ResetRetryCount resets the retry count for an operation
func (ssm *SessionStateManager) ResetRetryCount(ctx ADKInvocationContext, operation string) {
	sessionState := ctx.GetSessionState()
	retryKey := fmt.Sprintf("%s:retry_count", operation)
	sessionState.Set(retryKey, 0)
}

// getErrorHistoryList internal helper to get error history
func (ssm *SessionStateManager) getErrorHistoryList(ctx ADKInvocationContext) []map[string]interface{} {
	sessionState := ctx.GetSessionState()
	value, exists := sessionState.Get(ssm.keys.ErrorHistory)
	if !exists {
		return []map[string]interface{}{}
	}
	
	if history, ok := value.([]map[string]interface{}); ok {
		return history
	}
	
	// Try to convert from []interface{}
	if historyInterface, ok := value.([]interface{}); ok {
		history := make([]map[string]interface{}, 0, len(historyInterface))
		for _, item := range historyInterface {
			if entry, ok := item.(map[string]interface{}); ok {
				history = append(history, entry)
			}
		}
		return history
	}
	
	return []map[string]interface{}{}
}

// getRetryCount internal helper to get retry count
func (ssm *SessionStateManager) getRetryCount(ctx ADKInvocationContext, retryKey string) int {
	sessionState := ctx.GetSessionState()
	value, exists := sessionState.Get(retryKey)
	if !exists {
		return 0
	}
	
	if count, ok := value.(int); ok {
		return count
	}
	return 0
}

// Processing Jobs Management

// AddProcessingJob adds a processing job to the session state
func (ssm *SessionStateManager) AddProcessingJob(ctx ADKInvocationContext, jobID string, jobDetails map[string]interface{}) {
	sessionState := ctx.GetSessionState()
	
	jobs := ssm.getProcessingJobs(ctx)
	jobs[jobID] = map[string]interface{}{
		"details":    jobDetails,
		"status":     "running",
		"started":    time.Now(),
		"session_id": ctx.GetSessionID(),
		"user_id":    ctx.GetUserID(),
	}
	
	sessionState.Set(ssm.keys.ProcessingJobs, jobs)
	
	ctx.GetLogger().Info("SessionStateManager: added processing job",
		"jobID", jobID,
		"sessionID", ctx.GetSessionID(),
	)
}

// UpdateProcessingJob updates a processing job status
func (ssm *SessionStateManager) UpdateProcessingJob(ctx ADKInvocationContext, jobID string, status string, result interface{}) {
	sessionState := ctx.GetSessionState()
	
	jobs := ssm.getProcessingJobs(ctx)
	if job, exists := jobs[jobID]; exists {
		if jobMap, ok := job.(map[string]interface{}); ok {
			jobMap["status"] = status
			jobMap["completed"] = time.Now()
			if result != nil {
				jobMap["result"] = result
			}
			jobs[jobID] = jobMap
			sessionState.Set(ssm.keys.ProcessingJobs, jobs)
			
			ctx.GetLogger().Info("SessionStateManager: updated processing job",
				"jobID", jobID,
				"status", status,
				"sessionID", ctx.GetSessionID(),
			)
		}
	}
}

// GetProcessingJobs returns all processing jobs
func (ssm *SessionStateManager) GetProcessingJobs(ctx ADKInvocationContext) map[string]interface{} {
	return ssm.getProcessingJobs(ctx)
}

// getProcessingJobs internal helper to get processing jobs
func (ssm *SessionStateManager) getProcessingJobs(ctx ADKInvocationContext) map[string]interface{} {
	sessionState := ctx.GetSessionState()
	value, exists := sessionState.Get(ssm.keys.ProcessingJobs)
	if !exists {
		return make(map[string]interface{})
	}
	
	if jobs, ok := value.(map[string]interface{}); ok {
		return jobs
	}
	
	return make(map[string]interface{})
}

// Session Statistics

// GetSessionStatistics returns comprehensive session statistics
func (ssm *SessionStateManager) GetSessionStatistics(ctx ADKInvocationContext) map[string]interface{} {
	sessionState := ctx.GetSessionState()
	
	stats := map[string]interface{}{
		"session_id":      ctx.GetSessionID(),
		"user_id":         ctx.GetUserID(),
		"upload_count":    ssm.GetUploadCounter(ctx),
		"media_history":   len(ssm.GetMediaHistory(ctx)),
		"error_count":     len(ssm.GetErrorHistory(ctx)),
		"processing_jobs": len(ssm.GetProcessingJobs(ctx)),
		"state_keys":      len(sessionState.Keys()),
		"timestamp":       time.Now(),
	}
	
	// Add bucket info if available
	if bucket, exists := ssm.GetBucketFromSession(ctx); exists {
		stats["current_bucket"] = bucket
	}
	
	// Add user preferences count
	stats["user_preferences"] = len(ssm.GetUserPreferences(ctx))
	
	return stats
}

// CleanupExpiredData removes old data from session state
func (ssm *SessionStateManager) CleanupExpiredData(ctx ADKInvocationContext, maxAge time.Duration) {
	now := time.Now()
	
	// Clean up old media history entries
	history := ssm.GetMediaHistory(ctx)
	filteredHistory := []map[string]interface{}{}
	for _, entry := range history {
		if timestamp, ok := entry["timestamp"].(time.Time); ok {
			if now.Sub(timestamp) <= maxAge {
				filteredHistory = append(filteredHistory, entry)
			}
		}
	}
	if len(filteredHistory) != len(history) {
		sessionState := ctx.GetSessionState()
		sessionState.Set(ssm.keys.MediaHistory, filteredHistory)
		ctx.GetLogger().Info("SessionStateManager: cleaned up expired media history",
			"removed", len(history)-len(filteredHistory),
			"remaining", len(filteredHistory),
			"sessionID", ctx.GetSessionID(),
		)
	}
	
	// Clean up old error history entries
	errorHistory := ssm.GetErrorHistory(ctx)
	filteredErrors := []map[string]interface{}{}
	for _, entry := range errorHistory {
		if timestamp, ok := entry["timestamp"].(time.Time); ok {
			if now.Sub(timestamp) <= maxAge {
				filteredErrors = append(filteredErrors, entry)
			}
		}
	}
	if len(filteredErrors) != len(errorHistory) {
		sessionState := ctx.GetSessionState()
		sessionState.Set(ssm.keys.ErrorHistory, filteredErrors)
		ctx.GetLogger().Info("SessionStateManager: cleaned up expired error history",
			"removed", len(errorHistory)-len(filteredErrors),
			"remaining", len(filteredErrors),
			"sessionID", ctx.GetSessionID(),
		)
	}
}

// Global session state manager instance
var GlobalSessionStateManager = NewSessionStateManager()

// Convenience functions using global manager

// StoreBucketInSession stores bucket name using global manager
func StoreBucketInSession(ctx ADKInvocationContext, bucketName string) {
	GlobalSessionStateManager.StoreBucketInSession(ctx, bucketName)
}

// GetBucketFromSession gets bucket name using global manager
func GetBucketFromSession(ctx ADKInvocationContext) (string, bool) {
	return GlobalSessionStateManager.GetBucketFromSession(ctx)
}

// IncrementUploadCounter increments upload counter using global manager
func IncrementUploadCounter(ctx ADKInvocationContext) int {
	return GlobalSessionStateManager.IncrementUploadCounter(ctx)
}

// AddToMediaHistory adds to media history using global manager
func AddToMediaHistory(ctx ADKInvocationContext, operation string, details map[string]interface{}) {
	GlobalSessionStateManager.AddToMediaHistory(ctx, operation, details)
}

// RecordError records error using global manager
func RecordError(ctx ADKInvocationContext, operation string, err error) {
	GlobalSessionStateManager.RecordError(ctx, operation, err)
}
