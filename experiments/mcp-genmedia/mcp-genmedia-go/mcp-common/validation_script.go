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

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/storage"
	common "github.com/GoogleCloudPlatform/vertex-ai-creative-studio/experiments/mcp-genmedia/mcp-genmedia-go/mcp-common"
)

// ValidationResult represents the result of a validation test
type ValidationResult struct {
	TestName string
	Passed   bool
	Error    error
	Duration time.Duration
}

// ValidationSuite runs comprehensive validation tests for the media handling improvements
func main() {
	ctx := context.Background()
	
	log.Printf("Starting Media Handling Validation Suite...")
	log.Printf("Time: %s", time.Now().Format("2006-01-02 15:04:05"))
	
	results := make([]ValidationResult, 0)
	
	// Test 1: Bucket Resolution
	results = append(results, testBucketResolution(ctx))
	
	// Test 2: Bucket Validation
	results = append(results, testBucketValidation(ctx))
	
	// Test 3: User Image Upload Processing
	results = append(results, testUserImageUpload(ctx))
	
	// Test 4: Inline Data Processing
	results = append(results, testInlineDataProcessing(ctx))
	
	// Test 5: Session Management
	results = append(results, testSessionManagement(ctx))
	
	// Test 6: MCP Image Parameter Processing
	results = append(results, testMCPImageParameterProcessing(ctx))
	
	// Print results summary
	printValidationSummary(results)
}

func testBucketResolution(ctx context.Context) ValidationResult {
	start := time.Now()
	testName := "Bucket Resolution"
	
	log.Printf("Running test: %s", testName)
	
	client, err := storage.NewClient(ctx)
	if err != nil {
		return ValidationResult{testName, false, fmt.Errorf("failed to create GCS client: %w", err), time.Since(start)}
	}
	defer client.Close()
	
	// Test with empty user bucket (should fall back to environment or default)
	result := common.ResolveBucket(ctx, client, "")
	if result == nil {
		return ValidationResult{testName, false, fmt.Errorf("bucket resolution returned nil"), time.Since(start)}
	}
	
	log.Printf("Bucket resolved to: %s (source: %s, valid: %t)", result.BucketName, result.Source, result.IsValid)
	
	// Test with invalid bucket
	invalidResult := common.ResolveBucket(ctx, client, "invalid-bucket-name-12345")
	if invalidResult == nil {
		return ValidationResult{testName, false, fmt.Errorf("invalid bucket resolution returned nil"), time.Since(start)}
	}
	
	if invalidResult.IsValid {
		return ValidationResult{testName, false, fmt.Errorf("invalid bucket was marked as valid"), time.Since(start)}
	}
	
	log.Printf("Test %s: PASSED", testName)
	return ValidationResult{testName, true, nil, time.Since(start)}
}

func testBucketValidation(ctx context.Context) ValidationResult {
	start := time.Now()
	testName := "Bucket Validation"
	
	log.Printf("Running test: %s", testName)
	
	client, err := storage.NewClient(ctx)
	if err != nil {
		return ValidationResult{testName, false, fmt.Errorf("failed to create GCS client: %w", err), time.Since(start)}
	}
	defer client.Close()
	
	// Test bucket existence check
	exists := common.BucketExists(ctx, client, "")
	if exists {
		return ValidationResult{testName, false, fmt.Errorf("empty bucket name should not exist"), time.Since(start)}
	}
	
	// Test with retry
	existsWithRetry := common.BucketExistsWithRetry(ctx, client, "invalid-bucket-12345", 2)
	if existsWithRetry {
		return ValidationResult{testName, false, fmt.Errorf("invalid bucket should not exist with retry"), time.Since(start)}
	}
	
	log.Printf("Test %s: PASSED", testName)
	return ValidationResult{testName, true, nil, time.Since(start)}
}

func testUserImageUpload(ctx context.Context) ValidationResult {
	start := time.Now()
	testName := "User Image Upload Processing"
	
	log.Printf("Running test: %s", testName)
	
	// Create a small test image (1x1 PNG)
	testImageData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 dimensions
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0x00, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, // Image data
		0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82, // IEND chunk
	}
	
	// Test base64 encoding
	base64Data := base64.StdEncoding.EncodeToString(testImageData)
	
	// Test processing inline image data
	upload, err := common.ProcessInlineImageData(base64Data)
	if err != nil {
		return ValidationResult{testName, false, fmt.Errorf("failed to process inline image: %w", err), time.Since(start)}
	}
	
	if upload.MIMEType != "image/png" {
		return ValidationResult{testName, false, fmt.Errorf("expected PNG MIME type, got %s", upload.MIMEType), time.Since(start)}
	}
	
	// Test validation
	if err := common.ValidateImageUpload(upload); err != nil {
		return ValidationResult{testName, false, fmt.Errorf("image validation failed: %w", err), time.Since(start)}
	}
	
	log.Printf("Test %s: PASSED", testName)
	return ValidationResult{testName, true, nil, time.Since(start)}
}

func testInlineDataProcessing(ctx context.Context) ValidationResult {
	start := time.Now()
	testName := "Inline Data Processing"
	
	log.Printf("Running test: %s", testName)
	
	// Test data URL format
	dataURL := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="
	
	upload, err := common.ProcessInlineImageData(dataURL)
	if err != nil {
		return ValidationResult{testName, false, fmt.Errorf("failed to process data URL: %w", err), time.Since(start)}
	}
	
	if upload.MIMEType != "image/png" {
		return ValidationResult{testName, false, fmt.Errorf("expected PNG MIME type from data URL, got %s", upload.MIMEType), time.Since(start)}
	}
	
	// Test invalid data
	_, err = common.ProcessInlineImageData("invalid-data")
	if err == nil {
		return ValidationResult{testName, false, fmt.Errorf("invalid data should have failed"), time.Since(start)}
	}
	
	log.Printf("Test %s: PASSED", testName)
	return ValidationResult{testName, true, nil, time.Since(start)}
}

func testSessionManagement(ctx context.Context) ValidationResult {
	start := time.Now()
	testName := "Session Management"
	
	log.Printf("Running test: %s", testName)
	
	// Create session manager with short timeout for testing
	sm := common.NewSessionManager(1 * time.Second)
	defer sm.Shutdown()
	
	// Register a test resource
	cleanupCalled := false
	cleanupFunc := func() error {
		cleanupCalled = true
		return nil
	}
	
	err := sm.RegisterResource("test-resource", "test", "test-data", cleanupFunc)
	if err != nil {
		return ValidationResult{testName, false, fmt.Errorf("failed to register resource: %w", err), time.Since(start)}
	}
	
	// Get resource
	resource, exists := sm.GetResource("test-resource")
	if !exists {
		return ValidationResult{testName, false, fmt.Errorf("resource not found after registration"), time.Since(start)}
	}
	
	if resource.Type != "test" {
		return ValidationResult{testName, false, fmt.Errorf("unexpected resource type: %s", resource.Type), time.Since(start)}
	}
	
	// Test cleanup
	err = sm.UnregisterResource("test-resource")
	if err != nil {
		return ValidationResult{testName, false, fmt.Errorf("failed to unregister resource: %w", err), time.Since(start)}
	}
	
	if !cleanupCalled {
		return ValidationResult{testName, false, fmt.Errorf("cleanup function was not called"), time.Since(start)}
	}
	
	// Test stats
	stats := sm.GetStats()
	if stats["total_resources"].(int) != 0 {
		return ValidationResult{testName, false, fmt.Errorf("expected 0 resources after cleanup, got %d", stats["total_resources"]), time.Since(start)}
	}
	
	log.Printf("Test %s: PASSED", testName)
	return ValidationResult{testName, true, nil, time.Since(start)}
}

func testMCPImageParameterProcessing(ctx context.Context) ValidationResult {
	start := time.Now()
	testName := "MCP Image Parameter Processing"
	
	log.Printf("Running test: %s", testName)
	
	client, err := storage.NewClient(ctx)
	if err != nil {
		return ValidationResult{testName, false, fmt.Errorf("failed to create GCS client: %w", err), time.Since(start)}
	}
	defer client.Close()
	
	// Test GCS URI parameter
	gcsURI := "gs://test-bucket/test-image.png"
	result := common.ProcessMCPImageParameter(ctx, client, gcsURI, "test-bucket")
	
	if result.Error != nil {
		// This is expected since the bucket doesn't exist, but we should get a proper error
		log.Printf("Expected error for non-existent bucket: %v", result.Error)
	}
	
	if result.IsInlineData {
		return ValidationResult{testName, false, fmt.Errorf("GCS URI should not be detected as inline data"), time.Since(start)}
	}
	
	// Test invalid parameter
	invalidResult := common.ProcessMCPImageParameter(ctx, client, 123, "test-bucket")
	if invalidResult.Error == nil {
		return ValidationResult{testName, false, fmt.Errorf("invalid parameter type should have failed"), time.Since(start)}
	}
	
	log.Printf("Test %s: PASSED", testName)
	return ValidationResult{testName, true, nil, time.Since(start)}
}

func printValidationSummary(results []ValidationResult) {
	log.Printf("\n" + "="*60)
	log.Printf("VALIDATION SUMMARY")
	log.Printf("="*60)
	
	passed := 0
	failed := 0
	totalDuration := time.Duration(0)
	
	for _, result := range results {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
			failed++
		} else {
			passed++
		}
		
		totalDuration += result.Duration
		
		log.Printf("%-40s %s (%v)", result.TestName, status, result.Duration)
		if result.Error != nil {
			log.Printf("  Error: %v", result.Error)
		}
	}
	
	log.Printf("-"*60)
	log.Printf("Total Tests: %d", len(results))
	log.Printf("Passed: %d", passed)
	log.Printf("Failed: %d", failed)
	log.Printf("Total Duration: %v", totalDuration)
	log.Printf("Success Rate: %.1f%%", float64(passed)/float64(len(results))*100)
	
	if failed == 0 {
		log.Printf("\n🎉 ALL TESTS PASSED! Media handling improvements are working correctly.")
	} else {
		log.Printf("\n⚠️  %d test(s) failed. Please review the errors above.", failed)
	}
	
	log.Printf("="*60)
	
	// Environment information
	log.Printf("\nEnvironment Information:")
	log.Printf("GENMEDIA_BUCKET: %s", os.Getenv("GENMEDIA_BUCKET"))
	log.Printf("GOOGLE_CLOUD_PROJECT: %s", os.Getenv("GOOGLE_CLOUD_PROJECT"))
	log.Printf("GOOGLE_APPLICATION_CREDENTIALS: %s", os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
}
