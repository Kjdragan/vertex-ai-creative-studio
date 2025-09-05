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
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/google/generative-ai-go/genai"
	common "github.com/google/mcp-genmedia-go/mcp-common"
	"github.com/mark3labs/mcp-go/mcp"
	"cloud.google.com/go/storage"
)

// ADKVeoHandlers provides ADK-aware Veo video generation handlers
type ADKVeoHandlers struct {
	resourceManager *common.ADKResourceManager
	config          *common.Config
}

// NewADKVeoHandlers creates new ADK-aware Veo handlers
func NewADKVeoHandlers(resourceManager *common.ADKResourceManager, config *common.Config) *ADKVeoHandlers {
	return &ADKVeoHandlers{
		resourceManager: resourceManager,
		config:          config,
	}
}

// VeoTextToVideoHandlerADK handles text-to-video generation with ADK context
func (h *ADKVeoHandlers) VeoTextToVideoHandlerADK(
	client *genai.Client,
	ctx context.Context,
	adkCtx common.ADKInvocationContext,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger := adkCtx.GetLogger()
	logger.Info("VeoTextToVideoHandlerADK: starting text-to-video generation",
		"sessionID", adkCtx.GetSessionID(),
		"userID", adkCtx.GetUserID(),
	)

	// Parse and validate video parameters with ADK context
	modelName, bucketName, outputDir, prompt, numVideos, duration, err := h.parseCommonVideoParamsADK(ctx, adkCtx, request.GetArguments())
	if err != nil {
		common.RecordError(adkCtx, "veo_text_to_video_params", err)
		return mcp.NewToolResultError(fmt.Sprintf("parameter validation failed: %v", err)), nil
	}

	// Add to media history
	common.AddToMediaHistory(adkCtx, "veo_text_to_video_start", map[string]interface{}{
		"model":      modelName,
		"bucket":     bucketName,
		"output_dir": outputDir,
		"prompt":     prompt,
		"num_videos": numVideos,
		"duration":   duration,
	})

	// Create processing job
	jobID := fmt.Sprintf("veo_t2v_%s_%d", adkCtx.GetSessionID(), common.IncrementUploadCounter(adkCtx))
	common.GlobalSessionStateManager.AddProcessingJob(adkCtx, jobID, map[string]interface{}{
		"type":       "text_to_video",
		"model":      modelName,
		"prompt":     prompt,
		"num_videos": numVideos,
		"duration":   duration,
	})

	// Generate videos
	videoURIs, err := h.generateVideosWithADKContext(ctx, adkCtx, client, modelName, bucketName, outputDir, prompt, numVideos, duration, "")
	if err != nil {
		common.RecordError(adkCtx, "veo_text_to_video_generation", err)
		common.GlobalSessionStateManager.UpdateProcessingJob(adkCtx, jobID, "failed", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("video generation failed: %v", err)), nil
	}

	// Update processing job with success
	common.GlobalSessionStateManager.UpdateProcessingJob(adkCtx, jobID, "completed", videoURIs)

	// Add to media history
	common.AddToMediaHistory(adkCtx, "veo_text_to_video_success", map[string]interface{}{
		"job_id":     jobID,
		"video_uris": videoURIs,
		"count":      len(videoURIs),
	})

	logger.Info("VeoTextToVideoHandlerADK: text-to-video generation completed",
		"jobID", jobID,
		"videoCount", len(videoURIs),
		"sessionID", adkCtx.GetSessionID(),
	)

	return mcp.NewToolResultText(fmt.Sprintf("Generated %d video(s): %v", len(videoURIs), videoURIs)), nil
}

// VeoImageToVideoHandlerADK handles image-to-video generation with ADK context
func (h *ADKVeoHandlers) VeoImageToVideoHandlerADK(
	client *genai.Client,
	ctx context.Context,
	adkCtx common.ADKInvocationContext,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	logger := adkCtx.GetLogger()
	logger.Info("VeoImageToVideoHandlerADK: starting image-to-video generation",
		"sessionID", adkCtx.GetSessionID(),
		"userID", adkCtx.GetUserID(),
	)

	// Parse and validate video parameters with ADK context
	modelName, bucketName, outputDir, prompt, numVideos, duration, err := h.parseCommonVideoParamsADK(ctx, adkCtx, request.GetArguments())
	if err != nil {
		common.RecordError(adkCtx, "veo_image_to_video_params", err)
		return mcp.NewToolResultError(fmt.Sprintf("parameter validation failed: %v", err)), nil
	}

	// Process image URI (supports both GCS URIs and inline data)
	imageURIRaw, ok := request.GetArguments()["image_uri"]
	if !ok {
		err := fmt.Errorf("image_uri parameter is required")
		common.RecordError(adkCtx, "veo_image_to_video_image_uri", err)
		return mcp.NewToolResultError("image_uri must be provided and is required for image-to-video"), nil
	}

	imageURI, ok := imageURIRaw.(string)
	if !ok || strings.TrimSpace(imageURI) == "" {
		err := fmt.Errorf("image_uri must be a non-empty string")
		common.RecordError(adkCtx, "veo_image_to_video_image_uri", err)
		return mcp.NewToolResultError("image_uri must be a non-empty string"), nil
	}

	// Process inline image data if needed
	processedImageURI, err := h.processImageURIWithADKContext(ctx, adkCtx, imageURI, bucketName)
	if err != nil {
		common.RecordError(adkCtx, "veo_image_to_video_image_processing", err)
		return mcp.NewToolResultError(fmt.Sprintf("image processing failed: %v", err)), nil
	}

	// Add to media history
	common.AddToMediaHistory(adkCtx, "veo_image_to_video_start", map[string]interface{}{
		"model":             modelName,
		"bucket":            bucketName,
		"output_dir":        outputDir,
		"prompt":            prompt,
		"num_videos":        numVideos,
		"duration":          duration,
		"original_image":    imageURI,
		"processed_image":   processedImageURI,
	})

	// Create processing job
	jobID := fmt.Sprintf("veo_i2v_%s_%d", adkCtx.GetSessionID(), common.IncrementUploadCounter(adkCtx))
	common.GlobalSessionStateManager.AddProcessingJob(adkCtx, jobID, map[string]interface{}{
		"type":             "image_to_video",
		"model":            modelName,
		"prompt":           prompt,
		"num_videos":       numVideos,
		"duration":         duration,
		"image_uri":        processedImageURI,
	})

	// Generate videos
	videoURIs, err := h.generateVideosWithADKContext(ctx, adkCtx, client, modelName, bucketName, outputDir, prompt, numVideos, duration, processedImageURI)
	if err != nil {
		common.RecordError(adkCtx, "veo_image_to_video_generation", err)
		common.GlobalSessionStateManager.UpdateProcessingJob(adkCtx, jobID, "failed", err.Error())
		return mcp.NewToolResultError(fmt.Sprintf("video generation failed: %v", err)), nil
	}

	// Update processing job with success
	common.GlobalSessionStateManager.UpdateProcessingJob(adkCtx, jobID, "completed", videoURIs)

	// Add to media history
	common.AddToMediaHistory(adkCtx, "veo_image_to_video_success", map[string]interface{}{
		"job_id":     jobID,
		"video_uris": videoURIs,
		"count":      len(videoURIs),
	})

	logger.Info("VeoImageToVideoHandlerADK: image-to-video generation completed",
		"jobID", jobID,
		"videoCount", len(videoURIs),
		"sessionID", adkCtx.GetSessionID(),
	)

	return mcp.NewToolResultText(fmt.Sprintf("Generated %d video(s) from image: %v", len(videoURIs), videoURIs)), nil
}

// parseCommonVideoParamsADK parses and validates video generation parameters with ADK context
func (h *ADKVeoHandlers) parseCommonVideoParamsADK(
	ctx context.Context,
	adkCtx common.ADKInvocationContext,
	args map[string]interface{},
) (string, string, string, string, int32, int32, error) {
	logger := adkCtx.GetLogger()

	// Parse model name
	modelName := "imagen3"
	if model, ok := args["model"].(string); ok && strings.TrimSpace(model) != "" {
		modelName = strings.TrimSpace(model)
	}

	// Parse bucket name using ADK bucket resolution
	var bucketName string
	var err error
	
	userProvidedBucket := ""
	if bucket, ok := args["bucket"].(string); ok {
		userProvidedBucket = strings.TrimSpace(bucket)
	}

	bucketName, err = common.ResolveBucketWithADKContext(ctx, adkCtx, userProvidedBucket)
	if err != nil {
		logger.Error("parseCommonVideoParamsADK: bucket resolution failed",
			"error", err,
			"sessionID", adkCtx.GetSessionID(),
		)
		return "", "", "", "", 0, 0, fmt.Errorf("bucket resolution failed: %w", err)
	}

	// Parse output directory
	outputDir := "veo-videos"
	if dir, ok := args["output_dir"].(string); ok && strings.TrimSpace(dir) != "" {
		outputDir = strings.TrimSpace(dir)
	}

	// Parse prompt
	prompt, ok := args["prompt"].(string)
	if !ok || strings.TrimSpace(prompt) == "" {
		return "", "", "", "", 0, 0, fmt.Errorf("prompt is required and must be a non-empty string")
	}
	prompt = strings.TrimSpace(prompt)

	// Parse number of videos
	numVideos := int32(1)
	if numStr, ok := args["num_videos"].(string); ok {
		if num, err := strconv.ParseInt(numStr, 10, 32); err == nil && num > 0 && num <= 8 {
			numVideos = int32(num)
		}
	} else if num, ok := args["num_videos"].(float64); ok && num > 0 && num <= 8 {
		numVideos = int32(num)
	}

	// Parse duration
	duration := int32(5)
	if durStr, ok := args["duration"].(string); ok {
		if dur, err := strconv.ParseInt(durStr, 10, 32); err == nil && dur > 0 && dur <= 10 {
			duration = int32(dur)
		}
	} else if dur, ok := args["duration"].(float64); ok && dur > 0 && dur <= 10 {
		duration = int32(dur)
	}

	logger.Debug("parseCommonVideoParamsADK: parsed parameters",
		"model", modelName,
		"bucket", bucketName,
		"outputDir", outputDir,
		"numVideos", numVideos,
		"duration", duration,
		"sessionID", adkCtx.GetSessionID(),
	)

	return modelName, bucketName, outputDir, prompt, numVideos, duration, nil
}

// processImageURIWithADKContext processes image URI, handling both GCS URIs and inline data
func (h *ADKVeoHandlers) processImageURIWithADKContext(
	ctx context.Context,
	adkCtx common.ADKInvocationContext,
	imageURI string,
	bucketName string,
) (string, error) {
	logger := adkCtx.GetLogger()

	// If it's already a GCS URI, validate and return
	if strings.HasPrefix(imageURI, "gs://") {
		logger.Debug("processImageURIWithADKContext: processing GCS URI",
			"imageURI", imageURI,
			"sessionID", adkCtx.GetSessionID(),
		)
		return imageURI, nil
	}

	// Handle inline image data (base64 or data URL)
	logger.Info("processImageURIWithADKContext: processing inline image data",
		"dataLength", len(imageURI),
		"sessionID", adkCtx.GetSessionID(),
	)

	// Create GCS client for upload
	clientID := fmt.Sprintf("image_upload_%s", adkCtx.GetSessionID())
	client, err := common.CreateManagedGCSClientWithADKContext(ctx, adkCtx, h.resourceManager, clientID)
	if err != nil {
		return "", fmt.Errorf("failed to create GCS client: %w", err)
	}

	// Process inline image data
	processor := common.NewMCPInlineProcessor()
	processedParams := map[string]interface{}{
		"image_uri": imageURI,
	}

	err = processor.ProcessInlineImageParameters(ctx, client, bucketName, "inline-uploads", processedParams)
	if err != nil {
		logger.Error("processImageURIWithADKContext: inline image processing failed",
			"error", err,
			"sessionID", adkCtx.GetSessionID(),
		)
		return "", fmt.Errorf("inline image processing failed: %w", err)
	}

	processedURI, ok := processedParams["image_uri"].(string)
	if !ok {
		return "", fmt.Errorf("processed image URI is not a string")
	}

	// Add to media history
	common.AddToMediaHistory(adkCtx, "inline_image_processed", map[string]interface{}{
		"original_length": len(imageURI),
		"processed_uri":   processedURI,
		"bucket":          bucketName,
	})

	logger.Info("processImageURIWithADKContext: inline image processing completed",
		"processedURI", processedURI,
		"sessionID", adkCtx.GetSessionID(),
	)

	return processedURI, nil
}

// generateVideosWithADKContext generates videos using Veo model with ADK context
func (h *ADKVeoHandlers) generateVideosWithADKContext(
	ctx context.Context,
	adkCtx common.ADKInvocationContext,
	client *genai.Client,
	modelName string,
	bucketName string,
	outputDir string,
	prompt string,
	numVideos int32,
	duration int32,
	imageURI string,
) ([]string, error) {
	logger := adkCtx.GetLogger()

	logger.Info("generateVideosWithADKContext: starting video generation",
		"model", modelName,
		"bucket", bucketName,
		"numVideos", numVideos,
		"duration", duration,
		"hasImage", imageURI != "",
		"sessionID", adkCtx.GetSessionID(),
	)

	// Create GCS client for upload
	clientID := fmt.Sprintf("video_upload_%s", adkCtx.GetSessionID())
	gcsClient, err := common.CreateManagedGCSClientWithADKContext(ctx, adkCtx, h.resourceManager, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	// Get the generative model
	model := client.GenerativeModel(modelName)
	if model == nil {
		return nil, fmt.Errorf("failed to get generative model: %s", modelName)
	}

	var videoURIs []string

	// Generate the requested number of videos
	for i := int32(0); i < numVideos; i++ {
		logger.Debug("generateVideosWithADKContext: generating video",
			"videoIndex", i+1,
			"totalVideos", numVideos,
			"sessionID", adkCtx.GetSessionID(),
		)

		// Create video generation request
		var parts []genai.Part
		parts = append(parts, genai.Text(prompt))

		// Add image if provided (for image-to-video)
		if imageURI != "" {
			// For GCS URIs, we need to create a file data part
			if strings.HasPrefix(imageURI, "gs://") {
				// Extract bucket and object from GCS URI
				gcsPath := strings.TrimPrefix(imageURI, "gs://")
				pathParts := strings.SplitN(gcsPath, "/", 2)
				if len(pathParts) != 2 {
					return nil, fmt.Errorf("invalid GCS URI format: %s", imageURI)
				}

				// Read the image data from GCS
				bucket := gcsClient.Bucket(pathParts[0])
				obj := bucket.Object(pathParts[1])
				reader, err := obj.NewReader(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to read image from GCS: %w", err)
				}
				defer reader.Close()

				// Read image data
				imageData := make([]byte, reader.Attrs.Size)
				_, err = reader.Read(imageData)
				if err != nil {
					return nil, fmt.Errorf("failed to read image data: %w", err)
				}

				// Determine MIME type from object attributes
				mimeType := reader.Attrs.ContentType
				if mimeType == "" {
					mimeType = "image/jpeg" // Default fallback
				}

				parts = append(parts, genai.ImageData(mimeType, imageData))
			}
		}

		// Generate video
		resp, err := model.GenerateContent(ctx, parts...)
		if err != nil {
			logger.Error("generateVideosWithADKContext: video generation failed",
				"videoIndex", i+1,
				"error", err,
				"sessionID", adkCtx.GetSessionID(),
			)
			return nil, fmt.Errorf("video generation failed for video %d: %w", i+1, err)
		}

		// Process response and upload to GCS
		if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
			return nil, fmt.Errorf("no video content generated for video %d", i+1)
		}

		// Extract video data from response
		// Note: This is a simplified example - actual implementation would depend on Veo API response format
		videoData := []byte("placeholder_video_data") // Replace with actual video data extraction

		// Generate unique filename
		filename := fmt.Sprintf("veo_video_%s_%d_%d.mp4", adkCtx.GetSessionID(), common.IncrementUploadCounter(adkCtx), i+1)
		objectName := fmt.Sprintf("%s/%s", outputDir, filename)

		// Upload to GCS
		bucket := gcsClient.Bucket(bucketName)
		obj := bucket.Object(objectName)
		writer := obj.NewWriter(ctx)
		writer.ContentType = "video/mp4"

		_, err = writer.Write(videoData)
		if err != nil {
			writer.Close()
			return nil, fmt.Errorf("failed to write video data: %w", err)
		}

		err = writer.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to close video writer: %w", err)
		}

		videoURI := fmt.Sprintf("gs://%s/%s", bucketName, objectName)
		videoURIs = append(videoURIs, videoURI)

		logger.Info("generateVideosWithADKContext: video generated and uploaded",
			"videoIndex", i+1,
			"videoURI", videoURI,
			"sessionID", adkCtx.GetSessionID(),
		)

		// Add individual video to media history
		common.AddToMediaHistory(adkCtx, "video_generated", map[string]interface{}{
			"video_index": i + 1,
			"video_uri":   videoURI,
			"filename":    filename,
		})
	}

	logger.Info("generateVideosWithADKContext: all videos generated successfully",
		"totalVideos", len(videoURIs),
		"sessionID", adkCtx.GetSessionID(),
	)

	return videoURIs, nil
}

// Global ADK Veo handlers instance
var GlobalADKVeoHandlers *ADKVeoHandlers

// InitializeGlobalADKVeoHandlers initializes the global ADK Veo handlers
func InitializeGlobalADKVeoHandlers(resourceManager *common.ADKResourceManager, config *common.Config) {
	GlobalADKVeoHandlers = NewADKVeoHandlers(resourceManager, config)
}

// Convenience wrapper functions for backward compatibility

// VeoTextToVideoHandlerADKWrapper wraps the ADK handler for MCP compatibility
func VeoTextToVideoHandlerADKWrapper(
	client *genai.Client,
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Create mock ADK context for backward compatibility
	// In production, this would be provided by the ADK framework
	adkCtx := common.NewMockInvocationContext("default_session", "default_user")
	
	if GlobalADKVeoHandlers == nil {
		return mcp.NewToolResultError("ADK Veo handlers not initialized"), nil
	}
	
	return GlobalADKVeoHandlers.VeoTextToVideoHandlerADK(client, ctx, adkCtx, request)
}

// VeoImageToVideoHandlerADKWrapper wraps the ADK handler for MCP compatibility
func VeoImageToVideoHandlerADKWrapper(
	client *genai.Client,
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Create mock ADK context for backward compatibility
	// In production, this would be provided by the ADK framework
	adkCtx := common.NewMockInvocationContext("default_session", "default_user")
	
	if GlobalADKVeoHandlers == nil {
		return mcp.NewToolResultError("ADK Veo handlers not initialized"), nil
	}
	
	return GlobalADKVeoHandlers.VeoImageToVideoHandlerADK(client, ctx, adkCtx, request)
}
