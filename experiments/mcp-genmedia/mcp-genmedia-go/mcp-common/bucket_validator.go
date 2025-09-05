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
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
)

// BucketExists checks if a GCS bucket exists and is accessible.
// Returns true if the bucket exists and is accessible, false otherwise.
func BucketExists(ctx context.Context, client *storage.Client, bucketName string) bool {
	if bucketName == "" {
		log.Printf("BucketExists: empty bucket name provided")
		return false
	}

	bucket := client.Bucket(bucketName)
	
	// Use a timeout context for the operation
	opCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	
	_, err := bucket.Attrs(opCtx)
	if err != nil {
		log.Printf("BucketExists: bucket '%s' not accessible: %v", bucketName, err)
		return false
	}
	
	log.Printf("BucketExists: bucket '%s' is accessible", bucketName)
	return true
}

// ValidateBucketAccess performs comprehensive validation of bucket access.
// It checks existence, read permissions, and write permissions.
// Returns nil if all validations pass, error otherwise.
func ValidateBucketAccess(ctx context.Context, client *storage.Client, bucketName string) error {
	if bucketName == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}

	bucket := client.Bucket(bucketName)
	
	// Use a timeout context for operations
	opCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	
	// Check if bucket exists and get attributes
	attrs, err := bucket.Attrs(opCtx)
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok {
			switch gerr.Code {
			case 404:
				return fmt.Errorf("bucket '%s' does not exist", bucketName)
			case 403:
				return fmt.Errorf("access denied to bucket '%s' - check IAM permissions", bucketName)
			default:
				return fmt.Errorf("failed to access bucket '%s': %v", bucketName, err)
			}
		}
		return fmt.Errorf("failed to access bucket '%s': %v", bucketName, err)
	}
	
	log.Printf("ValidateBucketAccess: bucket '%s' exists in location '%s'", bucketName, attrs.Location)
	
	// Test read permission by attempting to list objects (with limit)
	query := &storage.Query{Prefix: "", MaxResults: 1}
	it := bucket.Objects(opCtx, query)
	_, err = it.Next()
	if err != nil && err.Error() != "no more items in iterator" {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 403 {
			return fmt.Errorf("no read permission for bucket '%s'", bucketName)
		}
		// Other errors are acceptable (empty bucket, etc.)
	}
	
	log.Printf("ValidateBucketAccess: read access confirmed for bucket '%s'", bucketName)
	
	// Test write permission by attempting to create a test object
	testObjectName := fmt.Sprintf("_validation_test_%d", time.Now().Unix())
	testObject := bucket.Object(testObjectName)
	
	writer := testObject.NewWriter(opCtx)
	writer.ContentType = "text/plain"
	
	testData := []byte("validation test")
	if _, err := writer.Write(testData); err != nil {
		writer.Close()
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 403 {
			return fmt.Errorf("no write permission for bucket '%s'", bucketName)
		}
		return fmt.Errorf("write test failed for bucket '%s': %v", bucketName, err)
	}
	
	if err := writer.Close(); err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 403 {
			return fmt.Errorf("no write permission for bucket '%s'", bucketName)
		}
		return fmt.Errorf("write test failed for bucket '%s': %v", bucketName, err)
	}
	
	// Clean up test object
	if err := testObject.Delete(opCtx); err != nil {
		log.Printf("ValidateBucketAccess: warning - failed to clean up test object '%s': %v", testObjectName, err)
	}
	
	log.Printf("ValidateBucketAccess: write access confirmed for bucket '%s'", bucketName)
	return nil
}

// BucketExistsWithRetry checks bucket existence with retry logic for transient failures.
// This is useful when dealing with eventual consistency or temporary network issues.
func BucketExistsWithRetry(ctx context.Context, client *storage.Client, bucketName string, maxRetries int) bool {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if BucketExists(ctx, client, bucketName) {
			return true
		}
		
		if attempt < maxRetries {
			backoffDuration := time.Duration(attempt) * time.Second
			log.Printf("BucketExistsWithRetry: attempt %d/%d failed for bucket '%s', retrying in %v", 
				attempt, maxRetries, bucketName, backoffDuration)
			
			select {
			case <-ctx.Done():
				log.Printf("BucketExistsWithRetry: context cancelled during retry for bucket '%s'", bucketName)
				return false
			case <-time.After(backoffDuration):
				// Continue to next attempt
			}
		}
	}
	
	log.Printf("BucketExistsWithRetry: all %d attempts failed for bucket '%s'", maxRetries, bucketName)
	return false
}
