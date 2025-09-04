# Google GenAI Media Master Repository - Current Status

**Last Updated:** September 4, 2025 13:36 UTC  
**Project:** MCP Server Environment Variables and OpenTelemetry Tracing Integration  
**Status:** ✅ FULLY OPERATIONAL - Environment Variable Warnings Eliminated

---

## Quick Context for New AI Assistants

This document serves as the **source of truth** for the current state of the Google GenAI Media Master Repository project. If you're a new AI assistant taking over this work, read this document first to understand where we currently stand.

## Current Project Focus

**Primary Objective:** Fix MCP Server Environment Variables and OpenTelemetry Tracing issues for stable media generation

**Key Achievement:** Successfully resolved all critical timeout and environment variable issues. **Latest:** Eliminated all environment variable warnings by ensuring all MCP servers receive all bucket path variables. System is now fully operational with clean startup logs.

## System Architecture Overview

### Core Components
- **ADK Web Server:** Running on `http://localhost:8000` via `start_adk.sh`
- **5 MCP Servers (Go-based):**
  - `mcp-imagen-go` (v1.10.0) - Image generation
  - `mcp-chirp3-go` (v0.1.0) - Text-to-speech (1,118 voices)
  - `mcp-veo-go` (v1.10.0) - Video generation  
  - `mcp-avtool-go` (v2.1.0) - Audio/video compositing
  - `mcp-lyria-go` (v1.3.0) - Music generation

### Key Directories
- **Main Project:** `/home/kjdrag/lrepos/google-genai-media-master-repo/`
- **MCP Servers:** `experiments/mcp-genmedia/sample-agents/adk/`
- **Documentation:** `KevinSideWork/Documentation_for_Projects/`

## Current System Status

### ✅ Working Features
- **Music Generation:** Lyria model generating music in ~23 seconds
- **GCS Upload:** Files saving to dedicated buckets (e.g., `gs://supple-synapse-lyria/`)
- **MCP Server Communication:** All 5 servers responding properly
- **Timeout Handling:** Extended to 180-300 seconds (was 55s)
- **Environment Variables:** Properly configured and passed to MCP servers

### 🟡 Minor Issues (Non-Critical)
- **MCP Session Cleanup:** Minor warnings during shutdown only
- **OpenTelemetry TracerProvider Override:** Single harmless warning during startup (expected behavior)

### 🚫 Known Limitations
- **OpenTelemetry Tracing:** Disabled (`ENABLE_OTEL_TRACING=false`) to prevent TracerProvider conflicts
- **Recitation Checks:** Some prompts may be blocked by content policy (use descriptive, non-generic prompts)

## Key Configuration Files

### Critical Files Modified
1. **`agent.py`** - MCP server configuration with timeouts and environment variables
2. **`export_env.sh`** - Environment variable export script
3. **`start_adk.sh`** - Server startup script
4. **`otel.go`** - OpenTelemetry configuration in Go servers

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

# Tracing (Currently Disabled)
ENABLE_OTEL_TRACING=false
ARIZE_SPACE_ID=U3BhY2U6ODU1MDp2L3Rp
ARIZE_API_KEY=ak-b78262b2-f932-43f7-a57f-7e5e5733ad2e-RWDT9Q8u6co9vx8j3Nr3xSUh2k-LJiuB
```

## Recent Major Changes (Last 7 Days)

### September 4, 2025
- **RESOLVED:** MCP server timeout issues (increased from 55s to 300s for Lyria)
- **RESOLVED:** OpenTelemetry context detachment errors (disabled tracing)
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
3. **OpenTelemetry:**
   - Disabled to eliminate context conflicts
   - Can be re-enabled after resolving TracerProvider issues

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
- Fix OpenTelemetry TracerProvider conflicts for re-enabling tracing
- Consider implementing health monitoring for MCP servers

### Long-term
- Implement health monitoring for MCP servers
- Add retry logic for transient failures
- Performance optimization for generation times

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

**Current Priority:** System is stable and operational with clean startup logs. Focus on:
1. Monitoring system health
2. Implementing new features or improvements
3. Maintaining this document and `lessonslearned.md` with any significant changes
4. Leveraging documented lessons to avoid repeating solved problems

**Key Success Metrics:**
- Music generation completing in <30 seconds
- No timeout errors
- Successful GCS uploads
- All 5 MCP servers responding
- Clean startup logs without environment variable warnings

**Contact Context:** User (kjdrag) is working on WSL environment, prefers environment variable configuration, and values permanent fixes over temporary workarounds.

---

*This document should be updated whenever significant changes are made to the system architecture, configuration, or status. Companion document `lessonslearned.md` contains technical coding lessons and framework insights.*
