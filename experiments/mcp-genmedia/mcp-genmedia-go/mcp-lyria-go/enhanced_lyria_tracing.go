// enhanced_lyria_tracing.go - Enhanced Arize tracing integration for Lyria MCP server
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

// LyriaTracer provides enhanced tracing for Lyria operations
type LyriaTracer struct {
	tracer trace.Tracer
}

// NewLyriaTracer creates a new Lyria tracer instance
func NewLyriaTracer() *LyriaTracer {
	return &LyriaTracer{
		tracer: otel.Tracer("mcp-lyria"),
	}
}

// TraceLyriaGeneration wraps Lyria music generation with comprehensive tracing
func (lt *LyriaTracer) TraceLyriaGeneration(ctx context.Context, prompt string, model string, parameters map[string]interface{}) (context.Context, trace.Span) {
	spanName := "lyria_generate_music"
	
	ctx, span := lt.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			// OpenInference semantic attributes
			attribute.String("openinference.span.kind", "TOOL"),
			attribute.String("input.value", prompt),
			attribute.String("tool_call.function.name", "lyria_generate_music"),
			
			// MCP-specific attributes
			attribute.String("mcp.tool.name", "lyria_generate_music"),
			attribute.String("mcp.server.name", "lyria"),
			attribute.String("mcp.operation", "text_to_music"),
			
			// Media generation attributes
			attribute.String("media.type", "music"),
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

	// Add Lyria-specific attributes
	if sampleCount, ok := parameters["sample_count"]; ok {
		span.SetAttributes(attribute.String("lyria.sample_count", fmt.Sprintf("%v", sampleCount)))
	}
	if seed, ok := parameters["seed"]; ok {
		span.SetAttributes(attribute.String("lyria.seed", fmt.Sprintf("%v", seed)))
	}
	if negativePrompt, ok := parameters["negative_prompt"]; ok {
		span.SetAttributes(attribute.String("lyria.negative_prompt", fmt.Sprintf("%v", negativePrompt)))
	}

	return ctx, span
}

// RecordLyriaRequest records the Lyria API request details
func (lt *LyriaTracer) RecordLyriaRequest(span trace.Span, endpoint string, instanceData map[string]interface{}) {
	span.SetAttributes(
		attribute.String("lyria.endpoint", endpoint),
		attribute.String("processing.stage", "api_request"),
	)
	
	if instanceJSON, err := json.Marshal(instanceData); err == nil {
		span.SetAttributes(attribute.String("lyria.instance_data", string(instanceJSON)))
	}
	
	span.AddEvent("lyria_api_request_sent", trace.WithAttributes(
		attribute.String("endpoint", endpoint),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordLyriaResponse records the Lyria API response details
func (lt *LyriaTracer) RecordLyriaResponse(span trace.Span, audioDataLength int, base64Length int) {
	span.SetAttributes(
		attribute.String("processing.stage", "api_response"),
		attribute.String("lyria.audio_data_length", fmt.Sprintf("%d", audioDataLength)),
		attribute.String("lyria.base64_length", fmt.Sprintf("%d", base64Length)),
	)
	
	span.AddEvent("lyria_api_response_received", trace.WithAttributes(
		attribute.String("audio_data_length", fmt.Sprintf("%d", audioDataLength)),
		attribute.String("base64_length", fmt.Sprintf("%d", base64Length)),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordGCSUpload records GCS upload operation details
func (lt *LyriaTracer) RecordGCSUpload(span trace.Span, bucket string, object string, size int64) {
	span.SetAttributes(
		attribute.String("processing.stage", "gcs_upload"),
		attribute.String("gcs.bucket", bucket),
		attribute.String("gcs.object", object),
		attribute.String("gcs.size_bytes", fmt.Sprintf("%d", size)),
		attribute.String("gcs.operation", "upload"),
	)
	
	span.AddEvent("gcs_upload_completed", trace.WithAttributes(
		attribute.String("bucket", bucket),
		attribute.String("object", object),
		attribute.String("size", fmt.Sprintf("%d", size)),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordLyriaSuccess records successful completion of Lyria generation
func (lt *LyriaTracer) RecordLyriaSuccess(span trace.Span, outputURI string, duration time.Duration, size int64) {
	span.SetAttributes(
		attribute.String("output.value", outputURI),
		attribute.String("media.output.uri", outputURI),
		attribute.String("mcp.result", outputURI),
		attribute.String("tool_call.function.output", outputURI),
		attribute.String("mcp.duration_ms", fmt.Sprintf("%.2f", duration.Seconds()*1000)),
		attribute.String("processing.status", "completed"),
		attribute.String("media.output.size_bytes", fmt.Sprintf("%d", size)),
	)

	span.SetStatus(codes.Ok, "Music generation completed successfully")
	
	span.AddEvent("lyria_generation_completed", trace.WithAttributes(
		attribute.String("output_uri", outputURI),
		attribute.String("duration_ms", fmt.Sprintf("%.2f", duration.Seconds()*1000)),
		attribute.String("size_bytes", fmt.Sprintf("%d", size)),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordLyriaError records an error in Lyria operation
func (lt *LyriaTracer) RecordLyriaError(span trace.Span, err error, stage string) {
	span.SetAttributes(
		attribute.String("mcp.error", err.Error()),
		attribute.String("processing.status", "failed"),
		attribute.String("processing.stage", stage),
	)
	
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	
	span.AddEvent("lyria_error_occurred", trace.WithAttributes(
		attribute.String("error", err.Error()),
		attribute.String("stage", stage),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordBucketResolution records bucket resolution details
func (lt *LyriaTracer) RecordBucketResolution(span trace.Span, resolvedBucket string, resolutionMethod string) {
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

// GetTraceInfo returns trace and span IDs for correlation
func (lt *LyriaTracer) GetTraceInfo(ctx context.Context) (string, string) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String(), span.SpanContext().SpanID().String()
	}
	return "", ""
}

// Example integration with existing Lyria handler:
/*
func lyriaGenerateMusicHandlerWithTracing(client *aiplatform.PredictionClient, ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tracer := NewLyriaTracer()
	
	// Extract parameters
	prompt := request.GetArguments()["prompt"].(string)
	model := "lyria-002"
	parameters := request.GetArguments()
	
	// Start tracing
	ctx, span := tracer.TraceLyriaGeneration(ctx, prompt, model, parameters)
	defer span.End()
	
	// Get trace ID for logging correlation
	traceID, spanID := tracer.GetTraceInfo(ctx)
	log.Printf("Starting Lyria generation - TraceID: %s, SpanID: %s", traceID, spanID)
	
	// Record bucket resolution
	bucket := resolveBucket(parameters)
	tracer.RecordBucketResolution(span, bucket, "environment_variable")
	
	// Prepare request
	instanceData := map[string]interface{}{
		"prompt": prompt,
		"sample_count": 1,
	}
	
	// Record API request
	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", projectID, location, model)
	tracer.RecordLyriaRequest(span, endpoint, instanceData)
	
	// Make API call
	startTime := time.Now()
	response, err := client.Predict(ctx, &aiplatformpb.PredictRequest{
		Endpoint: endpoint,
		Instances: []*structpb.Value{instanceValue},
	})
	
	if err != nil {
		tracer.RecordLyriaError(span, err, "api_request")
		return mcp.NewToolResultError(fmt.Sprintf("Lyria API error: %v", err)), nil
	}
	
	// Process response
	audioData := extractAudioData(response)
	tracer.RecordLyriaResponse(span, len(audioData), len(base64AudioData))
	
	// Upload to GCS
	objectName := generateObjectName()
	gcsURI := fmt.Sprintf("gs://%s/%s", bucket, objectName)
	
	if err := uploadToGCS(bucket, objectName, audioData); err != nil {
		tracer.RecordLyriaError(span, err, "gcs_upload")
		return mcp.NewToolResultError(fmt.Sprintf("GCS upload error: %v", err)), nil
	}
	
	// Record success
	duration := time.Since(startTime)
	tracer.RecordGCSUpload(span, bucket, objectName, int64(len(audioData)))
	tracer.RecordLyriaSuccess(span, gcsURI, duration, int64(len(audioData)))
	
	log.Printf("Lyria generation completed - TraceID: %s, Duration: %v, Output: %s", traceID, duration, gcsURI)
	
	return mcp.NewToolResultText(fmt.Sprintf("Music generation completed in %v. Uploaded to GCS: %s.", duration, gcsURI)), nil
}
*/
