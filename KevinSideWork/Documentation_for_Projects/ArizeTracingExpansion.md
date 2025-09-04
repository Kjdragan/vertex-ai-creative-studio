# Arize Tracing Implementation Plan for Google GenAI Media Master Repository

## Overview

This document outlines the comprehensive implementation plan for integrating Arize tracing across our Google GenAI media generation system. The implementation will provide end-to-end observability for the Python ADK agent, Go MCP servers, and Google Gemini 2.5-pro Vertex AI calls.

## Architecture Components

### 1. Python ADK Agent (`agent.py`)
- **Current State**: Basic OpenTelemetry setup with TracerProvider conflicts
- **Target State**: Full instrumentation with Arize's `register()` function to resolve conflicts
- **Key Instrumentations**:
  - `openinference-instrumentation-google-adk` for ADK framework
  - `openinference-instrumentation-vertexai` for Gemini calls
  - `openinference-instrumentation-mcp` for MCP server communication

### 2. Go MCP Servers (5 servers)
- **Current State**: Basic OTEL setup with endpoint and headers configured
- **Target State**: Enhanced with manual spans for media generation operations
- **Servers**: Imagen, Veo, Lyria, Chirp3, AVTool

### 3. Google Vertex AI Integration
- **Current State**: Direct Gemini 2.5-pro API calls without tracing
- **Target State**: Full instrumentation with OpenInference semantic conventions

## Implementation Plan

### Phase 1: Python ADK Agent Enhancement

#### 1.1 Install Required Dependencies
Add to `requirements.txt`:
```
arize-otel>=0.1.0
openinference-instrumentation-google-adk>=0.1.0
openinference-instrumentation-vertexai>=0.1.0
openinference-instrumentation-mcp>=0.1.0
openinference-semantic-conventions>=0.1.0
```

#### 1.2 TracerProvider Conflict Resolution
Replace existing OpenTelemetry setup in `agent.py` with Arize's `register()` function:

```python
from arize.otel import register
from openinference.instrumentation.google_adk import GoogleADKInstrumentor
from openinference.instrumentation.vertexai import VertexAIInstrumentor
from openinference.instrumentation.mcp import MCPInstrumentor

# Use Arize's register function to avoid TracerProvider conflicts
tracer_provider = register(
    space_id=os.getenv("ARIZE_SPACE_ID"),
    api_key=os.getenv("ARIZE_API_KEY"),
    project_name="google-genai-media-master",
    endpoint="https://otlp.arize.com/v1"
)

# Auto-instrument frameworks
GoogleADKInstrumentor().instrument(tracer_provider=tracer_provider)
VertexAIInstrumentor().instrument(tracer_provider=tracer_provider)
MCPInstrumentor().instrument(tracer_provider=tracer_provider)
```

#### 1.3 Manual Span Addition
Add manual spans for key operations:
- MCP server initialization
- Media generation requests
- GCS upload operations
- Error handling

### Phase 2: Go MCP Server Enhancement

#### 2.1 Manual Span Implementation
For each Go MCP server, enhance the existing OTEL setup with manual spans:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

// Get tracer for the service
tracer := otel.Tracer("mcp-imagen-server")

// Create spans for media generation operations
func generateImage(ctx context.Context, prompt string) error {
    ctx, span := tracer.Start(ctx, "imagen.generate_image",
        trace.WithAttributes(
            attribute.String("openinference.span.kind", "LLM"),
            attribute.String("llm.model_name", "imagen-3.0"),
            attribute.String("llm.provider", "google"),
            attribute.String("input.value", prompt),
        ),
    )
    defer span.End()
    
    // Add operation-specific attributes
    span.SetAttributes(
        attribute.String("media.type", "image"),
        attribute.String("media.format", "png"),
    )
    
    // Perform image generation
    result, err := performImageGeneration(ctx, prompt)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }
    
    // Add output attributes
    span.SetAttributes(
        attribute.String("output.value", result.URL),
        attribute.Int("media.size_bytes", result.SizeBytes),
    )
    
    return nil
}
```

#### 2.2 GCS Upload Tracing
Add spans for Google Cloud Storage operations:

```go
func uploadToGCS(ctx context.Context, data []byte, bucketPath string) error {
    ctx, span := tracer.Start(ctx, "gcs.upload",
        trace.WithAttributes(
            attribute.String("gcs.bucket_path", bucketPath),
            attribute.Int("gcs.object_size", len(data)),
        ),
    )
    defer span.End()
    
    // Perform upload with tracing
    // ...
}
```

### Phase 3: OpenInference Semantic Conventions

#### 3.1 LLM Spans
Use OpenInference semantic conventions for AI operations:

**Python Example:**
```python
from openinference.semconv.trace import SpanAttributes, OpenInferenceSpanKindValues

with tracer.start_as_current_span("gemini.generate_content") as span:
    span.set_attribute(SpanAttributes.OPENINFERENCE_SPAN_KIND, OpenInferenceSpanKindValues.LLM.value)
    span.set_attribute(SpanAttributes.LLM_MODEL_NAME, "gemini-2.5-pro")
    span.set_attribute(SpanAttributes.LLM_PROVIDER, "google")
    span.set_attribute(SpanAttributes.INPUT_VALUE, prompt)
    
    # Make API call
    response = model.generate_content(prompt)
    
    span.set_attribute(SpanAttributes.OUTPUT_VALUE, response.text)
    span.set_attribute(SpanAttributes.LLM_TOKEN_COUNT_PROMPT, response.usage_metadata.prompt_token_count)
    span.set_attribute(SpanAttributes.LLM_TOKEN_COUNT_COMPLETION, response.usage_metadata.candidates_token_count)
```

#### 3.2 Tool Spans
For MCP tool calls:

```python
span.set_attribute(SpanAttributes.OPENINFERENCE_SPAN_KIND, OpenInferenceSpanKindValues.TOOL.value)
span.set_attribute(SpanAttributes.TOOL_NAME, "generate_image")
span.set_attribute(SpanAttributes.TOOL_PARAMETERS, json.dumps(parameters))
```

### Phase 4: Environment Configuration

#### 4.1 Update Environment Variables
Ensure all required variables are set in `.env` and `export_env.sh`:

```bash
# Arize Configuration
export ARIZE_API_KEY="your-arize-api-key"
export ARIZE_SPACE_ID="your-arize-space-id"
export ARIZE_PROJECT_NAME="google-genai-media-master"

# OpenTelemetry Configuration
export ENABLE_OTEL_TRACING="true"
export OTEL_EXPORTER_OTLP_ENDPOINT="https://otlp.arize.com/v1"
export OTEL_EXPORTER_OTLP_TRACES_HEADERS="space_id=${ARIZE_SPACE_ID},api_key=${ARIZE_API_KEY}"

# Service Names for Go servers
export OTEL_SERVICE_NAME_IMAGEN="mcp-imagen-server"
export OTEL_SERVICE_NAME_VEO="mcp-veo-server"
export OTEL_SERVICE_NAME_LYRIA="mcp-lyria-server"
export OTEL_SERVICE_NAME_CHIRP3="mcp-chirp3-server"
export OTEL_SERVICE_NAME_AVTOOL="mcp-avtool-server"
```

#### 4.2 Pass Environment Variables to MCP Servers
Update `agent.py` to pass tracing configuration to all MCP servers:

```python
# Add to each MCP server configuration
"OTEL_SERVICE_NAME": f"mcp-{server_name}-server",
"OTEL_EXPORTER_OTLP_ENDPOINT": os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://otlp.arize.com/v1"),
"OTEL_EXPORTER_OTLP_TRACES_HEADERS": f"space_id={arize_space_id},api_key={arize_api_key}",
```

### Phase 5: Context Propagation

#### 5.1 Cross-Service Tracing
Ensure trace context propagates from Python agent to Go MCP servers:

**Python (agent.py):**
```python
from opentelemetry.propagate import inject
from opentelemetry import context

# Inject trace context into MCP requests
headers = {}
inject(headers, context=context.get_current())
# Pass headers to MCP server calls
```

**Go (MCP servers):**
```go
import (
    "go.opentelemetry.io/otel/propagation"
)

// Extract trace context from incoming requests
ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(headers))
```

### Phase 6: Error Handling and Monitoring

#### 6.1 Error Span Attributes
Standardize error handling across all components:

```python
try:
    # Operation
    pass
except Exception as e:
    span.record_exception(e)
    span.set_status(trace.Status(trace.StatusCode.ERROR, str(e)))
    span.set_attribute("error.type", type(e).__name__)
    span.set_attribute("error.message", str(e))
    raise
```

#### 6.2 Performance Metrics
Add performance-related attributes:

```python
span.set_attribute("operation.duration_ms", duration_ms)
span.set_attribute("media.generation_time_ms", generation_time)
span.set_attribute("gcs.upload_time_ms", upload_time)
```

## Implementation Timeline

### Week 1: Foundation
- [ ] Install Python dependencies
- [ ] Implement Arize `register()` function in agent.py
- [ ] Update environment configuration
- [ ] Test basic tracing functionality

### Week 2: Python Enhancement
- [ ] Add auto-instrumentation for Google ADK, Vertex AI, and MCP
- [ ] Implement manual spans for key operations
- [ ] Add OpenInference semantic conventions
- [ ] Test end-to-end Python tracing

### Week 3: Go Enhancement
- [ ] Enhance Go MCP servers with manual spans
- [ ] Implement media generation tracing
- [ ] Add GCS upload tracing
- [ ] Test cross-service context propagation

### Week 4: Integration & Testing
- [ ] End-to-end testing of complete tracing pipeline
- [ ] Performance optimization
- [ ] Documentation updates
- [ ] Monitoring setup in Arize dashboard

## Expected Benefits

### 1. Observability
- Complete visibility into media generation pipeline
- Request flow tracking across Python and Go services
- Performance bottleneck identification

### 2. Debugging
- Detailed error context and stack traces
- Request correlation across service boundaries
- Failed operation root cause analysis

### 3. Performance Monitoring
- Media generation latency tracking
- GCS upload performance metrics
- Resource utilization insights

### 4. Cost Analytics
- Token usage tracking for Gemini API calls
- Operation cost attribution
- Usage pattern analysis

## Success Metrics

1. **Trace Completeness**: 100% of media generation requests traced end-to-end
2. **Error Visibility**: All errors captured with full context
3. **Performance Insights**: Sub-second trace data availability in Arize
4. **Cost Tracking**: Accurate token and API usage metrics

## Risk Mitigation

### 1. Performance Impact
- Use BatchSpanProcessor for non-blocking operation
- Implement sampling for high-volume operations
- Monitor trace overhead

### 2. Configuration Complexity
- Centralized environment variable management
- Comprehensive documentation
- Automated configuration validation

### 3. Service Dependencies
- Graceful degradation when tracing is unavailable
- Toggle-based tracing enable/disable
- Fallback to console logging

## Next Steps

1. Review and approve this implementation plan
2. Begin Phase 1 implementation with Python ADK agent enhancement
3. Set up Arize dashboard for monitoring
4. Establish testing procedures for each phase

---

*This document serves as the definitive guide for implementing comprehensive Arize tracing across the Google GenAI Media Master Repository. It should be updated as implementation progresses and lessons are learned.*
