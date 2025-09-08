# How to Run the ADK GenMedia Agent

This guide provides comprehensive instructions for running the ADK GenMedia Agent, a powerful multimedia AI assistant that integrates image generation, text-to-speech, video generation, and audio/video processing capabilities.

## Project Overview

The **ADK GenMedia Agent** is built using Google's Agent Development Kit (ADK) and integrates multiple MCP (Model Context Protocol) toolsets to provide comprehensive generative media capabilities:

### Core Capabilities
- **Image Generation & Editing**: Create and modify images using Google's Imagen models
- **Text-to-Speech**: Generate natural speech using Chirp3-HD voices in multiple languages
- **Video Generation**: Create videos from text prompts or images using Veo models
- **Audio/Video Processing**: Professional media processing with FFmpeg integration

### Architecture
The agent consists of:
- **Python ADK Agent**: Main orchestration layer using Google's ADK framework
- **Go MCP Servers**: High-performance backend services for each media type
  - `mcp-imagen-go`: Image generation and editing
  - `mcp-chirp3-go`: Text-to-speech synthesis
  - `mcp-veo-go`: Video generation
  - `mcp-avtool-go`: Audio/video processing with FFmpeg

## Prerequisites

### System Requirements
- Python 3.13+
- Go 1.21+ (for MCP servers)
- FFmpeg (for audio/video processing)
- Google Cloud SDK (`gcloud`)

### Google Cloud Setup
1. **Project Configuration**: Ensure you have access to a Google Cloud project with the following APIs enabled:
   - Vertex AI API
   - Cloud Text-to-Speech API
   - Cloud Storage API

2. **Authentication**: 
   ```bash
   gcloud auth login
   gcloud config set project YOUR_PROJECT_ID
   ```

3. **Storage Bucket**: Create a GCS bucket for media outputs:
   ```bash
   gsutil mb gs://your-genmedia-bucket
   ```

## Installation & Setup

### 1. Navigate to Project Directory
```bash
cd /home/kjdrag/lrepos/google-genai-media-master-repo/experiments/mcp-genmedia/sample-agents/adk
```

### 2. Install Dependencies
```bash
# Install Python dependencies
pip install -r requirements.txt

# Or using uv (recommended)
uv sync
```

### 3. Configure Environment
Edit `genmedia-config.json` to set your project details:
```json
{
  "mcpServers": {
    "veo-go": {
      "command": "mcp-veo-go",
      "env": {
        "MCP_SERVER_REQUEST_TIMEOUT": "55000",
        "GENMEDIA_BUCKET": "your-genmedia-bucket",
        "PROJECT_ID": "your-project-id"
      }
    },
    "imagen-go": {
      "command": "mcp-imagen-go",
      "env": {
        "MCP_SERVER_REQUEST_TIMEOUT": "55000",
        "GENMEDIA_BUCKET": "your-genmedia-bucket",
        "PROJECT_ID": "your-project-id"
      }
    },
    "chirp3-go": {
      "command": "mcp-chirp3-go",
      "env": {
        "GENMEDIA_BUCKET": "your-genmedia-bucket",
        "PROJECT_ID": "your-project-id"
      }
    },
    "avtool-go": {
      "command": "mcp-avtool-go",
      "env": {
        "PROJECT_ID": "your-project-id",
        "MCP_SERVER_REQUEST_TIMEOUT": "55000"
      }
    }
  }
}
```

### 4. Build MCP Servers
Ensure the Go MCP servers are built and available in your PATH:
```bash
# Navigate to MCP servers directory
cd ../../../mcp-genmedia-go

# Build all servers
make build

# Or build individually
go build -o bin/mcp-imagen-go ./mcp-imagen-go
go build -o bin/mcp-chirp3-go ./mcp-chirp3-go
go build -o bin/mcp-veo-go ./mcp-veo-go
go build -o bin/mcp-avtool-go ./mcp-avtool-go

# Add to PATH or copy to system PATH
export PATH=$PATH:$(pwd)/bin
```

## Running the Agent

### Method 1: Web UI (Recommended)

The web UI provides an intuitive interface for interacting with the agent through a browser.

#### Start the Web Server
```bash
cd /home/kjdrag/lrepos/google-genai-media-master-repo/experiments/mcp-genmedia/sample-agents/adk

# Using virtual environment
.venv/bin/adk web genmedia_agent

# Or if ADK is installed globally
adk web genmedia_agent
```

#### Web UI Options
```bash
# Custom host and port
.venv/bin/adk web --host 0.0.0.0 --port 8080 genmedia_agent

# Enable auto-reload for development
.venv/bin/adk web --reload genmedia_agent

# With session persistence
.venv/bin/adk web --session_service_uri sqlite:///sessions.db genmedia_agent

# With cloud storage for artifacts
.venv/bin/adk web --artifact_service_uri gs://your-bucket genmedia_agent
```

#### Access the Web UI
1. Open your browser to `http://localhost:8000` (or your specified port)
2. You'll see the ADK Web Developer UI
3. Select the `genmedia_agent` from the available agents
4. Start chatting with the agent to generate media content

### Method 2: Command Line Interface

The CLI provides direct terminal interaction with the agent.

#### Start Interactive CLI
```bash
cd /home/kjdrag/lrepos/google-genai-media-master-repo/experiments/mcp-genmedia/sample-agents/adk

# Run interactive CLI
.venv/bin/adk run genmedia_agent
```

#### CLI Options
```bash
# Save session on exit
.venv/bin/adk run --save_session genmedia_agent

# Resume previous session
.venv/bin/adk run --resume session.json genmedia_agent

# Replay commands from file
.venv/bin/adk run --replay commands.json genmedia_agent
```

## Usage Examples

### Image Generation
```
User: Generate an image of a serene mountain landscape at sunset with a lake reflection

Agent: I'll create a beautiful mountain landscape image for you using Imagen.
[Generates image using imagen_t2i tool]
```

### Text-to-Speech
```
User: Convert this text to speech: "Welcome to our AI-powered media generation system"

Agent: I'll synthesize that text using our Chirp3-HD voice system.
[Generates audio using chirp_tts tool]
```

### Video Creation
```
User: Create a 10-second video of a cat playing with a ball of yarn

Agent: I'll generate that video using our Veo text-to-video system.
[Generates video using veo_t2v tool]
```

### Media Processing
```
User: Combine this audio file with this video file
[Provide GCS URIs for both files]

Agent: I'll merge the audio and video streams for you.
[Processes using ffmpeg_combine_audio_and_video tool]
```

## Advanced Configuration

### Environment Variables
Create a `.env` file in the agent directory:
```env
PROJECT_ID=your-project-id
GENMEDIA_BUCKET=your-bucket-name
GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
MCP_SERVER_REQUEST_TIMEOUT=60000
```

### Session Persistence
For production use, configure persistent sessions:
```bash
# SQLite database
adk web --session_service_uri sqlite:///sessions.db genmedia_agent

# Agent Engine (cloud)
adk web --session_service_uri agentengine://your-agent-engine-id genmedia_agent
```

### Artifact Storage
Configure cloud storage for generated media:
```bash
# Google Cloud Storage
adk web --artifact_service_uri gs://your-artifacts-bucket genmedia_agent
```

## Troubleshooting

### Common Issues

#### MCP Server Connection Errors
```bash
# Check if MCP servers are in PATH
which mcp-imagen-go mcp-chirp3-go mcp-veo-go mcp-avtool-go

# Test individual server
mcp-imagen-go --help
```

#### Authentication Issues
```bash
# Verify authentication
gcloud auth list
gcloud config get-value project

# Re-authenticate if needed
gcloud auth application-default login
```

#### Permission Errors
```bash
# Check project permissions
gcloud projects get-iam-policy YOUR_PROJECT_ID

# Verify API enablement
gcloud services list --enabled
```

#### Storage Access Issues
```bash
# Test bucket access
gsutil ls gs://your-bucket-name
gsutil cp test.txt gs://your-bucket-name/
```

### Debug Mode
Run with debug logging:
```bash
# Web UI with debug logging
adk web --log_level debug genmedia_agent

# CLI with verbose output
adk run genmedia_agent --verbose
```

### Performance Optimization

#### Timeout Configuration
Adjust timeouts for large media generation:
```json
{
  "mcpServers": {
    "veo-go": {
      "env": {
        "MCP_SERVER_REQUEST_TIMEOUT": "300000"
      }
    }
  }
}
```

#### Concurrent Processing
The agent supports concurrent tool execution for faster processing of multiple media requests.

## Development & Customization

### Adding New Tools
1. Implement new MCP server in Go
2. Add server configuration to `genmedia-config.json`
3. Update agent imports in `genmedia_agent/agent.py`

### Custom Prompts
Modify the agent's behavior by editing the instruction in `genmedia_agent/agent.py`:
```python
root_agent = LlmAgent(
    model='gemini-2.0-flash',
    name='genmedia_agent',
    instruction="""Your custom instructions here...""",
    tools=[imagen, chirp3, veo, avtool],
)
```

## Production Deployment

### Docker Deployment
```dockerfile
FROM python:3.13-slim
COPY . /app
WORKDIR /app
RUN pip install -r requirements.txt
CMD ["adk", "web", "--host", "0.0.0.0", "genmedia_agent"]
```

### Cloud Run Deployment
```bash
# Deploy to Google Cloud Run
adk deploy --platform cloud_run genmedia_agent
```

## Support & Resources

- **Agent Development Kit Documentation**: [Google ADK Docs](https://cloud.google.com/vertex-ai/generative-ai/docs/agent-builder)
- **MCP Protocol**: [Model Context Protocol](https://modelcontextprotocol.io/)
- **Vertex AI**: [Vertex AI Documentation](https://cloud.google.com/vertex-ai/docs)

For technical issues, check the logs and ensure all prerequisites are properly configured. The agent provides detailed error messages to help diagnose configuration problems.
