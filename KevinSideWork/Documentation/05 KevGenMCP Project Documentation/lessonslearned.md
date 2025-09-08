# Lessons Learned - Google GenAI Media Master Repository

*Last Updated: 2025-09-07*

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

## GenMedia Creative Studio (veo-app) — 2025-09-08

### ✅ Lesson: Signed URLs are required for browser rendering of GCS objects
Problem: The UI attempted to display `gs://` outputs by swapping to `https://storage.mtls.cloud.google.com/`, which failed in browsers due to authentication/CORS.

Solution: Generate V4 signed URLs server-side and render those. Preserve canonical `gs://` URIs for metadata and downstream processing (e.g., Gemini critique), but use display-ready HTTPS URLs for `<img>` sources.

### ✅ Lesson: Separate “source URI” from “display URL” in state
Maintain `gs://` in state for data lineage and logging. Add a parallel “display URLs” list derived at render time (signed or public). This keeps analytics clean while ensuring reliable UX.

### ✅ Lesson: Expect duplicate init logs under dev reload
Uvicorn’s reloader executes module-level initialization twice (parent + worker). Duplicate “Initiating Gemini client …” and Firebase init logs are expected and harmless in development.

### ✅ Lesson: Clarify timing metrics
If a single “generation_time” metric spans both image generation and critique, interpret it as total pipeline time. Consider separate metrics when needed.

### ✅ Lesson: CSP domains must cover storage endpoints
Ensure `img-src` and `media-src` include `https://storage.googleapis.com` (and any alternates) to allow signed URL fetches.

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
#### Veo Native Audio Shaping Prompt (Example)
Use this prompt when you want Veo 3 to generate and speak a specific line in its own native audio (no external TTS or overlays). This improves the chance that Veo produces aligned lips and an “alien” timbre, but exact verbatim speech is not guaranteed.

```
Create an 8-second, 16:9 video using the uploaded image artifact (use the latest item from session state `image_artifacts`, e.g., "artifact:user_image_0.jpg") with the Veo 3 model (veo-3.0-generate-001).

Audio policy (STRICT):
- Use ONLY the native, Veo-generated audio in this video.
- DO NOT call or use: chirp_tts, list_chirp_voices, lyria_generate_music, or any ffmpeg_* tool that mixes/overlays external audio.
- The ONLY allowed ffmpeg tool (if needed) is ffmpeg_get_media_info for verification.
- Do NOT add music, backing tracks, or any external/non-Veo audio.

Speech requirement (verbatim):
- The alien faces the camera; mouth motion clearly synchronized to speech.
- Speak EXACTLY the following line and nothing else:
  "Happy birthday to you... happy birthday to you,,, happy birthday dear Kevin, Happy birthday to you!"
- Timbre: deep, resonant, alien/monster tone; clear articulation (not whispering).
- No extra words or sounds.

Video parameters:
- Duration: 8 seconds
- Aspect ratio: 16:9
- Framing: Close-up on the alien facing the camera

Verification:
- After generation, call ffmpeg_get_media_info to confirm exactly 1 audio stream is present.
- Compare the spoken audio against the transcript. If it does not match EXACTLY, respond with:
  STATUS=NATIVE_AUDIO_MISMATCH
  MISMATCH_REASON=<brief reason>
  NATIVE_VIDEO_GS_URI=<gs://...>
  NATIVE_SIGNING=<call gcs_signed_url(gs_uri) and return {public_url, expires_at} if possible>
- DO NOT fall back to TTS unless I explicitly reply with the exact token: CONFIRM_TTS_FALLBACK

Deliverables (if match is exact):
- NATIVE_VIDEO_GS_URI=<gs://...>
- Call gcs_signed_url(gs_uri) and return: { public_url, expires_at }
- TOOLS_USED=<list>
- TRANSCRIPT_MATCH=TRUE
```

Notes
- __Why it helps__: Strongly biases the agent to rely only on Veo’s native audio so the voice matches the visual style/timbre, while enforcing a single audio stream.
- __Verification__: Use `ffmpeg_get_media_info` to ensure exactly 1 audio stream is present (AAC mono is common) and duration ~ video duration.
- __Limitations__: Veo-native speech is probabilistic; it may not be verbatim. Use TTS + combine when exact words are mandatory.
- __Related tools__: `veo_i2v`, `ffmpeg_get_media_info`, `gcs_signed_url()` (requires impersonation as documented above).

#### Field-Tested Prompt (2025-09-06)
Good example to shape Veo’s native audio without external TTS or overlays. Copy/paste as-is.

```
Create an 8-second, 16:9 video using the uploaded image artifact (use the latest item from session state `image_artifacts`, e.g., "artifact:user_image_0.jpg") with the Veo 3 model (veo-3.0-generate-001).

Audio policy (STRICT):
Use ONLY the native, Veo-generated audio in this video.
DO NOT call or use: chirp_tts, list_chirp_voices, lyria_generate_music, or any ffmpeg_* tool that mixes/overlays external audio.
The ONLY allowed ffmpeg tool (if needed) is ffmpeg_get_media_info for verification.
Do NOT add music, backing tracks, or any external/non-Veo audio.

Speech requirement (verbatim):
The alien faces the camera; mouth motion clearly synchronized to speech.
Speak EXACTLY the following line and nothing else: "Happy birthday to you... happy birthday to you,,, happy birthday dear Kevin, Happy birthday to you!"
Timbre: deep, resonant, alien/monster tone; clear articulation (not whispering).
No extra words or sounds.

Video parameters:
Duration: 8 seconds
Aspect ratio: 16:9
Framing: Close-up on the alien facing the camera

Verification:
After generation, call ffmpeg_get_media_info to confirm exactly 1 audio stream is present.
Compare the spoken audio against the transcript. If it does not match EXACTLY, respond with: STATUS=NATIVE_AUDIO_MISMATCH MISMATCH_REASON= NATIVE_VIDEO_GS_URI=gs://... NATIVE_SIGNING=<call gcs_signed_url(gs_uri) and return {public_url, expires_at} if possible>
DO NOT fall back to TTS unless I explicitly reply with the exact token: CONFIRM_TTS_FALLBACK

Deliverables (if match is exact):
NATIVE_VIDEO_GS_URI=gs://...
Call gcs_signed_url(gs_uri) and return: { public_url, expires_at }
TOOLS_USED=
TRANSCRIPT_MATCH=TRUE
```

Observed outcome snippet

```
STATUS=NATIVE_AUDIO_MISMATCH
MISMATCH_REASON=The Veo model's native audio generation is probabilistic and cannot be guaranteed to match the provided script verbatim. Although prompted with the exact text, the model may have ad-libbed or altered the speech. Programmatic verification of the spoken words is not possible.
NATIVE_VIDEO_GS_URI=gs://supple-synapse-media/veo_outputs/15520646249243698122/sample_0.mp4
NATIVE_SIGNING={"error": "Failed to generate signed URL. This is likely due to a security credential issue on the server side."}

ffmpeg_get_media_info (excerpt):
{
  "streams": [
    { "index": 0, "codec_type": "video", "width": 1280, "height": 720, "duration": "8.000000" },
    { "index": 1, "codec_type": "audio", "codec_name": "aac", "channels": 1, "sample_rate": "44100", "duration": "7.905011" }
  ],
  "format": { "duration": "8.000000", "nb_streams": 2 }
}
```

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

## 2025-09-06 — Handoff Snapshot and New Lessons

### Run Evaluation 2 Snapshot (Session `c60b1549-d001-4c18-b145-3eb0723a0914`)
- __Outcome__: Video and audio were successfully combined and uploaded.
- __Final object__: `gs://supple-synapse-media/xenomorph_singing_happy_birthday.mp4`
- __Console details__: https://console.cloud.google.com/storage/browser/_details/supple-synapse-media/xenomorph_singing_happy_birthday.mp4?project=supple-synapse-470916-a2
- __Authenticated viewer__: https://storage.cloud.google.com/supple-synapse-media/xenomorph_singing_happy_birthday.mp4
- __Signed URL__: Previously failed due to token-only ADC; IAM Service Account Credentials API is now enabled. The agent has been updated to use explicit IAM impersonation for signed URLs.

### Signed URL Generation via IAM Impersonation (Update)
- __Problem__: `gcs_signed_url()` failed with token-only ADC (error: needs a private key). Even with `SIGNED_URL_SERVICE_ACCOUNT` set and Token Creator granted, the library attempted local signing.
- __Fix__: Force IAM impersonation and pass impersonated credentials to `blob.generate_signed_url(...)`.
- __Code__: `experiments/mcp-genmedia/sample-agents/adk/genmedia_agent/agent.py` (function `gcs_signed_url`)

```python
from google.auth import default as google_auth_default
from google.auth.impersonated_credentials import Credentials as ImpersonatedCredentials
from google.cloud import storage
from datetime import timedelta

def gcs_signed_url(gs_uri: str, expiry_seconds: int = 3600) -> dict:
    signer_email = os.getenv("SIGNED_URL_SERVICE_ACCOUNT") or os.getenv("GOOGLE_IMPERSONATE_SERVICE_ACCOUNT")
    client = storage.Client()
    bucket_name, blob_path = gs_uri[len("gs://"):].split("/", 1)
    blob = client.bucket(bucket_name).blob(blob_path)

    creds, _ = google_auth_default(scopes=["https://www.googleapis.com/auth/cloud-platform"])
    imp_creds = ImpersonatedCredentials(
        source_credentials=creds,
        target_principal=signer_email,
        target_scopes=["https://www.googleapis.com/auth/cloud-platform"],
        lifetime=3600,
    )

    url = blob.generate_signed_url(
        version="v4",
        expiration=timedelta(seconds=int(expiry_seconds)),
        method="GET",
        service_account_email=signer_email,
        credentials=imp_creds,
    )
    return {"gs_uri": gs_uri, "public_url": url}
```

- __Requirements__:
  - Enable API: `iamcredentials.googleapis.com` (done)
  - Env: `SIGNED_URL_SERVICE_ACCOUNT=<sa>@<project>.iam.gserviceaccount.com`
  - Caller has `roles/iam.serviceAccountTokenCreator` on the signer SA
  - Restart ADK after enabling the API to ensure runtime picks it up: `experiments/mcp-genmedia/sample-agents/adk/start_adk.sh`

### Audio/Video Combine Preconditions (Callback Preflight)
- __Problem__: First combine used a pseudo audio path `artifact:chirp_tts_response.result.content[1]` causing "local input file ... does not exist".
- __Fixes in callback__: `experiments/mcp-genmedia/sample-agents/adk/genmedia_agent/simple_callback.py` (`before_tool_callback`)
  - Default `chirp_tts` to set `output_directory='.'` if not provided, so a local WAV is created.
  - Resolve `artifact:` audio placeholders by selecting the most recent `chirp_audio-*.wav`.
  - Validate local and `gs://` inputs and log warnings if missing.
  - Auto-set `output_gcs_bucket` from `GENMEDIA_BUCKET` for combine outputs.

### Next Steps Checklist (for new assistant)
- __Restart ADK__ to load code and API changes: `experiments/mcp-genmedia/sample-agents/adk/start_adk.sh`
- __Verify signed URL__:
  - Call `gcs_signed_url(gs_uri="gs://supple-synapse-media/xenomorph_singing_happy_birthday.mp4")`
  - Expect `{ public_url, expires_at }` in response.
- __Validate another i2v + TTS run__:
  - Ensure TTS produces a real file (callback sets it); combine should upload directly to GCS.
  - Agent should automatically return a clickable signed link for outputs.

### Files and Paths Touched (2025-09-06)
- `experiments/mcp-genmedia/sample-agents/adk/genmedia_agent/agent.py` — `gcs_signed_url` now uses explicit impersonation.
- `experiments/mcp-genmedia/sample-agents/adk/genmedia_agent/simple_callback.py` — TTS default output + combine preflight checks.
- `experiments/mcp-genmedia/sample-agents/adk/start_adk.sh` — sources `.env` and starts ADK with GCS ArtifactService.
- Reports:
  - `KevinSideWork/Documentation_for_Projects/run evaluation.md`
  - `KevinSideWork/Documentation_for_Projects/runEvaluation2.md` (includes Trace Addendum)

---

*This document should be updated whenever new technical lessons are learned or coding patterns are established. It serves as institutional knowledge for the project.*
