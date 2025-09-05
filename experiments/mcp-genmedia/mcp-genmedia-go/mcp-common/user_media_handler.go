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
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

const (
	// MaxImageSize is the maximum allowed image size in bytes (10MB)
	MaxImageSize = 10 * 1024 * 1024
	// UserUploadsPath is the default path for user uploads in the bucket
	UserUploadsPath = "user_uploads"
)

// UserImageUpload represents a user-uploaded image with metadata
type UserImageUpload struct {
	Data        []byte
	MIMEType    string
	OriginalExt string
	Size        int64
}

// UploadResult contains the result of an image upload operation
type UploadResult struct {
	GCSUri      string
	BucketName  string
	ObjectName  string
	MIMEType    string
	Size        int64
	UploadedAt  time.Time
}

// ProcessInlineImageData processes base64-encoded inline image data and validates it.
// Supports data URLs in the format: data:image/jpeg;base64,<data> or raw base64 data.
func ProcessInlineImageData(inlineData string) (*UserImageUpload, error) {
	if inlineData == "" {
		return nil, fmt.Errorf("inline image data cannot be empty")
	}

	var base64Data string
	var mimeType string

	// Check if it's a data URL
	if strings.HasPrefix(inlineData, "data:") {
		parts := strings.SplitN(inlineData, ",", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid data URL format")
		}

		// Parse the data URL header
		header := parts[0]
		base64Data = parts[1]

		// Extract MIME type from header (e.g., "data:image/jpeg;base64")
		if strings.Contains(header, ":") && strings.Contains(header, ";") {
			mimeStart := strings.Index(header, ":") + 1
			mimeEnd := strings.Index(header, ";")
			if mimeStart < mimeEnd {
				mimeType = header[mimeStart:mimeEnd]
			}
		}
	} else {
		// Assume it's raw base64 data
		base64Data = inlineData
	}

	// Decode base64 data
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 image data: %w", err)
	}

	// Validate image size
	if len(imageData) > MaxImageSize {
		return nil, fmt.Errorf("image size %d bytes exceeds maximum allowed size of %d bytes", len(imageData), MaxImageSize)
	}

	if len(imageData) == 0 {
		return nil, fmt.Errorf("decoded image data is empty")
	}

	// Detect MIME type if not provided
	if mimeType == "" {
		mimeType = detectImageMIMEType(imageData)
		if mimeType == "" {
			return nil, fmt.Errorf("could not detect image MIME type")
		}
	}

	// Validate MIME type
	if !isValidImageMIMEType(mimeType) {
		return nil, fmt.Errorf("unsupported image MIME type: %s", mimeType)
	}

	// Get file extension
	ext := getExtensionFromMIMEType(mimeType)

	upload := &UserImageUpload{
		Data:        imageData,
		MIMEType:    mimeType,
		OriginalExt: ext,
		Size:        int64(len(imageData)),
	}

	log.Printf("ProcessInlineImageData: processed %s image, size: %d bytes", mimeType, len(imageData))
	return upload, nil
}

// UploadUserImage uploads a processed user image to GCS and returns the upload result.
func UploadUserImage(ctx context.Context, client *storage.Client, upload *UserImageUpload, bucketName string) (*UploadResult, error) {
	if upload == nil {
		return nil, fmt.Errorf("upload data cannot be nil")
	}

	if bucketName == "" {
		return nil, fmt.Errorf("bucket name cannot be empty")
	}

	// Generate unique object name
	objectName, err := generateUniqueObjectName(upload.OriginalExt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate unique object name: %w", err)
	}

	// Upload to GCS
	fullObjectName := fmt.Sprintf("%s/%s", UserUploadsPath, objectName)
	err = UploadToGCS(ctx, bucketName, fullObjectName, upload.MIMEType, upload.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to GCS: %w", err)
	}

	result := &UploadResult{
		GCSUri:     fmt.Sprintf("gs://%s/%s", bucketName, fullObjectName),
		BucketName: bucketName,
		ObjectName: fullObjectName,
		MIMEType:   upload.MIMEType,
		Size:       upload.Size,
		UploadedAt: time.Now(),
	}

	log.Printf("UploadUserImage: uploaded image to %s (size: %d bytes)", result.GCSUri, result.Size)
	return result, nil
}

// ProcessAndUploadInlineImage is a convenience function that processes inline image data
// and uploads it to GCS in a single operation.
func ProcessAndUploadInlineImage(ctx context.Context, client *storage.Client, inlineData, bucketName string) (*UploadResult, error) {
	// Process inline image data
	upload, err := ProcessInlineImageData(inlineData)
	if err != nil {
		return nil, fmt.Errorf("failed to process inline image: %w", err)
	}

	// Upload to GCS
	result, err := UploadUserImage(ctx, client, upload, bucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	return result, nil
}

// detectImageMIMEType detects the MIME type of image data by examining the file header
func detectImageMIMEType(data []byte) string {
	if len(data) < 12 {
		return ""
	}

	// Check for common image file signatures
	if len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}

	if len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}

	if len(data) >= 6 && string(data[0:6]) == "GIF87a" || string(data[0:6]) == "GIF89a" {
		return "image/gif"
	}

	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}

	return ""
}

// isValidImageMIMEType checks if the MIME type is supported for image uploads
func isValidImageMIMEType(mimeType string) bool {
	supportedTypes := []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	}

	for _, supported := range supportedTypes {
		if mimeType == supported {
			return true
		}
	}
	return false
}

// getExtensionFromMIMEType returns the appropriate file extension for a MIME type
func getExtensionFromMIMEType(mimeType string) string {
	extensions, err := mime.ExtensionsByType(mimeType)
	if err != nil || len(extensions) == 0 {
		// Fallback to manual mapping
		switch mimeType {
		case "image/jpeg":
			return ".jpg"
		case "image/png":
			return ".png"
		case "image/gif":
			return ".gif"
		case "image/webp":
			return ".webp"
		default:
			return ".bin"
		}
	}
	return extensions[0]
}

// generateUniqueObjectName generates a unique object name for uploaded files
func generateUniqueObjectName(extension string) (string, error) {
	// Generate random bytes for uniqueness
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Create timestamp-based prefix
	timestamp := time.Now().UTC().Format("2006-01-02_15-04-05")
	
	// Encode random bytes as hex
	randomHex := fmt.Sprintf("%x", randomBytes)
	
	// Combine timestamp and random hex
	filename := fmt.Sprintf("user_image_%s_%s%s", timestamp, randomHex[:16], extension)
	
	return filename, nil
}

// ValidateImageUpload performs comprehensive validation of image upload data
func ValidateImageUpload(upload *UserImageUpload) error {
	if upload == nil {
		return fmt.Errorf("upload data cannot be nil")
	}

	if len(upload.Data) == 0 {
		return fmt.Errorf("image data cannot be empty")
	}

	if upload.Size > MaxImageSize {
		return fmt.Errorf("image size %d exceeds maximum allowed size %d", upload.Size, MaxImageSize)
	}

	if !isValidImageMIMEType(upload.MIMEType) {
		return fmt.Errorf("unsupported MIME type: %s", upload.MIMEType)
	}

	// Verify that detected MIME type matches the data
	detectedType := detectImageMIMEType(upload.Data)
	if detectedType != "" && detectedType != upload.MIMEType {
		return fmt.Errorf("MIME type mismatch: declared %s, detected %s", upload.MIMEType, detectedType)
	}

	return nil
}
