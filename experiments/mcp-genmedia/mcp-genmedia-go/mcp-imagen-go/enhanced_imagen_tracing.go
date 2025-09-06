// enhanced_imagen_tracing.go - Enhanced Arize tracing integration for Imagen MCP server
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

// ImagenTracer provides enhanced tracing for Imagen operations
type ImagenTracer struct {
	tracer trace.Tracer
}

// NewImagenTracer creates a new Imagen tracer instance
func NewImagenTracer() *ImagenTracer {
	return &ImagenTracer{
		tracer: otel.Tracer("mcp-imagen"),
	}
}

// TraceImagenGeneration wraps Imagen text-to-image generation with comprehensive tracing
func (it *ImagenTracer) TraceImagenGeneration(ctx context.Context, prompt string, model string, parameters map[string]interface{}) (context.Context, trace.Span) {
	spanName := "imagen_t2i"
	
	ctx, span := it.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			// OpenInference semantic attributes
			attribute.String("openinference.span.kind", "TOOL"),
			attribute.String("input.value", prompt),
			attribute.String("tool_call.function.name", "imagen_t2i"),
			
			// MCP-specific attributes
			attribute.String("mcp.tool.name", "imagen_t2i"),
			attribute.String("mcp.server.name", "imagen"),
			attribute.String("mcp.operation", "text_to_image"),
			
			// Media generation attributes
			attribute.String("media.type", "image"),
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

	// Add Imagen-specific attributes
	if numImages, ok := parameters["num_images"]; ok {
		span.SetAttributes(attribute.String("imagen.num_images", fmt.Sprintf("%v", numImages)))
	}
	if aspectRatio, ok := parameters["aspect_ratio"]; ok {
		span.SetAttributes(
			attribute.String("imagen.aspect_ratio", fmt.Sprintf("%v", aspectRatio)),
			attribute.String("media.aspect_ratio", fmt.Sprintf("%v", aspectRatio)),
		)
	}
	if seed, ok := parameters["seed"]; ok {
		span.SetAttributes(attribute.String("imagen.seed", fmt.Sprintf("%v", seed)))
	}

	return ctx, span
}

// TraceImagenInpainting wraps Imagen inpainting operations with comprehensive tracing
func (it *ImagenTracer) TraceImagenInpainting(ctx context.Context, operation string, imageURI string, prompt string, parameters map[string]interface{}) (context.Context, trace.Span) {
	spanName := fmt.Sprintf("imagen_edit_%s", operation)
	inputValue := fmt.Sprintf("Image: %s, Prompt: %s", imageURI, prompt)
	
	ctx, span := it.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			// OpenInference semantic attributes
			attribute.String("openinference.span.kind", "TOOL"),
			attribute.String("input.value", inputValue),
			attribute.String("tool_call.function.name", spanName),
			
			// MCP-specific attributes
			attribute.String("mcp.tool.name", spanName),
			attribute.String("mcp.server.name", "imagen"),
			attribute.String("mcp.operation", fmt.Sprintf("image_%s", operation)),
			
			// Media generation attributes
			attribute.String("media.type", "image"),
			attribute.String("media.prompt", prompt),
			attribute.String("processing.stage", "editing"),
			attribute.String("processing.status", "started"),
			
			// Image editing attributes
			attribute.String("imagen.operation", operation),
			attribute.String("imagen.input_image_uri", imageURI),
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

	// Add inpainting-specific attributes
	if maskMode, ok := parameters["mask_mode"]; ok {
		span.SetAttributes(attribute.String("imagen.mask_mode", fmt.Sprintf("%v", maskMode)))
	}
	if maskDilation, ok := parameters["mask_dilation"]; ok {
		span.SetAttributes(attribute.String("imagen.mask_dilation", fmt.Sprintf("%v", maskDilation)))
	}
	if segmentationClasses, ok := parameters["segmentation_classes"]; ok {
		span.SetAttributes(attribute.String("imagen.segmentation_classes", fmt.Sprintf("%v", segmentationClasses)))
	}

	return ctx, span
}

// RecordImagenRequest records Imagen API request details
func (it *ImagenTracer) RecordImagenRequest(span trace.Span, model string, requestData map[string]interface{}) {
	span.SetAttributes(
		attribute.String("processing.stage", "api_request"),
		attribute.String("imagen.model_used", model),
	)
	
	if requestJSON, err := json.Marshal(requestData); err == nil {
		span.SetAttributes(attribute.String("imagen.request_data", string(requestJSON)))
	}
	
	span.AddEvent("imagen_api_request_sent", trace.WithAttributes(
		attribute.String("model", model),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordImagenResponse records Imagen API response details
func (it *ImagenTracer) RecordImagenResponse(span trace.Span, imageCount int, totalSize int64) {
	span.SetAttributes(
		attribute.String("processing.stage", "api_response"),
		attribute.String("imagen.image_count", fmt.Sprintf("%d", imageCount)),
		attribute.String("imagen.total_size_bytes", fmt.Sprintf("%d", totalSize)),
	)
	
	span.AddEvent("imagen_api_response_received", trace.WithAttributes(
		attribute.String("image_count", fmt.Sprintf("%d", imageCount)),
		attribute.String("total_size", fmt.Sprintf("%d", totalSize)),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordImageResult records individual image generation results
func (it *ImagenTracer) RecordImageResult(span trace.Span, imageIndex int, gcsURI string, httpsURL string, size int64) {
	span.SetAttributes(
		attribute.String("processing.stage", "result_processing"),
		attribute.String(fmt.Sprintf("imagen.image_%d.gcs_uri", imageIndex), gcsURI),
		attribute.String(fmt.Sprintf("imagen.image_%d.https_url", imageIndex), httpsURL),
		attribute.String(fmt.Sprintf("imagen.image_%d.size_bytes", imageIndex), fmt.Sprintf("%d", size)),
	)
	
	span.AddEvent("imagen_image_result", trace.WithAttributes(
		attribute.String("image_index", fmt.Sprintf("%d", imageIndex)),
		attribute.String("gcs_uri", gcsURI),
		attribute.String("https_url", httpsURL),
		attribute.String("size_bytes", fmt.Sprintf("%d", size)),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordImagenSuccess records successful completion of Imagen generation
func (it *ImagenTracer) RecordImagenSuccess(span trace.Span, outputURIs []string, httpsURLs []string, duration time.Duration, imageCount int) {
	primaryOutput := ""
	primaryHTTPS := ""
	if len(outputURIs) > 0 {
		primaryOutput = outputURIs[0]
	}
	if len(httpsURLs) > 0 {
		primaryHTTPS = httpsURLs[0]
	}
	
	span.SetAttributes(
		attribute.String("output.value", primaryOutput),
		attribute.String("media.output.uri", primaryOutput),
		attribute.String("mcp.result", primaryOutput),
		attribute.String("tool_call.function.output", primaryOutput),
		attribute.String("mcp.duration_ms", fmt.Sprintf("%.2f", duration.Seconds()*1000)),
		attribute.String("processing.status", "completed"),
		attribute.String("imagen.image_count", fmt.Sprintf("%d", imageCount)),
		attribute.String("imagen.primary_https_url", primaryHTTPS),
	)

	// Add all output URIs and HTTPS URLs
	for i, uri := range outputURIs {
		span.SetAttributes(attribute.String(fmt.Sprintf("imagen.output_uri_%d", i), uri))
	}
	for i, url := range httpsURLs {
		span.SetAttributes(attribute.String(fmt.Sprintf("imagen.https_url_%d", i), url))
	}

	span.SetStatus(codes.Ok, "Image generation completed successfully")
	
	span.AddEvent("imagen_generation_completed", trace.WithAttributes(
		attribute.String("primary_output", primaryOutput),
		attribute.String("primary_https", primaryHTTPS),
		attribute.String("image_count", fmt.Sprintf("%d", imageCount)),
		attribute.String("total_duration", duration.String()),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordImagenError records an error in Imagen operation
func (it *ImagenTracer) RecordImagenError(span trace.Span, err error, stage string) {
	span.SetAttributes(
		attribute.String("mcp.error", err.Error()),
		attribute.String("processing.status", "failed"),
		attribute.String("processing.stage", stage),
	)
	
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	
	span.AddEvent("imagen_error_occurred", trace.WithAttributes(
		attribute.String("error", err.Error()),
		attribute.String("stage", stage),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordBucketResolution records bucket resolution details
func (it *ImagenTracer) RecordBucketResolution(span trace.Span, resolvedBucket string, resolutionMethod string) {
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

// RecordGCSUpload records GCS upload operation details
func (it *ImagenTracer) RecordGCSUpload(span trace.Span, bucket string, object string, size int64) {
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

// GetTraceInfo returns trace and span IDs for correlation
func (it *ImagenTracer) GetTraceInfo(ctx context.Context) (string, string) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String(), span.SpanContext().SpanID().String()
	}
	return "", ""
}
