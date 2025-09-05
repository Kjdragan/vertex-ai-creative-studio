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
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// ADKMemoryService provides memory service integration for media processing
type ADKMemoryService struct {
	memoryService ADKMemoryServiceInterface
	logger        ADKLogger
}

// ADKMemoryServiceInterface defines the interface for ADK memory service
type ADKMemoryServiceInterface interface {
	Store(ctx context.Context, key string, value interface{}) error
	Retrieve(ctx context.Context, key string) (interface{}, error)
	Search(ctx context.Context, query string, limit int) ([]MemorySearchResult, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
}

// MemorySearchResult represents a search result from memory service
type MemorySearchResult struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	Score     float64     `json:"score"`
	Timestamp time.Time   `json:"timestamp"`
}

// MediaProcessingMemory represents media processing history stored in memory
type MediaProcessingMemory struct {
	SessionID     string                 `json:"session_id"`
	UserID        string                 `json:"user_id"`
	JobID         string                 `json:"job_id"`
	JobType       string                 `json:"job_type"`
	Status        string                 `json:"status"`
	Parameters    map[string]interface{} `json:"parameters"`
	Results       interface{}            `json:"results"`
	Error         string                 `json:"error,omitempty"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time,omitempty"`
	Duration      time.Duration          `json:"duration,omitempty"`
	ResourceUsage map[string]interface{} `json:"resource_usage,omitempty"`
}

// UserPreferences represents user preferences stored in memory
type UserPreferences struct {
	UserID           string            `json:"user_id"`
	DefaultBucket    string            `json:"default_bucket,omitempty"`
	PreferredModels  map[string]string `json:"preferred_models,omitempty"`
	OutputSettings   map[string]interface{} `json:"output_settings,omitempty"`
	QualitySettings  map[string]interface{} `json:"quality_settings,omitempty"`
	NotificationPrefs map[string]bool   `json:"notification_prefs,omitempty"`
	LastUpdated      time.Time         `json:"last_updated"`
}

// BucketRecommendation represents bucket usage recommendations
type BucketRecommendation struct {
	BucketName      string    `json:"bucket_name"`
	UsageCount      int       `json:"usage_count"`
	LastUsed        time.Time `json:"last_used"`
	SuccessRate     float64   `json:"success_rate"`
	AverageLatency  float64   `json:"average_latency"`
	RecommendScore  float64   `json:"recommend_score"`
}

// NewADKMemoryService creates a new ADK memory service integration
func NewADKMemoryService(memoryService ADKMemoryServiceInterface, logger ADKLogger) *ADKMemoryService {
	return &ADKMemoryService{
		memoryService: memoryService,
		logger:        logger,
	}
}

// StoreMediaProcessingHistory stores media processing job history in memory
func (m *ADKMemoryService) StoreMediaProcessingHistory(ctx context.Context, adkCtx ADKInvocationContext, memory *MediaProcessingMemory) error {
	key := fmt.Sprintf("media_processing:%s:%s", memory.SessionID, memory.JobID)
	
	m.logger.Debug("StoreMediaProcessingHistory: storing processing history",
		"key", key,
		"jobType", memory.JobType,
		"status", memory.Status,
		"sessionID", adkCtx.GetSessionID(),
	)

	err := m.memoryService.Store(ctx, key, memory)
	if err != nil {
		m.logger.Error("StoreMediaProcessingHistory: failed to store processing history",
			"error", err,
			"key", key,
			"sessionID", adkCtx.GetSessionID(),
		)
		return fmt.Errorf("failed to store media processing history: %w", err)
	}

	// Also store in user-specific index for easy retrieval
	userKey := fmt.Sprintf("user_media_history:%s:%s", memory.UserID, memory.JobID)
	err = m.memoryService.Store(ctx, userKey, map[string]interface{}{
		"job_id":     memory.JobID,
		"session_id": memory.SessionID,
		"job_type":   memory.JobType,
		"status":     memory.Status,
		"start_time": memory.StartTime,
		"end_time":   memory.EndTime,
	})
	if err != nil {
		m.logger.Warn("StoreMediaProcessingHistory: failed to store user index",
			"error", err,
			"userKey", userKey,
			"sessionID", adkCtx.GetSessionID(),
		)
	}

	return nil
}

// RetrieveMediaProcessingHistory retrieves media processing job history from memory
func (m *ADKMemoryService) RetrieveMediaProcessingHistory(ctx context.Context, adkCtx ADKInvocationContext, jobID string) (*MediaProcessingMemory, error) {
	key := fmt.Sprintf("media_processing:%s:%s", adkCtx.GetSessionID(), jobID)
	
	m.logger.Debug("RetrieveMediaProcessingHistory: retrieving processing history",
		"key", key,
		"jobID", jobID,
		"sessionID", adkCtx.GetSessionID(),
	)

	value, err := m.memoryService.Retrieve(ctx, key)
	if err != nil {
		m.logger.Error("RetrieveMediaProcessingHistory: failed to retrieve processing history",
			"error", err,
			"key", key,
			"sessionID", adkCtx.GetSessionID(),
		)
		return nil, fmt.Errorf("failed to retrieve media processing history: %w", err)
	}

	// Convert to MediaProcessingMemory
	var memory MediaProcessingMemory
	if data, ok := value.(map[string]interface{}); ok {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal memory data: %w", err)
		}
		err = json.Unmarshal(jsonData, &memory)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal memory data: %w", err)
		}
	} else if memoryPtr, ok := value.(*MediaProcessingMemory); ok {
		memory = *memoryPtr
	} else {
		return nil, fmt.Errorf("invalid memory data type: %T", value)
	}

	return &memory, nil
}

// SearchMediaProcessingHistory searches media processing history by query
func (m *ADKMemoryService) SearchMediaProcessingHistory(ctx context.Context, adkCtx ADKInvocationContext, query string, limit int) ([]*MediaProcessingMemory, error) {
	m.logger.Debug("SearchMediaProcessingHistory: searching processing history",
		"query", query,
		"limit", limit,
		"sessionID", adkCtx.GetSessionID(),
	)

	results, err := m.memoryService.Search(ctx, query, limit)
	if err != nil {
		m.logger.Error("SearchMediaProcessingHistory: search failed",
			"error", err,
			"query", query,
			"sessionID", adkCtx.GetSessionID(),
		)
		return nil, fmt.Errorf("failed to search media processing history: %w", err)
	}

	var memories []*MediaProcessingMemory
	for _, result := range results {
		if data, ok := result.Value.(map[string]interface{}); ok {
			var memory MediaProcessingMemory
			jsonData, err := json.Marshal(data)
			if err != nil {
				m.logger.Warn("SearchMediaProcessingHistory: failed to marshal result",
					"error", err,
					"key", result.Key,
				)
				continue
			}
			err = json.Unmarshal(jsonData, &memory)
			if err != nil {
				m.logger.Warn("SearchMediaProcessingHistory: failed to unmarshal result",
					"error", err,
					"key", result.Key,
				)
				continue
			}
			memories = append(memories, &memory)
		}
	}

	m.logger.Info("SearchMediaProcessingHistory: search completed",
		"query", query,
		"resultsFound", len(memories),
		"sessionID", adkCtx.GetSessionID(),
	)

	return memories, nil
}

// StoreUserPreferences stores user preferences in memory
func (m *ADKMemoryService) StoreUserPreferences(ctx context.Context, adkCtx ADKInvocationContext, prefs *UserPreferences) error {
	key := fmt.Sprintf("user_preferences:%s", prefs.UserID)
	prefs.LastUpdated = time.Now()
	
	m.logger.Debug("StoreUserPreferences: storing user preferences",
		"key", key,
		"userID", prefs.UserID,
		"sessionID", adkCtx.GetSessionID(),
	)

	err := m.memoryService.Store(ctx, key, prefs)
	if err != nil {
		m.logger.Error("StoreUserPreferences: failed to store user preferences",
			"error", err,
			"key", key,
			"sessionID", adkCtx.GetSessionID(),
		)
		return fmt.Errorf("failed to store user preferences: %w", err)
	}

	return nil
}

// RetrieveUserPreferences retrieves user preferences from memory
func (m *ADKMemoryService) RetrieveUserPreferences(ctx context.Context, adkCtx ADKInvocationContext, userID string) (*UserPreferences, error) {
	key := fmt.Sprintf("user_preferences:%s", userID)
	
	m.logger.Debug("RetrieveUserPreferences: retrieving user preferences",
		"key", key,
		"userID", userID,
		"sessionID", adkCtx.GetSessionID(),
	)

	value, err := m.memoryService.Retrieve(ctx, key)
	if err != nil {
		m.logger.Debug("RetrieveUserPreferences: no preferences found, returning defaults",
			"userID", userID,
			"sessionID", adkCtx.GetSessionID(),
		)
		// Return default preferences if not found
		return &UserPreferences{
			UserID:           userID,
			PreferredModels:  make(map[string]string),
			OutputSettings:   make(map[string]interface{}),
			QualitySettings:  make(map[string]interface{}),
			NotificationPrefs: make(map[string]bool),
			LastUpdated:      time.Now(),
		}, nil
	}

	// Convert to UserPreferences
	var prefs UserPreferences
	if data, ok := value.(map[string]interface{}); ok {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal preferences data: %w", err)
		}
		err = json.Unmarshal(jsonData, &prefs)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal preferences data: %w", err)
		}
	} else if prefsPtr, ok := value.(*UserPreferences); ok {
		prefs = *prefsPtr
	} else {
		return nil, fmt.Errorf("invalid preferences data type: %T", value)
	}

	return &prefs, nil
}

// StoreBucketRecommendation stores bucket usage recommendation in memory
func (m *ADKMemoryService) StoreBucketRecommendation(ctx context.Context, adkCtx ADKInvocationContext, recommendation *BucketRecommendation) error {
	key := fmt.Sprintf("bucket_recommendation:%s:%s", adkCtx.GetUserID(), recommendation.BucketName)
	
	m.logger.Debug("StoreBucketRecommendation: storing bucket recommendation",
		"key", key,
		"bucketName", recommendation.BucketName,
		"usageCount", recommendation.UsageCount,
		"sessionID", adkCtx.GetSessionID(),
	)

	err := m.memoryService.Store(ctx, key, recommendation)
	if err != nil {
		m.logger.Error("StoreBucketRecommendation: failed to store bucket recommendation",
			"error", err,
			"key", key,
			"sessionID", adkCtx.GetSessionID(),
		)
		return fmt.Errorf("failed to store bucket recommendation: %w", err)
	}

	return nil
}

// GetBucketRecommendations retrieves bucket recommendations for a user
func (m *ADKMemoryService) GetBucketRecommendations(ctx context.Context, adkCtx ADKInvocationContext, limit int) ([]*BucketRecommendation, error) {
	prefix := fmt.Sprintf("bucket_recommendation:%s:", adkCtx.GetUserID())
	
	m.logger.Debug("GetBucketRecommendations: retrieving bucket recommendations",
		"prefix", prefix,
		"limit", limit,
		"sessionID", adkCtx.GetSessionID(),
	)

	keys, err := m.memoryService.List(ctx, prefix)
	if err != nil {
		m.logger.Error("GetBucketRecommendations: failed to list bucket recommendations",
			"error", err,
			"prefix", prefix,
			"sessionID", adkCtx.GetSessionID(),
		)
		return nil, fmt.Errorf("failed to list bucket recommendations: %w", err)
	}

	var recommendations []*BucketRecommendation
	for i, key := range keys {
		if i >= limit {
			break
		}

		value, err := m.memoryService.Retrieve(ctx, key)
		if err != nil {
			m.logger.Warn("GetBucketRecommendations: failed to retrieve recommendation",
				"error", err,
				"key", key,
			)
			continue
		}

		var recommendation BucketRecommendation
		if data, ok := value.(map[string]interface{}); ok {
			jsonData, err := json.Marshal(data)
			if err != nil {
				m.logger.Warn("GetBucketRecommendations: failed to marshal recommendation",
					"error", err,
					"key", key,
				)
				continue
			}
			err = json.Unmarshal(jsonData, &recommendation)
			if err != nil {
				m.logger.Warn("GetBucketRecommendations: failed to unmarshal recommendation",
					"error", err,
					"key", key,
				)
				continue
			}
		} else if recPtr, ok := value.(*BucketRecommendation); ok {
			recommendation = *recPtr
		} else {
			m.logger.Warn("GetBucketRecommendations: invalid recommendation data type",
				"type", fmt.Sprintf("%T", value),
				"key", key,
			)
			continue
		}

		recommendations = append(recommendations, &recommendation)
	}

	m.logger.Info("GetBucketRecommendations: retrieved bucket recommendations",
		"count", len(recommendations),
		"sessionID", adkCtx.GetSessionID(),
	)

	return recommendations, nil
}

// UpdateBucketUsageStats updates bucket usage statistics for recommendations
func (m *ADKMemoryService) UpdateBucketUsageStats(ctx context.Context, adkCtx ADKInvocationContext, bucketName string, success bool, latency time.Duration) error {
	key := fmt.Sprintf("bucket_recommendation:%s:%s", adkCtx.GetUserID(), bucketName)
	
	// Retrieve existing recommendation or create new one
	var recommendation *BucketRecommendation
	value, err := m.memoryService.Retrieve(ctx, key)
	if err != nil {
		// Create new recommendation
		recommendation = &BucketRecommendation{
			BucketName:     bucketName,
			UsageCount:     0,
			LastUsed:       time.Now(),
			SuccessRate:    0.0,
			AverageLatency: 0.0,
			RecommendScore: 0.0,
		}
	} else {
		// Convert existing recommendation
		if data, ok := value.(map[string]interface{}); ok {
			jsonData, err := json.Marshal(data)
			if err != nil {
				return fmt.Errorf("failed to marshal existing recommendation: %w", err)
			}
			err = json.Unmarshal(jsonData, recommendation)
			if err != nil {
				return fmt.Errorf("failed to unmarshal existing recommendation: %w", err)
			}
		} else if recPtr, ok := value.(*BucketRecommendation); ok {
			recommendation = recPtr
		} else {
			return fmt.Errorf("invalid existing recommendation data type: %T", value)
		}
	}

	// Update statistics
	recommendation.UsageCount++
	recommendation.LastUsed = time.Now()

	// Update success rate
	if success {
		recommendation.SuccessRate = (recommendation.SuccessRate*float64(recommendation.UsageCount-1) + 1.0) / float64(recommendation.UsageCount)
	} else {
		recommendation.SuccessRate = (recommendation.SuccessRate * float64(recommendation.UsageCount-1)) / float64(recommendation.UsageCount)
	}

	// Update average latency
	latencyMs := float64(latency.Milliseconds())
	recommendation.AverageLatency = (recommendation.AverageLatency*float64(recommendation.UsageCount-1) + latencyMs) / float64(recommendation.UsageCount)

	// Calculate recommendation score (success rate weighted by usage frequency and recency)
	recencyFactor := 1.0 / (1.0 + float64(time.Since(recommendation.LastUsed).Hours())/24.0) // Decay over days
	usageFactor := float64(recommendation.UsageCount) / 100.0                                  // Normalize usage count
	if usageFactor > 1.0 {
		usageFactor = 1.0
	}
	recommendation.RecommendScore = recommendation.SuccessRate * (0.7 + 0.2*usageFactor + 0.1*recencyFactor)

	// Store updated recommendation
	err = m.memoryService.Store(ctx, key, recommendation)
	if err != nil {
		m.logger.Error("UpdateBucketUsageStats: failed to store updated recommendation",
			"error", err,
			"key", key,
			"sessionID", adkCtx.GetSessionID(),
		)
		return fmt.Errorf("failed to store updated bucket recommendation: %w", err)
	}

	m.logger.Debug("UpdateBucketUsageStats: updated bucket usage statistics",
		"bucketName", bucketName,
		"usageCount", recommendation.UsageCount,
		"successRate", recommendation.SuccessRate,
		"recommendScore", recommendation.RecommendScore,
		"sessionID", adkCtx.GetSessionID(),
	)

	return nil
}

// Global ADK memory service instance
var GlobalADKMemoryService *ADKMemoryService

// InitializeGlobalADKMemoryService initializes the global ADK memory service
func InitializeGlobalADKMemoryService(memoryService ADKMemoryServiceInterface, logger ADKLogger) {
	GlobalADKMemoryService = NewADKMemoryService(memoryService, logger)
	log.Printf("GlobalADKMemoryService initialized")
}

// Convenience functions for global access

// StoreMediaProcessingHistoryGlobal stores media processing history using global service
func StoreMediaProcessingHistoryGlobal(ctx context.Context, adkCtx ADKInvocationContext, memory *MediaProcessingMemory) error {
	if GlobalADKMemoryService == nil {
		return fmt.Errorf("global ADK memory service not initialized")
	}
	return GlobalADKMemoryService.StoreMediaProcessingHistory(ctx, adkCtx, memory)
}

// RetrieveUserPreferencesGlobal retrieves user preferences using global service
func RetrieveUserPreferencesGlobal(ctx context.Context, adkCtx ADKInvocationContext, userID string) (*UserPreferences, error) {
	if GlobalADKMemoryService == nil {
		return nil, fmt.Errorf("global ADK memory service not initialized")
	}
	return GlobalADKMemoryService.RetrieveUserPreferences(ctx, adkCtx, userID)
}

// UpdateBucketUsageStatsGlobal updates bucket usage statistics using global service
func UpdateBucketUsageStatsGlobal(ctx context.Context, adkCtx ADKInvocationContext, bucketName string, success bool, latency time.Duration) error {
	if GlobalADKMemoryService == nil {
		return fmt.Errorf("global ADK memory service not initialized")
	}
	return GlobalADKMemoryService.UpdateBucketUsageStats(ctx, adkCtx, bucketName, success, latency)
}
