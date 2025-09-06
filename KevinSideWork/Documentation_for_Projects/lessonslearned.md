# Lessons Learned - Google GenAI Media Master Repository

*Last Updated: 2025-09-05*

This document serves as a technical reference and source of truth for coding lessons learned, framework insights, and solution patterns discovered during development. It helps prevent repeating solved problems and ensures consistent implementation patterns across the project.

## Table of Contents
1. [Environment Variable Management](#environment-variable-management)
2. [MCP Server Configuration](#mcp-server-configuration)
3. [OpenTelemetry & Tracing](#opentelemetry--tracing)
4. [ADK Session Management](#adk-session-management)
5. [Enhanced Arize Tracing](#enhanced-arize-tracing)
6. [Google Cloud Integration](#google-cloud-integration)
7. [Error Handling Patterns](#error-handling-patterns)
8. [Performance & Timeout Management](#performance--timeout-management)
9. [Security Best Practices](#security-best-practices)
10. [Development Workflow](#development-workflow)

---

## Environment Variable Management

### ✅ **Lesson: Pass All Environment Variables to All MCP Servers**
**Problem:** MCP servers were logging warnings like "Environment variable X_BUCKET_PATH not set or empty, using empty fallback" because each server only received its specific bucket path variable, but the Go code tried to read all bucket paths.

**Solution:** Pass all bucket path environment variables to all MCP servers in `agent.py`:
```python
env={
    "PROJECT_ID": os.getenv("PROJECT_ID"),
    "LOCATION": os.getenv("LOCATION"),
    "GENMEDIA_BUCKET": os.getenv("GENMEDIA_BUCKET"),
    "IMAGEN_BUCKET_PATH": os.getenv("IMAGEN_BUCKET_PATH"),
    "VEO_BUCKET_PATH": os.getenv("VEO_BUCKET_PATH"),
    "CHIRP3_BUCKET_PATH": os.getenv("CHIRP3_BUCKET_PATH"),
    "LYRIA_BUCKET_PATH": os.getenv("LYRIA_BUCKET_PATH"),
    "AVTOOL_BUCKET_PATH": os.getenv("AVTOOL_BUCKET_PATH"),
    # ... other variables
}
```

**Key Insight:** When Go MCP servers use a shared config module that reads multiple environment variables, all servers need access to all variables to avoid fallback warnings, even if they only use one specific variable.

### ✅ **Lesson: Environment Variable Export Pattern**
**Best Practice:** Always export environment variables in `export_env.sh` and define them in `.env`:
- Use consistent naming patterns (e.g., `SERVICE_BUCKET_PATH`)
- Include fallback values in Go code but avoid empty string fallbacks that trigger warnings
- Validate critical environment variables on startup

---

## MCP Server Configuration

### ✅ **Lesson: MCP Server Timeout Configuration**
**Problem:** MCP servers were timing out during media generation operations.

**Solution:** Set appropriate timeouts per server based on expected operation duration:
```python
# In agent.py MCPToolset configuration
timeout_seconds=180,  # For image generation
timeout_seconds=480,  # For video/music generation (longer operations)
```

**Key Insight:** Different media generation operations have vastly different completion times. Music and video generation require much longer timeouts than image generation.

### ✅ **Lesson: MCP Server Environment Variable Propagation**
**Pattern:** Always explicitly pass environment variables to MCP server processes rather than relying on inheritance:
```python
server_params=StdioServerParameters(
    command="/path/to/mcp-server",
    args=[],
    env={
        # Explicitly define all required environment variables
    }
)
```

---

## OpenTelemetry & Tracing

### ✅ **Lesson: OpenTelemetry TracerProvider Override Conflicts**
**Problem:** Multiple TracerProvider registrations caused override warnings and context detachment errors.

**Solution:** Add environment variable toggle to disable tracing when conflicts occur:
```bash
export ENABLE_OTEL_TRACING=false
```

**Go Implementation:**
```go
func setupTracing() {
    if os.Getenv("ENABLE_OTEL_TRACING") != "true" {
        log.Println("OpenTelemetry tracing disabled via ENABLE_OTEL_TRACING")
        return
    }
    // ... tracing setup
}
```

**Key Insight:** OpenTelemetry can conflict with existing tracing setups. Always provide a way to disable it cleanly.

### ✅ **Lesson: Arize Integration Headers**
**Pattern:** When integrating with Arize for tracing, ensure proper header configuration:
```go
headers := map[string]string{
    "authorization": fmt.Sprintf("Bearer %s", apiKey),
    "api_key":       apiKey,
    "space_id":      spaceID,
    "arize-space-id": spaceID,
    "arize-interface": interfaceType,
}
```

---

## ADK Session Management

### ✅ **Lesson: ADK Native Session Management Integration**
**Problem:** Custom session management was handling resource lifecycle but not leveraging ADK's built-in session capabilities.

**Solution:** Integrate with ADK's native SessionService and MemoryService:
```python
# In ADK configuration
session_service = VertexAiSessionService(project_id=project_id, location=location)
memory_service = VertexAiMemoryBankService(project_id=project_id, location=location)
```

**Key Components:**
- **SessionService**: Manages conversation threads and session lifecycle
- **MemoryService**: Provides cross-session knowledge persistence
- **InvocationContext**: Provides session context during agent execution

**Key Insight:** ADK's session management provides better integration with the agent lifecycle and enables cross-session memory persistence.

### ✅ **Lesson: Resource Management within ADK Sessions**
**Pattern:** Adapt custom resource management to work within ADK's session framework:
```go
// Maintain resource cleanup capabilities within ADK session lifecycle
type SessionResourceManager struct {
    sessionID string
    resources map[string]func() error
    timeout   time.Duration
}

func (srm *SessionResourceManager) RegisterCleanup(resourceID string, cleanup func() error) {
    srm.resources[resourceID] = cleanup
}
```

**Key Insight:** Resource management should complement ADK sessions rather than replace them.

---

## Enhanced Arize Tracing

### ✅ **Lesson: OpenInference Semantic Attributes**
**Problem:** Basic tracing lacked detail and standardized attributes for LLM/AI operations.

**Solution:** Implement OpenInference semantic conventions:
```go
span.SetAttributes(
    attribute.String("openinference.span.kind", "TOOL"),
    attribute.String("input.value", userPrompt),
    attribute.String("output.value", generatedContent),
    attribute.String("tool_call.function.name", toolName),
    attribute.String("tool_call.function.arguments", parametersJSON),
)
```

**Key Attributes:**
- `openinference.span.kind`: TOOL, LLM, CHAIN, AGENT, RETRIEVER, EMBEDDING
- `input.value`: User input or prompt
- `output.value`: Generated content or result
- `tool_call.function.*`: Tool execution details

### ✅ **Lesson: MCP-Specific Tracing Attributes**
**Pattern:** Add MCP-specific context to spans:
```go
span.SetAttributes(
    attribute.String("mcp.tool.name", toolName),
    attribute.String("mcp.server.name", serverName),
    attribute.String("mcp.operation", operationType),
    attribute.String("mcp.parameters", parametersJSON),
    attribute.String("mcp.result", resultURI),
)
```

### ✅ **Lesson: Processing Stage Tracking**
**Pattern:** Track detailed processing stages for better observability:
```go
// Update processing stage throughout operation
span.SetAttributes(attribute.String("processing.stage", "bucket_resolution"))
// ... bucket resolution logic
span.SetAttributes(attribute.String("processing.stage", "api_request"))
// ... API call
span.SetAttributes(attribute.String("processing.stage", "gcs_upload"))
// ... upload logic
```

**Key Stages:**
- `preparation`: Initial setup and validation
- `bucket_resolution`: Smart bucket resolution
- `api_request`: External API calls
- `response_processing`: Processing API responses
- `gcs_upload`: File upload operations
- `completion`: Final result processing
- `cleanup`: Resource cleanup

### ✅ **Lesson: Error Context in Tracing**
**Pattern:** Record detailed error context with processing stage:
```go
func RecordError(span trace.Span, err error, stage string) {
    span.SetAttributes(
        attribute.String("mcp.error", err.Error()),
        attribute.String("processing.status", "failed"),
        attribute.String("processing.stage", stage),
    )
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
}
```

**Key Insight:** Error context with processing stage enables faster debugging and root cause analysis.

### ✅ **Lesson: Trace Correlation IDs**
**Pattern:** Extract trace and span IDs for log correlation:
```go
func GetTraceInfo(ctx context.Context) (string, string) {
    span := trace.SpanFromContext(ctx)
    if span.SpanContext().IsValid() {
        return span.SpanContext().TraceID().String(), span.SpanContext().SpanID().String()
    }
    return "", ""
}

// Use in logging
traceID, spanID := tracer.GetTraceInfo(ctx)
log.Printf("Operation started - TraceID: %s, SpanID: %s", traceID, spanID)
```

**Key Insight:** Trace correlation IDs enable linking logs, metrics, and traces for comprehensive debugging.

---

## Google Cloud Integration

### ✅ **Lesson: GCS Bucket Path Patterns**
**Best Practice:** Use dedicated buckets per media type for organized storage:
- `gs://project-imagen` for images
- `gs://project-veo` for videos  
- `gs://project-lyria` for music
- `gs://project-chirp3` for TTS audio
- `gs://project-avtool` for composited media

**Key Insight:** Separate buckets improve organization, access control, and cost tracking.

### ✅ **Lesson: Smart Bucket Resolution**
**Problem:** Hardcoded bucket names caused bucket-not-found errors when buckets didn't exist.

**Solution:** Implement priority-based bucket resolution with validation:
```go
type BucketResolver struct {
    validator *BucketValidator
}

func (br *BucketResolver) ResolveBucket(userBucket, envBucket, defaultBucket string) (string, error) {
    // Priority: user-specified > environment > default
    candidates := []string{userBucket, envBucket, defaultBucket}
    
    for _, bucket := range candidates {
        if bucket != "" {
            if err := br.validator.ValidateBucket(bucket); err == nil {
                return bucket, nil
            }
        }
    }
    return "", errors.New("no valid bucket found")
}
```

**Key Insight:** Smart bucket resolution with validation eliminates runtime bucket errors.

### ✅ **Lesson: Inline Media Processing**
**Problem:** MCP handlers only accepted GCS URIs, limiting user experience with uploaded images.

**Solution:** Support both GCS URIs and inline base64/data URL images:
```go
func ProcessImageInput(input string) (gcsURI string, err error) {
    if strings.HasPrefix(input, "gs://") {
        return input, nil // Already a GCS URI
    }
    
    if strings.HasPrefix(input, "data:") || isBase64(input) {
        return uploadInlineImage(input) // Process and upload inline data
    }
    
    return "", errors.New("invalid image input format")
}
```

**Key Insight:** Supporting multiple input formats improves user experience without breaking existing functionality.

### ✅ **Lesson: ADK Artifact → GCS URI Conversion in Callbacks**
**Problem:** MCP tools reject `artifact:` URIs (require `gs://`). LLM sometimes referenced `artifact:user_image_0.jpg` directly.

**Solution:** Implement `before_tool_callback` to resolve `artifact:` references to `gs://` using ADK's `ArtifactService` and session context. Include robust fallbacks and auto-infer `mime_type` if missing (`.jpg`/`.jpeg` → `image/jpeg`, `.png` → `image/png`).

```python
# simple_callback.py (excerpt)
if 'image_uri' in args and str(args['image_uri']).startswith('artifact:'):
    gcs_uri = await handler.load_artifact_as_gcs_uri(tool_context, artifact_filename)
    if gcs_uri:
        args['image_uri'] = gcs_uri
        if not args.get('mime_type'):
            # infer from extension
            ...
```

**Key Insight:** Doing this at the callback layer guarantees MCP tools always receive valid `gs://` URIs while keeping prompts simple and artifact-centric.

### ✅ **Lesson: Default to GCS ArtifactService at Startup**
**Pattern:** Start ADK with `--artifact_service_uri gs://<bucket>` to ensure deterministic artifact URIs without requiring inline uploads.

```bash
# start_adk.sh
uv run adk web --artifact_service_uri "gs://${GENMEDIA_ARTIFACT_BUCKET}"
```

**Key Insight:** GCS-backed artifacts simplify resolution and make callback logic more reliable (list_versions, versioned paths), while still allowing a fallback for in-memory services.

### ✅ **Lesson: Signed URLs for Browser-Friendly Links**
**Problem:** Users want clickable HTTPS links instead of `gs://` URIs.

**Solution:** Add a local ADK `FunctionTool` `gcs_signed_url(gs_uri, expiry_seconds=3600)` that returns a short-lived HTTPS URL using GCS V4 signed URLs.

```python
# agent.py (excerpt)
from google.adk.tools import FunctionTool

def gcs_signed_url(gs_uri: str, expiry_seconds: int = 3600) -> dict:
    ... # returns {gs_uri, public_url, expires_at}

tools=[..., FunctionTool(gcs_signed_url)]
```

**Security Note:** Signed URLs expire automatically; prefer them over public buckets. Ensure `google-cloud-storage` is installed in the ADK runtime.

### ✅ **Lesson: Service Account Authentication**
**Pattern:** Use `GOOGLE_APPLICATION_CREDENTIALS` environment variable pointing to service account JSON:
```bash
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
```

**Security Note:** Never commit service account files to version control. Use environment variables and secure credential management.

---

## Error Handling Patterns

### ✅ **Lesson: Go Error Logging with Fallbacks**
**Pattern:** When implementing fallback values in Go, avoid empty string fallbacks that trigger unnecessary warnings:
```go
func GetEnv(key, fallback string) string {
    if value, exists := os.LookupEnv(key); exists && value != "" {
        return value
    }
    if fallback != "" {
        log.Printf("Environment variable %s not set, using fallback: %s", key, fallback)
    }
    return fallback
}
```

**Key Insight:** Only log fallback warnings when the fallback value is meaningful, not for expected missing variables.

### ✅ **Lesson: MCP Server Error Propagation**
**Pattern:** Ensure MCP servers return structured error responses that can be properly handled by the ADK framework:
```go
return &types.CallToolResult{
    Content: []types.Content{{
        Type: "text",
        Text: fmt.Sprintf("Error: %v", err),
    }},
    IsError: true,
}
```

---

## Performance & Timeout Management

### ✅ **Lesson: Media Generation Timeout Scaling**
**Guideline:** Set timeouts based on media complexity:
- **Images (Imagen):** 180 seconds
- **Audio (Chirp3/TTS):** 180 seconds  
- **Music (Lyria):** 480 seconds (complex generation)
- **Video (Veo):** 480 seconds (complex generation)
- **AV Compositing:** 300 seconds

**Key Insight:** Generative AI operations have unpredictable completion times. Always err on the side of longer timeouts for production systems.

### ✅ **Lesson: Voice Caching for TTS**
**Pattern:** Cache voice lists on startup to avoid repeated API calls:
```go
// Cache voices during server initialization
voices, err := fetchAvailableVoices()
if err == nil {
    log.Printf("Found and cached %d voices.", len(voices))
    cachedVoices = voices
}
```

---

## Security Best Practices

### ✅ **Lesson: API Key Management**
**Pattern:** Use environment variables for all API keys and sensitive data:
```bash
export ARIZE_API_KEY="your-key-here"
export ARIZE_SPACE_ID="your-space-id"
```

**Never:** Hardcode API keys in source code, even for development.

### ✅ **Lesson: GCS Access Control**
**Best Practice:** Use IAM roles and service accounts rather than bucket-level permissions for fine-grained access control.

---

## Development Workflow

### ✅ **Lesson: Documentation as Source of Truth**
**Pattern:** Maintain living documentation that serves as context for new AI assistants:
- `currentstatus.md` - Current project state and configuration
- `lessonslearned.md` - Technical lessons and coding patterns
- `runevaluation.md` - System evaluation and performance metrics

**Key Insight:** Well-maintained documentation enables seamless handoffs between AI assistants and team members.

### ✅ **Lesson: Environment Setup Scripts**
**Pattern:** Use shell scripts for consistent environment setup:
```bash
# export_env.sh
export PROJECT_ID="your-project"
export LOCATION="us-central1"
# ... all environment variables
```

**Best Practice:** Source the script before running the application: `source export_env.sh && ./start_adk.sh`

### ✅ **Lesson: Testing Media Generation**
**Pattern:** Test with simple prompts first to verify system functionality:
- "create a happy melody" (music)
- "generate a sunset image" (image)
- "create a short video of waves" (video)

**Key Insight:** Simple test cases help isolate configuration issues from prompt complexity issues.

---

## Framework-Specific Insights

### ADK (Agent Development Kit)
- **MCPToolset Configuration:** Always specify explicit timeouts and environment variables
- **Session Management:** Use native SessionService and MemoryService for proper lifecycle management
- **Error Handling:** Structured error responses work better than plain text errors
- **Resource Management:** Integrate custom resource cleanup with ADK session lifecycle
- **Cross-Session Memory:** Leverage MemoryService for knowledge persistence across conversations

### Go MCP Servers
- **Initialization:** Global client initialization should happen once during server startup
- **Error Logging:** Use structured logging with consistent format across all servers
- **Environment Variables:** Use shared config modules but ensure all servers get required variables

### Google Vertex AI
- **Model Selection:** Different models have different capabilities and limits
- **Timeout Handling:** Vertex AI operations can be slow; always set appropriate timeouts
- **Error Responses:** Parse structured error responses for better user feedback

---

## Common Pitfalls to Avoid

1. **❌ Don't:** Rely on environment variable inheritance for MCP servers
   **✅ Do:** Explicitly pass all required environment variables

2. **❌ Don't:** Use empty string fallbacks that trigger warnings
   **✅ Do:** Use meaningful fallbacks or handle missing variables gracefully

3. **❌ Don't:** Set uniform timeouts for all media generation operations
   **✅ Do:** Scale timeouts based on operation complexity

4. **❌ Don't:** Hardcode API keys or bucket names
   **✅ Do:** Use environment variables for all configuration

5. **❌ Don't:** Ignore OpenTelemetry conflicts
   **✅ Do:** Provide toggle mechanisms for tracing systems

6. **❌ Don't:** Use basic tracing without semantic attributes
   **✅ Do:** Implement OpenInference semantic conventions for LLM/AI operations

7. **❌ Don't:** Hardcode bucket names in MCP handlers
   **✅ Do:** Use smart bucket resolution with validation

8. **❌ Don't:** Only support GCS URIs for media inputs
   **✅ Do:** Support both GCS URIs and inline base64/data URL processing

9. **❌ Don't:** Implement custom session management when ADK provides native solutions
   **✅ Do:** Integrate with ADK SessionService and MemoryService

---

*This document should be updated whenever new technical lessons are learned or coding patterns are established. It serves as institutional knowledge for the project.*
