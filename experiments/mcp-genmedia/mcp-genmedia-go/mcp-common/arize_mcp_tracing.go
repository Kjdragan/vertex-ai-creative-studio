// arize_mcp_tracing.go - Enhanced Arize tracing for MCP operations with OpenInference semantic attributes
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// OpenInference semantic attributes for MCP operations
const (
	// Span kinds
	OpenInferenceSpanKind = "openinference.span.kind"
	SpanKindTool         = "TOOL"
	SpanKindAgent        = "AGENT"
	SpanKindChain        = "CHAIN"
	SpanKindLLM          = "LLM"
	SpanKindRetriever    = "RETRIEVER"
	SpanKindEmbedding    = "EMBEDDING"

	// Input/Output attributes
	InputValue  = "input.value"
	OutputValue = "output.value"
	InputMimeType = "input.mime_type"
	OutputMimeType = "output.mime_type"

	// Tool-specific attributes
	ToolCallFunctionName = "tool_call.function.name"
	ToolCallFunctionArgs = "tool_call.function.arguments"
	ToolCallFunctionOutput = "tool_call.function.output"

	// MCP-specific attributes
	MCPToolName = "mcp.tool.name"
	MCPRequestID = "mcp.request.id"
	MCPSessionID = "mcp.session.id"
	MCPServerName = "mcp.server.name"
	MCPOperation = "mcp.operation"
	MCPParameters = "mcp.parameters"
	MCPResult = "mcp.result"
	MCPError = "mcp.error"
	MCPDuration = "mcp.duration_ms"

	// Media generation attributes
	MediaType = "media.type"
	MediaModel = "media.model"
	MediaPrompt = "media.prompt"
	MediaOutputURI = "media.output.uri"
	MediaOutputSize = "media.output.size_bytes"
	MediaDuration = "media.duration_seconds"
	MediaAspectRatio = "media.aspect_ratio"
	MediaQuality = "media.quality"

	// GCS attributes
	GCSBucket = "gcs.bucket"
	GCSObject = "gcs.object"
	GCSOperation = "gcs.operation"
	GCSSize = "gcs.size_bytes"

	// Processing attributes
	ProcessingStage = "processing.stage"
	ProcessingJobID = "processing.job_id"
	ProcessingStatus = "processing.status"
	ProcessingRetryCount = "processing.retry_count"

	// User context attributes
	UserID = "user.id"
	SessionID = "session.id"
	RequestID = "request.id"
)

// MCPTracer provides enhanced tracing for MCP operations
type MCPTracer struct {
	tracer trace.Tracer
	serverName string
}

// NewMCPTracer creates a new MCP tracer instance
func NewMCPTracer(serverName string) *MCPTracer {
	return &MCPTracer{
		tracer: otel.Tracer(fmt.Sprintf("mcp-%s", serverName)),
		serverName: serverName,
	}
}

// MCPSpanConfig holds configuration for MCP spans
type MCPSpanConfig struct {
	ToolName     string
	Operation    string
	RequestID    string
	SessionID    string
	UserID       string
	Parameters   map[string]interface{}
	InputValue   string
	InputMimeType string
}

// StartMCPToolSpan starts a new span for MCP tool execution
func (t *MCPTracer) StartMCPToolSpan(ctx context.Context, config MCPSpanConfig) (context.Context, trace.Span) {
	spanName := fmt.Sprintf("%s_%s", t.serverName, config.ToolName)
	
	ctx, span := t.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String(OpenInferenceSpanKind, SpanKindTool),
			attribute.String(MCPToolName, config.ToolName),
			attribute.String(MCPServerName, t.serverName),
			attribute.String(MCPOperation, config.Operation),
			attribute.String(ToolCallFunctionName, config.ToolName),
		),
	)

	// Add context attributes if provided
	if config.RequestID != "" {
		span.SetAttributes(attribute.String(MCPRequestID, config.RequestID))
		span.SetAttributes(attribute.String(RequestID, config.RequestID))
	}
	if config.SessionID != "" {
		span.SetAttributes(attribute.String(MCPSessionID, config.SessionID))
		span.SetAttributes(attribute.String(SessionID, config.SessionID))
	}
	if config.UserID != "" {
		span.SetAttributes(attribute.String(UserID, config.UserID))
	}

	// Add input attributes
	if config.InputValue != "" {
		span.SetAttributes(attribute.String(InputValue, config.InputValue))
	}
	if config.InputMimeType != "" {
		span.SetAttributes(attribute.String(InputMimeType, config.InputMimeType))
	}

	// Add parameters as JSON
	if config.Parameters != nil {
		if paramsJSON, err := json.Marshal(config.Parameters); err == nil {
			span.SetAttributes(attribute.String(MCPParameters, string(paramsJSON)))
			span.SetAttributes(attribute.String(ToolCallFunctionArgs, string(paramsJSON)))
		}
	}

	return ctx, span
}

// MediaGenerationSpanConfig holds configuration for media generation spans
type MediaGenerationSpanConfig struct {
	MediaType    string // "image", "video", "audio", "music"
	Model        string
	Prompt       string
	Parameters   map[string]interface{}
	OutputBucket string
	JobID        string
}

// StartMediaGenerationSpan starts a span for media generation operations
func (t *MCPTracer) StartMediaGenerationSpan(ctx context.Context, parentSpan trace.Span, config MediaGenerationSpanConfig) (context.Context, trace.Span) {
	spanName := fmt.Sprintf("generate_%s", config.MediaType)
	
	ctx, span := t.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String(OpenInferenceSpanKind, SpanKindLLM), // Media generation is LLM-like
			attribute.String(MediaType, config.MediaType),
			attribute.String(MediaModel, config.Model),
			attribute.String(MediaPrompt, config.Prompt),
			attribute.String(InputValue, config.Prompt),
			attribute.String(ProcessingStage, "generation"),
		),
	)

	if config.OutputBucket != "" {
		span.SetAttributes(attribute.String(GCSBucket, config.OutputBucket))
	}
	if config.JobID != "" {
		span.SetAttributes(attribute.String(ProcessingJobID, config.JobID))
	}

	// Add model parameters
	if config.Parameters != nil {
		if paramsJSON, err := json.Marshal(config.Parameters); err == nil {
			span.SetAttributes(attribute.String("model.parameters", string(paramsJSON)))
		}
	}

	return ctx, span
}

// GCSOperationSpanConfig holds configuration for GCS operation spans
type GCSOperationSpanConfig struct {
	Operation string // "upload", "download", "validate"
	Bucket    string
	Object    string
	Size      int64
}

// StartGCSOperationSpan starts a span for GCS operations
func (t *MCPTracer) StartGCSOperationSpan(ctx context.Context, config GCSOperationSpanConfig) (context.Context, trace.Span) {
	spanName := fmt.Sprintf("gcs_%s", config.Operation)
	
	ctx, span := t.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String(OpenInferenceSpanKind, SpanKindTool),
			attribute.String(GCSOperation, config.Operation),
			attribute.String(GCSBucket, config.Bucket),
			attribute.String(GCSObject, config.Object),
		),
	)

	if config.Size > 0 {
		span.SetAttributes(attribute.String(GCSSize, fmt.Sprintf("%d", config.Size)))
	}

	return ctx, span
}

// RecordMediaGenerationResult records the result of media generation
func (t *MCPTracer) RecordMediaGenerationResult(span trace.Span, outputURI string, size int64, duration time.Duration, metadata map[string]interface{}) {
	span.SetAttributes(
		attribute.String(OutputValue, outputURI),
		attribute.String(MediaOutputURI, outputURI),
		attribute.String(MCPResult, outputURI),
		attribute.String(ToolCallFunctionOutput, outputURI),
		attribute.String(MCPDuration, fmt.Sprintf("%.2f", duration.Seconds()*1000)),
		attribute.String(ProcessingStatus, "completed"),
	)

	if size > 0 {
		span.SetAttributes(attribute.String(MediaOutputSize, fmt.Sprintf("%d", size)))
	}

	// Add metadata attributes
	if metadata != nil {
		for key, value := range metadata {
			if strValue, ok := value.(string); ok {
				span.SetAttributes(attribute.String(fmt.Sprintf("media.%s", key), strValue))
			}
		}
	}

	span.SetStatus(codes.Ok, "Media generation completed successfully")
}

// RecordMCPError records an error in MCP operation
func (t *MCPTracer) RecordMCPError(span trace.Span, err error, stage string) {
	span.SetAttributes(
		attribute.String(MCPError, err.Error()),
		attribute.String(ProcessingStatus, "failed"),
		attribute.String(ProcessingStage, stage),
	)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// AddProcessingEvent adds a processing event to the span
func (t *MCPTracer) AddProcessingEvent(span trace.Span, eventName string, attributes map[string]string) {
	attrs := make([]attribute.KeyValue, 0, len(attributes))
	for key, value := range attributes {
		attrs = append(attrs, attribute.String(key, value))
	}
	span.AddEvent(eventName, trace.WithAttributes(attrs...))
}

// RecordBucketResolution records bucket resolution details
func (t *MCPTracer) RecordBucketResolution(span trace.Span, resolvedBucket string, resolutionMethod string, validated bool) {
	span.SetAttributes(
		attribute.String(GCSBucket, resolvedBucket),
		attribute.String("bucket.resolution_method", resolutionMethod),
		attribute.Bool("bucket.validated", validated),
	)
	
	t.AddProcessingEvent(span, "bucket_resolved", map[string]string{
		"bucket": resolvedBucket,
		"method": resolutionMethod,
		"validated": fmt.Sprintf("%t", validated),
	})
}

// RecordInlineDataProcessing records inline data processing details
func (t *MCPTracer) RecordInlineDataProcessing(span trace.Span, dataType string, originalSize int64, processedSize int64) {
	span.SetAttributes(
		attribute.String("inline_data.type", dataType),
		attribute.String("inline_data.original_size", fmt.Sprintf("%d", originalSize)),
		attribute.String("inline_data.processed_size", fmt.Sprintf("%d", processedSize)),
	)
	
	t.AddProcessingEvent(span, "inline_data_processed", map[string]string{
		"type": dataType,
		"original_size": fmt.Sprintf("%d", originalSize),
		"processed_size": fmt.Sprintf("%d", processedSize),
	})
}

// RecordRetryAttempt records a retry attempt
func (t *MCPTracer) RecordRetryAttempt(span trace.Span, attempt int, reason string) {
	span.SetAttributes(attribute.String(ProcessingRetryCount, fmt.Sprintf("%d", attempt)))
	
	t.AddProcessingEvent(span, "retry_attempt", map[string]string{
		"attempt": fmt.Sprintf("%d", attempt),
		"reason": reason,
	})
}

// Helper functions for common MCP operations

// TraceImageGeneration wraps image generation with comprehensive tracing
func (t *MCPTracer) TraceImageGeneration(ctx context.Context, prompt string, model string, parameters map[string]interface{}) (context.Context, trace.Span) {
	config := MCPSpanConfig{
		ToolName:   "imagen_t2i",
		Operation:  "text_to_image",
		InputValue: prompt,
		Parameters: parameters,
	}
	
	ctx, span := t.StartMCPToolSpan(ctx, config)
	
	// Start nested media generation span
	mediaConfig := MediaGenerationSpanConfig{
		MediaType:  "image",
		Model:      model,
		Prompt:     prompt,
		Parameters: parameters,
	}
	
	return t.StartMediaGenerationSpan(ctx, span, mediaConfig)
}

// TraceVideoGeneration wraps video generation with comprehensive tracing
func (t *MCPTracer) TraceVideoGeneration(ctx context.Context, prompt string, model string, imageURI string, parameters map[string]interface{}) (context.Context, trace.Span) {
	operation := "text_to_video"
	toolName := "veo_t2v"
	inputValue := prompt
	
	if imageURI != "" {
		operation = "image_to_video"
		toolName = "veo_i2v"
		inputValue = fmt.Sprintf("Image: %s, Prompt: %s", imageURI, prompt)
		parameters["image_uri"] = imageURI
	}
	
	config := MCPSpanConfig{
		ToolName:   toolName,
		Operation:  operation,
		InputValue: inputValue,
		Parameters: parameters,
	}
	
	ctx, span := t.StartMCPToolSpan(ctx, config)
	
	// Start nested media generation span
	mediaConfig := MediaGenerationSpanConfig{
		MediaType:  "video",
		Model:      model,
		Prompt:     prompt,
		Parameters: parameters,
	}
	
	return t.StartMediaGenerationSpan(ctx, span, mediaConfig)
}

// TraceMusicGeneration wraps music generation with comprehensive tracing
func (t *MCPTracer) TraceMusicGeneration(ctx context.Context, prompt string, model string, parameters map[string]interface{}) (context.Context, trace.Span) {
	config := MCPSpanConfig{
		ToolName:   "lyria_generate_music",
		Operation:  "text_to_music",
		InputValue: prompt,
		Parameters: parameters,
	}
	
	ctx, span := t.StartMCPToolSpan(ctx, config)
	
	// Start nested media generation span
	mediaConfig := MediaGenerationSpanConfig{
		MediaType:  "music",
		Model:      model,
		Prompt:     prompt,
		Parameters: parameters,
	}
	
	return t.StartMediaGenerationSpan(ctx, span, mediaConfig)
}

// TraceFFmpegOperation wraps FFmpeg operations with tracing
func (t *MCPTracer) TraceFFmpegOperation(ctx context.Context, operation string, inputFiles []string, outputFile string, parameters map[string]interface{}) (context.Context, trace.Span) {
	inputValue := strings.Join(inputFiles, ", ")
	
	config := MCPSpanConfig{
		ToolName:   fmt.Sprintf("ffmpeg_%s", operation),
		Operation:  operation,
		InputValue: inputValue,
		Parameters: parameters,
	}
	
	ctx, span := t.StartMCPToolSpan(ctx, config)
	
	// Add FFmpeg-specific attributes
	span.SetAttributes(
		attribute.String("ffmpeg.operation", operation),
		attribute.String("ffmpeg.input_files", inputValue),
		attribute.String("ffmpeg.output_file", outputFile),
	)
	
	return ctx, span
}

// GetTraceID returns the current trace ID for correlation
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID returns the current span ID for correlation
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}
