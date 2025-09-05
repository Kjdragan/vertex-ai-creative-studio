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
	"time"
)

// ADKServiceManager manages all ADK service initialization and lifecycle
type ADKServiceManager struct {
	config          *Config
	logger          ADKLogger
	resourceManager *ADKResourceManager
	memoryService   *ADKMemoryService
	errorHandler    *ADKErrorHandler
	initialized     bool
}

// NewADKServiceManager creates a new ADK service manager
func NewADKServiceManager() (*ADKServiceManager, error) {
	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	logger := NewMockLogger() // In production, this would be the real ADK logger

	return &ADKServiceManager{
		config:      config,
		logger:      logger,
		initialized: false,
	}, nil
}

// Initialize initializes all ADK services
func (m *ADKServiceManager) Initialize(ctx context.Context) error {
	if m.initialized {
		return fmt.Errorf("ADK services already initialized")
	}

	m.logger.Info("Initializing ADK services...")

	// Validate environment
	if err := m.validateEnvironment(); err != nil {
		return fmt.Errorf("environment validation failed: %w", err)
	}

	// Initialize error handler
	m.errorHandler = NewADKErrorHandler(m.logger)
	InitializeGlobalADKErrorHandler(m.logger)
	m.logger.Info("ADK error handler initialized")

	// Initialize resource manager
	m.resourceManager = NewADKResourceManager(m.logger)
	InitializeGlobalADKResourceManager(m.logger)
	m.logger.Info("ADK resource manager initialized")

	// Initialize memory service
	memoryServiceInterface := NewMockMemoryService() // In production, use real ADK memory service
	m.memoryService = NewADKMemoryService(memoryServiceInterface, m.logger)
	InitializeGlobalADKMemoryService(memoryServiceInterface, m.logger)
	m.logger.Info("ADK memory service initialized")

	// Initialize session state manager
	InitializeGlobalSessionStateManager()
	m.logger.Info("ADK session state manager initialized")

	// Initialize Veo handlers
	InitializeGlobalADKVeoHandlers(m.resourceManager, m.config)
	m.logger.Info("ADK Veo handlers initialized")

	m.initialized = true
	m.logger.Info("All ADK services initialized successfully")

	return nil
}

// validateEnvironment validates required environment variables and settings
func (m *ADKServiceManager) validateEnvironment() error {
	requiredEnvVars := []string{
		"GOOGLE_CLOUD_PROJECT",
		"GOOGLE_CLOUD_LOCATION",
	}

	missingVars := []string{}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			missingVars = append(missingVars, envVar)
		}
	}

	if len(missingVars) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	// Validate optional but recommended variables
	if os.Getenv("GENMEDIA_BUCKET") == "" {
		m.logger.Warn("GENMEDIA_BUCKET not set, will use default bucket")
	}

	if os.Getenv("GOOGLE_API_KEY") == "" && os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		m.logger.Warn("Neither GOOGLE_API_KEY nor GOOGLE_APPLICATION_CREDENTIALS set")
	}

	return nil
}

// Shutdown gracefully shuts down all ADK services
func (m *ADKServiceManager) Shutdown(ctx context.Context) error {
	if !m.initialized {
		return nil
	}

	m.logger.Info("Shutting down ADK services...")

	// Create a timeout context for shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Shutdown resource manager (cleanup all resources)
	if m.resourceManager != nil {
		// Create a mock ADK context for cleanup
		mockCtx := NewMockInvocationContext("shutdown_session", "system")
		err := m.resourceManager.CleanupSessionResources(shutdownCtx, mockCtx)
		if err != nil {
			m.logger.Error("Failed to cleanup resources during shutdown", "error", err)
		}
	}

	m.initialized = false
	m.logger.Info("ADK services shutdown completed")

	return nil
}

// GetResourceManager returns the resource manager
func (m *ADKServiceManager) GetResourceManager() *ADKResourceManager {
	return m.resourceManager
}

// GetMemoryService returns the memory service
func (m *ADKServiceManager) GetMemoryService() *ADKMemoryService {
	return m.memoryService
}

// GetErrorHandler returns the error handler
func (m *ADKServiceManager) GetErrorHandler() *ADKErrorHandler {
	return m.errorHandler
}

// GetLogger returns the logger
func (m *ADKServiceManager) GetLogger() ADKLogger {
	return m.logger
}

// IsInitialized returns whether services are initialized
func (m *ADKServiceManager) IsInitialized() bool {
	return m.initialized
}

// HealthCheck performs a health check on all ADK services
func (m *ADKServiceManager) HealthCheck(ctx context.Context) error {
	if !m.initialized {
		return fmt.Errorf("ADK services not initialized")
	}

	// Create a test ADK context
	testCtx := NewMockInvocationContext("health_check", "system")

	// Test resource manager
	if m.resourceManager == nil {
		return fmt.Errorf("resource manager is nil")
	}

	// Test memory service
	if m.memoryService == nil {
		return fmt.Errorf("memory service is nil")
	}

	// Test basic memory operations
	testKey := "health_check_test"
	testValue := map[string]interface{}{
		"timestamp": time.Now(),
		"status":    "healthy",
	}

	err := m.memoryService.memoryService.Store(ctx, testKey, testValue)
	if err != nil {
		return fmt.Errorf("memory service store test failed: %w", err)
	}

	_, err = m.memoryService.memoryService.Retrieve(ctx, testKey)
	if err != nil {
		return fmt.Errorf("memory service retrieve test failed: %w", err)
	}

	// Cleanup test data
	err = m.memoryService.memoryService.Delete(ctx, testKey)
	if err != nil {
		m.logger.Warn("Failed to cleanup health check test data", "error", err)
	}

	// Test session state
	if GlobalSessionStateManager == nil {
		return fmt.Errorf("global session state manager is nil")
	}

	// Test bucket resolution
	_, err = ResolveBucketWithADKContext(ctx, testCtx, "")
	if err != nil {
		return fmt.Errorf("bucket resolution test failed: %w", err)
	}

	m.logger.Info("ADK services health check passed")
	return nil
}

// Global service manager instance
var GlobalADKServiceManager *ADKServiceManager

// InitializeADKServices initializes all ADK services globally
func InitializeADKServices(ctx context.Context) error {
	if GlobalADKServiceManager != nil {
		return fmt.Errorf("ADK services already initialized")
	}

	manager, err := NewADKServiceManager()
	if err != nil {
		return fmt.Errorf("failed to create ADK service manager: %w", err)
	}

	err = manager.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize ADK services: %w", err)
	}

	GlobalADKServiceManager = manager
	log.Printf("Global ADK services initialized successfully")
	return nil
}

// ShutdownADKServices shuts down all ADK services globally
func ShutdownADKServices(ctx context.Context) error {
	if GlobalADKServiceManager == nil {
		return nil
	}

	err := GlobalADKServiceManager.Shutdown(ctx)
	GlobalADKServiceManager = nil
	return err
}

// GetGlobalADKServiceManager returns the global ADK service manager
func GetGlobalADKServiceManager() *ADKServiceManager {
	return GlobalADKServiceManager
}

// ADKHealthCheckHandler provides HTTP health check endpoint
func ADKHealthCheckHandler(ctx context.Context) (map[string]interface{}, error) {
	response := map[string]interface{}{
		"timestamp": time.Now(),
		"service":   "ADK Session Management",
	}

	if GlobalADKServiceManager == nil {
		response["status"] = "error"
		response["message"] = "ADK services not initialized"
		return response, fmt.Errorf("ADK services not initialized")
	}

	err := GlobalADKServiceManager.HealthCheck(ctx)
	if err != nil {
		response["status"] = "error"
		response["message"] = err.Error()
		return response, err
	}

	response["status"] = "healthy"
	response["message"] = "All ADK services operational"
	response["components"] = map[string]string{
		"resource_manager": "healthy",
		"memory_service":   "healthy",
		"error_handler":    "healthy",
		"session_state":    "healthy",
		"veo_handlers":     "healthy",
	}

	return response, nil
}

// ADKServiceInfo provides information about ADK service status
func ADKServiceInfo() map[string]interface{} {
	info := map[string]interface{}{
		"timestamp": time.Now(),
		"service":   "ADK Session Management Integration",
		"version":   "1.0.0",
	}

	if GlobalADKServiceManager == nil {
		info["initialized"] = false
		info["status"] = "not_initialized"
		return info
	}

	info["initialized"] = GlobalADKServiceManager.IsInitialized()
	info["status"] = "initialized"

	// Add component status
	components := map[string]bool{
		"resource_manager":      GlobalADKResourceManager != nil,
		"memory_service":        GlobalADKMemoryService != nil,
		"error_handler":         GlobalADKErrorHandler != nil,
		"session_state_manager": GlobalSessionStateManager != nil,
		"veo_handlers":          GlobalADKVeoHandlers != nil,
	}
	info["components"] = components

	return info
}

// SetupADKEnvironment sets up the ADK environment with default values
func SetupADKEnvironment() {
	// Set default values for ADK environment variables if not already set
	envDefaults := map[string]string{
		"GOOGLE_CLOUD_PROJECT":  "supple-synapse-470916-a2",
		"GOOGLE_CLOUD_LOCATION": "us-central1",
		"GENMEDIA_BUCKET":       "supple-synapse-media",
	}

	for key, defaultValue := range envDefaults {
		if os.Getenv(key) == "" {
			os.Setenv(key, defaultValue)
			log.Printf("Set default environment variable: %s=%s", key, defaultValue)
		}
	}

	// Ensure ADK-specific environment variables
	if os.Getenv("GOOGLE_GENAI_USE_VERTEXAI") == "" {
		os.Setenv("GOOGLE_GENAI_USE_VERTEXAI", "true")
		log.Printf("Set GOOGLE_GENAI_USE_VERTEXAI=true for ADK integration")
	}
}
