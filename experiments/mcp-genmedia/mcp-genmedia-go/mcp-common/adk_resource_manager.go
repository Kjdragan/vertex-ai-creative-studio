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
	"sync"
	"time"

	"cloud.google.com/go/storage"
)

// ADKInvocationContext represents the ADK invocation context interface
// This interface abstracts ADK's InvocationContext for our Go implementation
type ADKInvocationContext interface {
	GetSessionState() ADKSessionState
	GetMemoryService() ADKMemoryService
	GetSessionID() string
	GetUserID() string
	GetLogger() ADKLogger
}

// ADKSessionState represents session state management interface
type ADKSessionState interface {
	Set(key string, value interface{})
	Get(key string) (interface{}, bool)
	Delete(key string)
	Keys() []string
}

// ADKMemoryService represents memory service interface
type ADKMemoryService interface {
	AddToMemory(key string, value interface{}) error
	SearchMemory(query string) ([]interface{}, error)
	AddSessionToMemory(sessionData interface{}) error
}

// ADKLogger represents logging interface
type ADKLogger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// InMemorySessionState provides a simple in-memory implementation
type InMemorySessionState struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

func NewInMemorySessionState() *InMemorySessionState {
	return &InMemorySessionState{
		data: make(map[string]interface{}),
	}
}

func (s *InMemorySessionState) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *InMemorySessionState) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.data[key]
	return value, exists
}

func (s *InMemorySessionState) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

func (s *InMemorySessionState) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

// InMemoryMemoryService provides a simple in-memory implementation
type InMemoryMemoryService struct {
	mu   sync.RWMutex
	data map[string][]interface{}
}

func NewInMemoryMemoryService() *InMemoryMemoryService {
	return &InMemoryMemoryService{
		data: make(map[string][]interface{}),
	}
}

func (m *InMemoryMemoryService) AddToMemory(key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = append(m.data[key], value)
	return nil
}

func (m *InMemoryMemoryService) SearchMemory(query string) ([]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Simple search implementation - in production this would be more sophisticated
	for key, values := range m.data {
		if key == query {
			return values, nil
		}
	}
	return []interface{}{}, nil
}

func (m *InMemoryMemoryService) AddSessionToMemory(sessionData interface{}) error {
	return m.AddToMemory("sessions", sessionData)
}

// SimpleLogger provides basic logging implementation
type SimpleLogger struct{}

func (l *SimpleLogger) Info(msg string, args ...interface{}) {
	log.Printf("[INFO] "+msg, args...)
}

func (l *SimpleLogger) Error(msg string, args ...interface{}) {
	log.Printf("[ERROR] "+msg, args...)
}

func (l *SimpleLogger) Debug(msg string, args ...interface{}) {
	log.Printf("[DEBUG] "+msg, args...)
}

// MockInvocationContext provides a mock implementation for testing
type MockInvocationContext struct {
	sessionState  ADKSessionState
	memoryService ADKMemoryService
	sessionID     string
	userID        string
	logger        ADKLogger
}

func NewMockInvocationContext(sessionID, userID string) *MockInvocationContext {
	return &MockInvocationContext{
		sessionState:  NewInMemorySessionState(),
		memoryService: NewInMemoryMemoryService(),
		sessionID:     sessionID,
		userID:        userID,
		logger:        &SimpleLogger{},
	}
}

func (m *MockInvocationContext) GetSessionState() ADKSessionState {
	return m.sessionState
}

func (m *MockInvocationContext) GetMemoryService() ADKMemoryService {
	return m.memoryService
}

func (m *MockInvocationContext) GetSessionID() string {
	return m.sessionID
}

func (m *MockInvocationContext) GetUserID() string {
	return m.userID
}

func (m *MockInvocationContext) GetLogger() ADKLogger {
	return m.logger
}

// ADKResourceManager manages resources within ADK's session lifecycle
type ADKResourceManager struct {
	mu        sync.RWMutex
	resources map[string]*SessionResource
	timeout   time.Duration
	cleanupCh chan string
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// NewADKResourceManager creates a new ADK-aware resource manager
func NewADKResourceManager(timeout time.Duration) *ADKResourceManager {
	if timeout <= 0 {
		timeout = 30 * time.Minute // Default timeout
	}

	arm := &ADKResourceManager{
		resources: make(map[string]*SessionResource),
		timeout:   timeout,
		cleanupCh: make(chan string, 100),
		stopCh:    make(chan struct{}),
	}

	// Start cleanup goroutine
	arm.wg.Add(1)
	go arm.cleanupWorker()

	log.Printf("ADKResourceManager: initialized with timeout %v", timeout)
	return arm
}

// RegisterResourceWithContext registers a resource with ADK session context
func (arm *ADKResourceManager) RegisterResourceWithContext(
	ctx ADKInvocationContext,
	resourceID string,
	resourceType string,
	resource interface{},
	cleanupFunc func() error,
) error {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	if _, exists := arm.resources[resourceID]; exists {
		return fmt.Errorf("resource with ID '%s' already exists", resourceID)
	}

	// Store resource reference in ADK session state
	sessionState := ctx.GetSessionState()
	sessionState.Set(fmt.Sprintf("resource:%s", resourceID), resourceID)
	sessionState.Set(fmt.Sprintf("resource:%s:type", resourceID), resourceType)
	sessionState.Set(fmt.Sprintf("resource:%s:created", resourceID), time.Now())

	// Register in local resource map
	arm.resources[resourceID] = &SessionResource{
		ID:          resourceID,
		Type:        resourceType,
		Resource:    resource,
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		CleanupFunc: cleanupFunc,
	}

	logger := ctx.GetLogger()
	logger.Info("ADKResourceManager: registered resource with context",
		"resourceID", resourceID,
		"resourceType", resourceType,
		"sessionID", ctx.GetSessionID(),
		"userID", ctx.GetUserID(),
	)

	return nil
}

// GetResourceWithContext retrieves a resource and updates context
func (arm *ADKResourceManager) GetResourceWithContext(
	ctx ADKInvocationContext,
	resourceID string,
) (*SessionResource, bool) {
	arm.mu.Lock()
	defer arm.mu.Unlock()

	resource, exists := arm.resources[resourceID]
	if exists {
		resource.LastUsed = time.Now()
		
		// Update session state with last used time
		sessionState := ctx.GetSessionState()
		sessionState.Set(fmt.Sprintf("resource:%s:last_used", resourceID), time.Now())
		
		logger := ctx.GetLogger()
		logger.Debug("ADKResourceManager: resource accessed",
			"resourceID", resourceID,
			"sessionID", ctx.GetSessionID(),
		)
	}
	return resource, exists
}

// UnregisterResourceWithContext removes a resource and updates context
func (arm *ADKResourceManager) UnregisterResourceWithContext(
	ctx ADKInvocationContext,
	resourceID string,
) error {
	arm.mu.Lock()
	resource, exists := arm.resources[resourceID]
	if !exists {
		arm.mu.Unlock()
		return fmt.Errorf("resource '%s' not found", resourceID)
	}
	delete(arm.resources, resourceID)
	arm.mu.Unlock()

	// Remove from session state
	sessionState := ctx.GetSessionState()
	sessionState.Delete(fmt.Sprintf("resource:%s", resourceID))
	sessionState.Delete(fmt.Sprintf("resource:%s:type", resourceID))
	sessionState.Delete(fmt.Sprintf("resource:%s:created", resourceID))
	sessionState.Delete(fmt.Sprintf("resource:%s:last_used", resourceID))

	logger := ctx.GetLogger()
	logger.Info("ADKResourceManager: unregistered resource",
		"resourceID", resourceID,
		"sessionID", ctx.GetSessionID(),
	)

	return arm.cleanupResource(resource)
}

// CleanupSessionResources cleans up all resources for a session
func (arm *ADKResourceManager) CleanupSessionResources(ctx ADKInvocationContext) error {
	sessionState := ctx.GetSessionState()
	logger := ctx.GetLogger()
	
	// Find all resource keys for this session
	resourceKeys := []string{}
	for _, key := range sessionState.Keys() {
		if len(key) > 9 && key[:9] == "resource:" && key[len(key)-5:] != ":type" && 
		   key[len(key)-8:] != ":created" && key[len(key)-10:] != ":last_used" {
			resourceKeys = append(resourceKeys, key)
		}
	}

	var lastError error
	cleanedCount := 0

	for _, key := range resourceKeys {
		resourceID := key[9:] // Remove "resource:" prefix
		if err := arm.UnregisterResourceWithContext(ctx, resourceID); err != nil {
			logger.Error("ADKResourceManager: failed to cleanup session resource",
				"resourceID", resourceID,
				"error", err,
				"sessionID", ctx.GetSessionID(),
			)
			lastError = err
		} else {
			cleanedCount++
		}
	}

	logger.Info("ADKResourceManager: cleaned up session resources",
		"cleanedCount", cleanedCount,
		"sessionID", ctx.GetSessionID(),
	)

	return lastError
}

// StoreResourceMetadataInMemory stores resource usage in ADK memory
func (arm *ADKResourceManager) StoreResourceMetadataInMemory(
	ctx ADKInvocationContext,
	resourceID string,
	operation string,
	metadata map[string]interface{},
) error {
	memoryService := ctx.GetMemoryService()
	
	memoryEntry := map[string]interface{}{
		"resource_id": resourceID,
		"operation":   operation,
		"metadata":    metadata,
		"timestamp":   time.Now(),
		"session_id":  ctx.GetSessionID(),
		"user_id":     ctx.GetUserID(),
	}

	return memoryService.AddToMemory("resource_usage_history", memoryEntry)
}

// GetResourceUsageFromMemory retrieves resource usage history
func (arm *ADKResourceManager) GetResourceUsageFromMemory(
	ctx ADKInvocationContext,
	query string,
) ([]interface{}, error) {
	memoryService := ctx.GetMemoryService()
	return memoryService.SearchMemory(fmt.Sprintf("resource_usage_history %s", query))
}

// CleanupExpiredResources removes resources that haven't been used within the timeout period
func (arm *ADKResourceManager) CleanupExpiredResources() int {
	arm.mu.Lock()
	now := time.Now()
	expiredIDs := make([]string, 0)

	for id, resource := range arm.resources {
		if now.Sub(resource.LastUsed) > arm.timeout {
			expiredIDs = append(expiredIDs, id)
		}
	}

	// Remove expired resources from map
	expiredResources := make([]*SessionResource, 0, len(expiredIDs))
	for _, id := range expiredIDs {
		if resource, exists := arm.resources[id]; exists {
			expiredResources = append(expiredResources, resource)
			delete(arm.resources, id)
		}
	}
	arm.mu.Unlock()

	// Cleanup expired resources
	cleanedCount := 0
	for _, resource := range expiredResources {
		if err := arm.cleanupResource(resource); err != nil {
			log.Printf("ADKResourceManager: failed to cleanup expired resource '%s': %v", resource.ID, err)
		} else {
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		log.Printf("ADKResourceManager: cleaned up %d expired resources", cleanedCount)
	}

	return cleanedCount
}

// Shutdown stops the resource manager and cleans up all resources
func (arm *ADKResourceManager) Shutdown() error {
	log.Printf("ADKResourceManager: shutting down...")

	// Stop cleanup worker
	close(arm.stopCh)
	arm.wg.Wait()

	// Cleanup all remaining resources
	return arm.cleanupAll()
}

// cleanupWorker runs in a goroutine to periodically clean up expired resources
func (arm *ADKResourceManager) cleanupWorker() {
	defer arm.wg.Done()

	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			arm.CleanupExpiredResources()
		case <-arm.stopCh:
			log.Printf("ADKResourceManager: cleanup worker stopping")
			return
		}
	}
}

// cleanupAll removes all resources and performs cleanup
func (arm *ADKResourceManager) cleanupAll() error {
	arm.mu.Lock()
	allResources := make([]*SessionResource, 0, len(arm.resources))
	for _, resource := range arm.resources {
		allResources = append(allResources, resource)
	}
	arm.resources = make(map[string]*SessionResource)
	arm.mu.Unlock()

	var lastError error
	cleanedCount := 0

	for _, resource := range allResources {
		if err := arm.cleanupResource(resource); err != nil {
			log.Printf("ADKResourceManager: failed to cleanup resource '%s': %v", resource.ID, err)
			lastError = err
		} else {
			cleanedCount++
		}
	}

	log.Printf("ADKResourceManager: cleaned up %d resources during shutdown", cleanedCount)
	return lastError
}

// cleanupResource performs cleanup for a single resource
func (arm *ADKResourceManager) cleanupResource(resource *SessionResource) error {
	if resource.CleanupFunc != nil {
		if err := resource.CleanupFunc(); err != nil {
			return fmt.Errorf("cleanup function failed for resource '%s': %w", resource.ID, err)
		}
	}

	log.Printf("ADKResourceManager: cleaned up resource '%s' of type '%s'", resource.ID, resource.Type)
	return nil
}

// ADKGCSClientManager provides managed GCS clients with ADK context
type ADKGCSClientManager struct {
	resourceManager *ADKResourceManager
}

// NewADKGCSClientManager creates a new ADK-aware GCS client manager
func NewADKGCSClientManager(resourceManager *ADKResourceManager) *ADKGCSClientManager {
	return &ADKGCSClientManager{
		resourceManager: resourceManager,
	}
}

// GetOrCreateClientWithContext gets an existing GCS client or creates a new one with ADK context
func (gcm *ADKGCSClientManager) GetOrCreateClientWithContext(
	ctx context.Context,
	adkCtx ADKInvocationContext,
	clientID string,
) (*storage.Client, error) {
	// Try to get existing client
	if resource, exists := gcm.resourceManager.GetResourceWithContext(adkCtx, clientID); exists {
		if client, ok := resource.Resource.(*storage.Client); ok {
			adkCtx.GetLogger().Debug("ADKGCSClientManager: reusing existing client",
				"clientID", clientID,
				"sessionID", adkCtx.GetSessionID(),
			)
			return client, nil
		}
	}

	// Create new client
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	// Register for cleanup with ADK context
	cleanupFunc := func() error {
		return client.Close()
	}

	err = gcm.resourceManager.RegisterResourceWithContext(
		adkCtx,
		clientID,
		"gcs_client",
		client,
		cleanupFunc,
	)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to register GCS client: %w", err)
	}

	// Store metadata in memory
	metadata := map[string]interface{}{
		"client_id": clientID,
		"created":   time.Now(),
	}
	gcm.resourceManager.StoreResourceMetadataInMemory(adkCtx, clientID, "gcs_client_created", metadata)

	adkCtx.GetLogger().Info("ADKGCSClientManager: created new client",
		"clientID", clientID,
		"sessionID", adkCtx.GetSessionID(),
	)

	return client, nil
}

// CreateManagedGCSClientWithADKContext creates a GCS client with ADK session management
func CreateManagedGCSClientWithADKContext(
	ctx context.Context,
	adkCtx ADKInvocationContext,
	resourceManager *ADKResourceManager,
	clientID string,
) (*storage.Client, error) {
	manager := NewADKGCSClientManager(resourceManager)
	return manager.GetOrCreateClientWithContext(ctx, adkCtx, clientID)
}
