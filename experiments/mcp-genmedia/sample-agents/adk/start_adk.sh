#!/bin/bash

set -euo pipefail

# Resolve script directory and source .env so .env values take precedence
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/genmedia_agent/.env"
if [ -f "${ENV_FILE}" ]; then
  # shellcheck disable=SC1090
  set -a
  . "${ENV_FILE}"
  set +a
fi

# Export environment variables for MCP servers (defaults only; .env wins)
export GOOGLE_GENAI_USE_VERTEXAI=${GOOGLE_GENAI_USE_VERTEXAI:-True}
export GOOGLE_CLOUD_PROJECT=${GOOGLE_CLOUD_PROJECT:-supple-synapse-470916-a2}
export GOOGLE_CLOUD_LOCATION=${GOOGLE_CLOUD_LOCATION:-us-central1}
export PROJECT_ID=${PROJECT_ID:-supple-synapse-470916-a2}
export LOCATION=${LOCATION:-us-central1}
export GENMEDIA_BUCKET=${GENMEDIA_BUCKET:-supple-synapse-media}
export IMAGEN_BUCKET_PATH=${IMAGEN_BUCKET_PATH:-gs://supple-synapse-imagen}
export VEO_BUCKET_PATH=${VEO_BUCKET_PATH:-gs://supple-synapse-veo}
export LYRIA_BUCKET_PATH=${LYRIA_BUCKET_PATH:-gs://supple-synapse-lyria}
export CHIRP3_BUCKET_PATH=${CHIRP3_BUCKET_PATH:-gs://supple-synapse-chirp3}
export AVTOOL_BUCKET_PATH=${AVTOOL_BUCKET_PATH:-gs://supple-synapse-avtool}
export GOOGLE_APPLICATION_CREDENTIALS=${GOOGLE_APPLICATION_CREDENTIALS:-/home/kjdrag/.config/gcloud/application_default_credentials.json}
export MCP_SERVER_REQUEST_TIMEOUT=${MCP_SERVER_REQUEST_TIMEOUT:-55000}
export AGENT_MODEL=${AGENT_MODEL:-gemini-2.5-pro}
export ARIZE_SPACE_ID=${ARIZE_SPACE_ID:-U3BhY2U6ODU1MDp2L3Rp}
export ARIZE_API_KEY=${ARIZE_API_KEY:-ak-b78262b2-f932-43f7-a57f-7e5e5733ad2e-RWDT9Q8u6co9vx8j3Nr3xSUh2k-LJiuB}
export ARIZE_PROJECT_NAME=${ARIZE_PROJECT_NAME:-genmedia-adk}
export ARIZE_INTERFACE=${ARIZE_INTERFACE:-adk}

# ADK Artifacts: default to GCS-backed persistence so callbacks can resolve gs:// URIs
export USE_GCS_ARTIFACTS=${USE_GCS_ARTIFACTS:-true}
# Artifact bucket (separate from GENMEDIA_BUCKET if you want). Defaults to supple-synapse-media
export GENMEDIA_ARTIFACT_BUCKET=${GENMEDIA_ARTIFACT_BUCKET:-supple-synapse-media}

# Start the ADK web server (prefer GCS ArtifactService by default)
if [ "${USE_GCS_ARTIFACTS}" = "true" ]; then
  uv run adk web --artifact_service_uri "gs://${GENMEDIA_ARTIFACT_BUCKET}"
else
  uv run adk web
fi
