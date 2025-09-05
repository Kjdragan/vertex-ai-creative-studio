# Google Cloud Storage Integration Guide

**Date:** 2025-09-05  
**Purpose:** Comprehensive guide for programmatic GCS operations to fix media handling issues  
**Target Languages:** Python, Go  

## Overview

This guide documents best practices for Google Cloud Storage integration based on official documentation research. It addresses the specific issues identified in our media handling evaluation, focusing on bucket validation, user media uploads, and robust error handling.

## Authentication Methods

### 1. Application Default Credentials (ADC) - Recommended

**Python:**
```python
from google.cloud import storage

# Uses ADC automatically - no explicit credentials needed
client = storage.Client()
```

**Go:**
```go
import "cloud.google.com/go/storage"

// Uses ADC automatically
client, err := storage.NewClient(ctx)
if err != nil {
    log.Fatal(err)
}
```

### 2. Service Account Key File

**Python:**
```python
from google.cloud import storage

client = storage.Client.from_service_account_json('path/to/keyfile.json')
# OR
client = storage.Client()  # With GOOGLE_APPLICATION_CREDENTIALS env var
```

**Go:**
```go
import (
    "cloud.google.com/go/storage"
    "google.golang.org/api/option"
)

client, err := storage.NewClient(ctx, option.WithCredentialsFile("path/to/keyfile.json"))
```

### 3. Explicit Credentials Object

**Python:**
```python
from google.cloud import storage
from google.oauth2 import service_account

credentials = service_account.Credentials.from_service_account_file('path/to/keyfile.json')
client = storage.Client(credentials=credentials)
```

**Go:**
```go
import (
    "cloud.google.com/go/auth/credentials"
    "google.golang.org/api/option"
)

creds, err := credentials.DetectDefault(&credentials.DetectOptions{...})
client, err := storage.NewClient(ctx, option.WithAuthCredentials(creds))
```

## Bucket Operations

### 1. Bucket Validation and Existence Check

**Python:**
```python
def bucket_exists(client, bucket_name):
    """Check if a bucket exists and is accessible."""
    try:
        bucket = client.get_bucket(bucket_name)
        return True
    except Exception as e:
        print(f"Bucket {bucket_name} not accessible: {e}")
        return False

# Usage
if bucket_exists(client, "my-bucket"):
    print("Bucket is accessible")
```

**Go:**
```go
func bucketExists(ctx context.Context, client *storage.Client, bucketName string) bool {
    bucket := client.Bucket(bucketName)
    _, err := bucket.Attrs(ctx)
    return err == nil
}

// Usage
if bucketExists(ctx, client, "my-bucket") {
    fmt.Println("Bucket is accessible")
}
```

### 2. Smart Bucket Resolution (Fix for Issue #1)

**Python:**
```python
import os
from google.cloud import storage

def resolve_bucket(client, user_bucket=None):
    """Resolve bucket with fallback hierarchy."""
    # Priority 1: User-specified bucket (if exists)
    if user_bucket and bucket_exists(client, user_bucket):
        return user_bucket
    
    # Priority 2: Environment variable
    env_bucket = os.getenv('GENMEDIA_BUCKET')
    if env_bucket and bucket_exists(client, env_bucket):
        return env_bucket
    
    # Priority 3: Default fallback
    default_bucket = "supple-synapse-media"
    if bucket_exists(client, default_bucket):
        return default_bucket
    
    raise ValueError("No accessible bucket found")

# Usage in Veo handler
try:
    bucket_name = resolve_bucket(client, user_provided_bucket)
    print(f"Using bucket: {bucket_name}")
except ValueError as e:
    return {"error": str(e)}
```

**Go:**
```go
func resolveBucket(ctx context.Context, client *storage.Client, userBucket string) (string, error) {
    // Priority 1: User-specified bucket
    if userBucket != "" && bucketExists(ctx, client, userBucket) {
        return userBucket, nil
    }
    
    // Priority 2: Environment variable
    envBucket := os.Getenv("GENMEDIA_BUCKET")
    if envBucket != "" && bucketExists(ctx, client, envBucket) {
        return envBucket, nil
    }
    
    // Priority 3: Default fallback
    defaultBucket := "supple-synapse-media"
    if bucketExists(ctx, client, defaultBucket) {
        return defaultBucket, nil
    }
    
    return "", fmt.Errorf("no accessible bucket found")
}
```

### 3. Bucket Creation (If Needed)

**Python:**
```python
def create_bucket_if_not_exists(client, bucket_name, project_id, location="us-central1"):
    """Create bucket if it doesn't exist."""
    try:
        bucket = client.get_bucket(bucket_name)
        return bucket
    except Exception:
        # Bucket doesn't exist, create it
        bucket = client.bucket(bucket_name)
        bucket = client.create_bucket(bucket, project=project_id, location=location)
        return bucket
```

**Go:**
```go
func createBucketIfNotExists(ctx context.Context, client *storage.Client, bucketName, projectID, location string) error {
    bucket := client.Bucket(bucketName)
    
    // Check if bucket exists
    if _, err := bucket.Attrs(ctx); err == nil {
        return nil // Bucket exists
    }
    
    // Create bucket
    attrs := &storage.BucketAttrs{
        Location: location,
    }
    return bucket.Create(ctx, projectID, attrs)
}
```

## Object Upload Operations

### 1. Upload from Memory (Fix for Issue #2)

**Python:**
```python
def upload_bytes_to_gcs(client, bucket_name, object_name, data, content_type=None):
    """Upload bytes data to GCS."""
    try:
        bucket = client.bucket(bucket_name)
        blob = bucket.blob(object_name)
        
        if content_type:
            blob.content_type = content_type
            
        blob.upload_from_string(data)
        
        return f"gs://{bucket_name}/{object_name}"
    except Exception as e:
        raise Exception(f"Upload failed: {e}")

# Usage for user image uploads
def save_user_image(client, image_data, filename=None, bucket_name=None):
    """Save user-provided image data to GCS."""
    if not filename:
        filename = f"user_uploads/{uuid.uuid4()}.png"
    
    bucket_name = resolve_bucket(client, bucket_name)
    
    return upload_bytes_to_gcs(
        client, 
        bucket_name, 
        filename, 
        image_data, 
        content_type="image/png"
    )
```

**Go:**
```go
func uploadBytesToGCS(ctx context.Context, client *storage.Client, bucketName, objectName string, data []byte, contentType string) (string, error) {
    bucket := client.Bucket(bucketName)
    obj := bucket.Object(objectName)
    
    w := obj.NewWriter(ctx)
    if contentType != "" {
        w.ContentType = contentType
    }
    
    if _, err := w.Write(data); err != nil {
        w.Close()
        return "", fmt.Errorf("write failed: %v", err)
    }
    
    if err := w.Close(); err != nil {
        return "", fmt.Errorf("close failed: %v", err)
    }
    
    return fmt.Sprintf("gs://%s/%s", bucketName, objectName), nil
}
```

### 2. Upload from File

**Python:**
```python
def upload_file_to_gcs(client, bucket_name, source_file_path, destination_name):
    """Upload a file to GCS."""
    try:
        bucket = client.bucket(bucket_name)
        blob = bucket.blob(destination_name)
        
        blob.upload_from_filename(source_file_path)
        
        return f"gs://{bucket_name}/{destination_name}"
    except Exception as e:
        raise Exception(f"File upload failed: {e}")
```

**Go:**
```go
func uploadFileToGCS(ctx context.Context, client *storage.Client, bucketName, sourceFile, destName string) (string, error) {
    file, err := os.Open(sourceFile)
    if err != nil {
        return "", fmt.Errorf("open file failed: %v", err)
    }
    defer file.Close()
    
    bucket := client.Bucket(bucketName)
    obj := bucket.Object(destName)
    w := obj.NewWriter(ctx)
    
    if _, err := io.Copy(w, file); err != nil {
        w.Close()
        return "", fmt.Errorf("copy failed: %v", err)
    }
    
    if err := w.Close(); err != nil {
        return "", fmt.Errorf("close failed: %v", err)
    }
    
    return fmt.Sprintf("gs://%s/%s", bucketName, destName), nil
}
```

## Object Download Operations

### 1. Download to Memory

**Python:**
```python
def download_blob_to_memory(client, bucket_name, blob_name):
    """Download blob content to memory."""
    try:
        bucket = client.bucket(bucket_name)
        blob = bucket.blob(blob_name)
        
        content = blob.download_as_bytes()
        return content
    except Exception as e:
        raise Exception(f"Download failed: {e}")
```

**Go:**
```go
func downloadBlobToMemory(ctx context.Context, client *storage.Client, bucketName, blobName string) ([]byte, error) {
    bucket := client.Bucket(bucketName)
    obj := bucket.Object(blobName)
    
    r, err := obj.NewReader(ctx)
    if err != nil {
        return nil, fmt.Errorf("new reader failed: %v", err)
    }
    defer r.Close()
    
    data, err := io.ReadAll(r)
    if err != nil {
        return nil, fmt.Errorf("read failed: %v", err)
    }
    
    return data, nil
}
```

### 2. Download to File

**Python:**
```python
def download_blob_to_file(client, bucket_name, blob_name, destination_path):
    """Download blob to local file."""
    try:
        bucket = client.bucket(bucket_name)
        blob = bucket.blob(blob_name)
        
        blob.download_to_filename(destination_path)
        return destination_path
    except Exception as e:
        raise Exception(f"Download to file failed: {e}")
```

**Go:**
```go
func downloadBlobToFile(ctx context.Context, client *storage.Client, bucketName, blobName, destPath string) error {
    bucket := client.Bucket(bucketName)
    obj := bucket.Object(blobName)
    
    r, err := obj.NewReader(ctx)
    if err != nil {
        return fmt.Errorf("new reader failed: %v", err)
    }
    defer r.Close()
    
    f, err := os.Create(destPath)
    if err != nil {
        return fmt.Errorf("create file failed: %v", err)
    }
    defer f.Close()
    
    if _, err := io.Copy(f, r); err != nil {
        return fmt.Errorf("copy failed: %v", err)
    }
    
    return nil
}
```

## Error Handling and Retry Mechanisms

### 1. Common GCS Error Types

**Python:**
```python
from google.cloud import exceptions
from google.api_core import retry
import time

def handle_gcs_errors(operation_func, *args, **kwargs):
    """Generic error handler with retry logic."""
    max_retries = 3
    base_delay = 1
    
    for attempt in range(max_retries):
        try:
            return operation_func(*args, **kwargs)
        except exceptions.NotFound:
            raise Exception("Bucket or object not found")
        except exceptions.Forbidden:
            raise Exception("Access denied - check permissions")
        except exceptions.TooManyRequests:
            if attempt < max_retries - 1:
                delay = base_delay * (2 ** attempt)
                time.sleep(delay)
                continue
            raise Exception("Rate limit exceeded")
        except Exception as e:
            if attempt < max_retries - 1:
                delay = base_delay * (2 ** attempt)
                time.sleep(delay)
                continue
            raise Exception(f"Operation failed after {max_retries} attempts: {e}")
```

**Go:**
```go
import (
    "context"
    "time"
    "google.golang.org/api/googleapi"
)

func handleGCSErrors(ctx context.Context, operation func() error) error {
    maxRetries := 3
    baseDelay := time.Second
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        err := operation()
        if err == nil {
            return nil
        }
        
        // Check for specific error types
        if gerr, ok := err.(*googleapi.Error); ok {
            switch gerr.Code {
            case 404:
                return fmt.Errorf("bucket or object not found")
            case 403:
                return fmt.Errorf("access denied - check permissions")
            case 429:
                if attempt < maxRetries-1 {
                    delay := baseDelay * time.Duration(1<<attempt)
                    time.Sleep(delay)
                    continue
                }
                return fmt.Errorf("rate limit exceeded")
            }
        }
        
        // Generic retry for other errors
        if attempt < maxRetries-1 {
            delay := baseDelay * time.Duration(1<<attempt)
            time.Sleep(delay)
            continue
        }
        
        return fmt.Errorf("operation failed after %d attempts: %v", maxRetries, err)
    }
    
    return nil
}
```

### 2. Built-in Retry Decorators

**Python:**
```python
from google.api_core import retry

# Use built-in retry decorator
@retry.Retry(predicate=retry.if_transient_error)
def upload_with_retry(client, bucket_name, object_name, data):
    """Upload with automatic retry on transient errors."""
    bucket = client.bucket(bucket_name)
    blob = bucket.blob(object_name)
    blob.upload_from_string(data)
    return f"gs://{bucket_name}/{object_name}"
```

## Practical Implementation for Media Handling Issues

### 1. Enhanced Veo Handler with Bucket Resolution

**Python:**
```python
import os
from google.cloud import storage

class EnhancedVeoHandler:
    def __init__(self):
        self.storage_client = storage.Client()
    
    def process_image_to_video(self, image_uri=None, image_data=None, prompt="", user_bucket=None):
        """Process image to video with robust bucket handling."""
        try:
            # Resolve bucket
            bucket_name = self.resolve_bucket(user_bucket)
            
            # Handle user-uploaded image data
            if image_data and not image_uri:
                image_uri = self.save_user_image(image_data, bucket_name)
            
            # Validate image exists
            if not self.object_exists(image_uri):
                raise ValueError(f"Image not found: {image_uri}")
            
            # Process video generation
            video_uri = self.generate_video(image_uri, prompt, bucket_name)
            
            return {
                "success": True,
                "video_uri": video_uri,
                "bucket_used": bucket_name
            }
            
        except Exception as e:
            return {
                "success": False,
                "error": str(e),
                "suggestion": "Check bucket permissions and image accessibility"
            }
    
    def resolve_bucket(self, user_bucket=None):
        """Smart bucket resolution with fallbacks."""
        return resolve_bucket(self.storage_client, user_bucket)
    
    def save_user_image(self, image_data, bucket_name):
        """Save user image data to GCS."""
        filename = f"user_uploads/{uuid.uuid4()}.png"
        return upload_bytes_to_gcs(
            self.storage_client,
            bucket_name,
            filename,
            image_data,
            content_type="image/png"
        )
    
    def object_exists(self, gcs_uri):
        """Check if GCS object exists."""
        # Parse gs://bucket/object
        parts = gcs_uri.replace("gs://", "").split("/", 1)
        bucket_name, object_name = parts[0], parts[1]
        
        try:
            bucket = self.storage_client.bucket(bucket_name)
            blob = bucket.blob(object_name)
            return blob.exists()
        except Exception:
            return False
```

### 2. Go Implementation for MCP Server

**Go:**
```go
package main

import (
    "context"
    "fmt"
    "os"
    "cloud.google.com/go/storage"
)

type EnhancedVeoHandler struct {
    storageClient *storage.Client
    ctx           context.Context
}

func NewEnhancedVeoHandler(ctx context.Context) (*EnhancedVeoHandler, error) {
    client, err := storage.NewClient(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to create storage client: %v", err)
    }
    
    return &EnhancedVeoHandler{
        storageClient: client,
        ctx:           ctx,
    }, nil
}

func (h *EnhancedVeoHandler) ProcessImageToVideo(imageURI, imageData, prompt, userBucket string) map[string]interface{} {
    // Resolve bucket
    bucketName, err := h.resolveBucket(userBucket)
    if err != nil {
        return map[string]interface{}{
            "success": false,
            "error":   err.Error(),
        }
    }
    
    // Handle user-uploaded image data
    if imageData != "" && imageURI == "" {
        uri, err := h.saveUserImage([]byte(imageData), bucketName)
        if err != nil {
            return map[string]interface{}{
                "success": false,
                "error":   fmt.Sprintf("Failed to save user image: %v", err),
            }
        }
        imageURI = uri
    }
    
    // Validate image exists
    if !h.objectExists(imageURI) {
        return map[string]interface{}{
            "success": false,
            "error":   fmt.Sprintf("Image not found: %s", imageURI),
        }
    }
    
    // Process video generation (placeholder)
    videoURI := h.generateVideo(imageURI, prompt, bucketName)
    
    return map[string]interface{}{
        "success":     true,
        "video_uri":   videoURI,
        "bucket_used": bucketName,
    }
}

func (h *EnhancedVeoHandler) resolveBucket(userBucket string) (string, error) {
    return resolveBucket(h.ctx, h.storageClient, userBucket)
}
```

## Environment Configuration

### Required Environment Variables

```bash
# Primary bucket for media storage
export GENMEDIA_BUCKET=supple-synapse-media

# Google Cloud credentials (if not using ADC)
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json

# Project configuration
export GOOGLE_CLOUD_PROJECT=your-project-id
export GOOGLE_CLOUD_LOCATION=us-central1
```

### Validation Script

**Python:**
```python
def validate_gcs_setup():
    """Validate GCS configuration and permissions."""
    try:
        client = storage.Client()
        
        # Test bucket access
        bucket_name = os.getenv('GENMEDIA_BUCKET', 'supple-synapse-media')
        if not bucket_exists(client, bucket_name):
            print(f"❌ Bucket {bucket_name} not accessible")
            return False
        
        # Test upload permission
        test_blob = f"test_uploads/validation_{uuid.uuid4()}.txt"
        test_data = b"validation test"
        
        try:
            upload_bytes_to_gcs(client, bucket_name, test_blob, test_data)
            print(f"✅ Upload test successful")
            
            # Cleanup test file
            bucket = client.bucket(bucket_name)
            bucket.blob(test_blob).delete()
            
        except Exception as e:
            print(f"❌ Upload test failed: {e}")
            return False
        
        print("✅ GCS setup validation successful")
        return True
        
    except Exception as e:
        print(f"❌ GCS setup validation failed: {e}")
        return False

if __name__ == "__main__":
    validate_gcs_setup()
```

## Best Practices Summary

### 1. Authentication
- Use Application Default Credentials (ADC) in production
- Set `GOOGLE_APPLICATION_CREDENTIALS` for service account keys
- Validate credentials before operations

### 2. Bucket Management
- Always validate bucket existence before operations
- Implement fallback bucket resolution
- Use environment variables for configuration

### 3. Error Handling
- Implement exponential backoff for retries
- Handle specific GCS error codes appropriately
- Provide meaningful error messages to users

### 4. Performance
- Use streaming uploads/downloads for large files
- Implement connection pooling for high-throughput scenarios
- Consider parallel uploads for multiple files

### 5. Security
- Use IAM roles instead of service account keys when possible
- Implement least-privilege access
- Validate file types and sizes before upload

This guide provides the foundation for implementing robust GCS operations that will resolve the media handling issues identified in our system.
