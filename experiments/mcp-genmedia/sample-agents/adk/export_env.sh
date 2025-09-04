#!/bin/bash

# Export environment variables for MCP servers
export GOOGLE_GENAI_USE_VERTEXAI=True
export GOOGLE_CLOUD_PROJECT=supple-synapse-470916-a2
export GOOGLE_CLOUD_LOCATION=us-central1
export PROJECT_ID=supple-synapse-470916-a2
export LOCATION=us-central1
export GENMEDIA_BUCKET=supple-synapse-media

# Media-specific separate buckets for organized storage
export IMAGEN_BUCKET_PATH=gs://supple-synapse-imagen
export VEO_BUCKET_PATH=gs://supple-synapse-veo
export LYRIA_BUCKET_PATH=gs://supple-synapse-lyria
export CHIRP3_BUCKET_PATH=gs://supple-synapse-chirp3
export AVTOOL_BUCKET_PATH=gs://supple-synapse-avtool

export GOOGLE_APPLICATION_CREDENTIALS=/home/kjdrag/.config/gcloud/application_default_credentials.json
export MCP_SERVER_REQUEST_TIMEOUT=55000
export AGENT_MODEL=gemini-2.5-pro

# Arize configuration
export ARIZE_SPACE_ID=U3BhY2U6ODU1MDp2L3Rp
export ARIZE_API_KEY=ak-b78262b2-f932-43f7-a57f-7e5e5733ad2e-RWDT9Q8u6co9vx8j3Nr3xSUh2k-LJiuB
export ARIZE_PROJECT_NAME=genmedia-adk
export ARIZE_INTERFACE=adk

# Set to "false" to disable OpenTelemetry tracing entirely
export ENABLE_OTEL_TRACING=false
