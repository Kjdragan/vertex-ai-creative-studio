# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


import os

# as of google-adk==1.3.0, StdioConnectionParams
from dotenv import load_dotenv
from google.adk.agents import LlmAgent
from google.adk.tools.function_tool import FunctionTool
from google.adk.tools.mcp_tool.mcp_toolset import (
    MCPToolset,
    StdioConnectionParams,
    StdioServerParameters,
)
from google.adk.artifacts import GcsArtifactService
from google.adk.sessions import InMemorySessionService
from .simple_callback import before_agent_callback, before_tool_callback
from datetime import datetime, timedelta
try:
    from google.cloud import storage  # type: ignore
except Exception:  # pragma: no cover
    storage = None  # type: ignore
try:
    from google.auth import default as google_auth_default  # type: ignore
except Exception:  # pragma: no cover
    google_auth_default = None  # type: ignore
try:
    from google.auth.impersonated_credentials import Credentials as ImpersonatedCredentials  # type: ignore
except Exception:  # pragma: no cover
    ImpersonatedCredentials = None  # type: ignore
from typing import Optional

# Arize OpenInference instrumentation for ADK
from arize.otel import register
from openinference.instrumentation.google_adk import GoogleADKInstrumentor

load_dotenv()

def gcs_signed_url(gs_uri: str, expiry_seconds: int = 3600) -> dict:
    """
    Generate a time-limited signed HTTPS URL for a given GCS object.

    Args:
      gs_uri: The GCS URI of the object, e.g. "gs://bucket/path/to/object".
      expiry_seconds: Number of seconds the URL should remain valid (default 3600).

    Returns:
      A dict with keys:
        - gs_uri: original gs:// URI
        - public_url: signed HTTPS URL
        - expires_at: ISO8601 timestamp when the URL expires
    """
    if not gs_uri or not gs_uri.startswith("gs://"):
        return {"error": "Invalid gs_uri. Must start with gs://"}

    try:
        if storage is None:
            return {"error": "google-cloud-storage is not installed in the ADK environment. Please install 'google-cloud-storage' to enable signed URL generation."}
        # Parse bucket and blob path
        without_scheme = gs_uri[len("gs://"):]
        parts = without_scheme.split("/", 1)
        if len(parts) != 2 or not parts[0] or not parts[1]:
            return {"error": "Invalid gs_uri format. Expected gs://<bucket>/<object>"}
        bucket_name, blob_path = parts[0], parts[1]

        client = storage.Client()
        bucket = client.bucket(bucket_name)
        blob = bucket.blob(blob_path)

        # Signing strategy:
        # 1) If SIGNED_URL_SERVICE_ACCOUNT or GOOGLE_IMPERSONATE_SERVICE_ACCOUNT is set and google_auth_default is available,
        #    attempt remote signing via IAMCredentials using ADC (user or workload identity) with signBlob on the target SA.
        # 2) Otherwise, attempt local signing. This requires ADC to be a service account with a private key (e.g., key file).

        signer_email = os.getenv("SIGNED_URL_SERVICE_ACCOUNT") or os.getenv("GOOGLE_IMPERSONATE_SERVICE_ACCOUNT")
        url: Optional[str] = None
        err_primary: Optional[Exception] = None

        if signer_email and google_auth_default is not None:
            try:
                creds, _ = google_auth_default(scopes=["https://www.googleapis.com/auth/cloud-platform"])
                # Prefer explicit IAM impersonation for signing when available.
                if ImpersonatedCredentials is not None:
                    imp_creds = ImpersonatedCredentials(
                        source_credentials=creds,
                        target_principal=signer_email,
                        target_scopes=["https://www.googleapis.com/auth/cloud-platform"],
                        lifetime=3600,
                    )
                    url = blob.generate_signed_url(
                        version="v4",
                        expiration=timedelta(seconds=int(expiry_seconds)),
                        method="GET",
                        # Provide both the target SA email and the impersonated credentials for remote signing
                        service_account_email=signer_email,
                        credentials=imp_creds,
                    )
                else:
                    # Fallback: let storage library attempt remote signing using ADC directly
                    url = blob.generate_signed_url(
                        version="v4",
                        expiration=timedelta(seconds=int(expiry_seconds)),
                        method="GET",
                        service_account_email=signer_email,
                        credentials=creds,
                    )
            except Exception as e1:  # fallback to local signing
                err_primary = e1

        if url is None:
            url = blob.generate_signed_url(
                version="v4",
                expiration=timedelta(seconds=int(expiry_seconds)),
                method="GET",
            )
        expires_at = (datetime.utcnow() + timedelta(seconds=int(expiry_seconds))).isoformat() + "Z"
        return {"gs_uri": gs_uri, "public_url": url, "expires_at": expires_at}
    except Exception as e:
        # Add guidance for common misconfiguration: user ADC can't sign.
        guidance = ""
        if "private key" in str(e) or "sign" in str(e).lower():
            guidance = (
                " Ensure your ADK runtime uses sign-capable credentials. Options: "
                "(1) Set GOOGLE_APPLICATION_CREDENTIALS to a Service Account JSON with Storage Object Viewer and iam.serviceAccounts.signBlob; or "
                "(2) Set SIGNED_URL_SERVICE_ACCOUNT=<service-account-email> (or GOOGLE_IMPERSONATE_SERVICE_ACCOUNT) and grant your current principal 'Service Account Token Creator' on that service account."
            )
        return {"error": f"Failed to generate signed URL: {e}.{guidance}"}

# Register with Arize AX using environment variables
tracer_provider = register(
    space_id=os.getenv("ARIZE_SPACE_ID"),
    api_key=os.getenv("ARIZE_API_KEY"),
    project_name=os.getenv("ARIZE_PROJECT_NAME", "genmedia-adk")
)

# Instrument Google ADK for automatic tracing
GoogleADKInstrumentor().instrument(tracer_provider=tracer_provider)

project_id = os.getenv("GOOGLE_CLOUD_PROJECT")

# MCP Client (STDIO)
# assumes you've installed the MCP server on your path
imagen = MCPToolset(
    connection_params=StdioConnectionParams(
        server_params=StdioServerParameters(
            command="/home/kjdrag/go/bin/mcp-imagen-go",
            args=[],
            env={
                "PROJECT_ID": os.getenv("PROJECT_ID"),
                "LOCATION": os.getenv("LOCATION"),
                "GENMEDIA_BUCKET": os.getenv("GENMEDIA_BUCKET"),
                "IMAGEN_BUCKET_PATH": os.getenv("IMAGEN_BUCKET_PATH"),
                "VEO_BUCKET_PATH": os.getenv("VEO_BUCKET_PATH"),
                "CHIRP3_BUCKET_PATH": os.getenv("CHIRP3_BUCKET_PATH"),
                "LYRIA_BUCKET_PATH": os.getenv("LYRIA_BUCKET_PATH"),
                "AVTOOL_BUCKET_PATH": os.getenv("AVTOOL_BUCKET_PATH"),
                "GOOGLE_APPLICATION_CREDENTIALS": os.getenv("GOOGLE_APPLICATION_CREDENTIALS"),
                "ARIZE_API_KEY": os.getenv("ARIZE_API_KEY"),
                "ARIZE_SPACE_ID": os.getenv("ARIZE_SPACE_ID"),
                "ARIZE_PROJECT_NAME": os.getenv("ARIZE_PROJECT_NAME"),
                "ARIZE_INTERFACE": os.getenv("ARIZE_INTERFACE"),
                "OTEL_EXPORTER_OTLP_TRACES_HEADERS": os.getenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", ""),
                "OTEL_EXPORTER_OTLP_HEADERS": os.getenv("OTEL_EXPORTER_OTLP_HEADERS", ""),
                "OTEL_EXPORTER_OTLP_ENDPOINT": os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
                "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": os.getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", ""),
                "ENABLE_OTEL_TRACING": os.getenv("ENABLE_OTEL_TRACING", "false"),
            },
        ),
        timeout=180,
    ),
)

chirp3 = MCPToolset(
    connection_params=StdioConnectionParams(
        server_params=StdioServerParameters(
            command="/home/kjdrag/go/bin/mcp-chirp3-go",
            args=[],
            env={
                "PROJECT_ID": os.getenv("PROJECT_ID"),
                "LOCATION": os.getenv("LOCATION"),
                "GENMEDIA_BUCKET": os.getenv("GENMEDIA_BUCKET"),
                "IMAGEN_BUCKET_PATH": os.getenv("IMAGEN_BUCKET_PATH"),
                "VEO_BUCKET_PATH": os.getenv("VEO_BUCKET_PATH"),
                "CHIRP3_BUCKET_PATH": os.getenv("CHIRP3_BUCKET_PATH"),
                "LYRIA_BUCKET_PATH": os.getenv("LYRIA_BUCKET_PATH"),
                "AVTOOL_BUCKET_PATH": os.getenv("AVTOOL_BUCKET_PATH"),
                "GOOGLE_APPLICATION_CREDENTIALS": os.getenv("GOOGLE_APPLICATION_CREDENTIALS"),
                "ARIZE_API_KEY": os.getenv("ARIZE_API_KEY"),
                "ARIZE_SPACE_ID": os.getenv("ARIZE_SPACE_ID"),
                "ARIZE_PROJECT_NAME": os.getenv("ARIZE_PROJECT_NAME"),
                "ARIZE_INTERFACE": os.getenv("ARIZE_INTERFACE"),
                "OTEL_EXPORTER_OTLP_TRACES_HEADERS": os.getenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", ""),
                "OTEL_EXPORTER_OTLP_HEADERS": os.getenv("OTEL_EXPORTER_OTLP_HEADERS", ""),
                "OTEL_EXPORTER_OTLP_ENDPOINT": os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
                "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": os.getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", ""),
                "ENABLE_OTEL_TRACING": os.getenv("ENABLE_OTEL_TRACING", "false"),
            },
        ),
        timeout=180,
    ),
)

veo = MCPToolset(
    connection_params=StdioConnectionParams(
        server_params=StdioServerParameters(
            command="/home/kjdrag/go/bin/mcp-veo-go",
            args=[],
            env={
                "PROJECT_ID": os.getenv("PROJECT_ID"),
                "LOCATION": os.getenv("LOCATION"),
                "GENMEDIA_BUCKET": os.getenv("GENMEDIA_BUCKET"),
                "IMAGEN_BUCKET_PATH": os.getenv("IMAGEN_BUCKET_PATH"),
                "VEO_BUCKET_PATH": os.getenv("VEO_BUCKET_PATH"),
                "CHIRP3_BUCKET_PATH": os.getenv("CHIRP3_BUCKET_PATH"),
                "LYRIA_BUCKET_PATH": os.getenv("LYRIA_BUCKET_PATH"),
                "AVTOOL_BUCKET_PATH": os.getenv("AVTOOL_BUCKET_PATH"),
                "GOOGLE_APPLICATION_CREDENTIALS": os.getenv("GOOGLE_APPLICATION_CREDENTIALS"),
                "ARIZE_API_KEY": os.getenv("ARIZE_API_KEY"),
                "ARIZE_SPACE_ID": os.getenv("ARIZE_SPACE_ID"),
                "ARIZE_PROJECT_NAME": os.getenv("ARIZE_PROJECT_NAME"),
                "ARIZE_INTERFACE": os.getenv("ARIZE_INTERFACE"),
                "OTEL_EXPORTER_OTLP_TRACES_HEADERS": os.getenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", ""),
                "OTEL_EXPORTER_OTLP_HEADERS": os.getenv("OTEL_EXPORTER_OTLP_HEADERS", ""),
                "OTEL_EXPORTER_OTLP_ENDPOINT": os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
                "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": os.getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", ""),
                "ENABLE_OTEL_TRACING": os.getenv("ENABLE_OTEL_TRACING", "false"),
            },
        ),
        timeout=480,
    ),
)

avtool = MCPToolset(
    connection_params=StdioConnectionParams(
        server_params=StdioServerParameters(
            command="/home/kjdrag/go/bin/mcp-avtool-go",
            args=[],
            env={
                "PROJECT_ID": os.getenv("PROJECT_ID"),
                "LOCATION": os.getenv("LOCATION"),
                "GENMEDIA_BUCKET": os.getenv("GENMEDIA_BUCKET"),
                "IMAGEN_BUCKET_PATH": os.getenv("IMAGEN_BUCKET_PATH"),
                "VEO_BUCKET_PATH": os.getenv("VEO_BUCKET_PATH"),
                "CHIRP3_BUCKET_PATH": os.getenv("CHIRP3_BUCKET_PATH"),
                "LYRIA_BUCKET_PATH": os.getenv("LYRIA_BUCKET_PATH"),
                "AVTOOL_BUCKET_PATH": os.getenv("AVTOOL_BUCKET_PATH"),
                "GOOGLE_APPLICATION_CREDENTIALS": os.getenv("GOOGLE_APPLICATION_CREDENTIALS"),
                "ARIZE_API_KEY": os.getenv("ARIZE_API_KEY"),
                "ARIZE_SPACE_ID": os.getenv("ARIZE_SPACE_ID"),
                "ARIZE_PROJECT_NAME": os.getenv("ARIZE_PROJECT_NAME"),
                "ARIZE_INTERFACE": os.getenv("ARIZE_INTERFACE"),
                "OTEL_EXPORTER_OTLP_TRACES_HEADERS": os.getenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", ""),
                "OTEL_EXPORTER_OTLP_HEADERS": os.getenv("OTEL_EXPORTER_OTLP_HEADERS", ""),
                "OTEL_EXPORTER_OTLP_ENDPOINT": os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
                "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": os.getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", ""),
                "ENABLE_OTEL_TRACING": os.getenv("ENABLE_OTEL_TRACING", "false"),
            },
        ),
        timeout=300,
    ),
)

lyria = MCPToolset(
    connection_params=StdioConnectionParams(
        server_params=StdioServerParameters(
            command="/home/kjdrag/go/bin/mcp-lyria-go",
            args=[],
            env={
                "PROJECT_ID": os.getenv("PROJECT_ID"),
                "LOCATION": os.getenv("LOCATION"),
                "GENMEDIA_BUCKET": os.getenv("GENMEDIA_BUCKET"),
                "IMAGEN_BUCKET_PATH": os.getenv("IMAGEN_BUCKET_PATH"),
                "VEO_BUCKET_PATH": os.getenv("VEO_BUCKET_PATH"),
                "CHIRP3_BUCKET_PATH": os.getenv("CHIRP3_BUCKET_PATH"),
                "LYRIA_BUCKET_PATH": os.getenv("LYRIA_BUCKET_PATH"),
                "AVTOOL_BUCKET_PATH": os.getenv("AVTOOL_BUCKET_PATH"),
                "GOOGLE_APPLICATION_CREDENTIALS": os.getenv("GOOGLE_APPLICATION_CREDENTIALS"),
                "ARIZE_API_KEY": os.getenv("ARIZE_API_KEY"),
                "ARIZE_SPACE_ID": os.getenv("ARIZE_SPACE_ID"),
                "ARIZE_PROJECT_NAME": os.getenv("ARIZE_PROJECT_NAME"),
                "ARIZE_INTERFACE": os.getenv("ARIZE_INTERFACE"),
                "OTEL_EXPORTER_OTLP_TRACES_HEADERS": os.getenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", ""),
                "OTEL_EXPORTER_OTLP_HEADERS": os.getenv("OTEL_EXPORTER_OTLP_HEADERS", ""),
                "OTEL_EXPORTER_OTLP_ENDPOINT": os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
                "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT": os.getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", ""),
                "ENABLE_OTEL_TRACING": os.getenv("ENABLE_OTEL_TRACING", "false"),
            },
        ),
        timeout=180,
    ),
)


root_agent = LlmAgent(
    model=os.getenv("AGENT_MODEL", "gemini-2.0-flash"),
    name='genmedia_agent',
        instruction="""You're a creative assistant that can help users with creating audio, images, video, and music via your generative media tools. You also have the ability to composit these using your available tools.
        
        IMPORTANT: When users upload images, the system automatically saves them as artifacts. You can reference these artifacts in your tool calls:
        1. Check session state for 'image_artifacts' to see available uploaded images
        2. Use artifact filenames with "artifact:" prefix when calling MCP tools like veo_i2v
        3. Example: veo_i2v with image_uri="artifact:user_image_0.jpg"
        
        The image artifact callback automatically handles uploaded images, so you don't need to manually save them.
        
        DO NOT create fake GCS URIs. Always use the artifact system for uploaded images.
        
        After generation, always include a clickable HTTPS link by calling the local tool gcs_signed_url(gs_uri) on every gs:// output object and return both gs_uri and public_url. If the tool fails, explain why and provide the gs:// URI.

        Feel free to be helpful in your suggestions, based on the information you know or can retrieve from your tools.
        If you're asked to translate into other languages, please do.
        """,
    tools=[
       imagen, chirp3, veo, avtool, lyria, FunctionTool(gcs_signed_url),
    ],
    before_agent_callback=before_agent_callback,
    before_tool_callback=before_tool_callback,
)
