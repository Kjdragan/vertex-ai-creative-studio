# Model Configuration Exhibit - ADK GenMedia Project

## Overview
This exhibit documents all locations where AI models are configured across the ADK GenMedia project. The system uses a distributed configuration approach with different models configured in different layers.

## Configuration Hierarchy

### 1. **Agent LLM Model Configuration**

**Location**: `/experiments/mcp-genmedia/sample-agents/adk/genmedia_agent/agent.py`
**Line**: 97
```python
root_agent = LlmAgent(
    model=os.getenv("AGENT_MODEL", "gemini-2.0-flash"),  # ← AGENT MODEL FROM ENV
    name='genmedia_agent',
    instruction="""...""",
    tools=[imagen, chirp3, veo, avtool],
)
```

**Environment Configuration**: `/experiments/mcp-genmedia/sample-agents/adk/genmedia_agent/.env`
**Line**: 12
```bash
AGENT_MODEL=gemini-2.5-pro
```

**Purpose**: Sets the primary reasoning model for the agent
**Current Value**: `gemini-2.5-pro`
**Centralized**: ✅ Yes - configurable via environment variable

---

### 2. **Imagen Model Configurations**

#### **Default Model Location**
**Location**: `/experiments/mcp-genmedia/mcp-genmedia-go/mcp-common/models.go`
**Lines**: 35-56
```go
var SupportedImagenModels = map[string]ImagenModelInfo{
    "imagen-4.0-generate-001": {  // ← DEFAULT IMAGEN MODEL
        CanonicalName: "imagen-4.0-generate-001",
        MaxImages:     4,
        Aliases:       []string{"Imagen 4", "Imagen4"},
    },
    "imagen-3.0-generate-002": {
        CanonicalName: "imagen-3.0-generate-002",
        MaxImages:     4,
        Aliases:       []string{"Imagen 3"},
    },
    // ... more models
}
```

**Purpose**: Central registry of all supported Imagen models and their capabilities
**Default**: `imagen-4.0-generate-001` (first in map, used as default)
**Centralized**: ✅ Yes - single source of truth for all Imagen models

#### **Available Models**
- `imagen-4.0-generate-001` (Max Images: 4, Aliases: "Imagen 4", "Imagen4") **← DEFAULT**
- `imagen-3.0-generate-002` (Max Images: 4, Aliases: "Imagen 3")
- `imagen-4.0-fast-generate-001` (Max Images: 4, Aliases: "Imagen 4 Fast", "Imagen4 Fast")
- `imagen-4.0-ultra-generate-001` (Max Images: 1, Aliases: "Imagen 4 Ultra", "Imagen4 Ultra")

---

### 3. **Veo Model Configurations**

#### **Default Model Location**
**Location**: `/experiments/mcp-genmedia/mcp-genmedia-go/mcp-common/models.go`
**Lines**: 111-139
```go
var SupportedVeoModels = map[string]VeoModelInfo{
    "veo-3.0-generate-001": {  // ← DEFAULT VEO MODEL
        CanonicalName:         "veo-3.0-generate-001",
        Aliases:               []string{"Veo 3"},
        MinDuration:           8,
        MaxDuration:           8,
        DefaultDuration:       8,
        MaxVideos:             2,
        SupportedAspectRatios: []string{"16:9"},
    },
    // ... more models
}
```

**Purpose**: Central registry of all supported Veo models and their capabilities
**Default**: `veo-3.0-generate-001` (first in map, used as default)
**Centralized**: ✅ Yes - single source of truth for all Veo models

#### **Available Models**
- `veo-3.0-generate-001` (Duration: 8-8s, Max Videos: 2, Ratios: 16:9, Aliases: "Veo 3") **← DEFAULT**
- `veo-2.0-generate-001` (Duration: 5-8s, Max Videos: 4, Ratios: 16:9, 9:16, Aliases: "Veo 2")
- `veo-3.0-generate-preview` (Duration: 8-8s, Max Videos: 2, Ratios: 16:9, Aliases: "Veo 3")
- `veo-3.0-fast-generate-preview` (Duration: 8-8s, Max Videos: 2, Ratios: 16:9, Aliases: "Veo 3 Fast")

---

### 4. **Chirp3 TTS Model Configuration**

**Location**: `/experiments/mcp-genmedia/mcp-genmedia-go/mcp-chirp3-go/chirp3.go`
**Line**: 40
**Implementation**: Dynamic API calls to Google Cloud Text-to-Speech with hardcoded default
**Default Voice**: `en-US-Chirp3-HD-Achernar` (hardcoded constant)
**Centralized**: ⚠️ Partial - default voice centralized, available voices fetched dynamically

```go
const (
    serviceName             = "mcp-chirp3-go"
    timeFormatForFilename = "20060102-150405"
    defaultChirpVoiceName = "en-US-Chirp3-HD-Achernar"  // ← DEFAULT VOICE
)
```

#### **How It Works**
- At startup, the MCP server queries Google Cloud TTS API to fetch all available Chirp3-HD voices
- Voices are cached in memory for the session
- Default voice is hardcoded as a Go constant
- No central registry like Imagen/Veo models - voices are dynamic from API

#### **Voice Selection Process**
1. If `voice_name` parameter provided → use that specific voice
2. If not provided → try default voice (`en-US-Chirp3-HD-Achernar`)
3. If default not available → use first available Chirp3-HD voice
4. If no voices available → return error

---

### 5. **Environment-Based Configuration**

#### **Primary Environment File**
**Location**: `/experiments/mcp-genmedia/sample-agents/adk/genmedia_agent/.env`
```bash
# Google Cloud Configuration
GOOGLE_GENAI_USE_VERTEXAI=True
GOOGLE_CLOUD_PROJECT=supple-synapse-470916-a2
GOOGLE_CLOUD_LOCATION=us-central1
PROJECT_ID=supple-synapse-470916-a2
LOCATION=us-central1
GENMEDIA_BUCKET=supple-synapse-media

# MCP Server Configuration
MCP_SERVER_REQUEST_TIMEOUT=55000
```

**Purpose**: Runtime configuration for Google Cloud services and MCP servers
**Centralized**: ✅ Yes - single environment file for the agent

#### **MCP Server Configuration Template**
**Location**: `/experiments/mcp-genmedia/sample-agents/adk/genmedia-config.json`
```json
{
  "mcpServers": {
    "veo-go": {
      "command": "mcp-veo-go",
      "env": {
        "MCP_SERVER_REQUEST_TIMEOUT": "55000",
        "GENMEDIA_BUCKET": "YOUR_BUCKET_NAME",
        "PROJECT_ID": "YOUR_PROJECT"
      }
    }
    // ... other servers
  }
}
```

**Purpose**: Template for MCP server environment configuration
**Status**: Template only - not actively used in current implementation
**Centralized**: ✅ Yes - but not currently utilized

---

## Configuration Summary Table

| Component | Model/Service | Configuration Location | Centralized | Default Value |
|-----------|---------------|------------------------|-------------|---------------|
| **Agent** | Gemini | `agent.py:97` + `.env:12` | ✅ | `gemini-2.5-pro` |
| **Imagen** | Image Generation | `models.go:35-56` | ✅ | `imagen-4.0-generate-001` |
| **Veo** | Video Generation | `models.go:111-139` | ✅ | `veo-3.0-generate-001` |
| **Chirp3** | Text-to-Speech | `chirp3.go:40` | ⚠️ | `en-US-Chirp3-HD-Achernar` |
| **Environment** | Runtime Config | `.env` | ✅ | See file |
| **MCP Servers** | Server Config | `genmedia-config.json` | ✅ | Template only |

---

## Recommendations for Centralization

### **High Priority**
1. **Create Central Model Configuration**
   ```yaml
   # models.yaml
   agent:
     model: "gemini-2.0-flash"
   
   imagen:
     default: "imagen-3.0-generate-002"
   
   veo:
     default: "veo-2.0-generate-001"
   
   chirp3:
     default_voice: "en-US-Chirp3-HD-Zephyr"
   ```

2. **Implement Configuration Loading**
   - Load model configs from central file
   - Support environment variable overrides
   - Validate model availability at startup

### **Medium Priority**
3. **Activate MCP Server Configuration**
   - Use `genmedia-config.json` for MCP server settings
   - Centralize timeout and environment configurations

### **Low Priority**
4. **Runtime Model Switching**
   - Add API endpoints to change models dynamically
   - Implement model performance monitoring
   - Add A/B testing capabilities

---

## Current Limitations

1. **Agent Model**: Hardcoded in Python, requires code change to modify
2. **Chirp3 Voices**: No central voice registry, relies on API discovery
3. **MCP Configuration**: Template exists but not actively used
4. **Model Validation**: No startup validation of model availability
5. **Environment Sync**: Multiple places to update project/bucket settings

---

## File Locations Quick Reference

```
experiments/mcp-genmedia/
├── sample-agents/adk/
│   ├── genmedia_agent/
│   │   ├── agent.py              # Agent model: gemini-2.0-flash
│   │   └── .env                  # Environment configuration
│   └── genmedia-config.json      # MCP server template (unused)
└── mcp-genmedia-go/
    └── mcp-common/
        └── models.go             # Imagen & Veo model registry
```

This exhibit shows that while some model configurations are centralized (Imagen/Veo in `models.go`), others are scattered across different files, creating potential maintenance challenges.
