# ADK Artifact Solution for Inline Image Processing

## Overview

This document describes the complete solution for handling uploaded inline images in the ADK-MCP system using ADK's native Artifact system. This solution eliminates the issue where Gemini LLM would hallucinate fake GCS URIs instead of properly processing uploaded image data.

## Problem Statement

Previously, when users uploaded images to the ADK agent:
1. ADK received the inline image data but didn't forward it to MCP tools
2. Gemini LLM would hallucinate non-existent GCS URIs like `gs://cloud-samples-data/generative-ai/image/backyard.jpeg`
3. MCP tools would fail because the fake URIs didn't exist
4. Video generation from uploaded images was impossible

## Solution Architecture

The solution uses ADK's native Artifact system to:
1. **Automatically capture** uploaded images via ADK callbacks
2. **Save images as artifacts** with proper versioning and persistence
3. **Reference artifacts** in MCP tool calls using `artifact:` prefix
4. **Process artifacts** in MCP handlers by loading from the artifact service

## Components

### 1. ADK Artifact Handler (`artifact_handler.py`)
- **Purpose**: Core functionality for saving and loading image artifacts
- **Key Features**:
  - Supports all standard image formats (JPEG, PNG, GIF, WebP, BMP, TIFF)
  - Automatic MIME type detection and validation
  - Descriptive filename generation
  - Error handling for missing artifact service

### 2. Image Artifact Callback (`image_artifact_callback.py`)
- **Purpose**: Automatically intercepts user messages with images
- **Key Features**:
  - Processes uploaded images in real-time
  - Saves images as artifacts with descriptive names
  - Updates session state with artifact references
  - Provides feedback to users about processed images

### 3. ADK Artifact Processor (`adk_artifact_processor.go`)
- **Purpose**: MCP handler middleware for processing artifact references
- **Key Features**:
  - Detects `artifact:` prefixed parameters
  - Falls back to existing inline data processing
  - Maintains backward compatibility with GCS URIs

### 4. Runner Configuration (`runner_config.py`)
- **Purpose**: Proper ADK Runner setup with artifact service
- **Key Features**:
  - Configurable GCS or in-memory artifact storage
  - Environment variable based configuration
  - Fallback mechanisms for service initialization

## Workflow

### User Upload Process
1. **User uploads image** → Image appears in message parts
2. **ImageArtifactCallback intercepts** → Automatically processes image
3. **Artifact saved** → Image stored with filename like `user_image_0.jpg`
4. **Session state updated** → Artifact reference stored for agent access
5. **User informed** → Confirmation message about saved artifact

### Agent Processing
1. **Agent receives request** → Checks session state for available artifacts
2. **Tool call made** → Uses `artifact:user_image_0.jpg` as parameter
3. **MCP handler processes** → ADKArtifactProcessor detects artifact reference
4. **Artifact loaded** → Image data retrieved from artifact service
5. **Media generation** → Video/audio/image processing proceeds normally

## Configuration

### Environment Variables
```bash
# Artifact Service Configuration
USE_GCS_ARTIFACTS=true                    # Use GCS for persistent storage
GENMEDIA_ARTIFACT_BUCKET=supple-synapse-media  # GCS bucket for artifacts

# Existing MCP Configuration (unchanged)
IMAGEN_BUCKET_PATH=gs://supple-synapse-media/imagen/
VEO_BUCKET_PATH=gs://supple-synapse-media/veo/
# ... other bucket paths
```

### ADK Runner Setup
```python
from google.adk.artifacts import GcsArtifactService
from google.adk.runners import Runner

# Configure artifact service
artifact_service = GcsArtifactService(bucket_name="supple-synapse-media")

# Create runner with artifact support
runner = Runner(
    agent=root_agent,
    app_name="genmedia_agent",
    session_service=session_service,
    artifact_service=artifact_service  # Critical for artifact functionality
)
```

## Usage Examples

### For Users
```
User: [Uploads image: vacation_photo.jpg] "Create a video from this image"

System: [Image uploaded and saved as artifact: user_image_0.jpg]

Agent: I'll create a video from your uploaded image using the Veo model.
       [Calls veo_i2v with image_uri="artifact:user_image_0.jpg"]
```

### For Developers
```python
# In agent callbacks
async def process_user_message(context: CallbackContext, message: types.Content):
    # Images are automatically processed by ImageArtifactCallback
    session_state = context.get_session_state()
    artifacts = session_state.get('image_artifacts', {})
    print(f"Available image artifacts: {list(artifacts.values())}")

# In MCP tool calls
veo_result = await veo_i2v(
    image_uri="artifact:user_image_0.jpg",  # Reference saved artifact
    prompt="Create a cinematic video",
    duration=5
)
```

## Benefits

### ✅ Eliminates Hallucinated URIs
- No more fake `gs://cloud-samples-data/` URIs
- Agent uses actual uploaded image data
- Reliable video generation from user images

### ✅ Proper ADK Integration
- Uses ADK's native artifact system
- Leverages ADK's versioning and persistence
- Integrates with ADK session management

### ✅ Scalable Storage
- GCS-backed artifacts for production
- In-memory artifacts for development
- Automatic cleanup and lifecycle management

### ✅ Enhanced User Experience
- Seamless image upload workflow
- Automatic processing with feedback
- No manual GCS URI management required

## Migration Guide

### From Previous Implementation
1. **Update agent configuration** → Add `callbacks=[image_artifact_callback]`
2. **Configure artifact service** → Use `runner_config.py` pattern
3. **Update instructions** → Remove fake URI warnings, add artifact guidance
4. **Test workflow** → Verify image upload → artifact → video generation

### Environment Setup
```bash
# Add to .env file
USE_GCS_ARTIFACTS=true
GENMEDIA_ARTIFACT_BUCKET=supple-synapse-media

# Ensure GCS permissions for artifact bucket
gcloud auth application-default login
```

## Testing

### Manual Test Workflow
1. Start ADK agent with artifact configuration
2. Upload an image through the interface
3. Verify artifact is saved (check logs for confirmation)
4. Request video generation from uploaded image
5. Confirm MCP handler processes artifact correctly
6. Verify video is generated successfully

### Expected Log Output
```
INFO: Processed uploaded image as artifact: user_image_0.jpg
INFO: Updated session state with 1 image artifacts
INFO: Processing ADK artifact reference: user_image_0.jpg
INFO: Veo i2v request successful with artifact image
```

## Troubleshooting

### Common Issues

**"ArtifactService not configured"**
- Ensure `artifact_service` is passed to Runner initialization
- Check GCS credentials and bucket permissions

**"Artifact not found"**
- Verify artifact was saved successfully in callback
- Check session state for artifact references
- Ensure consistent filename usage

**"Unsupported image type"**
- Verify image MIME type is in supported list
- Check image file is not corrupted
- Ensure proper base64 encoding for inline data

## Future Enhancements

1. **Multi-image support** → Handle multiple uploaded images per message
2. **Artifact cleanup** → Automatic deletion of old artifacts
3. **Cross-session artifacts** → User-scoped artifacts with `user:` prefix
4. **Artifact metadata** → Enhanced metadata for better organization
5. **Direct GCS integration** → Seamless GCS URI generation from artifacts

## Conclusion

The ADK Artifact solution provides a robust, scalable, and user-friendly approach to handling uploaded images in the generative media pipeline. By leveraging ADK's native capabilities, it eliminates the core issue of hallucinated URIs while providing a foundation for enhanced media processing workflows.
