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
	"strings"

	"cloud.google.com/go/storage"
)

// MCPInlineDataResult represents the result of processing inline data in MCP requests
type MCPInlineDataResult struct {
	IsInlineData bool
	GCSUri       string
	MIMEType     string
	UploadResult *UploadResult
	Error        error
}

// ProcessMCPImageParameter processes an image parameter from MCP requests.
// It detects whether the parameter contains inline data (base64) or a GCS URI,
// and handles each case appropriately.
// 
// For inline data: processes and uploads to GCS, returns the new GCS URI
// For GCS URIs: validates and returns the URI as-is
func ProcessMCPImageParameter(ctx context.Context, client *storage.Client, imageParam interface{}, bucketName string) *MCPInlineDataResult {
	result := &MCPInlineDataResult{
		IsInlineData: false,
	}
	
	// Extract string value from parameter
	imageStr, ok := imageParam.(string)
	if !ok || strings.TrimSpace(imageStr) == "" {
		result.Error = fmt.Errorf("image parameter must be a non-empty string")
		return result
	}
	
	imageStr = strings.TrimSpace(imageStr)
	
	// Check if it's a GCS URI
	if strings.HasPrefix(imageStr, "gs://") {
		log.Printf("ProcessMCPImageParameter: detected GCS URI: %s", imageStr)
		result.GCSUri = imageStr
		result.MIMEType = inferMIMETypeFromGCSURI(imageStr)
		return result
	}
	
	// Check if it's inline data (base64 or data URL)
	if isInlineImageData(imageStr) {
		log.Printf("ProcessMCPImageParameter: detected inline image data (length: %d)", len(imageStr))
		result.IsInlineData = true
		
		// Process and upload inline data
		uploadResult, err := ProcessAndUploadInlineImage(ctx, client, imageStr, bucketName)
		if err != nil {
			result.Error = fmt.Errorf("failed to process inline image data: %w", err)
			return result
		}
		
		result.GCSUri = uploadResult.GCSUri
		result.MIMEType = uploadResult.MIMEType
		result.UploadResult = uploadResult
		
		log.Printf("ProcessMCPImageParameter: uploaded inline data to %s", result.GCSUri)
		return result
	}
	
	// If it's neither GCS URI nor inline data, treat as error
	result.Error = fmt.Errorf("image parameter must be either a GCS URI (gs://) or inline image data (base64/data URL)")
	return result
}

// ProcessMCPImageParameterWithBucketResolution combines image parameter processing with smart bucket resolution.
// This is the recommended function for MCP handlers as it handles both bucket resolution and image processing.
func ProcessMCPImageParameterWithBucketResolution(ctx context.Context, client *storage.Client, imageParam interface{}, userBucket string) *MCPInlineDataResult {
	// Resolve bucket first
	bucketResult := ResolveBucket(ctx, client, userBucket)
	if !bucketResult.IsValid {
		return &MCPInlineDataResult{
			Error: fmt.Errorf("bucket resolution failed: %v", bucketResult.Error),
		}
	}
	
	log.Printf("ProcessMCPImageParameterWithBucketResolution: using bucket '%s' from source '%s'", 
		bucketResult.BucketName, bucketResult.Source)
	
	// Process image parameter with resolved bucket
	return ProcessMCPImageParameter(ctx, client, imageParam, bucketResult.BucketName)
}

// isInlineImageData determines if a string contains inline image data
func isInlineImageData(data string) bool {
	// Check for data URL format
	if strings.HasPrefix(data, "data:image/") {
		return true
	}
	
	// Check for raw base64 data (heuristic: long string with base64 characters)
	if len(data) > 100 && isLikelyBase64(data) {
		return true
	}
	
	return false
}

// isLikelyBase64 performs a heuristic check to determine if a string is likely base64 encoded
func isLikelyBase64(data string) bool {
	// Base64 strings should only contain these characters
	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
	
	// Check if at least 90% of characters are valid base64 characters
	validCount := 0
	for _, char := range data {
		for _, validChar := range validChars {
			if char == validChar {
				validCount++
				break
			}
		}
	}
	
	ratio := float64(validCount) / float64(len(data))
	return ratio >= 0.9
}

// inferMIMETypeFromGCSURI infers MIME type from a GCS URI based on file extension
func inferMIMETypeFromGCSURI(gcsURI string) string {
	if strings.HasSuffix(strings.ToLower(gcsURI), ".png") {
		return "image/png"
	}
	if strings.HasSuffix(strings.ToLower(gcsURI), ".jpg") || strings.HasSuffix(strings.ToLower(gcsURI), ".jpeg") {
		return "image/jpeg"
	}
	if strings.HasSuffix(strings.ToLower(gcsURI), ".gif") {
		return "image/gif"
	}
	if strings.HasSuffix(strings.ToLower(gcsURI), ".webp") {
		return "image/webp"
	}
	return ""
}

// CreateMCPImageProcessingSummary creates a summary of the image processing operation for logging
func CreateMCPImageProcessingSummary(result *MCPInlineDataResult, originalParam string) string {
	summary := "MCP Image Processing Summary:\n"
	summary += fmt.Sprintf("  Original parameter length: %d\n", len(originalParam))
	summary += fmt.Sprintf("  Is inline data: %t\n", result.IsInlineData)
	
	if result.Error != nil {
		summary += fmt.Sprintf("  Error: %v\n", result.Error)
	} else {
		summary += fmt.Sprintf("  Resolved GCS URI: %s\n", result.GCSUri)
		summary += fmt.Sprintf("  MIME type: %s\n", result.MIMEType)
		
		if result.UploadResult != nil {
			summary += fmt.Sprintf("  Upload size: %d bytes\n", result.UploadResult.Size)
			summary += fmt.Sprintf("  Upload time: %s\n", result.UploadResult.UploadedAt.Format("2006-01-02 15:04:05"))
		}
	}
	
	return summary
}
