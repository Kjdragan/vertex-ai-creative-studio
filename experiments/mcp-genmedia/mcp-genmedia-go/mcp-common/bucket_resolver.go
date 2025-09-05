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
	"log"
	"os"
	"strings"

	"cloud.google.com/go/storage"
)

const (
	// DefaultBucketName is the fallback bucket name when no other options are available
	DefaultBucketName = "supple-synapse-media"
	// GenmediaBucketEnvVar is the environment variable for the primary bucket
	GenmediaBucketEnvVar = "GENMEDIA_BUCKET"
)

// BucketResolutionResult contains the result of bucket resolution
type BucketResolutionResult struct {
	BucketName string
	Source     string // "user_provided", "environment", "default"
	IsValid    bool
	Error      error
}

// ResolveBucket implements smart bucket resolution with the following priority:
// 1. User-provided bucket (if specified and valid)
// 2. Environment variable GENMEDIA_BUCKET (if set and valid)
// 3. Default bucket (supple-synapse-media)
// Returns the resolved bucket name and metadata about the resolution process.
func ResolveBucket(ctx context.Context, client *storage.Client, userBucket string) *BucketResolutionResult {
	log.Printf("ResolveBucket: starting resolution with user bucket: '%s'", userBucket)
	
	// Priority 1: User-provided bucket
	if userBucket != "" {
		cleanBucket := cleanBucketName(userBucket)
		log.Printf("ResolveBucket: checking user-provided bucket: '%s'", cleanBucket)
		
		if BucketExists(ctx, client, cleanBucket) {
			log.Printf("ResolveBucket: user-provided bucket '%s' is valid", cleanBucket)
			return &BucketResolutionResult{
				BucketName: cleanBucket,
				Source:     "user_provided",
				IsValid:    true,
				Error:      nil,
			}
		}
		
		log.Printf("ResolveBucket: user-provided bucket '%s' is not accessible", cleanBucket)
		return &BucketResolutionResult{
			BucketName: cleanBucket,
			Source:     "user_provided",
			IsValid:    false,
			Error:      fmt.Errorf("user-provided bucket '%s' does not exist or is not accessible", cleanBucket),
		}
	}
	
	// Priority 2: Environment variable
	envBucket := os.Getenv(GenmediaBucketEnvVar)
	if envBucket != "" {
		cleanBucket := cleanBucketName(envBucket)
		log.Printf("ResolveBucket: checking environment bucket: '%s'", cleanBucket)
		
		if BucketExists(ctx, client, cleanBucket) {
			log.Printf("ResolveBucket: environment bucket '%s' is valid", cleanBucket)
			return &BucketResolutionResult{
				BucketName: cleanBucket,
				Source:     "environment",
				IsValid:    true,
				Error:      nil,
			}
		}
		
		log.Printf("ResolveBucket: environment bucket '%s' is not accessible, falling back to default", cleanBucket)
	}
	
	// Priority 3: Default bucket
	log.Printf("ResolveBucket: checking default bucket: '%s'", DefaultBucketName)
	
	if BucketExists(ctx, client, DefaultBucketName) {
		log.Printf("ResolveBucket: default bucket '%s' is valid", DefaultBucketName)
		return &BucketResolutionResult{
			BucketName: DefaultBucketName,
			Source:     "default",
			IsValid:    true,
			Error:      nil,
		}
	}
	
	log.Printf("ResolveBucket: default bucket '%s' is not accessible", DefaultBucketName)
	return &BucketResolutionResult{
		BucketName: DefaultBucketName,
		Source:     "default",
		IsValid:    false,
		Error:      fmt.Errorf("no accessible bucket found - tried user: '%s', env: '%s', default: '%s'", 
			userBucket, envBucket, DefaultBucketName),
	}
}

// ResolveBucketWithValidation performs bucket resolution and comprehensive validation.
// This is the recommended function for production use as it ensures the bucket is fully accessible.
func ResolveBucketWithValidation(ctx context.Context, client *storage.Client, userBucket string) *BucketResolutionResult {
	result := ResolveBucket(ctx, client, userBucket)
	
	if !result.IsValid {
		return result
	}
	
	// Perform comprehensive validation
	log.Printf("ResolveBucketWithValidation: performing comprehensive validation for bucket '%s'", result.BucketName)
	
	if err := ValidateBucketAccess(ctx, client, result.BucketName); err != nil {
		log.Printf("ResolveBucketWithValidation: validation failed for bucket '%s': %v", result.BucketName, err)
		result.IsValid = false
		result.Error = fmt.Errorf("bucket '%s' validation failed: %v", result.BucketName, err)
		return result
	}
	
	log.Printf("ResolveBucketWithValidation: bucket '%s' passed comprehensive validation", result.BucketName)
	return result
}

// GetResolvedBucketURI returns a fully qualified GCS URI for the resolved bucket with optional path.
// If subPath is provided, it will be appended to the bucket URI.
func GetResolvedBucketURI(ctx context.Context, client *storage.Client, userBucket, subPath string) (string, error) {
	result := ResolveBucket(ctx, client, userBucket)
	
	if !result.IsValid {
		return "", result.Error
	}
	
	bucketURI := fmt.Sprintf("gs://%s", result.BucketName)
	
	if subPath != "" {
		// Clean and normalize the subPath
		cleanPath := strings.Trim(subPath, "/")
		if cleanPath != "" {
			bucketURI = fmt.Sprintf("%s/%s", bucketURI, cleanPath)
		}
	}
	
	log.Printf("GetResolvedBucketURI: resolved to '%s' from source: %s", bucketURI, result.Source)
	return bucketURI, nil
}

// cleanBucketName removes common prefixes and suffixes to extract the bucket name
func cleanBucketName(bucket string) string {
	// Remove gs:// prefix if present
	if strings.HasPrefix(bucket, "gs://") {
		bucket = strings.TrimPrefix(bucket, "gs://")
	}
	
	// Remove trailing slashes and paths
	if idx := strings.Index(bucket, "/"); idx != -1 {
		bucket = bucket[:idx]
	}
	
	return strings.TrimSpace(bucket)
}

// CreateBucketResolutionSummary creates a human-readable summary of the bucket resolution process.
// This is useful for logging and debugging purposes.
func CreateBucketResolutionSummary(result *BucketResolutionResult, userBucket string) string {
	envBucket := os.Getenv(GenmediaBucketEnvVar)
	
	summary := fmt.Sprintf("Bucket Resolution Summary:\n")
	summary += fmt.Sprintf("  User provided: '%s'\n", userBucket)
	summary += fmt.Sprintf("  Environment (%s): '%s'\n", GenmediaBucketEnvVar, envBucket)
	summary += fmt.Sprintf("  Default: '%s'\n", DefaultBucketName)
	summary += fmt.Sprintf("  Resolved to: '%s' (source: %s)\n", result.BucketName, result.Source)
	summary += fmt.Sprintf("  Valid: %t\n", result.IsValid)
	
	if result.Error != nil {
		summary += fmt.Sprintf("  Error: %v\n", result.Error)
	}
	
	return summary
}
