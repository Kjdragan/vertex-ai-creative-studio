# GenMedia Agent Tools Documentation

This document provides comprehensive documentation for all the tools available in the `genmedia_agent` system. The agent integrates multiple MCP (Model Context Protocol) toolsets to provide generative media capabilities including image generation, text-to-speech, video generation, and audio/video processing.

## Overview

The `genmedia_agent` is built using Google's ADK (Agent Development Kit) and integrates four main MCP toolsets:
- **Imagen**: Image generation and editing
- **Chirp3**: Text-to-speech synthesis
- **Veo**: Video generation
- **AVTool**: Audio/video processing with FFmpeg

## Image Generation Tools (Imagen)

### imagen_t2i
**Description**: Generates images from text prompts using Google's Imagen models.

**Parameters**:
- `prompt` (required, string): Text prompt for image generation
- `model` (optional, string, default: "imagen-3.0-generate-002"): Imagen model to use
- `num_images` (optional, number, default: 1, range: 1-4): Number of images to generate
- `aspect_ratio` (optional, string, default: "1:1"): Aspect ratio (e.g., "1:1", "16:9", "9:16")
- `gcs_bucket_uri` (optional, string): GCS URI prefix to store generated images
- `output_directory` (optional, string): Local directory to save images

**Returns**: Generated images as base64 data or saved to specified locations with metadata.

### imagen_edit_inpainting_insert
**Description**: Adds content to a masked area of an image using inpainting.

**Parameters**:
- `prompt` (required, string): Description of content to add
- `image_uri` (required, string): GCS URI of the image to edit
- `mask_mode` (required, string): Masking mode (e.g., "MASK_MODE_FOREGROUND", "MASK_MODE_SEMANTIC")
- `mask_dilation` (optional, number): Dilation to apply to the mask
- `segmentation_classes` (optional, array): Segmentation classes for semantic masking

**Returns**: Edited image with new content inserted in the masked area.

### imagen_edit_inpainting_remove
**Description**: Removes content from a masked area of an image using inpainting.

**Parameters**:
- `image_uri` (required, string): GCS URI of the image to edit
- `mask_mode` (required, string): Masking mode (e.g., "MASK_MODE_FOREGROUND", "MASK_MODE_SEMANTIC")
- `mask_dilation` (optional, number): Dilation to apply to the mask
- `segmentation_classes` (optional, array): Segmentation classes for semantic masking

**Returns**: Edited image with content removed from the masked area.

## Text-to-Speech Tools (Chirp3)

### chirp_tts
**Description**: Synthesizes speech from text using Google Cloud TTS with Chirp3-HD voices.

**Parameters**:
- `text` (required, string): Text to synthesize into speech
- `voice_name` (optional, string): Specific Chirp3-HD voice name (default: "en-US-Chirp3-HD-Zephyr")
- `output_filename_prefix` (optional, string, default: "chirp_audio"): Prefix for output WAV filename
- `output_directory` (optional, string): Local directory to save audio file
- `pronunciations` (optional, array): Custom pronunciations in format "phrase:phonetic_representation"
- `pronunciation_encoding` (optional, string, default: "ipa"): Phonetic encoding ("ipa" or "xsampa")

**Returns**: Synthesized audio as WAV data, optionally saved locally.

### list_chirp_voices
**Description**: Lists available Chirp3-HD voices filtered by language.

**Parameters**:
- `language` (required, string): Language filter (descriptive name or BCP-47 code)

**Returns**: JSON list of available voices with metadata (name, language code, gender).

## Video Generation Tools (Veo)

### veo_t2v
**Description**: Generates videos from text prompts using Veo models.

**Parameters**:
- `prompt` (required, string): Text prompt for video generation
- `bucket` (optional, string): GCS bucket for saving videos (uses GENMEDIA_BUCKET if not provided)
- `output_directory` (optional, string): Local directory to download videos
- `model` (optional, string, default: "veo-2.0-generate-001"): Veo model to use
- `num_videos` (optional, number, default: 1): Number of videos to generate
- `aspect_ratio` (optional, string, default: "16:9"): Video aspect ratio
- `duration` (optional, number, default: 5): Video duration in seconds

**Returns**: Generated video saved to GCS and optionally downloaded locally.

### veo_i2v
**Description**: Generates videos from input images using Veo models.

**Parameters**:
- `image_uri` (required, string): GCS URI of input image
- `mime_type` (optional, string): Image MIME type ("image/jpeg" or "image/png")
- `prompt` (optional, string): Text prompt to guide video generation
- `bucket` (optional, string): GCS bucket for saving videos
- `output_directory` (optional, string): Local directory to download videos
- `model` (optional, string, default: "veo-2.0-generate-001"): Veo model to use
- `num_videos` (optional, number, default: 1): Number of videos to generate
- `aspect_ratio` (optional, string, default: "16:9"): Video aspect ratio
- `duration` (optional, number, default: 5): Video duration in seconds

**Returns**: Generated video from image input, saved to GCS and optionally downloaded locally.

## Audio/Video Processing Tools (AVTool/FFmpeg)

### ffmpeg_get_media_info
**Description**: Extracts media information from files using ffprobe.

**Parameters**:
- `input_media_uri` (required, string): URI of input media file (local path or gs://)

**Returns**: JSON output containing streams, format, and metadata information.

### ffmpeg_convert_audio_wav_to_mp3
**Description**: Converts WAV audio files to MP3 format.

**Parameters**:
- `input_audio_uri` (required, string): URI of input WAV audio file
- `output_file_name` (optional, string): Desired output MP3 filename
- `output_local_dir` (optional, string): Local directory to save output
- `output_gcs_bucket` (optional, string): GCS bucket to upload output

**Returns**: Converted MP3 file with processing status and location information.

### ffmpeg_video_to_gif
**Description**: Creates GIF animations from video files using two-pass palette generation.

**Parameters**:
- `input_video_uri` (required, string): URI of input video file
- `scale_width_factor` (optional, number, default: 0.33): Width scaling factor (0.33 = 33%)
- `fps` (optional, number, default: 15, range: 1-50): Frames per second for output GIF
- `output_file_name` (optional, string): Desired output GIF filename
- `output_local_dir` (optional, string): Local directory to save output
- `output_gcs_bucket` (optional, string): GCS bucket to upload output

**Returns**: High-quality GIF created using optimized palette generation.

### ffmpeg_combine_audio_and_video
**Description**: Merges separate audio and video streams into a single video file.

**Parameters**:
- `input_video_uri` (required, string): URI of input video file
- `input_audio_uri` (required, string): URI of input audio file
- `output_file_name` (optional, string): Desired output video filename
- `output_local_dir` (optional, string): Local directory to save output
- `output_gcs_bucket` (optional, string): GCS bucket to upload output

**Returns**: Combined video file with synchronized audio and video streams.

### ffmpeg_overlay_image_on_video
**Description**: Overlays an image onto a video at specified coordinates.

**Parameters**:
- `input_video_uri` (required, string): URI of input video file
- `input_image_uri` (required, string): URI of input image file
- `x_coordinate` (optional, number, default: 0): X coordinate for overlay (top-left)
- `y_coordinate` (optional, number, default: 0): Y coordinate for overlay (top-left)
- `output_file_name` (optional, string): Desired output video filename
- `output_local_dir` (optional, string): Local directory to save output
- `output_gcs_bucket` (optional, string): GCS bucket to upload output

**Returns**: Video with image overlay applied at specified position.

### ffmpeg_concatenate_media_files
**Description**: Joins multiple media files into a single file with format standardization.

**Parameters**:
- `input_media_uris` (required, array): Array of URIs for input media files
- `output_file_name` (optional, string): Desired output filename
- `output_local_dir` (optional, string): Local directory to save output
- `output_gcs_bucket` (optional, string): GCS bucket to upload output

**Returns**: Concatenated media file. For WAV output, requires compatible PCM inputs. Other formats are standardized to MP4/AAC before concatenation.

### ffmpeg_adjust_volume
**Description**: Adjusts audio volume by specified decibel amount.

**Parameters**:
- `input_audio_uri` (required, string): URI of input audio file
- `volume_db_change` (required, number): Volume change in dB (e.g., -10 for -10dB, +5 for +5dB)
- `output_file_name` (optional, string): Desired output audio filename
- `output_local_dir` (optional, string): Local directory to save output
- `output_gcs_bucket` (optional, string): GCS bucket to upload output

**Returns**: Audio file with adjusted volume level.

### ffmpeg_layer_audio_files
**Description**: Mixes multiple audio files together into a single audio stream.

**Parameters**:
- `input_audio_uris` (required, array): Array of URIs for input audio files to layer
- `output_file_name` (optional, string): Desired output mixed audio filename
- `output_local_dir` (optional, string): Local directory to save output
- `output_gcs_bucket` (optional, string): GCS bucket to upload output

**Returns**: Mixed audio file combining all input audio streams using FFmpeg's amix filter.

## Common Parameters and Behaviors

### File Handling
- **Local Paths**: Absolute paths to local files
- **GCS URIs**: Google Cloud Storage URIs in format `gs://bucket/path`
- **Output Handling**: Tools support both local saving and GCS upload
- **Automatic Cleanup**: Temporary files are automatically cleaned up after processing

### Default Behaviors
- If `output_gcs_bucket` is not provided, tools use the `GENMEDIA_BUCKET` environment variable
- Output filenames are auto-generated with timestamps if not specified
- Tools preserve input file formats when possible or convert to appropriate defaults

### Error Handling
- All tools provide detailed error messages for invalid parameters
- Timeout handling for long-running operations (e.g., 3 minutes for image generation)
- Graceful fallbacks when preferred options are unavailable

### Performance Considerations
- Image generation: 3-minute timeout per request
- Audio synthesis: 30-second timeout per request
- Video processing: Varies by operation complexity and duration
- All operations support cancellation via context

## Usage Examples

### Generate an image
```json
{
  "tool": "imagen_t2i",
  "parameters": {
    "prompt": "A serene mountain landscape at sunset",
    "num_images": 2,
    "aspect_ratio": "16:9"
  }
}
```

### Create speech from text
```json
{
  "tool": "chirp_tts",
  "parameters": {
    "text": "Hello, this is a test of the text-to-speech system.",
    "voice_name": "en-US-Chirp3-HD-Zephyr",
    "output_directory": "/tmp/audio"
  }
}
```

### Generate video from text
```json
{
  "tool": "veo_t2v",
  "parameters": {
    "prompt": "A cat playing with a ball of yarn",
    "duration": 10,
    "aspect_ratio": "1:1"
  }
}
```

### Process media files
```json
{
  "tool": "ffmpeg_combine_audio_and_video",
  "parameters": {
    "input_video_uri": "gs://my-bucket/video.mp4",
    "input_audio_uri": "gs://my-bucket/audio.wav",
    "output_file_name": "combined_media.mp4"
  }
}
```

This comprehensive toolset enables the `genmedia_agent` to handle complex multimedia workflows, from content generation to post-processing, all integrated through a unified MCP interface.
