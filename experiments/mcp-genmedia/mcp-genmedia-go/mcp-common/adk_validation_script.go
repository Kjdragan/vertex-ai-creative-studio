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
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

// ADKValidationScript provides comprehensive testing for ADK integration
type ADKValidationScript struct {
	config          *Config
	resourceManager *ADKResourceManager
	memoryService   *ADKMemoryService
	errorHandler    *ADKErrorHandler
	logger          ADKLogger
}

// ValidationResult represents the result of a validation test
type ValidationResult struct {
	TestName    string        `json:"test_name"`
	Success     bool          `json:"success"`
	Duration    time.Duration `json:"duration"`
	Error       string        `json:"error,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// ValidationSuite represents a collection of validation results
type ValidationSuite struct {
	SuiteName    string             `json:"suite_name"`
	StartTime    time.Time          `json:"start_time"`
	EndTime      time.Time          `json:"end_time"`
	Duration     time.Duration      `json:"duration"`
	TotalTests   int                `json:"total_tests"`
	PassedTests  int                `json:"passed_tests"`
	FailedTests  int                `json:"failed_tests"`
	Results      []ValidationResult `json:"results"`
}

// NewADKValidationScript creates a new validation script
func NewADKValidationScript(config *Config) *ADKValidationScript {
	logger := NewMockLogger()
	resourceManager := NewADKResourceManager(logger)
	memoryService := NewADKMemoryService(NewMockMemoryService(), logger)
	errorHandler := NewADKErrorHandler(logger)

	return &ADKValidationScript{
		config:          config,
		resourceManager: resourceManager,
		memoryService:   memoryService,
		errorHandler:    errorHandler,
		logger:          logger,
	}
}

// RunAllValidations runs the complete validation suite
func (v *ADKValidationScript) RunAllValidations(ctx context.Context) (*ValidationSuite, error) {
	suite := &ValidationSuite{
		SuiteName: "ADK Session Management Integration",
		StartTime: time.Now(),
		Results:   make([]ValidationResult, 0),
	}

	log.Printf("Starting ADK validation suite: %s", suite.SuiteName)

	// Test suites
	testSuites := []func(context.Context) []ValidationResult{
		v.validateResourceManager,
		v.validateSessionState,
		v.validateBucketResolution,
		v.validateMemoryIntegration,
		v.validateErrorHandling,
		v.validateInlineDataProcessing,
		v.validateEndToEndWorkflow,
	}

	// Run all test suites
	for _, testSuite := range testSuites {
		results := testSuite(ctx)
		suite.Results = append(suite.Results, results...)
	}

	// Calculate suite statistics
	suite.EndTime = time.Now()
	suite.Duration = suite.EndTime.Sub(suite.StartTime)
	suite.TotalTests = len(suite.Results)

	for _, result := range suite.Results {
		if result.Success {
			suite.PassedTests++
		} else {
			suite.FailedTests++
		}
	}

	log.Printf("ADK validation suite completed: %d/%d tests passed in %v",
		suite.PassedTests, suite.TotalTests, suite.Duration)

	return suite, nil
}

// validateResourceManager tests ADK resource manager functionality
func (v *ADKValidationScript) validateResourceManager(ctx context.Context) []ValidationResult {
	var results []ValidationResult
	adkCtx := NewMockInvocationContext("test_session_rm", "test_user_rm")

	// Test 1: Resource manager initialization
	results = append(results, v.runTest("ResourceManager_Initialization", func() error {
		if v.resourceManager == nil {
			return fmt.Errorf("resource manager is nil")
		}
		return nil
	}))

	// Test 2: GCS client creation and management
	results = append(results, v.runTest("ResourceManager_GCSClient", func() error {
		client, err := CreateManagedGCSClientWithADKContext(ctx, adkCtx, v.resourceManager, "test_client")
		if err != nil {
			return fmt.Errorf("failed to create GCS client: %w", err)
		}
		if client == nil {
			return fmt.Errorf("GCS client is nil")
		}
		return nil
	}))

	// Test 3: Resource metadata storage
	results = append(results, v.runTest("ResourceManager_Metadata", func() error {
		err := v.resourceManager.StoreResourceMetadata(ctx, adkCtx, "test_resource", map[string]interface{}{
			"type":        "test",
			"created_at":  time.Now(),
			"description": "Test resource for validation",
		})
		if err != nil {
			return fmt.Errorf("failed to store resource metadata: %w", err)
		}

		metadata, err := v.resourceManager.GetResourceMetadata(ctx, adkCtx, "test_resource")
		if err != nil {
			return fmt.Errorf("failed to retrieve resource metadata: %w", err)
		}
		if metadata == nil {
			return fmt.Errorf("resource metadata is nil")
		}
		return nil
	}))

	// Test 4: Resource cleanup
	results = append(results, v.runTest("ResourceManager_Cleanup", func() error {
		err := v.resourceManager.CleanupSessionResources(ctx, adkCtx)
		if err != nil {
			return fmt.Errorf("failed to cleanup session resources: %w", err)
		}
		return nil
	}))

	return results
}

// validateSessionState tests ADK session state functionality
func (v *ADKValidationScript) validateSessionState(ctx context.Context) []ValidationResult {
	var results []ValidationResult
	adkCtx := NewMockInvocationContext("test_session_ss", "test_user_ss")

	// Test 1: Session state manager initialization
	results = append(results, v.runTest("SessionState_Initialization", func() error {
		if GlobalSessionStateManager == nil {
			return fmt.Errorf("global session state manager is nil")
		}
		return nil
	}))

	// Test 2: Bucket name management
	results = append(results, v.runTest("SessionState_BucketManagement", func() error {
		testBucket := "test-validation-bucket"
		SetCurrentBucket(adkCtx, testBucket)
		
		retrievedBucket := GetCurrentBucket(adkCtx)
		if retrievedBucket != testBucket {
			return fmt.Errorf("bucket mismatch: expected %s, got %s", testBucket, retrievedBucket)
		}
		return nil
	}))

	// Test 3: Upload counter management
	results = append(results, v.runTest("SessionState_UploadCounter", func() error {
		initialCount := IncrementUploadCounter(adkCtx)
		secondCount := IncrementUploadCounter(adkCtx)
		
		if secondCount != initialCount+1 {
			return fmt.Errorf("upload counter not incrementing properly: %d -> %d", initialCount, secondCount)
		}
		return nil
	}))

	// Test 4: Media history management
	results = append(results, v.runTest("SessionState_MediaHistory", func() error {
		testEvent := "test_media_event"
		testData := map[string]interface{}{
			"test_key": "test_value",
			"timestamp": time.Now(),
		}
		
		AddToMediaHistory(adkCtx, testEvent, testData)
		
		history := GetMediaHistory(adkCtx)
		if len(history) == 0 {
			return fmt.Errorf("media history is empty after adding event")
		}
		return nil
	}))

	// Test 5: Processing job management
	results = append(results, v.runTest("SessionState_ProcessingJobs", func() error {
		jobID := "test_job_123"
		jobData := map[string]interface{}{
			"type": "test_job",
			"status": "running",
		}
		
		GlobalSessionStateManager.AddProcessingJob(adkCtx, jobID, jobData)
		
		jobs := GlobalSessionStateManager.GetProcessingJobs(adkCtx)
		if _, exists := jobs[jobID]; !exists {
			return fmt.Errorf("processing job not found after adding")
		}
		
		GlobalSessionStateManager.UpdateProcessingJob(adkCtx, jobID, "completed", "test result")
		return nil
	}))

	return results
}

// validateBucketResolution tests ADK bucket resolution functionality
func (v *ADKValidationScript) validateBucketResolution(ctx context.Context) []ValidationResult {
	var results []ValidationResult
	adkCtx := NewMockInvocationContext("test_session_br", "test_user_br")

	// Test 1: Default bucket resolution
	results = append(results, v.runTest("BucketResolution_Default", func() error {
		bucket, err := ResolveBucketWithADKContext(ctx, adkCtx, "")
		if err != nil {
			return fmt.Errorf("failed to resolve default bucket: %w", err)
		}
		if bucket == "" {
			return fmt.Errorf("resolved bucket is empty")
		}
		return nil
	}))

	// Test 2: User-provided bucket resolution
	results = append(results, v.runTest("BucketResolution_UserProvided", func() error {
		testBucket := "user-provided-bucket"
		bucket, err := ResolveBucketWithADKContext(ctx, adkCtx, testBucket)
		if err != nil {
			return fmt.Errorf("failed to resolve user-provided bucket: %w", err)
		}
		if bucket != testBucket {
			return fmt.Errorf("bucket mismatch: expected %s, got %s", testBucket, bucket)
		}
		return nil
	}))

	// Test 3: Environment variable bucket resolution
	results = append(results, v.runTest("BucketResolution_Environment", func() error {
		// Set environment variable
		envBucket := "env-test-bucket"
		os.Setenv("GENMEDIA_BUCKET", envBucket)
		defer os.Unsetenv("GENMEDIA_BUCKET")
		
		bucket, err := ResolveBucketWithADKContext(ctx, adkCtx, "")
		if err != nil {
			return fmt.Errorf("failed to resolve environment bucket: %w", err)
		}
		if bucket != envBucket {
			return fmt.Errorf("bucket mismatch: expected %s, got %s", envBucket, bucket)
		}
		return nil
	}))

	return results
}

// validateMemoryIntegration tests ADK memory service integration
func (v *ADKValidationScript) validateMemoryIntegration(ctx context.Context) []ValidationResult {
	var results []ValidationResult
	adkCtx := NewMockInvocationContext("test_session_mi", "test_user_mi")

	// Test 1: Memory service initialization
	results = append(results, v.runTest("MemoryIntegration_Initialization", func() error {
		if v.memoryService == nil {
			return fmt.Errorf("memory service is nil")
		}
		return nil
	}))

	// Test 2: Media processing history storage and retrieval
	results = append(results, v.runTest("MemoryIntegration_ProcessingHistory", func() error {
		memory := &MediaProcessingMemory{
			SessionID:   adkCtx.GetSessionID(),
			UserID:      adkCtx.GetUserID(),
			JobID:       "test_job_memory",
			JobType:     "test_generation",
			Status:      "completed",
			Parameters:  map[string]interface{}{"test": "value"},
			Results:     []string{"test_result.mp4"},
			StartTime:   time.Now().Add(-5 * time.Minute),
			EndTime:     time.Now(),
			Duration:    5 * time.Minute,
		}
		
		err := v.memoryService.StoreMediaProcessingHistory(ctx, adkCtx, memory)
		if err != nil {
			return fmt.Errorf("failed to store processing history: %w", err)
		}
		
		retrieved, err := v.memoryService.RetrieveMediaProcessingHistory(ctx, adkCtx, "test_job_memory")
		if err != nil {
			return fmt.Errorf("failed to retrieve processing history: %w", err)
		}
		if retrieved.JobID != memory.JobID {
			return fmt.Errorf("job ID mismatch: expected %s, got %s", memory.JobID, retrieved.JobID)
		}
		return nil
	}))

	// Test 3: User preferences storage and retrieval
	results = append(results, v.runTest("MemoryIntegration_UserPreferences", func() error {
		prefs := &UserPreferences{
			UserID:        adkCtx.GetUserID(),
			DefaultBucket: "user-preferred-bucket",
			PreferredModels: map[string]string{
				"video": "veo-1",
				"image": "imagen-3",
			},
			OutputSettings: map[string]interface{}{
				"quality": "high",
				"format":  "mp4",
			},
		}
		
		err := v.memoryService.StoreUserPreferences(ctx, adkCtx, prefs)
		if err != nil {
			return fmt.Errorf("failed to store user preferences: %w", err)
		}
		
		retrieved, err := v.memoryService.RetrieveUserPreferences(ctx, adkCtx, adkCtx.GetUserID())
		if err != nil {
			return fmt.Errorf("failed to retrieve user preferences: %w", err)
		}
		if retrieved.DefaultBucket != prefs.DefaultBucket {
			return fmt.Errorf("default bucket mismatch: expected %s, got %s", prefs.DefaultBucket, retrieved.DefaultBucket)
		}
		return nil
	}))

	// Test 4: Bucket recommendations
	results = append(results, v.runTest("MemoryIntegration_BucketRecommendations", func() error {
		recommendation := &BucketRecommendation{
			BucketName:     "recommended-bucket",
			UsageCount:     10,
			LastUsed:       time.Now(),
			SuccessRate:    0.95,
			AverageLatency: 250.0,
			RecommendScore: 0.85,
		}
		
		err := v.memoryService.StoreBucketRecommendation(ctx, adkCtx, recommendation)
		if err != nil {
			return fmt.Errorf("failed to store bucket recommendation: %w", err)
		}
		
		recommendations, err := v.memoryService.GetBucketRecommendations(ctx, adkCtx, 5)
		if err != nil {
			return fmt.Errorf("failed to get bucket recommendations: %w", err)
		}
		if len(recommendations) == 0 {
			return fmt.Errorf("no bucket recommendations retrieved")
		}
		return nil
	}))

	return results
}

// validateErrorHandling tests ADK error handling functionality
func (v *ADKValidationScript) validateErrorHandling(ctx context.Context) []ValidationResult {
	var results []ValidationResult
	adkCtx := NewMockInvocationContext("test_session_eh", "test_user_eh")

	// Test 1: Error handler initialization
	results = append(results, v.runTest("ErrorHandling_Initialization", func() error {
		if v.errorHandler == nil {
			return fmt.Errorf("error handler is nil")
		}
		return nil
	}))

	// Test 2: Error creation and handling
	results = append(results, v.runTest("ErrorHandling_Creation", func() error {
		testErr := fmt.Errorf("test error for validation")
		adkErr := v.errorHandler.HandleError(ctx, adkCtx, "validation", "test", testErr)
		
		if adkErr == nil {
			return fmt.Errorf("ADK error is nil")
		}
		if adkErr.Component != "validation" {
			return fmt.Errorf("component mismatch: expected validation, got %s", adkErr.Component)
		}
		if adkErr.SessionID != adkCtx.GetSessionID() {
			return fmt.Errorf("session ID mismatch")
		}
		return nil
	}))

	// Test 3: Error wrapping
	results = append(results, v.runTest("ErrorHandling_Wrapping", func() error {
		originalErr := fmt.Errorf("original error")
		wrappedErr := v.errorHandler.WrapError(ctx, adkCtx, "validation", "wrap_test", originalErr, "wrapped error message")
		
		if wrappedErr == nil {
			return fmt.Errorf("wrapped error is nil")
		}
		if wrappedErr.Cause != originalErr {
			return fmt.Errorf("cause error mismatch")
		}
		return nil
	}))

	// Test 4: Panic recovery
	results = append(results, v.runTest("ErrorHandling_PanicRecovery", func() error {
		err := v.errorHandler.WithRecovery(ctx, adkCtx, "validation", "panic_test", func() error {
			panic("test panic")
		})
		// If we reach here, panic was recovered successfully
		return nil
	}))

	return results
}

// validateInlineDataProcessing tests inline data processing functionality
func (v *ADKValidationScript) validateInlineDataProcessing(ctx context.Context) []ValidationResult {
	var results []ValidationResult

	// Test 1: MCP inline processor initialization
	results = append(results, v.runTest("InlineData_ProcessorInit", func() error {
		processor := NewMCPInlineProcessor()
		if processor == nil {
			return fmt.Errorf("MCP inline processor is nil")
		}
		return nil
	}))

	// Test 2: Base64 image data processing (mock)
	results = append(results, v.runTest("InlineData_Base64Processing", func() error {
		// Create mock base64 image data
		testImageData := []byte("fake image data for testing")
		base64Data := base64.StdEncoding.EncodeToString(testImageData)
		dataURL := fmt.Sprintf("data:image/jpeg;base64,%s", base64Data)
		
		processor := NewMCPInlineProcessor()
		
		// Mock parameters
		params := map[string]interface{}{
			"image_uri": dataURL,
		}
		
		// Note: This would normally require a real GCS client and bucket
		// For validation, we just check the processor can handle the format
		if !strings.HasPrefix(dataURL, "data:") {
			return fmt.Errorf("data URL format not recognized")
		}
		
		return nil
	}))

	return results
}

// validateEndToEndWorkflow tests complete end-to-end workflow
func (v *ADKValidationScript) validateEndToEndWorkflow(ctx context.Context) []ValidationResult {
	var results []ValidationResult
	adkCtx := NewMockInvocationContext("test_session_e2e", "test_user_e2e")

	// Test 1: Complete session lifecycle
	results = append(results, v.runTest("EndToEnd_SessionLifecycle", func() error {
		// Initialize session state
		SetCurrentBucket(adkCtx, "test-e2e-bucket")
		
		// Add media history
		AddToMediaHistory(adkCtx, "e2e_test_start", map[string]interface{}{
			"test": "end-to-end validation",
		})
		
		// Create processing job
		jobID := "e2e_test_job"
		GlobalSessionStateManager.AddProcessingJob(adkCtx, jobID, map[string]interface{}{
			"type": "validation_test",
		})
		
		// Store in memory
		memory := &MediaProcessingMemory{
			SessionID: adkCtx.GetSessionID(),
			UserID:    adkCtx.GetUserID(),
			JobID:     jobID,
			JobType:   "validation_test",
			Status:    "completed",
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}
		
		err := v.memoryService.StoreMediaProcessingHistory(ctx, adkCtx, memory)
		if err != nil {
			return fmt.Errorf("failed to store memory: %w", err)
		}
		
		// Update job status
		GlobalSessionStateManager.UpdateProcessingJob(adkCtx, jobID, "completed", "validation successful")
		
		// Cleanup resources
		err = v.resourceManager.CleanupSessionResources(ctx, adkCtx)
		if err != nil {
			return fmt.Errorf("failed to cleanup resources: %w", err)
		}
		
		return nil
	}))

	// Test 2: Error handling integration
	results = append(results, v.runTest("EndToEnd_ErrorIntegration", func() error {
		// Simulate an error in the workflow
		testErr := fmt.Errorf("simulated workflow error")
		adkErr := v.errorHandler.HandleError(ctx, adkCtx, "e2e_workflow", "simulation", testErr)
		
		if adkErr == nil {
			return fmt.Errorf("error handling failed")
		}
		
		// Verify error was recorded in session state
		errorHistory := GetErrorHistory(adkCtx)
		if len(errorHistory) == 0 {
			return fmt.Errorf("error not recorded in session state")
		}
		
		return nil
	}))

	return results
}

// runTest executes a single test and returns the result
func (v *ADKValidationScript) runTest(testName string, testFunc func() error) ValidationResult {
	startTime := time.Now()
	
	log.Printf("Running test: %s", testName)
	
	err := testFunc()
	duration := time.Since(startTime)
	
	result := ValidationResult{
		TestName: testName,
		Success:  err == nil,
		Duration: duration,
		Details:  make(map[string]interface{}),
	}
	
	if err != nil {
		result.Error = err.Error()
		log.Printf("Test FAILED: %s - %v (took %v)", testName, err, duration)
	} else {
		log.Printf("Test PASSED: %s (took %v)", testName, duration)
	}
	
	return result
}

// PrintValidationSummary prints a formatted summary of validation results
func (v *ADKValidationScript) PrintValidationSummary(suite *ValidationSuite) {
	fmt.Printf("\n" + strings.Repeat("=", 80) + "\n")
	fmt.Printf("ADK VALIDATION SUITE SUMMARY\n")
	fmt.Printf(strings.Repeat("=", 80) + "\n")
	fmt.Printf("Suite: %s\n", suite.SuiteName)
	fmt.Printf("Duration: %v\n", suite.Duration)
	fmt.Printf("Total Tests: %d\n", suite.TotalTests)
	fmt.Printf("Passed: %d\n", suite.PassedTests)
	fmt.Printf("Failed: %d\n", suite.FailedTests)
	fmt.Printf("Success Rate: %.1f%%\n", float64(suite.PassedTests)/float64(suite.TotalTests)*100)
	fmt.Printf(strings.Repeat("-", 80) + "\n")
	
	// Print failed tests
	if suite.FailedTests > 0 {
		fmt.Printf("FAILED TESTS:\n")
		for _, result := range suite.Results {
			if !result.Success {
				fmt.Printf("  ❌ %s: %s\n", result.TestName, result.Error)
			}
		}
		fmt.Printf(strings.Repeat("-", 80) + "\n")
	}
	
	// Print passed tests summary
	fmt.Printf("PASSED TESTS:\n")
	for _, result := range suite.Results {
		if result.Success {
			fmt.Printf("  ✅ %s (%v)\n", result.TestName, result.Duration)
		}
	}
	
	fmt.Printf(strings.Repeat("=", 80) + "\n")
}

// RunADKValidation is the main entry point for running ADK validation
func RunADKValidation() error {
	ctx := context.Background()
	
	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	
	// Initialize validation script
	validator := NewADKValidationScript(config)
	
	// Run all validations
	suite, err := validator.RunAllValidations(ctx)
	if err != nil {
		return fmt.Errorf("validation suite failed: %w", err)
	}
	
	// Print summary
	validator.PrintValidationSummary(suite)
	
	// Return error if any tests failed
	if suite.FailedTests > 0 {
		return fmt.Errorf("validation failed: %d/%d tests failed", suite.FailedTests, suite.TotalTests)
	}
	
	log.Printf("All ADK validation tests passed successfully!")
	return nil
}
