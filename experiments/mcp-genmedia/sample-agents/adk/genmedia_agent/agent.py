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
from google.adk.tools.mcp_tool.mcp_toolset import (
    MCPToolset,
    StdioConnectionParams,
    StdioServerParameters,
)

# Arize OpenInference instrumentation for ADK
from arize.otel import register
from openinference.instrumentation.google_adk import GoogleADKInstrumentor

load_dotenv()

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
                "CHIRP3_BUCKET_PATH": os.getenv("CHIRP3_BUCKET_PATH"),
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
                "VEO_BUCKET_PATH": os.getenv("VEO_BUCKET_PATH"),
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
                "LYRIA_BUCKET_PATH": os.getenv("LYRIA_BUCKET_PATH"),
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
        Feel free to be helpful in your suggestions, based on the information you know or can retrieve from your tools.
        If you're asked to translate into other languages, please do.
        """,
    tools=[
       imagen, chirp3, veo, avtool, lyria,
    ],
)
