// enhanced_veo_tracing.go - Enhanced Arize tracing integration for Veo MCP server
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// VeoTracer provides enhanced tracing for Veo operations
type VeoTracer struct {
	tracer trace.Tracer
}

// NewVeoTracer creates a new Veo tracer instance
func NewVeoTracer() *VeoTracer {
	return &VeoTracer{
		tracer: otel.Tracer("mcp-veo"),
	}
}

// TraceVeoTextToVideo wraps Veo text-to-video generation with comprehensive tracing
func (vt *VeoTracer) TraceVeoTextToVideo(ctx context.Context, prompt string, model string, parameters map[string]interface{}) (context.Context, trace.Span) {
	spanName := "veo_t2v"
	
	ctx, span := vt.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			// OpenInference semantic attributes
			attribute.String("openinference.span.kind", "TOOL"),
			attribute.String("input.value", prompt),
			attribute.String("tool_call.function.name", "veo_t2v"),
			
			// MCP-specific attributes
			attribute.String("mcp.tool.name", "veo_t2v"),
			attribute.String("mcp.server.name", "veo"),
			attribute.String("mcp.operation", "text_to_video"),
			
			// Media generation attributes
			attribute.String("media.type", "video"),
			attribute.String("media.model", model),
			attribute.String("media.prompt", prompt),
			attribute.String("processing.stage", "generation"),
			attribute.String("processing.status", "started"),
		),
	)

	// Add parameters as JSON
	if parameters != nil {
		if paramsJSON, err := json.Marshal(parameters); err == nil {
			span.SetAttributes(
				attribute.String("mcp.parameters", string(paramsJSON)),
				attribute.String("tool_call.function.arguments", string(paramsJSON)),
			)
		}
	}

	// Add Veo-specific attributes
	if duration, ok := parameters["duration"]; ok {
		span.SetAttributes(attribute.String("veo.duration", fmt.Sprintf("%v", duration)))
	}
	if aspectRatio, ok := parameters["aspect_ratio"]; ok {
		span.SetAttributes(attribute.String("veo.aspect_ratio", fmt.Sprintf("%v", aspectRatio)))
		span.SetAttributes(attribute.String("media.aspect_ratio", fmt.Sprintf("%v", aspectRatio)))
	}
	if numVideos, ok := parameters["num_videos"]; ok {
		span.SetAttributes(attribute.String("veo.num_videos", fmt.Sprintf("%v", numVideos)))
	}

	return ctx, span
}

// TraceVeoImageToVideo wraps Veo image-to-video generation with comprehensive tracing
func (vt *VeoTracer) TraceVeoImageToVideo(ctx context.Context, imageURI string, prompt string, model string, parameters map[string]interface{}) (context.Context, trace.Span) {
	spanName := "veo_i2v"
	inputValue := fmt.Sprintf("Image: %s, Prompt: %s", imageURI, prompt)
	
	ctx, span := vt.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			// OpenInference semantic attributes
			attribute.String("openinference.span.kind", "TOOL"),
			attribute.String("input.value", inputValue),
			attribute.String("tool_call.function.name", "veo_i2v"),
			
			// MCP-specific attributes
			attribute.String("mcp.tool.name", "veo_i2v"),
			attribute.String("mcp.server.name", "veo"),
			attribute.String("mcp.operation", "image_to_video"),
			
			// Media generation attributes
			attribute.String("media.type", "video"),
			attribute.String("media.model", model),
			attribute.String("media.prompt", prompt),
			attribute.String("processing.stage", "generation"),
			attribute.String("processing.status", "started"),
			
			// Image input attributes
			attribute.String("veo.image_uri", imageURI),
			attribute.String("input.image.uri", imageURI),
		),
	)

	// Add parameters as JSON
	if parameters != nil {
		if paramsJSON, err := json.Marshal(parameters); err == nil {
			span.SetAttributes(
				attribute.String("mcp.parameters", string(paramsJSON)),
				attribute.String("tool_call.function.arguments", string(paramsJSON)),
			)
		}
	}

	// Add image MIME type if available
	if mimeType, ok := parameters["mime_type"]; ok {
		span.SetAttributes(
			attribute.String("veo.image_mime_type", fmt.Sprintf("%v", mimeType)),
			attribute.String("input.mime_type", fmt.Sprintf("%v", mimeType)),
		)
	}

	return ctx, span
}

// RecordVeoOperation records Veo API operation details
func (vt *VeoTracer) RecordVeoOperation(span trace.Span, operationName string, model string, duration int32, aspectRatio string) {
	span.SetAttributes(
		attribute.String("processing.stage", "api_request"),
		attribute.String("veo.operation_name", operationName),
		attribute.String("veo.model_used", model),
		attribute.String("veo.video_duration", fmt.Sprintf("%d", duration)),
		attribute.String("veo.aspect_ratio", aspectRatio),
		attribute.String("media.duration_seconds", fmt.Sprintf("%d", duration)),
		attribute.String("media.aspect_ratio", aspectRatio),
	)
	
	span.AddEvent("veo_operation_initiated", trace.WithAttributes(
		attribute.String("operation", operationName),
		attribute.String("model", model),
		attribute.String("duration", fmt.Sprintf("%d", duration)),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordVeoPolling records polling attempts for long-running operations
func (vt *VeoTracer) RecordVeoPolling(span trace.Span, operationName string, attempt int, elapsed time.Duration) {
	span.SetAttributes(
		attribute.String("processing.stage", "polling"),
		attribute.String("veo.polling_attempt", fmt.Sprintf("%d", attempt)),
		attribute.String("veo.elapsed_time", elapsed.String()),
	)
	
	span.AddEvent("veo_operation_polling", trace.WithAttributes(
		attribute.String("operation", operationName),
		attribute.String("attempt", fmt.Sprintf("%d", attempt)),
		attribute.String("elapsed", elapsed.String()),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordVeoCompletion records successful completion of Veo operation
func (vt *VeoTracer) RecordVeoCompletion(span trace.Span, operationName string, totalDuration time.Duration, videoCount int) {
	span.SetAttributes(
		attribute.String("processing.stage", "completion"),
		attribute.String("veo.total_duration", totalDuration.String()),
		attribute.String("veo.video_count", fmt.Sprintf("%d", videoCount)),
	)
	
	span.AddEvent("veo_operation_completed", trace.WithAttributes(
		attribute.String("operation", operationName),
		attribute.String("total_duration", totalDuration.String()),
		attribute.String("video_count", fmt.Sprintf("%d", videoCount)),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordVeoVideoResult records individual video generation results
func (vt *VeoTracer) RecordVeoVideoResult(span trace.Span, videoIndex int, gcsURI string, size int64) {
	span.SetAttributes(
		attribute.String("processing.stage", "result_processing"),
		attribute.String(fmt.Sprintf("veo.video_%d.uri", videoIndex), gcsURI),
		attribute.String(fmt.Sprintf("veo.video_%d.size_bytes", videoIndex), fmt.Sprintf("%d", size)),
	)
	
	span.AddEvent("veo_video_result", trace.WithAttributes(
		attribute.String("video_index", fmt.Sprintf("%d", videoIndex)),
		attribute.String("gcs_uri", gcsURI),
		attribute.String("size_bytes", fmt.Sprintf("%d", size)),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordVeoSuccess records successful completion of Veo generation
func (vt *VeoTracer) RecordVeoSuccess(span trace.Span, outputURIs []string, totalDuration time.Duration, videoCount int) {
	primaryOutput := ""
	if len(outputURIs) > 0 {
		primaryOutput = outputURIs[0]
	}
	
	span.SetAttributes(
		attribute.String("output.value", primaryOutput),
		attribute.String("media.output.uri", primaryOutput),
		attribute.String("mcp.result", primaryOutput),
		attribute.String("tool_call.function.output", primaryOutput),
		attribute.String("mcp.duration_ms", fmt.Sprintf("%.2f", totalDuration.Seconds()*1000)),
		attribute.String("processing.status", "completed"),
		attribute.String("veo.video_count", fmt.Sprintf("%d", videoCount)),
	)

	// Add all output URIs
	for i, uri := range outputURIs {
		span.SetAttributes(attribute.String(fmt.Sprintf("veo.output_uri_%d", i), uri))
	}

	span.SetStatus(codes.Ok, "Video generation completed successfully")
	
	span.AddEvent("veo_generation_completed", trace.WithAttributes(
		attribute.String("primary_output", primaryOutput),
		attribute.String("video_count", fmt.Sprintf("%d", videoCount)),
		attribute.String("total_duration", totalDuration.String()),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordVeoError records an error in Veo operation
func (vt *VeoTracer) RecordVeoError(span trace.Span, err error, stage string, operationName string) {
	span.SetAttributes(
		attribute.String("mcp.error", err.Error()),
		attribute.String("processing.status", "failed"),
		attribute.String("processing.stage", stage),
		attribute.String("veo.failed_operation", operationName),
	)
	
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	
	span.AddEvent("veo_error_occurred", trace.WithAttributes(
		attribute.String("error", err.Error()),
		attribute.String("stage", stage),
		attribute.String("operation", operationName),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordBucketResolution records bucket resolution details
func (vt *VeoTracer) RecordBucketResolution(span trace.Span, resolvedBucket string, resolutionMethod string) {
	span.SetAttributes(
		attribute.String("gcs.bucket", resolvedBucket),
		attribute.String("bucket.resolution_method", resolutionMethod),
	)
	
	span.AddEvent("bucket_resolved", trace.WithAttributes(
		attribute.String("bucket", resolvedBucket),
		attribute.String("method", resolutionMethod),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordInlineImageProcessing records inline image processing details
func (vt *VeoTracer) RecordInlineImageProcessing(span trace.Span, originalSize int64, processedSize int64, mimeType string) {
	span.SetAttributes(
		attribute.String("processing.stage", "inline_image_processing"),
		attribute.String("inline_image.original_size", fmt.Sprintf("%d", originalSize)),
		attribute.String("inline_image.processed_size", fmt.Sprintf("%d", processedSize)),
		attribute.String("inline_image.mime_type", mimeType),
	)
	
	span.AddEvent("inline_image_processed", trace.WithAttributes(
		attribute.String("original_size", fmt.Sprintf("%d", originalSize)),
		attribute.String("processed_size", fmt.Sprintf("%d", processedSize)),
		attribute.String("mime_type", mimeType),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// GetTraceInfo returns trace and span IDs for correlation
func (vt *VeoTracer) GetTraceInfo(ctx context.Context) (string, string) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String(), span.SpanContext().SpanID().String()
	}
	return "", ""
}

// Example integration with existing Veo handlers:
/*
func veoTextToVideoHandlerWithTracing(client *genai.Client, ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tracer := NewVeoTracer()
	
	// Extract parameters
	prompt := request.GetArguments()["prompt"].(string)
	model := request.GetArguments()["model"].(string)
	parameters := request.GetArguments()
	
	// Start tracing
	ctx, span := tracer.TraceVeoTextToVideo(ctx, prompt, model, parameters)
	defer span.End()
	
	// Get trace ID for logging correlation
	traceID, spanID := tracer.GetTraceInfo(ctx)
	log.Printf("Starting Veo T2V generation - TraceID: %s, SpanID: %s", traceID, spanID)
	
	// Record bucket resolution
	bucket := resolveBucket(parameters)
	tracer.RecordBucketResolution(span, bucket, "smart_resolution")
	
	// Initiate video generation
	startTime := time.Now()
	operationName, err := initiateVideoGeneration(ctx, client, prompt, model, parameters)
	if err != nil {
		tracer.RecordVeoError(span, err, "initiation", "")
		return mcp.NewToolResultError(fmt.Sprintf("Failed to initiate video generation: %v", err)), nil
	}
	
	tracer.RecordVeoOperation(span, operationName, model, duration, aspectRatio)
	
	// Poll for completion
	attempt := 0
	for {
		attempt++
		elapsed := time.Since(startTime)
		tracer.RecordVeoPolling(span, operationName, attempt, elapsed)
		
		completed, videos, err := checkOperationStatus(ctx, operationName)
		if err != nil {
			tracer.RecordVeoError(span, err, "polling", operationName)
			return mcp.NewToolResultError(fmt.Sprintf("Operation polling failed: %v", err)), nil
		}
		
		if completed {
			totalDuration := time.Since(startTime)
			tracer.RecordVeoCompletion(span, operationName, totalDuration, len(videos))
			
			// Record individual video results
			var outputURIs []string
			for i, video := range videos {
				tracer.RecordVeoVideoResult(span, i, video.URI, video.Size)
				outputURIs = append(outputURIs, video.URI)
			}
			
			tracer.RecordVeoSuccess(span, outputURIs, totalDuration, len(videos))
			
			log.Printf("Veo T2V completed - TraceID: %s, Duration: %v, Videos: %d", traceID, totalDuration, len(videos))
			
			return mcp.NewToolResultText(fmt.Sprintf("Generated %d video(s) using model %s. This took about %v. Videos saved to GCS: %s.", 
				len(videos), model, totalDuration, strings.Join(outputURIs, ", "))), nil
		}
		
		time.Sleep(15 * time.Second)
	}
}
*/
