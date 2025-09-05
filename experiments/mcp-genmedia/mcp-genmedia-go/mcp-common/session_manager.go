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

// SessionResource represents a managed resource within an MCP session
type SessionResource struct {
	ID          string
	Type        string // "gcs_client", "temp_file", "upload_stream", etc.
	Resource    interface{}
	CreatedAt   time.Time
	LastUsed    time.Time
	CleanupFunc func() error
}

// SessionManager manages resources and cleanup for MCP sessions
type SessionManager struct {
	mu        sync.RWMutex
	resources map[string]*SessionResource
	timeout   time.Duration
	cleanupCh chan string
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// NewSessionManager creates a new session manager with the specified timeout
func NewSessionManager(timeout time.Duration) *SessionManager {
	if timeout <= 0 {
		timeout = 30 * time.Minute // Default timeout
	}

	sm := &SessionManager{
		resources: make(map[string]*SessionResource),
		timeout:   timeout,
		cleanupCh: make(chan string, 100),
		stopCh:    make(chan struct{}),
	}

	// Start cleanup goroutine
	sm.wg.Add(1)
	go sm.cleanupWorker()

	log.Printf("SessionManager: initialized with timeout %v", timeout)
	return sm
}

// RegisterResource registers a resource for automatic cleanup
func (sm *SessionManager) RegisterResource(id, resourceType string, resource interface{}, cleanupFunc func() error) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.resources[id]; exists {
		return fmt.Errorf("resource with ID '%s' already exists", id)
	}

	sm.resources[id] = &SessionResource{
		ID:          id,
		Type:        resourceType,
		Resource:    resource,
		CreatedAt:   time.Now(),
		LastUsed:    time.Now(),
		CleanupFunc: cleanupFunc,
	}

	log.Printf("SessionManager: registered resource '%s' of type '%s'", id, resourceType)
	return nil
}

// GetResource retrieves a resource and updates its last used time
func (sm *SessionManager) GetResource(id string) (*SessionResource, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	resource, exists := sm.resources[id]
	if exists {
		resource.LastUsed = time.Now()
	}
	return resource, exists
}

// UnregisterResource removes a resource and performs cleanup
func (sm *SessionManager) UnregisterResource(id string) error {
	sm.mu.Lock()
	resource, exists := sm.resources[id]
	if !exists {
		sm.mu.Unlock()
		return fmt.Errorf("resource '%s' not found", id)
	}
	delete(sm.resources, id)
	sm.mu.Unlock()

	return sm.cleanupResource(resource)
}

// CleanupExpiredResources removes resources that haven't been used within the timeout period
func (sm *SessionManager) CleanupExpiredResources() int {
	sm.mu.Lock()
	now := time.Now()
	expiredIDs := make([]string, 0)

	for id, resource := range sm.resources {
		if now.Sub(resource.LastUsed) > sm.timeout {
			expiredIDs = append(expiredIDs, id)
		}
	}

	// Remove expired resources from map
	expiredResources := make([]*SessionResource, 0, len(expiredIDs))
	for _, id := range expiredIDs {
		if resource, exists := sm.resources[id]; exists {
			expiredResources = append(expiredResources, resource)
			delete(sm.resources, id)
		}
	}
	sm.mu.Unlock()

	// Cleanup expired resources
	cleanedCount := 0
	for _, resource := range expiredResources {
		if err := sm.cleanupResource(resource); err != nil {
			log.Printf("SessionManager: failed to cleanup expired resource '%s': %v", resource.ID, err)
		} else {
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		log.Printf("SessionManager: cleaned up %d expired resources", cleanedCount)
	}

	return cleanedCount
}

// CleanupAll removes all resources and performs cleanup
func (sm *SessionManager) CleanupAll() error {
	sm.mu.Lock()
	allResources := make([]*SessionResource, 0, len(sm.resources))
	for _, resource := range sm.resources {
		allResources = append(allResources, resource)
	}
	sm.resources = make(map[string]*SessionResource)
	sm.mu.Unlock()

	var lastError error
	cleanedCount := 0

	for _, resource := range allResources {
		if err := sm.cleanupResource(resource); err != nil {
			log.Printf("SessionManager: failed to cleanup resource '%s': %v", resource.ID, err)
			lastError = err
		} else {
			cleanedCount++
		}
	}

	log.Printf("SessionManager: cleaned up %d resources during shutdown", cleanedCount)
	return lastError
}

// Shutdown stops the session manager and cleans up all resources
func (sm *SessionManager) Shutdown() error {
	log.Printf("SessionManager: shutting down...")

	// Stop cleanup worker
	close(sm.stopCh)
	sm.wg.Wait()

	// Cleanup all remaining resources
	return sm.CleanupAll()
}

// GetStats returns statistics about managed resources
func (sm *SessionManager) GetStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	typeCount := make(map[string]int)
	totalResources := len(sm.resources)
	oldestResource := time.Now()
	newestResource := time.Time{}

	for _, resource := range sm.resources {
		typeCount[resource.Type]++
		if resource.CreatedAt.Before(oldestResource) {
			oldestResource = resource.CreatedAt
		}
		if resource.CreatedAt.After(newestResource) {
			newestResource = resource.CreatedAt
		}
	}

	return map[string]interface{}{
		"total_resources": totalResources,
		"resource_types":  typeCount,
		"timeout":         sm.timeout.String(),
		"oldest_resource": oldestResource,
		"newest_resource": newestResource,
	}
}

// cleanupWorker runs in a goroutine to periodically clean up expired resources
func (sm *SessionManager) cleanupWorker() {
	defer sm.wg.Done()

	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.CleanupExpiredResources()
		case <-sm.stopCh:
			log.Printf("SessionManager: cleanup worker stopping")
			return
		}
	}
}

// cleanupResource performs cleanup for a single resource
func (sm *SessionManager) cleanupResource(resource *SessionResource) error {
	if resource.CleanupFunc != nil {
		if err := resource.CleanupFunc(); err != nil {
			return fmt.Errorf("cleanup function failed for resource '%s': %w", resource.ID, err)
		}
	}

	log.Printf("SessionManager: cleaned up resource '%s' of type '%s'", resource.ID, resource.Type)
	return nil
}

// GCSClientManager provides managed GCS clients with automatic cleanup
type GCSClientManager struct {
	sessionManager *SessionManager
}

// NewGCSClientManager creates a new GCS client manager
func NewGCSClientManager(sessionManager *SessionManager) *GCSClientManager {
	return &GCSClientManager{
		sessionManager: sessionManager,
	}
}

// GetOrCreateClient gets an existing GCS client or creates a new one
func (gcm *GCSClientManager) GetOrCreateClient(ctx context.Context, clientID string) (*storage.Client, error) {
	// Try to get existing client
	if resource, exists := gcm.sessionManager.GetResource(clientID); exists {
		if client, ok := resource.Resource.(*storage.Client); ok {
			log.Printf("GCSClientManager: reusing existing client '%s'", clientID)
			return client, nil
		}
	}

	// Create new client
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	// Register for cleanup
	cleanupFunc := func() error {
		return client.Close()
	}

	err = gcm.sessionManager.RegisterResource(clientID, "gcs_client", client, cleanupFunc)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to register GCS client: %w", err)
	}

	log.Printf("GCSClientManager: created new client '%s'", clientID)
	return client, nil
}

// CreateManagedGCSClient creates a GCS client with automatic session management
func CreateManagedGCSClient(ctx context.Context, sessionManager *SessionManager, clientID string) (*storage.Client, error) {
	manager := NewGCSClientManager(sessionManager)
	return manager.GetOrCreateClient(ctx, clientID)
}
