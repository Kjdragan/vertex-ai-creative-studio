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
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

// ADKBucketResolver provides ADK-aware bucket resolution with memory integration
type ADKBucketResolver struct {
	resourceManager *ADKResourceManager
	stateManager    *SessionStateManager
}

// NewADKBucketResolver creates a new ADK-aware bucket resolver
func NewADKBucketResolver(resourceManager *ADKResourceManager, stateManager *SessionStateManager) *ADKBucketResolver {
	return &ADKBucketResolver{
		resourceManager: resourceManager,
		stateManager:    stateManager,
	}
}

// ResolveBucketWithADKContext resolves bucket name using ADK session and memory services
func (abr *ADKBucketResolver) ResolveBucketWithADKContext(
	ctx context.Context,
	adkCtx ADKInvocationContext,
	userProvidedBucket string,
) (string, error) {
	logger := adkCtx.GetLogger()
	
	// Step 1: Check user-provided bucket (highest priority)
	if strings.TrimSpace(userProvidedBucket) != "" {
		logger.Debug("ADKBucketResolver: using user-provided bucket",
			"bucket", userProvidedBucket,
			"sessionID", adkCtx.GetSessionID(),
		)
		
		// Validate and store in session state
		if err := abr.validateAndStoreBucket(ctx, adkCtx, userProvidedBucket); err != nil {
			return "", fmt.Errorf("user-provided bucket validation failed: %w", err)
		}
		
		return userProvidedBucket, nil
	}
	
	// Step 2: Check session state for previously resolved bucket
	if bucket, exists := abr.stateManager.GetBucketFromSession(adkCtx); exists {
		logger.Debug("ADKBucketResolver: using bucket from session state",
			"bucket", bucket,
			"sessionID", adkCtx.GetSessionID(),
		)
		return bucket, nil
	}
	
	// Step 3: Check ADK memory for user bucket preferences
	bucket, err := abr.getBucketFromMemory(adkCtx)
	if err == nil && bucket != "" {
		logger.Info("ADKBucketResolver: using bucket from memory",
			"bucket", bucket,
			"sessionID", adkCtx.GetSessionID(),
		)
		
		// Store in session state for future use
		abr.stateManager.StoreBucketInSession(adkCtx, bucket)
		return bucket, nil
	}
	
	// Step 4: Check user preferences in session state
	if bucket := abr.getBucketFromUserPreferences(adkCtx); bucket != "" {
		logger.Info("ADKBucketResolver: using bucket from user preferences",
			"bucket", bucket,
			"sessionID", adkCtx.GetSessionID(),
		)
		
		// Validate and store
		if err := abr.validateAndStoreBucket(ctx, adkCtx, bucket); err != nil {
			logger.Error("ADKBucketResolver: user preference bucket validation failed",
				"bucket", bucket,
				"error", err,
				"sessionID", adkCtx.GetSessionID(),
			)
		} else {
			return bucket, nil
		}
	}
	
	// Step 5: Fall back to environment variable
	if envBucket := os.Getenv("GENMEDIA_BUCKET"); envBucket != "" {
		logger.Info("ADKBucketResolver: using bucket from environment",
			"bucket", envBucket,
			"sessionID", adkCtx.GetSessionID(),
		)
		
		// Validate and store
		if err := abr.validateAndStoreBucket(ctx, adkCtx, envBucket); err != nil {
			logger.Error("ADKBucketResolver: environment bucket validation failed",
				"bucket", envBucket,
				"error", err,
				"sessionID", adkCtx.GetSessionID(),
			)
		} else {
			return envBucket, nil
		}
	}
	
	// Step 6: Use default bucket as last resort
	defaultBucket := "supple-synapse-media"
	logger.Info("ADKBucketResolver: using default bucket",
		"bucket", defaultBucket,
		"sessionID", adkCtx.GetSessionID(),
	)
	
	// Validate and store default bucket
	if err := abr.validateAndStoreBucket(ctx, adkCtx, defaultBucket); err != nil {
		return "", fmt.Errorf("default bucket validation failed: %w", err)
	}
	
	return defaultBucket, nil
}

// validateAndStoreBucket validates bucket access and stores in session state
func (abr *ADKBucketResolver) validateAndStoreBucket(
	ctx context.Context,
	adkCtx ADKInvocationContext,
	bucketName string,
) error {
	logger := adkCtx.GetLogger()
	
	// Create GCS client for validation
	clientID := fmt.Sprintf("bucket_validation_%s", adkCtx.GetSessionID())
	client, err := CreateManagedGCSClientWithADKContext(ctx, adkCtx, abr.resourceManager, clientID)
	if err != nil {
		return fmt.Errorf("failed to create GCS client for validation: %w", err)
	}
	
	// Validate bucket access
	bucket := client.Bucket(bucketName)
	
	// Check if bucket exists and is accessible
	_, err = bucket.Attrs(ctx)
	if err != nil {
		logger.Error("ADKBucketResolver: bucket validation failed",
			"bucket", bucketName,
			"error", err,
			"sessionID", adkCtx.GetSessionID(),
		)
		
		// Record validation failure in session state
		abr.stateManager.RecordError(adkCtx, "bucket_validation", err)
		return fmt.Errorf("bucket '%s' validation failed: %w", bucketName, err)
	}
	
	// Store validated bucket in session state
	abr.stateManager.StoreBucketInSession(adkCtx, bucketName)
	
	// Store bucket validation success in memory
	validationDetails := map[string]interface{}{
		"bucket_name":     bucketName,
		"validation_time": time.Now(),
		"client_id":       clientID,
		"status":          "success",
	}
	
	err = abr.resourceManager.StoreResourceMetadataInMemory(
		adkCtx,
		fmt.Sprintf("bucket_validation_%s", bucketName),
		"bucket_validation",
		validationDetails,
	)
	if err != nil {
		logger.Error("ADKBucketResolver: failed to store validation in memory",
			"bucket", bucketName,
			"error", err,
			"sessionID", adkCtx.GetSessionID(),
		)
	}
	
	// Add to media history
	abr.stateManager.AddToMediaHistory(adkCtx, "bucket_validation", map[string]interface{}{
		"bucket_name": bucketName,
		"status":      "success",
	})
	
	logger.Info("ADKBucketResolver: bucket validation successful",
		"bucket", bucketName,
		"sessionID", adkCtx.GetSessionID(),
	)
	
	return nil
}

// getBucketFromMemory retrieves bucket preference from ADK memory service
func (abr *ADKBucketResolver) getBucketFromMemory(adkCtx ADKInvocationContext) (string, error) {
	memoryService := adkCtx.GetMemoryService()
	logger := adkCtx.GetLogger()
	
	// Search for user bucket preferences in memory
	results, err := memoryService.SearchMemory("user bucket preference")
	if err != nil {
		logger.Debug("ADKBucketResolver: memory search failed",
			"error", err,
			"sessionID", adkCtx.GetSessionID(),
		)
		return "", err
	}
	
	if len(results) == 0 {
		return "", nil
	}
	
	// Extract bucket from most recent memory entry
	for i := len(results) - 1; i >= 0; i-- {
		if entry, ok := results[i].(map[string]interface{}); ok {
			if bucket, exists := entry["bucket_name"]; exists {
				if bucketStr, ok := bucket.(string); ok && bucketStr != "" {
					logger.Debug("ADKBucketResolver: found bucket in memory",
						"bucket", bucketStr,
						"sessionID", adkCtx.GetSessionID(),
					)
					return bucketStr, nil
				}
			}
		}
	}
	
	return "", nil
}

// getBucketFromUserPreferences retrieves bucket from user preferences in session state
func (abr *ADKBucketResolver) getBucketFromUserPreferences(adkCtx ADKInvocationContext) string {
	preferences := abr.stateManager.GetUserPreferences(adkCtx)
	
	// Check for last used bucket
	if lastBucket, exists := preferences["last_used_bucket"]; exists {
		if bucketStr, ok := lastBucket.(string); ok && bucketStr != "" {
			return bucketStr
		}
	}
	
	// Check for default bucket preference
	if defaultBucket, exists := preferences["default_bucket"]; exists {
		if bucketStr, ok := defaultBucket.(string); ok && bucketStr != "" {
			return bucketStr
		}
	}
	
	return ""
}

// StoreBucketPreferenceInMemory stores user bucket preference in ADK memory
func (abr *ADKBucketResolver) StoreBucketPreferenceInMemory(
	adkCtx ADKInvocationContext,
	bucketName string,
) error {
	memoryService := adkCtx.GetMemoryService()
	logger := adkCtx.GetLogger()
	
	// Create memory entry for bucket preference
	preferenceEntry := map[string]interface{}{
		"bucket_name":     bucketName,
		"preference_type": "user_bucket_preference",
		"timestamp":       time.Now(),
		"session_id":      adkCtx.GetSessionID(),
		"user_id":         adkCtx.GetUserID(),
	}
	
	err := memoryService.AddToMemory("user_bucket_preference", preferenceEntry)
	if err != nil {
		logger.Error("ADKBucketResolver: failed to store bucket preference in memory",
			"bucket", bucketName,
			"error", err,
			"sessionID", adkCtx.GetSessionID(),
		)
		return err
	}
	
	// Also update user preferences in session state
	abr.stateManager.SetUserPreference(adkCtx, "default_bucket", bucketName)
	abr.stateManager.SetUserPreference(adkCtx, "bucket_preference_updated", time.Now())
	
	logger.Info("ADKBucketResolver: stored bucket preference in memory",
		"bucket", bucketName,
		"sessionID", adkCtx.GetSessionID(),
	)
	
	return nil
}

// GetBucketValidationHistory retrieves bucket validation history from memory
func (abr *ADKBucketResolver) GetBucketValidationHistory(
	adkCtx ADKInvocationContext,
) ([]interface{}, error) {
	return abr.resourceManager.GetResourceUsageFromMemory(adkCtx, "bucket_validation")
}

// RecommendBucketFromHistory recommends a bucket based on usage history
func (abr *ADKBucketResolver) RecommendBucketFromHistory(
	adkCtx ADKInvocationContext,
) (string, error) {
	logger := adkCtx.GetLogger()
	
	// Get bucket validation history
	history, err := abr.GetBucketValidationHistory(adkCtx)
	if err != nil {
		return "", err
	}
	
	if len(history) == 0 {
		return "", nil
	}
	
	// Count bucket usage frequency
	bucketCounts := make(map[string]int)
	for _, entry := range history {
		if entryMap, ok := entry.(map[string]interface{}); ok {
			if metadata, exists := entryMap["metadata"]; exists {
				if metadataMap, ok := metadata.(map[string]interface{}); ok {
					if bucketName, exists := metadataMap["bucket_name"]; exists {
						if bucketStr, ok := bucketName.(string); ok {
							bucketCounts[bucketStr]++
						}
					}
				}
			}
		}
	}
	
	// Find most frequently used bucket
	var recommendedBucket string
	maxCount := 0
	for bucket, count := range bucketCounts {
		if count > maxCount {
			maxCount = count
			recommendedBucket = bucket
		}
	}
	
	if recommendedBucket != "" {
		logger.Info("ADKBucketResolver: recommended bucket from history",
			"bucket", recommendedBucket,
			"usageCount", maxCount,
			"sessionID", adkCtx.GetSessionID(),
		)
	}
	
	return recommendedBucket, nil
}

// Global ADK bucket resolver instance
var GlobalADKBucketResolver *ADKBucketResolver

// InitializeGlobalADKBucketResolver initializes the global resolver
func InitializeGlobalADKBucketResolver(resourceManager *ADKResourceManager) {
	GlobalADKBucketResolver = NewADKBucketResolver(resourceManager, GlobalSessionStateManager)
}

// Convenience functions using global resolver

// ResolveBucketWithADKContext resolves bucket using global resolver
func ResolveBucketWithADKContext(
	ctx context.Context,
	adkCtx ADKInvocationContext,
	userProvidedBucket string,
) (string, error) {
	if GlobalADKBucketResolver == nil {
		return "", fmt.Errorf("global ADK bucket resolver not initialized")
	}
	return GlobalADKBucketResolver.ResolveBucketWithADKContext(ctx, adkCtx, userProvidedBucket)
}

// StoreBucketPreferenceInMemory stores bucket preference using global resolver
func StoreBucketPreferenceInMemory(adkCtx ADKInvocationContext, bucketName string) error {
	if GlobalADKBucketResolver == nil {
		return fmt.Errorf("global ADK bucket resolver not initialized")
	}
	return GlobalADKBucketResolver.StoreBucketPreferenceInMemory(adkCtx, bucketName)
}
