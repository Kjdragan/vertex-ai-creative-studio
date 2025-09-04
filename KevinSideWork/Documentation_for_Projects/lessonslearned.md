# Lessons Learned - Google GenAI Media Master Repository

*Last Updated: 2025-09-04*

This document serves as a technical reference and source of truth for coding lessons learned, framework insights, and solution patterns discovered during development. It helps prevent repeating solved problems and ensures consistent implementation patterns across the project.

## Table of Contents
1. [Environment Variable Management](#environment-variable-management)
2. [MCP Server Configuration](#mcp-server-configuration)
3. [OpenTelemetry & Tracing](#opentelemetry--tracing)
4. [Google Cloud Integration](#google-cloud-integration)
5. [Error Handling Patterns](#error-handling-patterns)
6. [Performance & Timeout Management](#performance--timeout-management)
7. [Security Best Practices](#security-best-practices)
8. [Development Workflow](#development-workflow)

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

## Google Cloud Integration

### ✅ **Lesson: GCS Bucket Path Patterns**
**Best Practice:** Use dedicated buckets per media type for organized storage:
- `gs://project-imagen` for images
- `gs://project-veo` for videos  
- `gs://project-lyria` for music
- `gs://project-chirp3` for TTS audio
- `gs://project-avtool` for composited media

**Key Insight:** Separate buckets improve organization, access control, and cost tracking.

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
- **Session Management:** Sessions are created automatically; focus on tool configuration
- **Error Handling:** Structured error responses work better than plain text errors

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

---

*This document should be updated whenever new technical lessons are learned or coding patterns are established. It serves as institutional knowledge for the project.*
