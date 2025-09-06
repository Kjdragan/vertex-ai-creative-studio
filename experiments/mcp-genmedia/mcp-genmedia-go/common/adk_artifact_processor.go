package common

import (
	"context"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/storage"
)

// ADKArtifactProcessor handles ADK artifact references in MCP tool parameters
type ADKArtifactProcessor struct {
	gcsClient *storage.Client
}

// NewADKArtifactProcessor creates a new artifact processor
func NewADKArtifactProcessor(gcsClient *storage.Client) *ADKArtifactProcessor {
	return &ADKArtifactProcessor{
		gcsClient: gcsClient,
	}
}

// ProcessArtifactParameter processes a parameter that might be an ADK artifact reference
// Returns the GCS URI if it's an artifact, or the original parameter if it's already a GCS URI
func (p *ADKArtifactProcessor) ProcessArtifactParameter(ctx context.Context, param interface{}, userBucket string) (*ImageProcessingResult, error) {
	if param == nil {
		return &ImageProcessingResult{
			Error: fmt.Errorf("parameter is nil"),
		}, nil
	}

	paramStr, ok := param.(string)
	if !ok {
		// If it's not a string, try to process as inline data
		return ProcessMCPImageParameterWithBucketResolution(ctx, p.gcsClient, param, userBucket), nil
	}

	// Check if this is an artifact reference (starts with "artifact:")
	if strings.HasPrefix(paramStr, "artifact:") {
		artifactFilename := strings.TrimPrefix(paramStr, "artifact:")
		log.Printf("Processing ADK artifact reference: %s", artifactFilename)
		
		// For now, we'll return an error indicating that artifact loading needs to be implemented
		// In a full implementation, this would interface with the ADK artifact service
		return &ImageProcessingResult{
			Error: fmt.Errorf("ADK artifact processing not yet implemented for artifact: %s. Please use direct GCS URIs or inline image data", artifactFilename),
		}, nil
	}

	// If it's a regular GCS URI or other string, process normally
	return ProcessMCPImageParameterWithBucketResolution(ctx, p.gcsClient, param, userBucket), nil
}

// ProcessArtifactParameterAsync processes artifact parameters asynchronously
func (p *ADKArtifactProcessor) ProcessArtifactParameterAsync(ctx context.Context, param interface{}, userBucket string) <-chan *ImageProcessingResult {
	resultChan := make(chan *ImageProcessingResult, 1)
	
	go func() {
		defer close(resultChan)
		result, _ := p.ProcessArtifactParameter(ctx, param, userBucket)
		resultChan <- result
	}()
	
	return resultChan
}

// IsArtifactReference checks if a parameter is an ADK artifact reference
func IsArtifactReference(param interface{}) bool {
	if paramStr, ok := param.(string); ok {
		return strings.HasPrefix(paramStr, "artifact:")
	}
	return false
}

// ExtractArtifactFilename extracts the filename from an artifact reference
func ExtractArtifactFilename(param interface{}) (string, error) {
	if paramStr, ok := param.(string); ok {
		if strings.HasPrefix(paramStr, "artifact:") {
			return strings.TrimPrefix(paramStr, "artifact:"), nil
		}
		return "", fmt.Errorf("parameter is not an artifact reference")
	}
	return "", fmt.Errorf("parameter is not a string")
}
