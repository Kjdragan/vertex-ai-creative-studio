# Media Handling Implementation Plan

**Date:** 2025-09-05  
**Status:** Implementation Ready  
**Priority:** CRITICAL - System cannot process user-uploaded content  

## Executive Summary

This plan implements fixes for the three critical media handling issues identified in our evaluation, incorporating best practices from the GCS Integration Guide research. The plan is structured with granular, testable tasks to ensure measurable progress.

## Issues to Address

1. **Bucket Configuration Failures** - Agent uses non-existent buckets instead of environment variables
2. **User Media Upload Handling** - Cannot process user-uploaded images with inline_data
3. **MCP Session Instability** - Resource management issues causing system crashes

## Implementation Phases

### Phase 1: Bucket Resolution System (Critical - Week 1)

#### Task 1.1: Create Bucket Validation Module
**Deliverable:** `bucket_validator.go` and `bucket_validator.py`
**Test Criteria:** 
- ✅ Function returns `true` for existing buckets
- ✅ Function returns `false` for non-existent buckets
- ✅ Handles authentication errors gracefully

**Implementation:**
```go
// bucket_validator.go
func BucketExists(ctx context.Context, client *storage.Client, bucketName string) bool
func ValidateBucketAccess(ctx context.Context, client *storage.Client, bucketName string) error
```

```python
# bucket_validator.py
def bucket_exists(client: storage.Client, bucket_name: str) -> bool
def validate_bucket_access(client: storage.Client, bucket_name: str) -> None
```

**Testing Tasks:**
- [ ] Test with existing bucket `supple-synapse-media`
- [ ] Test with non-existent bucket `fake-bucket-name`
- [ ] Test with bucket without permissions
- [ ] Test with invalid bucket names

#### Task 1.2: Implement Smart Bucket Resolution
**Deliverable:** `bucket_resolver.go` and `bucket_resolver.py`
**Test Criteria:**
- ✅ Uses user-specified bucket if exists and accessible
- ✅ Falls back to `GENMEDIA_BUCKET` environment variable
- ✅ Falls back to `supple-synapse-media` default
- ✅ Throws clear error if no buckets accessible

**Implementation:**
```go
func ResolveBucket(ctx context.Context, client *storage.Client, userBucket string) (string, error)
```

```python
def resolve_bucket(client: storage.Client, user_bucket: str = None) -> str
```

**Testing Tasks:**
- [ ] Test priority 1: Valid user bucket
- [ ] Test priority 2: Invalid user bucket, valid env bucket
- [ ] Test priority 3: Invalid user/env, valid default bucket
- [ ] Test error case: All buckets invalid

#### Task 1.3: Update Veo Handler Bucket Logic
**Deliverable:** Modified `veo.go` and `handlers.go`
**Test Criteria:**
- ✅ No more hardcoded bucket names
- ✅ Uses `ResolveBucket()` function
- ✅ Proper error messages for bucket issues

**Files to Modify:**
- `experiments/mcp-genmedia/mcp-genmedia-go/handlers.go`
- `experiments/mcp-genmedia/mcp-genmedia-go/veo.go`

**Testing Tasks:**
- [ ] Test Veo i2v with valid bucket resolution
- [ ] Test Veo i2v with invalid bucket (should fail gracefully)
- [ ] Test Veo t2v with bucket resolution
- [ ] Verify no `genmedia-agent-yo-out` references remain

#### Task 1.4: Environment Variable Integration
**Deliverable:** Updated environment handling
**Test Criteria:**
- ✅ `GENMEDIA_BUCKET` properly read and used
- ✅ Validation script confirms environment setup
- ✅ Clear error messages for missing environment variables

**Testing Tasks:**
- [ ] Test with `GENMEDIA_BUCKET=supple-synapse-media`
- [ ] Test with `GENMEDIA_BUCKET=invalid-bucket`
- [ ] Test with missing `GENMEDIA_BUCKET`
- [ ] Run validation script and confirm output

### Phase 2: User Media Upload System (Critical - Week 1-2)

#### Task 2.1: Create User Image Upload Handler
**Deliverable:** `user_media_handler.go` and `user_media_handler.py`
**Test Criteria:**
- ✅ Accepts byte data and MIME type
- ✅ Generates unique filenames
- ✅ Uploads to resolved bucket
- ✅ Returns GCS URI

**Implementation:**
```go
func SaveUserImage(ctx context.Context, client *storage.Client, imageData []byte, mimeType, bucketName string) (string, error)
func GenerateUniqueFilename(prefix, extension string) string
```

```python
def save_user_image(client: storage.Client, image_data: bytes, mime_type: str, bucket_name: str = None) -> str
def generate_unique_filename(prefix: str, extension: str) -> str
```

**Testing Tasks:**
- [ ] Test PNG image upload (valid data)
- [ ] Test JPEG image upload (valid data)
- [ ] Test invalid image data
- [ ] Test filename generation uniqueness
- [ ] Verify GCS URI format correctness

#### Task 2.2: Implement Inline Data Processing
**Deliverable:** `inline_data_processor.go` and MCP tool updates
**Test Criteria:**
- ✅ Parses `inline_data` from MCP requests
- ✅ Extracts MIME type and binary data
- ✅ Validates image format
- ✅ Integrates with upload handler

**Implementation:**
```go
type InlineDataRequest struct {
    DisplayName string `json:"display_name"`
    MimeType    string `json:"mime_type"`
    Data        []byte `json:"data"`
}

func ProcessInlineData(request InlineDataRequest) (string, error)
```

**Testing Tasks:**
- [ ] Test with PNG inline data
- [ ] Test with JPEG inline data
- [ ] Test with invalid MIME type
- [ ] Test with corrupted image data
- [ ] Test with oversized images

#### Task 2.3: Update Veo i2v Handler for User Uploads
**Deliverable:** Enhanced `veo_i2v` MCP tool
**Test Criteria:**
- ✅ Accepts both `image_uri` and `inline_data`
- ✅ Processes user uploads before video generation
- ✅ Maintains backward compatibility
- ✅ Clear error messages for upload failures

**MCP Tool Signature:**
```go
type VeoI2VRequest struct {
    ImageURI    string           `json:"image_uri,omitempty"`
    InlineData  *InlineDataRequest `json:"inline_data,omitempty"`
    Prompt      string           `json:"prompt"`
    Bucket      string           `json:"bucket,omitempty"`
    // ... other fields
}
```

**Testing Tasks:**
- [ ] Test with existing `image_uri` (backward compatibility)
- [ ] Test with `inline_data` only (new functionality)
- [ ] Test with both `image_uri` and `inline_data` (should prioritize URI)
- [ ] Test error handling for invalid uploads
- [ ] Verify video generation works with uploaded images

#### Task 2.4: Create Upload Validation System
**Deliverable:** `upload_validator.go` and validation rules
**Test Criteria:**
- ✅ File size limits enforced
- ✅ MIME type validation
- ✅ Image format verification
- ✅ Security checks for malicious content

**Implementation:**
```go
type UploadLimits struct {
    MaxFileSize   int64
    AllowedTypes  []string
    MaxDimensions ImageDimensions
}

func ValidateUpload(data []byte, mimeType string, limits UploadLimits) error
```

**Testing Tasks:**
- [ ] Test file size limits (5MB, 10MB, 50MB)
- [ ] Test allowed MIME types (image/png, image/jpeg)
- [ ] Test blocked MIME types (application/exe, text/html)
- [ ] Test image dimension limits
- [ ] Test malformed image headers

### Phase 3: Session Stability Improvements (High - Week 2)

#### Task 3.1: Implement Proper Resource Management
**Deliverable:** `session_manager.go` and cleanup handlers
**Test Criteria:**
- ✅ Proper context cancellation
- ✅ Resource cleanup on errors
- ✅ No memory leaks during operations
- ✅ Graceful shutdown handling

**Implementation:**
```go
type SessionManager struct {
    contexts map[string]context.Context
    cancels  map[string]context.CancelFunc
    mutex    sync.RWMutex
}

func (sm *SessionManager) CreateSession(id string) (context.Context, context.CancelFunc)
func (sm *SessionManager) CleanupSession(id string)
```

**Testing Tasks:**
- [ ] Test session creation and cleanup
- [ ] Test concurrent session handling
- [ ] Test cleanup on context cancellation
- [ ] Monitor memory usage during operations
- [ ] Test graceful shutdown scenarios

#### Task 3.2: Add Connection Pooling and Retry Logic
**Deliverable:** Enhanced MCP client management
**Test Criteria:**
- ✅ Connection reuse for multiple operations
- ✅ Automatic retry on transient failures
- ✅ Exponential backoff implementation
- ✅ Circuit breaker for persistent failures

**Implementation:**
```go
type MCPClientPool struct {
    clients map[string]*MCPClient
    config  PoolConfig
    mutex   sync.RWMutex
}

func (pool *MCPClientPool) GetClient(serverName string) (*MCPClient, error)
func (pool *MCPClientPool) RetryOperation(operation func() error, maxRetries int) error
```

**Testing Tasks:**
- [ ] Test connection reuse across operations
- [ ] Test retry on network failures
- [ ] Test exponential backoff timing
- [ ] Test circuit breaker activation
- [ ] Test pool cleanup and resource management

#### Task 3.3: Improve Error Recovery and Logging
**Deliverable:** Enhanced error handling and diagnostics
**Test Criteria:**
- ✅ Structured logging for debugging
- ✅ Error context preservation
- ✅ Recovery from partial failures
- ✅ Health check endpoints

**Implementation:**
```go
type ErrorContext struct {
    Operation   string
    SessionID   string
    Timestamp   time.Time
    StackTrace  string
    UserContext map[string]interface{}
}

func LogStructuredError(ctx context.Context, err error, operation string)
func RecoverFromPanic(ctx context.Context, operation string) error
```

**Testing Tasks:**
- [ ] Test error logging format and content
- [ ] Test panic recovery mechanisms
- [ ] Test health check endpoint responses
- [ ] Verify error context preservation
- [ ] Test diagnostic information collection

### Phase 4: Basic Validation (Optional - As Needed)

#### Task 4.1: Simple Validation Script
**Deliverable:** `scripts/validate_setup.go` - Basic environment validation
**Purpose:** Quick verification during development

**Simple validation features:**
```go
func validateEnvironment() error {
    // Check GENMEDIA_BUCKET env var
    // Test GCS credentials
    // Verify bucket access
}

func quickFunctionalTest() error {
    // Test bucket resolution
    // Test basic upload
    // Return success/failure
}
```

**Development Testing:**
- Manual testing during implementation
- Simple validation script for environment setup
- Basic functionality verification

## Implementation Timeline

### Week 1: Critical Fixes
**Days 1-2:** Bucket Resolution System (Tasks 1.1-1.4)
**Days 3-5:** User Media Upload System (Tasks 2.1-2.2)

### Week 2: Enhancement and Stability
**Days 1-2:** Complete User Upload System (Tasks 2.3-2.4)
**Days 3-5:** Session Stability Improvements (Tasks 3.1-3.3)

### Week 3: Polish and Documentation
**Days 1-2:** Basic validation and cleanup
**Days 3-5:** Documentation and deployment preparation

## Success Criteria

### Functional Requirements
- [ ] **Zero bucket-not-found errors** - 100% bucket resolution success
- [ ] **User upload support** - Handle inline_data from MCP requests
- [ ] **Session stability** - <1% MCP session failures
- [ ] **Backward compatibility** - Existing workflows unchanged

### Performance Requirements
- [ ] **Upload speed** - <5 seconds for images up to 10MB
- [ ] **Processing time** - No degradation in video generation speed
- [ ] **Memory usage** - No memory leaks during extended operations
- [ ] **Concurrent handling** - Support 10+ simultaneous uploads

### Quality Requirements
- [ ] **Error messages** - Clear, actionable error descriptions
- [ ] **Logging** - Structured logs for debugging
- [ ] **Documentation** - Updated API documentation
- [ ] **Testing** - 90%+ test coverage for new code

## Risk Mitigation

### High-Risk Areas
1. **Breaking existing workflows** - Comprehensive backward compatibility testing
2. **Performance degradation** - Benchmarking before/after implementation
3. **Security vulnerabilities** - Input validation and file type checking

### Rollback Plan
1. **Feature flags** - Ability to disable new functionality
2. **Database backups** - Configuration state preservation
3. **Quick revert** - Automated rollback scripts
4. **Monitoring** - Real-time health checks during deployment

## Dependencies

### External Dependencies
- Google Cloud Storage client libraries (latest versions)
- MCP protocol implementation updates
- Environment variable configuration

### Internal Dependencies
- Access to `supple-synapse-media` bucket
- Development/staging environment setup
- Testing infrastructure availability

## Deliverables Checklist

### Code Deliverables
- [ ] `bucket_validator.go` and `bucket_validator.py`
- [ ] `bucket_resolver.go` and `bucket_resolver.py`
- [ ] `user_media_handler.go` and `user_media_handler.py`
- [ ] `inline_data_processor.go`
- [ ] `session_manager.go`
- [ ] Updated MCP handlers (`veo.go`, `handlers.go`)
- [ ] Integration test suite
- [ ] Validation scripts

### Documentation Deliverables
- [ ] Updated API documentation
- [ ] Deployment guide
- [ ] Troubleshooting guide
- [ ] Performance tuning guide

### Testing Deliverables
- [ ] Unit test coverage reports
- [ ] Integration test results
- [ ] Performance benchmark results
- [ ] Security validation reports

## Next Steps

1. **Environment Setup** - Ensure development environment has proper GCS access
2. **Task Assignment** - Assign specific tasks to team members
3. **Sprint Planning** - Break tasks into daily sprints
4. **Progress Tracking** - Daily standup meetings to track completion
5. **Code Reviews** - Peer review process for all changes

This plan provides granular, testable tasks that will systematically resolve the media handling issues while maintaining system stability and performance.
