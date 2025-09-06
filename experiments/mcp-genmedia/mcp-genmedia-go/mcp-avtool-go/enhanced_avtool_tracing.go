// enhanced_avtool_tracing.go - Enhanced Arize tracing integration for AVTool MCP server
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

// AVToolTracer provides enhanced tracing for AVTool operations
type AVToolTracer struct {
	tracer trace.Tracer
}

// NewAVToolTracer creates a new AVTool tracer instance
func NewAVToolTracer() *AVToolTracer {
	return &AVToolTracer{
		tracer: otel.Tracer("mcp-avtool"),
	}
}

// TraceFFmpegOperation wraps FFmpeg operations with comprehensive tracing
func (at *AVToolTracer) TraceFFmpegOperation(ctx context.Context, operation string, inputFiles []string, outputFile string, parameters map[string]interface{}) (context.Context, trace.Span) {
	spanName := fmt.Sprintf("ffmpeg_%s", operation)
	inputValue := strings.Join(inputFiles, ", ")
	
	ctx, span := at.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			// OpenInference semantic attributes
			attribute.String("openinference.span.kind", "TOOL"),
			attribute.String("input.value", inputValue),
			attribute.String("tool_call.function.name", spanName),
			
			// MCP-specific attributes
			attribute.String("mcp.tool.name", spanName),
			attribute.String("mcp.server.name", "avtool"),
			attribute.String("mcp.operation", operation),
			
			// FFmpeg-specific attributes
			attribute.String("ffmpeg.operation", operation),
			attribute.String("ffmpeg.input_files", inputValue),
			attribute.String("ffmpeg.output_file", outputFile),
			attribute.String("processing.stage", "preparation"),
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

	// Add input file details
	for i, file := range inputFiles {
		span.SetAttributes(attribute.String(fmt.Sprintf("ffmpeg.input_file_%d", i), file))
	}

	return ctx, span
}

// RecordFileDownload records GCS file download details
func (at *AVToolTracer) RecordFileDownload(span trace.Span, gcsURI string, localPath string, size int64) {
	span.SetAttributes(
		attribute.String("processing.stage", "file_download"),
		attribute.String("gcs.operation", "download"),
		attribute.String("gcs.source_uri", gcsURI),
		attribute.String("ffmpeg.local_path", localPath),
		attribute.String("gcs.size_bytes", fmt.Sprintf("%d", size)),
	)
	
	span.AddEvent("file_downloaded", trace.WithAttributes(
		attribute.String("source", gcsURI),
		attribute.String("destination", localPath),
		attribute.String("size", fmt.Sprintf("%d", size)),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordFFmpegExecution records FFmpeg command execution details
func (at *AVToolTracer) RecordFFmpegExecution(span trace.Span, command string, outputPath string) {
	span.SetAttributes(
		attribute.String("processing.stage", "ffmpeg_execution"),
		attribute.String("ffmpeg.command", command),
		attribute.String("ffmpeg.output_path", outputPath),
	)
	
	span.AddEvent("ffmpeg_command_started", trace.WithAttributes(
		attribute.String("command", command),
		attribute.String("output_path", outputPath),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordFFmpegCompletion records successful FFmpeg execution
func (at *AVToolTracer) RecordFFmpegCompletion(span trace.Span, duration time.Duration, outputSize int64, ffmpegOutput string) {
	span.SetAttributes(
		attribute.String("processing.stage", "ffmpeg_completion"),
		attribute.String("ffmpeg.execution_duration", duration.String()),
		attribute.String("ffmpeg.output_size_bytes", fmt.Sprintf("%d", outputSize)),
	)
	
	// Add last few lines of FFmpeg output for debugging
	lines := strings.Split(strings.TrimSpace(ffmpegOutput), "\n")
	if len(lines) > 0 {
		lastLines := lines
		if len(lines) > 3 {
			lastLines = lines[len(lines)-3:]
		}
		span.SetAttributes(attribute.String("ffmpeg.output_tail", strings.Join(lastLines, " | ")))
	}
	
	span.AddEvent("ffmpeg_command_completed", trace.WithAttributes(
		attribute.String("duration", duration.String()),
		attribute.String("output_size", fmt.Sprintf("%d", outputSize)),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordFileUpload records GCS file upload details
func (at *AVToolTracer) RecordFileUpload(span trace.Span, localPath string, gcsURI string, size int64) {
	span.SetAttributes(
		attribute.String("processing.stage", "file_upload"),
		attribute.String("gcs.operation", "upload"),
		attribute.String("gcs.destination_uri", gcsURI),
		attribute.String("ffmpeg.source_path", localPath),
		attribute.String("gcs.size_bytes", fmt.Sprintf("%d", size)),
	)
	
	span.AddEvent("file_uploaded", trace.WithAttributes(
		attribute.String("source", localPath),
		attribute.String("destination", gcsURI),
		attribute.String("size", fmt.Sprintf("%d", size)),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordAVToolSuccess records successful completion of AVTool operation
func (at *AVToolTracer) RecordAVToolSuccess(span trace.Span, outputURI string, totalDuration time.Duration, outputSize int64) {
	span.SetAttributes(
		attribute.String("output.value", outputURI),
		attribute.String("media.output.uri", outputURI),
		attribute.String("mcp.result", outputURI),
		attribute.String("tool_call.function.output", outputURI),
		attribute.String("mcp.duration_ms", fmt.Sprintf("%.2f", totalDuration.Seconds()*1000)),
		attribute.String("processing.status", "completed"),
		attribute.String("media.output.size_bytes", fmt.Sprintf("%d", outputSize)),
	)

	span.SetStatus(codes.Ok, "AVTool operation completed successfully")
	
	span.AddEvent("avtool_operation_completed", trace.WithAttributes(
		attribute.String("output_uri", outputURI),
		attribute.String("total_duration", totalDuration.String()),
		attribute.String("output_size", fmt.Sprintf("%d", outputSize)),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordAVToolError records an error in AVTool operation
func (at *AVToolTracer) RecordAVToolError(span trace.Span, err error, stage string) {
	span.SetAttributes(
		attribute.String("mcp.error", err.Error()),
		attribute.String("processing.status", "failed"),
		attribute.String("processing.stage", stage),
	)
	
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	
	span.AddEvent("avtool_error_occurred", trace.WithAttributes(
		attribute.String("error", err.Error()),
		attribute.String("stage", stage),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordCleanup records temporary file cleanup
func (at *AVToolTracer) RecordCleanup(span trace.Span, cleanedPaths []string) {
	span.SetAttributes(
		attribute.String("processing.stage", "cleanup"),
		attribute.String("cleanup.paths_count", fmt.Sprintf("%d", len(cleanedPaths))),
	)
	
	for i, path := range cleanedPaths {
		span.SetAttributes(attribute.String(fmt.Sprintf("cleanup.path_%d", i), path))
	}
	
	span.AddEvent("temporary_files_cleaned", trace.WithAttributes(
		attribute.String("paths_count", fmt.Sprintf("%d", len(cleanedPaths))),
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// RecordMediaInfo records media file information
func (at *AVToolTracer) RecordMediaInfo(span trace.Span, mediaInfo map[string]interface{}) {
	span.SetAttributes(attribute.String("processing.stage", "media_info"))
	
	if mediaInfoJSON, err := json.Marshal(mediaInfo); err == nil {
		span.SetAttributes(attribute.String("ffmpeg.media_info", string(mediaInfoJSON)))
	}
	
	// Extract common media properties
	if duration, ok := mediaInfo["duration"]; ok {
		span.SetAttributes(attribute.String("media.duration_seconds", fmt.Sprintf("%v", duration)))
	}
	if width, ok := mediaInfo["width"]; ok {
		span.SetAttributes(attribute.String("media.width", fmt.Sprintf("%v", width)))
	}
	if height, ok := mediaInfo["height"]; ok {
		span.SetAttributes(attribute.String("media.height", fmt.Sprintf("%v", height)))
	}
	if codec, ok := mediaInfo["codec"]; ok {
		span.SetAttributes(attribute.String("media.codec", fmt.Sprintf("%v", codec)))
	}
	
	span.AddEvent("media_info_extracted", trace.WithAttributes(
		attribute.String("timestamp", time.Now().Format(time.RFC3339)),
	))
}

// GetTraceInfo returns trace and span IDs for correlation
func (at *AVToolTracer) GetTraceInfo(ctx context.Context) (string, string) {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String(), span.SpanContext().SpanID().String()
	}
	return "", ""
}

// Operation-specific tracing helpers

// TraceCombineAudioVideo traces audio/video combination operations
func (at *AVToolTracer) TraceCombineAudioVideo(ctx context.Context, videoURI string, audioURI string, outputFile string, parameters map[string]interface{}) (context.Context, trace.Span) {
	inputFiles := []string{videoURI, audioURI}
	ctx, span := at.TraceFFmpegOperation(ctx, "combine_audio_and_video", inputFiles, outputFile, parameters)
	
	// Add specific attributes for audio/video combination
	span.SetAttributes(
		attribute.String("ffmpeg.video_input", videoURI),
		attribute.String("ffmpeg.audio_input", audioURI),
		attribute.String("ffmpeg.combination_type", "audio_video"),
	)
	
	return ctx, span
}

// TraceVideoToGif traces video to GIF conversion operations
func (at *AVToolTracer) TraceVideoToGif(ctx context.Context, videoURI string, outputFile string, parameters map[string]interface{}) (context.Context, trace.Span) {
	inputFiles := []string{videoURI}
	ctx, span := at.TraceFFmpegOperation(ctx, "video_to_gif", inputFiles, outputFile, parameters)
	
	// Add specific attributes for GIF conversion
	if fps, ok := parameters["fps"]; ok {
		span.SetAttributes(attribute.String("ffmpeg.gif_fps", fmt.Sprintf("%v", fps)))
	}
	if scaleFactor, ok := parameters["scale_width_factor"]; ok {
		span.SetAttributes(attribute.String("ffmpeg.gif_scale_factor", fmt.Sprintf("%v", scaleFactor)))
	}
	
	span.SetAttributes(
		attribute.String("ffmpeg.conversion_type", "video_to_gif"),
		attribute.String("media.output_format", "gif"),
	)
	
	return ctx, span
}

// TraceAudioConversion traces audio format conversion operations
func (at *AVToolTracer) TraceAudioConversion(ctx context.Context, inputURI string, outputFile string, fromFormat string, toFormat string, parameters map[string]interface{}) (context.Context, trace.Span) {
	inputFiles := []string{inputURI}
	ctx, span := at.TraceFFmpegOperation(ctx, fmt.Sprintf("convert_audio_%s_to_%s", fromFormat, toFormat), inputFiles, outputFile, parameters)
	
	// Add specific attributes for audio conversion
	span.SetAttributes(
		attribute.String("ffmpeg.input_format", fromFormat),
		attribute.String("ffmpeg.output_format", toFormat),
		attribute.String("ffmpeg.conversion_type", "audio_format"),
		attribute.String("media.input_format", fromFormat),
		attribute.String("media.output_format", toFormat),
	)
	
	return ctx, span
}

// TraceConcatenateMedia traces media file concatenation operations
func (at *AVToolTracer) TraceConcatenateMedia(ctx context.Context, inputURIs []string, outputFile string, parameters map[string]interface{}) (context.Context, trace.Span) {
	ctx, span := at.TraceFFmpegOperation(ctx, "concatenate_media_files", inputURIs, outputFile, parameters)
	
	// Add specific attributes for concatenation
	span.SetAttributes(
		attribute.String("ffmpeg.input_count", fmt.Sprintf("%d", len(inputURIs))),
		attribute.String("ffmpeg.operation_type", "concatenation"),
	)
	
	return ctx, span
}
