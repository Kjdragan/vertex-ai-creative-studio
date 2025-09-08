# GenMedia Creative Studio

## Project Overview

This repository contains the GenMedia Creative Studio, a suite of web applications designed to showcase the creative capabilities of Google Cloud's Vertex AI and its generative AI models, including Imagen and Veo. The primary application is built with Mesop, a Python-based UI framework, and there are numerous experimental applications in the `experiments` directory.

The main application provides a user-friendly interface for generating images from text prompts, with features for prompt rewriting and multimodal evaluation of the generated images using Gemini.

The `experiments` directory contains a variety of standalone applications and workflows, including:

*   **veo-app:** The next-generation of the Creative Studio, with a focus on video generation using the Veo model.
*   **Storycraft:** An AI-powered video storyboard generation platform.
*   **Promptlandia:** A web application for analyzing and refining prompts.
*   **And many more:** The `experiments` directory is rich with tools for image and video generation, analysis, and workflow automation.

## Building and Running

### Main Application

To run the main GenMedia Creative Studio application:

1.  **Set up the environment:**
    *   Create a `.env` file from the `env_template` and populate it with your Google Cloud Project ID and a Cloud Storage bucket name.
    *   Create a Python virtual environment and activate it.

2.  **Install dependencies:**
    ```bash
    pip install -r requirements.txt
    ```

3.  **Run the application:**
    ```bash
    mesop main.py
    ```

### Veo App (v.Next)

The `veo-app` is a more advanced application and has its own deployment and development process.

**Development:**

1.  **Navigate to the `veo-app` directory:**
    ```bash
    cd experiments/veo-app
    ```

2.  **Set up the environment:**
    *   Use `uv` to sync the requirements to a virtual environment: `uv sync`
    *   Create a `.env` file from the `dotenv.template` and add your `PROJECT_ID`.

3.  **Run the application:**
    ```bash
    uv run main.py
    ```
    Or for development with hot-reloading:
    ```bash
    source .venv/bin/activate
    mesop main.py
    ```

**Deployment:**

The `veo-app` is designed to be deployed to Google Cloud Run using Terraform and Cloud Build. Detailed instructions are available in the `experiments/veo-app/README.md` file.

## Development Conventions

*   **Framework:** The main application and many of the experiments are built with the [Mesop](https://google.github.io/mesop/) framework.
*   **Configuration:** Application configuration is managed in Python files within the `config` directory (e.g., `config/default.py`).
*   **State Management:** Mesop's `@me.stateclass` is used for managing application state.
*   **Modularity:** The project is organized into modules for models, prompts, and UI components.
*   **Experiments:** New features and applications are developed in the `experiments` directory before being integrated into the main application.

## ADK Agent and MCPs

The `experiments/mcp-genmedia` directory contains the ADK (Agent Development Kit) agent and a suite of Go-based MCPs (Model-serving Control-plane Proxies) for multimodal media generation.

### ADK Agent

The ADK agent, located in `experiments/mcp-genmedia/sample-agents/adk`, is a Python-based agent that orchestrates the MCPs to perform various media generation tasks. It is configured via the `genmedia-config.json` file, which defines the commands and environment variables for each MCP.

To run the ADK agent:

1.  **Navigate to the ADK agent directory:**
    ```bash
    cd experiments/mcp-genmedia/sample-agents/adk
    ```

2.  **Set up the environment:**
    *   Source the `export_env.sh` script to set the required environment variables.
    *   Create a `.env` file in the `genmedia_agent` directory with your specific environment variables.

3.  **Start the agent:**
    ```bash
    ./start_adk.sh
    ```

### MCPs (Model-serving Control-plane Proxies)

The MCPs, located in `experiments/mcp-genmedia/mcp-genmedia-go`, are a set of Go-based servers that provide a standardized interface to various Google media generation models, including:

*   **Veo:** For video generation.
*   **Imagen:** For image generation.
*   **Chirp3:** For text-to-speech.
*   **Lyria:** For music generation.
*   **AVTool:** For audio/video processing.

These MCPs are started by the ADK agent as defined in the `genmedia-config.json` file. They provide a layer of abstraction between the agent and the underlying models, making it easier to develop and maintain the agent.