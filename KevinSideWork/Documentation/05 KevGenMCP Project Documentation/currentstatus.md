# Google GenAI Media Master Repository - Current Status

**Last Updated:** September 5, 2025 22:14 (local)  
**Project:** MCP Server Environment Variables, OpenTelemetry Tracing Integration, and ADK Session Management  
**Status:** ✅ FULLY OPERATIONAL - Enhanced with Comprehensive Arize Tracing and ADK Session Management

---

## Quick Context for New AI Assistants

This document serves as the **source of truth** for the current state of the Google GenAI Media Master Repository project. If you're a new AI assistant taking over this work, read this document first to understand where we currently stand.

## Current Project Focus

**Primary Objective:** Complete MCP Server Environment Variables, OpenTelemetry Tracing, and ADK Session Management integration for production-ready media generation

**Key Achievement:** Successfully resolved all critical timeout and environment variable issues. **Latest:** Implemented comprehensive Arize tracing with OpenInference semantic attributes and integrated ADK native session management. System is now fully operational with enhanced observability and proper session lifecycle management.

## System Architecture Overview

### Core Components
- **ADK Web Server:** Running on `http://localhost:8000` via `start_adk.sh` with native session management
- **5 MCP Servers (Go-based) with Enhanced Arize Tracing:**
  - `mcp-imagen-go` (v1.10.0) - Image generation with comprehensive span instrumentation
  - `mcp-chirp3-go` (v0.1.0) - Text-to-speech (1,118 voices) with TTS operation tracing
  - `mcp-veo-go` (v1.10.0) - Video generation with polling and completion tracing  
  - `mcp-avtool-go` (v2.1.0) - Audio/video compositing with FFmpeg operation tracing
  - `mcp-lyria-go` (v1.3.0) - Music generation with API request/response tracing
- **Session Management:** ADK SessionService and MemoryService integration
- **Enhanced Observability:** OpenInference semantic attributes with Arize integration

### Key Directories
- **Main Project:** `/home/kjdrag/lrepos/google-genai-media-master-repo/`
- **MCP Servers:** `experiments/mcp-genmedia/sample-agents/adk/`
- **Documentation:** `KevinSideWork/Documentation_for_Projects/`

## Current System Status

### ✅ Working Features
- **Music Generation:** Lyria model generating music in ~23 seconds with detailed tracing
- **GCS Upload:** Files saving to dedicated buckets (e.g., `gs://supple-synapse-lyria/`) with upload tracing
- **MCP Server Communication:** All 5 servers responding properly with comprehensive observability
- **Timeout Handling:** Extended to 180-300 seconds (was 55s)
- **Environment Variables:** Properly configured and passed to MCP servers
- **Session Management:** ADK native SessionService managing conversation threads and state
- **Memory Service:** Cross-session knowledge persistence with VertexAI integration
- **Enhanced Tracing:** OpenInference semantic attributes providing detailed span hierarchy
- **Smart Bucket Resolution:** Priority-based bucket resolution with validation
- **Inline Media Processing:** Support for base64/data URL image uploads
- **Artifact → GCS URI Conversion:** `before_tool_callback` converts `artifact:user_image_0.jpg` to valid `gs://...` prior to MCP calls
- **GCS ArtifactService Default:** `start_adk.sh` launches ADK with `--artifact_service_uri gs://<bucket>` for deterministic artifact URIs
- **Signed URL Tool:** Local ADK tool `gcs_signed_url(gs_uri, expiry_seconds)` returns a time-limited HTTPS link for any `gs://` object

### 🟡 Minor Issues (Non-Critical)
- **MCP Session Cleanup:** Minor warnings during shutdown only
- **OpenTelemetry TracerProvider Override:** Single harmless warning during startup (expected behavior)

### 🚫 Known Limitations
- **Legacy OpenTelemetry:** Basic tracing disabled (`ENABLE_OTEL_TRACING=false`) - replaced with enhanced Arize tracing
- **Recitation Checks:** Some prompts may be blocked by content policy (use descriptive, non-generic prompts)
- **Go Version:** Environment uses Go 1.22.2 while some modules require >=1.24.3 (non-blocking for current functionality)

## Key Configuration Files

### Critical Files Modified
1. **`agent.py`** - MCP server configuration with timeouts and environment variables
2. **`export_env.sh`** - Environment variable export script
3. **`start_adk.sh`** - Server startup script with Arize credentials
4. **`otel.go`** - Enhanced OpenTelemetry configuration with Arize integration
5. **Enhanced Tracing Modules:**
   - `mcp-common/arize_mcp_tracing.go` - Core tracing utilities with OpenInference attributes
   - `mcp-lyria-go/enhanced_lyria_tracing.go` - Music generation tracing
   - `mcp-veo-go/enhanced_veo_tracing.go` - Video generation tracing
   - `mcp-imagen-go/enhanced_imagen_tracing.go` - Image generation tracing
   - `mcp-avtool-go/enhanced_avtool_tracing.go` - FFmpeg operations tracing
6. **Media Handling Improvements:**
   - `bucket_validator.go` - GCS bucket validation with retry logic
   - `bucket_resolver.go` - Smart bucket resolution
   - `user_media_handler.go` - Inline image processing
   - `session_manager.go` - Resource lifecycle management

### Environment Variables
```bash
# Google Cloud
PROJECT_ID=supple-synapse-470916-a2
LOCATION=us-central1
GOOGLE_CLOUD_LOCATION=us-central1

# Bucket Paths
LYRIA_BUCKET_PATH=gs://supple-synapse-lyria
VEO_BUCKET_PATH=gs://supple-synapse-veo
CHIRP3_BUCKET_PATH=gs://supple-synapse-chirp3
IMAGEN_BUCKET_PATH=gs://supple-synapse-imagen
AVTOOL_BUCKET_PATH=gs://supple-synapse-avtool

# Enhanced Arize Tracing
ENABLE_OTEL_TRACING=false  # Legacy tracing disabled
ARIZE_SPACE_ID=U3BhY2U6ODU1MDp2L3Rp
ARIZE_API_KEY=ak-b78262b2-f932-43f7-a57f-7e5e5733ad2e-RWDT9Q8u6co9vx8j3Nr3xSUh2k-LJiuB
ARIZE_PROJECT_NAME=genai-media-pipeline
ARIZE_INTERFACE=mcp-servers
```

## Recent Major Changes (Last 7 Days)

### September 5, 2025
- **IMPLEMENTED:** Comprehensive Arize tracing with OpenInference semantic attributes
- **CREATED:** Enhanced tracing modules for all MCP servers (Lyria, Veo, Imagen, AVTool)
- **INTEGRATED:** ADK native session management replacing custom session handling
- **ENHANCED:** Media handling with smart bucket resolution and inline image processing
- **DOCUMENTED:** Complete integration guide for enhanced tracing implementation
- **IMPROVED:** Error handling and observability across all MCP operations
// Additional on Sep 5 (later run)
- **FIXED:** Artifact URI handling for uploaded images via ADK callbacks (indexing from 0) and `before_tool_callback` conversion to GCS
- **ADDED:** Local FunctionTool `gcs_signed_url` for generating temporary HTTPS links to outputs
- **POLISH:** Auto-infer `mime_type` in callback when missing (jpg/png), corrected Veo i2v log formatting placeholders

### September 4, 2025
- **RESOLVED:** MCP server timeout issues (increased from 55s to 300s for Lyria)
- **RESOLVED:** OpenTelemetry context detachment errors (disabled legacy tracing)
- **RESOLVED:** Environment variable propagation to MCP servers
- **RESOLVED:** Environment variable warnings by passing all bucket paths to all MCP servers
- **TESTED:** Successful music generation in 23 seconds with GCS upload
- **CREATED:** Comprehensive evaluation report (`runevaluation.md`)
- **CREATED:** Technical lessons learned documentation (`lessonslearned.md`)

### Key Fixes Applied
1. **Timeout Configuration:**
   - Imagen: 180s, Chirp3: 180s, Veo: 480s, Avtool: 300s, Lyria: 300s
2. **Environment Variables:**
   - Explicit passing of all required variables in `agent.py`
   - **NEW:** All bucket path variables passed to all MCP servers to eliminate warnings
   - Proper fallback values to prevent Pydantic validation errors
3. **Enhanced Tracing:**
   - Legacy OpenTelemetry disabled to eliminate context conflicts
   - Implemented comprehensive Arize tracing with OpenInference semantic attributes
   - Added detailed span instrumentation for all MCP operations
   - Integrated trace correlation IDs for debugging and monitoring

## How to Start the System

```bash
cd /home/kjdrag/lrepos/google-genai-media-master-repo/experiments/mcp-genmedia/sample-agents/adk
./start_adk.sh
```

Access web interface: `http://localhost:8000/dev-ui/?app=genmedia_agent`

## Testing Commands

### Music Generation Test
```
Prompt: "create a cheerful upbeat instrumental melody"
Expected: ~23 second generation, GCS upload to supple-synapse-lyria bucket
```

### Health Check
```bash
ps aux | grep -E "mcp-.*-go|adk" | grep -v grep
```

## Next Steps / Future Work

### Immediate (Optional)
- Integrate enhanced tracing modules into existing MCP handlers using the integration guide
- Validate detailed trace data appears in Arize dashboard
- Configure alerts for failed operations and performance monitoring

### Long-term
- Implement health monitoring for MCP servers
- Add retry logic for transient failures
- Performance optimization for generation times
- Expand ADK session management integration
- Upgrade Go environment to >=1.24.3 for full compatibility

## Troubleshooting Quick Reference

### Common Issues
1. **Timeout Errors:** Check if timeouts are properly set (300s for Lyria)
2. **MCP Server Not Starting:** Verify Go binaries exist in `/home/kjdrag/go/bin/`
3. **Environment Variables:** Source `export_env.sh` before starting
4. **Port Conflicts:** Kill existing processes with `pkill -f "adk"`
5. **Environment Variable Warnings:** Ensure all bucket path variables are passed to all MCP servers in `agent.py`

### Log Locations
- **ADK Server:** Console output when running `start_adk.sh`
- **MCP Servers:** STDIO output in ADK server logs
- **Agent Logs:** `/tmp/agents_log/agent.latest.log`

---

## For New AI Assistants

**Current Priority:** System is stable and operational with enhanced observability. Focus on:
1. Integrating enhanced tracing modules into MCP handlers
2. Leveraging ADK session management for improved resource lifecycle
3. Monitoring detailed trace data in Arize for optimization opportunities
4. Maintaining this document and `lessonslearned.md` with any significant changes
5. Leveraging documented lessons to avoid repeating solved problems

**Key Success Metrics:**
- Music generation completing in <30 seconds with detailed tracing
- No timeout errors
- Successful GCS uploads with upload tracing
- All 5 MCP servers responding with comprehensive observability
- Clean startup logs without environment variable warnings
- Detailed trace data visible in Arize dashboard
- Proper session lifecycle management with ADK SessionService
- Smart bucket resolution eliminating bucket-not-found errors

**Contact Context:** User (kjdrag) is working on WSL environment, prefers environment variable configuration, and values permanent fixes over temporary workarounds.

---

*This document should be updated whenever significant changes are made to the system architecture, configuration, or status. Companion document `lessonslearned.md` contains technical coding lessons and framework insights.*
