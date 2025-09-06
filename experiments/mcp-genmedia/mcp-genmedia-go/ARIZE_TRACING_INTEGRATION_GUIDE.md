# Arize Tracing Integration Guide for MCP Operations

## Overview

This guide provides comprehensive instructions for integrating enhanced Arize tracing with OpenInference semantic attributes into your existing MCP handlers. The enhanced tracing provides detailed observability for media generation operations with proper span hierarchy and correlation.

## Problem Analysis

Based on your trace ID `7c4a86e38b5e2ca9e912ff447ac700f5`, the current tracing lacks detail because:

1. **Missing OpenInference Semantic Attributes**: Current spans don't include proper `openinference.span.kind`, `input.value`, `output.value`, etc.
2. **Insufficient MCP Context**: Missing MCP-specific attributes like `mcp.tool.name`, `mcp.operation`, `mcp.parameters`
3. **No Processing Stage Tracking**: Missing detailed stage-by-stage tracing (bucket resolution, API calls, GCS operations)
4. **Limited Error Context**: Errors lack detailed context about which stage failed and why

## Enhanced Tracing Components

### 1. Core Tracing Library (`arize_mcp_tracing.go`)
- **MCPTracer**: Main tracing interface with OpenInference semantic attributes
- **Span Configuration**: Structured configuration for different operation types
- **Error Handling**: Comprehensive error recording with context
- **Event Tracking**: Processing events and milestones

### 2. Service-Specific Tracers
- **LyriaTracer** (`enhanced_lyria_tracing.go`): Music generation tracing
- **VeoTracer** (`enhanced_veo_tracing.go`): Video generation tracing  
- **ImagenTracer** (`enhanced_imagen_tracing.go`): Image generation tracing
- **AVToolTracer** (`enhanced_avtool_tracing.go`): FFmpeg operations tracing

## Integration Steps

### Step 1: Import Enhanced Tracing

```go
// Add to your existing MCP handler imports
import (
    // ... existing imports
    "go.opentelemetry.io/otel/trace"
    "context"
    "time"
)

// Import the enhanced tracer for your service
// For Lyria: use LyriaTracer
// For Veo: use VeoTracer  
// For Imagen: use ImagenTracer
// For AVTool: use AVToolTracer
```

### Step 2: Initialize Tracer in Handler

```go
func lyriaGenerateMusicHandler(client *aiplatform.PredictionClient, ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Initialize the enhanced tracer
    tracer := NewLyriaTracer()
    
    // Extract parameters
    prompt := request.GetArguments()["prompt"].(string)
    model := "lyria-002"
    parameters := request.GetArguments()
    
    // Start comprehensive tracing
    ctx, span := tracer.TraceLyriaGeneration(ctx, prompt, model, parameters)
    defer span.End()
    
    // Get trace ID for logging correlation
    traceID, spanID := tracer.GetTraceInfo(ctx)
    log.Printf("Starting Lyria generation - TraceID: %s, SpanID: %s", traceID, spanID)
    
    // Continue with existing handler logic...
}
```

### Step 3: Add Stage-by-Stage Tracing

```go
func lyriaGenerateMusicHandler(client *aiplatform.PredictionClient, ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    tracer := NewLyriaTracer()
    ctx, span := tracer.TraceLyriaGeneration(ctx, prompt, model, parameters)
    defer span.End()
    
    // 1. Record bucket resolution
    bucket := resolveBucket(parameters)
    tracer.RecordBucketResolution(span, bucket, "environment_variable")
    
    // 2. Record API request
    instanceData := map[string]interface{}{
        "prompt": prompt,
        "sample_count": 1,
    }
    endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", projectID, location, model)
    tracer.RecordLyriaRequest(span, endpoint, instanceData)
    
    // 3. Make API call with timing
    startTime := time.Now()
    response, err := client.Predict(ctx, &aiplatformpb.PredictRequest{
        Endpoint: endpoint,
        Instances: []*structpb.Value{instanceValue},
    })
    
    if err != nil {
        tracer.RecordLyriaError(span, err, "api_request")
        return mcp.NewToolResultError(fmt.Sprintf("Lyria API error: %v", err)), nil
    }
    
    // 4. Record API response
    audioData := extractAudioData(response)
    tracer.RecordLyriaResponse(span, len(audioData), len(base64AudioData))
    
    // 5. Record GCS upload
    objectName := generateObjectName()
    gcsURI := fmt.Sprintf("gs://%s/%s", bucket, objectName)
    
    if err := uploadToGCS(bucket, objectName, audioData); err != nil {
        tracer.RecordLyriaError(span, err, "gcs_upload")
        return mcp.NewToolResultError(fmt.Sprintf("GCS upload error: %v", err)), nil
    }
    
    // 6. Record success
    duration := time.Since(startTime)
    tracer.RecordGCSUpload(span, bucket, objectName, int64(len(audioData)))
    tracer.RecordLyriaSuccess(span, gcsURI, duration, int64(len(audioData)))
    
    return mcp.NewToolResultText(fmt.Sprintf("Music generation completed in %v. Uploaded to GCS: %s.", duration, gcsURI)), nil
}
```

### Step 4: Integrate with Existing Error Handling

```go
// Replace existing error returns with traced errors
if err != nil {
    tracer.RecordLyriaError(span, err, "api_request")
    return mcp.NewToolResultError(fmt.Sprintf("Lyria API error: %v", err)), nil
}

// Add retry tracing if applicable
for attempt := 1; attempt <= maxRetries; attempt++ {
    if attempt > 1 {
        tracer.RecordRetryAttempt(span, attempt, "api_timeout")
    }
    // ... retry logic
}
```

## Service-Specific Integration Examples

### Lyria Music Generation

```go
func lyriaGenerateMusicHandler(client *aiplatform.PredictionClient, ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    tracer := NewLyriaTracer()
    
    prompt := request.GetArguments()["prompt"].(string)
    model := "lyria-002"
    parameters := request.GetArguments()
    
    ctx, span := tracer.TraceLyriaGeneration(ctx, prompt, model, parameters)
    defer span.End()
    
    traceID, _ := tracer.GetTraceInfo(ctx)
    log.Printf("Lyria generation started - TraceID: %s", traceID)
    
    // Bucket resolution
    bucket := resolveBucket(parameters)
    tracer.RecordBucketResolution(span, bucket, "smart_resolution")
    
    // API request
    startTime := time.Now()
    response, err := callLyriaAPI(ctx, prompt, model)
    if err != nil {
        tracer.RecordLyriaError(span, err, "api_request")
        return mcp.NewToolResultError(err.Error()), nil
    }
    
    // Process response
    audioData := extractAudioData(response)
    tracer.RecordLyriaResponse(span, len(audioData), len(base64Data))
    
    // Upload to GCS
    gcsURI, err := uploadToGCS(bucket, audioData)
    if err != nil {
        tracer.RecordLyriaError(span, err, "gcs_upload")
        return mcp.NewToolResultError(err.Error()), nil
    }
    
    // Success
    duration := time.Since(startTime)
    tracer.RecordLyriaSuccess(span, gcsURI, duration, int64(len(audioData)))
    
    return mcp.NewToolResultText(fmt.Sprintf("Generated music: %s", gcsURI)), nil
}
```

### Veo Video Generation

```go
func veoTextToVideoHandler(client *genai.Client, ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    tracer := NewVeoTracer()
    
    prompt := request.GetArguments()["prompt"].(string)
    model := request.GetArguments()["model"].(string)
    parameters := request.GetArguments()
    
    ctx, span := tracer.TraceVeoTextToVideo(ctx, prompt, model, parameters)
    defer span.End()
    
    traceID, _ := tracer.GetTraceInfo(ctx)
    log.Printf("Veo T2V started - TraceID: %s", traceID)
    
    // Bucket resolution
    bucket := resolveBucket(parameters)
    tracer.RecordBucketResolution(span, bucket, "smart_resolution")
    
    // Initiate operation
    startTime := time.Now()
    operationName, err := initiateVideoGeneration(ctx, client, prompt, model, parameters)
    if err != nil {
        tracer.RecordVeoError(span, err, "initiation", "")
        return mcp.NewToolResultError(err.Error()), nil
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
            return mcp.NewToolResultError(err.Error()), nil
        }
        
        if completed {
            totalDuration := time.Since(startTime)
            tracer.RecordVeoCompletion(span, operationName, totalDuration, len(videos))
            
            var outputURIs []string
            for i, video := range videos {
                tracer.RecordVeoVideoResult(span, i, video.URI, video.Size)
                outputURIs = append(outputURIs, video.URI)
            }
            
            tracer.RecordVeoSuccess(span, outputURIs, totalDuration, len(videos))
            return mcp.NewToolResultText(fmt.Sprintf("Generated videos: %s", strings.Join(outputURIs, ", "))), nil
        }
        
        time.Sleep(15 * time.Second)
    }
}
```

### AVTool FFmpeg Operations

```go
func ffmpegCombineAudioVideoHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    tracer := NewAVToolTracer()
    
    videoURI := request.GetArguments()["input_video_uri"].(string)
    audioURI := request.GetArguments()["input_audio_uri"].(string)
    outputFile := request.GetArguments()["output_file_name"].(string)
    parameters := request.GetArguments()
    
    ctx, span := tracer.TraceCombineAudioVideo(ctx, videoURI, audioURI, outputFile, parameters)
    defer span.End()
    
    traceID, _ := tracer.GetTraceInfo(ctx)
    log.Printf("FFmpeg combine started - TraceID: %s", traceID)
    
    // Download files
    videoPath, err := downloadFromGCS(videoURI)
    if err != nil {
        tracer.RecordAVToolError(span, err, "video_download")
        return mcp.NewToolResultError(err.Error()), nil
    }
    tracer.RecordFileDownload(span, videoURI, videoPath, getFileSize(videoPath))
    
    audioPath, err := downloadFromGCS(audioURI)
    if err != nil {
        tracer.RecordAVToolError(span, err, "audio_download")
        return mcp.NewToolResultError(err.Error()), nil
    }
    tracer.RecordFileDownload(span, audioURI, audioPath, getFileSize(audioPath))
    
    // Execute FFmpeg
    startTime := time.Now()
    command := buildFFmpegCommand(videoPath, audioPath, outputPath)
    tracer.RecordFFmpegExecution(span, command, outputPath)
    
    output, err := executeFFmpeg(command)
    if err != nil {
        tracer.RecordAVToolError(span, err, "ffmpeg_execution")
        return mcp.NewToolResultError(err.Error()), nil
    }
    
    duration := time.Since(startTime)
    outputSize := getFileSize(outputPath)
    tracer.RecordFFmpegCompletion(span, duration, outputSize, output)
    
    // Upload result
    gcsURI, err := uploadToGCS(outputPath)
    if err != nil {
        tracer.RecordAVToolError(span, err, "result_upload")
        return mcp.NewToolResultError(err.Error()), nil
    }
    tracer.RecordFileUpload(span, outputPath, gcsURI, outputSize)
    
    // Cleanup
    cleanupPaths := []string{videoPath, audioPath, outputPath}
    cleanup(cleanupPaths)
    tracer.RecordCleanup(span, cleanupPaths)
    
    // Success
    totalDuration := time.Since(startTime)
    tracer.RecordAVToolSuccess(span, gcsURI, totalDuration, outputSize)
    
    return mcp.NewToolResultText(fmt.Sprintf("Combined audio and video: %s", gcsURI)), nil
}
```

## Key OpenInference Attributes

The enhanced tracing automatically adds these critical attributes:

### Core Attributes
- `openinference.span.kind`: "TOOL", "LLM", "CHAIN", "AGENT"
- `input.value`: User prompt or input data
- `output.value`: Generated content URI or result
- `tool_call.function.name`: MCP tool name
- `tool_call.function.arguments`: JSON parameters
- `tool_call.function.output`: Result URI or content

### MCP-Specific Attributes
- `mcp.tool.name`: MCP tool identifier
- `mcp.server.name`: MCP server name
- `mcp.operation`: Operation type (e.g., "text_to_music")
- `mcp.parameters`: JSON request parameters
- `mcp.result`: Operation result
- `mcp.duration_ms`: Operation duration

### Media Generation Attributes
- `media.type`: "image", "video", "music", "audio"
- `media.model`: Model used for generation
- `media.prompt`: Generation prompt
- `media.output.uri`: Generated content GCS URI
- `media.output.size_bytes`: File size
- `media.duration_seconds`: Media duration

### Processing Attributes
- `processing.stage`: Current processing stage
- `processing.status`: "started", "completed", "failed"
- `processing.job_id`: Processing job identifier
- `processing.retry_count`: Number of retry attempts

## Benefits of Enhanced Tracing

### 1. Complete Observability
- **End-to-End Visibility**: Track operations from request to completion
- **Stage-by-Stage Tracking**: See exactly where operations succeed or fail
- **Resource Usage**: Monitor GCS operations, file sizes, processing times

### 2. Better Debugging
- **Correlation IDs**: Link logs, metrics, and traces with trace/span IDs
- **Error Context**: Detailed error information with processing stage
- **Performance Insights**: Identify bottlenecks in media generation pipeline

### 3. Arize Integration
- **Rich Dashboards**: Automatic dashboards for media generation operations
- **Alert Configuration**: Set up alerts for failed operations or performance issues
- **Usage Analytics**: Track model usage, success rates, processing times

### 4. Production Monitoring
- **SLA Monitoring**: Track operation success rates and latencies
- **Resource Optimization**: Identify opportunities for performance improvement
- **Cost Tracking**: Monitor API usage and GCS operations

## Implementation Checklist

- [ ] Import enhanced tracing libraries
- [ ] Initialize service-specific tracer in handlers
- [ ] Add bucket resolution tracing
- [ ] Add API request/response tracing
- [ ] Add GCS operation tracing
- [ ] Add error handling with stage context
- [ ] Add success recording with metrics
- [ ] Test trace correlation with logs
- [ ] Verify Arize dashboard shows detailed traces
- [ ] Configure alerts for failed operations

## Testing Enhanced Tracing

1. **Run Operations**: Execute media generation operations
2. **Check Trace IDs**: Verify trace IDs appear in logs
3. **Validate Arize**: Confirm detailed traces appear in Arize dashboard
4. **Test Error Scenarios**: Verify error tracing with proper context
5. **Performance Validation**: Confirm tracing doesn't impact performance

The enhanced tracing will provide the detailed observability you need to understand exactly what's happening in your MCP operations, making debugging and monitoring much more effective.
