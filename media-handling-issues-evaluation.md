# Media Handling Issues - Evaluation & Fix Plan

**Date:** 2025-09-05  
**Issue Type:** User Media Upload & File Handling Failures  
**Severity:** HIGH - System cannot process user-uploaded content

## Executive Summary

The agent encountered multiple critical failures when attempting to work with user-uploaded media and custom bucket configurations. The system failed at two key points:
1. **Bucket Resolution Issues** - Agent used non-existent buckets
2. **User Media Upload Handling** - Cannot save or process user-provided images

## Issue Analysis

### 🚨 Critical Issue #1: Bucket Configuration Problems

**Error Pattern:**
```
Bucket "genmedia-agent-yo-out" not found for operation OP_INITIATE_RESUMABLE_WRITE (code: 5)
Bucket "genmedia-agent-yo-in" not found for operation OP_INITIATE_READ_OBJECT (code: 5)
```

**Root Cause:**
- Agent hardcoded non-existent bucket names (`genmedia-agent-yo-out`, `genmedia-agent-yo-in`)
- Failed to use configured environment variable `GENMEDIA_BUCKET=supple-synapse-media`
- No fallback mechanism when custom buckets don't exist

### 🚨 Critical Issue #2: User Media Upload Handling

**Error Pattern:**
```
"I am sorry, I cannot save the image to the required bucket. Please upload the image to the `genmedia-agent-yo-in` bucket and I will try again."
```

**Root Cause:**
- Agent cannot process inline image data from user uploads
- No mechanism to save user-provided images to GCS
- Agent expects images to already exist in specific GCS locations

### ⚠️ Issue #3: MCP Session Instability

**Error Pattern:**
```
anyio.ClosedResourceError
asyncio.exceptions.CancelledError: Cancelled by cancel scope
```

**Root Cause:**
- MCP sessions being cancelled during operations
- Session cleanup errors causing instability
- Resource management issues during concurrent operations

## Technical Analysis

### Bucket Resolution Logic Flaws

1. **Hardcoded Bucket Names**: Agent generated bucket names like `genmedia-agent-yo-out` without validation
2. **Environment Variable Ignored**: `GENMEDIA_BUCKET` properly set but not used as fallback
3. **No Bucket Validation**: No pre-flight checks for bucket existence

### User Media Processing Gaps

1. **No Upload Handler**: Missing functionality to accept and store user images
2. **Inline Data Processing**: Cannot handle `inline_data` with `mime_type: image/png`
3. **Temporary Storage**: No mechanism for temporary user media storage

### Session Management Issues

1. **Resource Cleanup**: Improper cleanup causing `ClosedResourceError`
2. **Concurrent Operations**: Session conflicts during multiple tool calls
3. **Error Recovery**: No graceful recovery from session failures

## Comprehensive Fix Plan

### Phase 1: Bucket Management (Priority: CRITICAL)

#### 1.1 Implement Smart Bucket Resolution
```go
// Pseudo-code for bucket resolution logic
func resolveBucket(userBucket string) string {
    if userBucket != "" && bucketExists(userBucket) {
        return userBucket
    }
    
    defaultBucket := os.Getenv("GENMEDIA_BUCKET")
    if defaultBucket != "" && bucketExists(defaultBucket) {
        return defaultBucket
    }
    
    return "supple-synapse-media" // Final fallback
}
```

#### 1.2 Add Bucket Validation
- Pre-flight bucket existence checks
- Graceful fallback to environment variables
- Clear error messages for missing buckets

#### 1.3 Update Veo Handler Logic
- Remove hardcoded bucket names
- Use environment variable `GENMEDIA_BUCKET` as default
- Implement bucket validation before operations

### Phase 2: User Media Upload Support (Priority: CRITICAL)

#### 2.1 Implement Image Upload Handler
```python
# New MCP tool for handling user uploads
async def save_user_image(
    image_data: bytes,
    mime_type: str,
    filename: str = None,
    bucket: str = None
) -> str:
    """Save user-provided image data to GCS and return URI"""
    # Generate unique filename if not provided
    # Upload to specified or default bucket
    # Return gs:// URI for downstream processing
```

#### 2.2 Extend Agent Capabilities
- Add image upload preprocessing
- Support for inline_data handling
- Automatic conversion to GCS URIs

#### 2.3 Update Tool Descriptions
- Document user upload support
- Provide clear instructions for media handling
- Add examples for different input types

### Phase 3: Session Stability (Priority: HIGH)

#### 3.1 Improve Resource Management
- Proper async context management
- Graceful session cleanup
- Resource leak prevention

#### 3.2 Add Error Recovery
- Session reconnection logic
- Retry mechanisms for transient failures
- Better error propagation

#### 3.3 Concurrent Operation Handling
- Session pooling for multiple tools
- Operation queuing for resource conflicts
- Timeout management improvements

### Phase 4: Enhanced Error Handling (Priority: MEDIUM)

#### 4.1 User-Friendly Error Messages
- Clear explanations of bucket issues
- Helpful suggestions for resolution
- Step-by-step troubleshooting guides

#### 4.2 Diagnostic Tools
- Bucket validation commands
- Media processing status checks
- System health indicators

#### 4.3 Fallback Mechanisms
- Alternative processing paths
- Graceful degradation options
- Recovery suggestions

## Implementation Priority

### Immediate (Week 1)
1. **Fix bucket resolution logic** - Use environment variables properly
2. **Add bucket validation** - Prevent non-existent bucket errors
3. **Implement basic user image upload** - Core functionality for user media

### Short-term (Week 2-3)
1. **Improve session management** - Reduce MCP session errors
2. **Add comprehensive error handling** - Better user experience
3. **Create diagnostic tools** - Easier troubleshooting

### Long-term (Month 1)
1. **Advanced media processing** - Support multiple formats
2. **Performance optimization** - Faster upload/processing
3. **Enhanced monitoring** - Proactive issue detection

## Success Metrics

### Technical Metrics
- **Zero bucket-not-found errors** - 100% bucket resolution success
- **User upload success rate** - >95% for supported formats
- **Session stability** - <1% MCP session failures

### User Experience Metrics
- **Workflow completion rate** - >90% for user media workflows
- **Error recovery time** - <30 seconds average
- **User satisfaction** - Clear error messages and guidance

## Risk Assessment

### High Risk
- **Breaking existing workflows** - Changes to bucket logic
- **Performance impact** - Additional validation overhead
- **Compatibility issues** - MCP session changes

### Mitigation Strategies
- **Gradual rollout** - Feature flags for new functionality
- **Comprehensive testing** - All media types and scenarios
- **Rollback plan** - Quick revert capability

## Conclusion

The media handling issues stem from fundamental gaps in bucket management and user media processing. The fix plan addresses these systematically, prioritizing critical functionality while maintaining system stability.

**Key Success Factors:**
1. **Proper bucket resolution** - Use environment variables and validation
2. **User media support** - Handle inline uploads and processing
3. **Session stability** - Robust resource management
4. **Error handling** - Clear messages and recovery paths

**Estimated Timeline:** 2-3 weeks for core fixes, 1 month for complete solution

**Resource Requirements:** 1-2 developers, testing infrastructure, staging environment
