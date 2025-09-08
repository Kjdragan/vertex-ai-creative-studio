# Genmedia MCPs and ADK Agent

This directory contains the core components for the Generative Media Model Context Protocol (MCP) servers and the Agent Development Kit (ADK) agent that orchestrates them.

## Project Overview

The project consists of two main parts:
1.  **Go-based MCP Servers (`mcp-genmedia-go`):** A collection of servers that provide a standardized MCP interface to Google's generative media APIs.
2.  **Python-based ADK Agent (`sample-agents/adk`):** An agent that uses the MCP servers to perform various media generation and composition tasks.

## Core Capabilities

The MCP servers provide access to the following Google Cloud generative AI models:

*   **Imagen:** High-quality image generation and editing.
*   **Veo:** Text-to-video and image-to-video generation.
*   **Chirp:** Text-to-speech synthesis.
*   **Lyria:** Music generation.
*   **AVTool:** A utility for audio/video compositing, concatenation, and other ffmpeg-based operations.

## Getting Started

### Running the ADK Agent

The primary way to interact with the MCPs is through the ADK agent.

1.  **Navigate to the agent directory:**
    ```bash
    cd experiments/mcp-genmedia/sample-agents/adk
    ```

2.  **Set Environment Variables:** The agent and MCPs require environment variables to be set, most importantly `PROJECT_ID`. You can source the provided script to set them:
    ```bash
    source ./export_env.sh
    ```
    You may also need to create a `.env` file inside the `genmedia_agent` directory for additional configuration.

3.  **Start the Agent:**
    ```bash
    ./start_adk.sh
    ```
    This script uses the `genmedia-config.json` file to start the relevant MCP servers as background processes and then starts the ADK agent.

### Configuration

*   **`genmedia-config.json`:** Located in `sample-agents/adk`, this is the central configuration file for the ADK agent. It defines which MCP servers to start and what tools are available to the agent.
*   **`mcp-genmedia-go/`:** Each subdirectory here (e.g., `mcp-imagen-go`, `mcp-veo-go`) contains the source code for an individual MCP server. Refer to the `GEMINI.md` inside `mcp-genmedia-go` for detailed development and testing instructions for the Go servers.

### Image Artifact Handling

The ADK agent has a robust system for handling image uploads, as detailed in `ADK_ARTIFACT_SOLUTION.md`.
*   When a user uploads an image, it is automatically saved as an "artifact".
*   To use this image in a tool call, use the `artifact:<filename>` prefix (e.g., `image_uri="artifact:user_image_0.jpg"`).
*   This avoids issues with the LLM hallucinating non-existent Google Cloud Storage URIs.

### Testing

*   **End-to-End:** The `test_end_to_end.py` script in the root of this directory can be used to run a full workflow test.
*   **Artifact Workflow:** The `test_artifact_workflow.py` script specifically tests the image artifact upload and processing pipeline.
*   **Manual MCP Testing:** For manual testing of the Go-based MCP servers, use the `mcptools` binary as described in `mcp-genmedia-go/GEMINI.md`.
