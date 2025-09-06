"""
ADK Artifact Handler for Inline Image Processing

This module provides functionality to handle uploaded images using ADK's artifact system,
enabling proper processing of inline image data for video generation and other media operations.
"""

import asyncio
import base64
import io
import logging
from typing import Optional, Dict, Any, Union
import google.genai.types as types
from google.adk.agents.callback_context import CallbackContext
from google.adk.tools.tool_context import ToolContext

logger = logging.getLogger(__name__)

class ADKArtifactHandler:
    """Handles ADK artifacts for media processing workflows."""
    
    def __init__(self):
        self.supported_image_types = {
            'image/jpeg', 'image/jpg', 'image/png', 'image/gif', 
            'image/webp', 'image/bmp', 'image/tiff'
        }
    
    async def save_uploaded_image_as_artifact(
        self, 
        context: Union[CallbackContext, ToolContext], 
        image_part: types.Part, 
        filename: Optional[str] = None
    ) -> str:
        """
        Save an uploaded image as an ADK artifact.
        
        Args:
            context: ADK context (CallbackContext or ToolContext)
            image_part: The image Part from the user's message
            filename: Optional custom filename, defaults to "uploaded_image.{ext}"
            
        Returns:
            str: The artifact filename that can be used to reference the image
            
        Raises:
            ValueError: If the image format is not supported or context lacks artifact service
        """
        if not image_part or not image_part.inline_data:
            raise ValueError("Invalid image part: no inline data found")
        
        mime_type = image_part.inline_data.mime_type
        if mime_type not in self.supported_image_types:
            raise ValueError(f"Unsupported image type: {mime_type}")
        
        # Generate filename if not provided
        if not filename:
            ext = self._get_extension_from_mime_type(mime_type)
            filename = f"uploaded_image{ext}"
        
        try:
            # Save the image as an artifact
            version = await context.save_artifact(filename=filename, artifact=image_part)
            logger.info(f"Saved image artifact '{filename}' as version {version}")
            return filename
            
        except ValueError as e:
            if "ArtifactService" in str(e):
                raise ValueError("ADK ArtifactService not configured. Please configure GcsArtifactService in the Runner.")
            raise
        except Exception as e:
            logger.error(f"Failed to save image artifact: {e}")
            raise
    
    async def load_artifact_as_gcs_uri(
        self, 
        context: Union[CallbackContext, ToolContext], 
        artifact_filename: str
    ) -> Optional[str]:
        """
        Load an artifact and return its GCS URI if using GcsArtifactService.
        
        Args:
            context: ADK context
            artifact_filename: The artifact filename to load
            
        Returns:
            Optional[str]: GCS URI if available, None if artifact not found
        """
        try:
            # Check if artifact service is available
            if not hasattr(context, '_invocation_context') or not context._invocation_context.artifact_service:
                logger.error("No artifact service available in context")
                return None
            
            artifact_service = context._invocation_context.artifact_service
            
            # If using GcsArtifactService, we can construct the GCS URI directly
            if hasattr(artifact_service, 'bucket_name'):
                # For GcsArtifactService, construct the GCS URI based on the service's bucket and naming convention
                bucket_name = artifact_service.bucket_name

                # The GCS path structure used by GcsArtifactService is:
                # {app_name}/{user_id}/{session_id}/{filename}/{version}
                app_name = context._invocation_context.app_name
                user_id = context._invocation_context.user_id
                session_id = context._invocation_context.session.id

                # Determine the latest version available for this artifact
                try:
                    versions = await artifact_service.list_versions(
                        app_name=app_name,
                        user_id=user_id,
                        session_id=session_id,
                        filename=artifact_filename,
                    )
                    version = max(versions) if versions else 0
                except Exception as lv_err:
                    logger.warning(
                        f"Failed to list versions for '{artifact_filename}', defaulting to 0: {lv_err}"
                    )
                    version = 0

                gcs_path = f"{app_name}/{user_id}/{session_id}/{artifact_filename}/{version}"
                gcs_uri = f"gs://{bucket_name}/{gcs_path}"

                logger.info(
                    f"Constructed GCS URI for artifact '{artifact_filename}' (version {version}): {gcs_uri}"
                )
                return gcs_uri
            
            else:
                # For InMemoryArtifactService or other services, load the artifact and convert inline data
                artifact = await context.load_artifact(filename=artifact_filename)
                if not artifact:
                    logger.warning(f"Artifact '{artifact_filename}' not found")
                    return None
                
                logger.info(f"Loaded artifact '{artifact_filename}' from non-GCS service")
                
                # Check if the artifact has inline_data
                if hasattr(artifact, 'inline_data') and artifact.inline_data:
                    # For non-GCS services, we need to upload to a bucket for MCP tools to use
                    import base64
                    import os
                    from google.cloud import storage
                    
                    try:
                        # Get the image data (it's already bytes in inline_data.data)
                        image_data = artifact.inline_data.data
                        
                        # Get bucket name from environment
                        bucket_name = os.getenv('GENMEDIA_ARTIFACT_BUCKET', 'supple-synapse-media')
                        
                        # Create GCS client and upload
                        gcs_client = storage.Client()
                        bucket = gcs_client.bucket(bucket_name)
                        
                        # Use artifact filename as blob name with artifacts prefix
                        blob_name = f"artifacts/{artifact_filename}"
                        blob = bucket.blob(blob_name)
                        blob.upload_from_string(image_data, content_type=artifact.inline_data.mime_type)
                        
                        gcs_uri = f"gs://{bucket_name}/{blob_name}"
                        logger.info(f"Uploaded artifact '{artifact_filename}' to GCS: {gcs_uri}")
                        return gcs_uri
                        
                    except Exception as upload_error:
                        logger.error(f"Failed to upload artifact to GCS: {upload_error}")
                        return None
                
                logger.warning(f"Artifact '{artifact_filename}' has no inline_data")
                return None
            
        except Exception as e:
            logger.error(f"Failed to process artifact '{artifact_filename}': {e}")
            return None
    
    def _get_extension_from_mime_type(self, mime_type: str) -> str:
        """Get file extension from MIME type."""
        mime_to_ext = {
            'image/jpeg': '.jpg',
            'image/jpg': '.jpg', 
            'image/png': '.png',
            'image/gif': '.gif',
            'image/webp': '.webp',
            'image/bmp': '.bmp',
            'image/tiff': '.tiff'
        }
        return mime_to_ext.get(mime_type, '.jpg')
    
    async def save_uploaded_image_as_artifact(
        self, 
        context: Union[CallbackContext, ToolContext], 
        image_part: 'types.Part', 
        filename: str
    ) -> str:
        """
        Save an uploaded image as an ADK artifact.
        
        Args:
            context: ADK context with artifact service
            image_part: The image part containing inline_data
            filename: Desired filename for the artifact
            
        Returns:
            str: The artifact filename that was saved
        """
        try:
            # Save the image part directly as an artifact using ADK's built-in system
            version = await context.save_artifact(filename=filename, artifact=image_part)
            logger.info(f"Saved image as artifact '{filename}' version {version}")
            return filename
            
        except Exception as e:
            logger.error(f"Failed to save image as artifact '{filename}': {e}")
            raise

    async def process_message_images(
        self, 
        context: Union[CallbackContext, ToolContext], 
        message_parts: list
    ) -> Dict[str, str]:
        """
        Process all images in a message and save them as artifacts.
        
        Args:
            context: ADK context
            message_parts: List of message parts that may contain images
            
        Returns:
            Dict[str, str]: Mapping of original part index to artifact filename
        """
        image_artifacts: Dict[str, str] = {}
        
        # Index images starting from 0, ignoring non-image parts
        img_index = 0
        for idx, part in enumerate(message_parts):
            if (
                hasattr(part, 'inline_data')
                and part.inline_data
                and part.inline_data.mime_type in self.supported_image_types
            ):
                try:
                    # Generate a deterministic filename for the Nth image
                    extension = self._get_extension_from_mime_type(part.inline_data.mime_type)[1:]
                    filename = f"user_image_{img_index}.{extension}"
                    
                    # Save the image as an ADK artifact
                    artifact_filename = await self.save_uploaded_image_as_artifact(
                        context, part, filename
                    )
                    # Map logical image index to filename
                    image_artifacts[str(img_index)] = artifact_filename
                    logger.info(
                        f"Processed message image index {img_index} (part {idx}) as artifact '{artifact_filename}'"
                    )
                    img_index += 1
                except Exception as e:
                    logger.error(f"Failed to process message image at part {idx}: {e}")
        
        return image_artifacts

# Global instance for easy access
artifact_handler = ADKArtifactHandler()

async def handle_uploaded_images(context: Union[CallbackContext, ToolContext], message_parts: list) -> Dict[str, str]:
    """
    Convenience function to handle uploaded images in a message.
    
    Args:
        context: ADK context
        message_parts: Message parts from user input
        
    Returns:
        Dict[str, str]: Mapping of image indices to artifact filenames
    """
    return await artifact_handler.process_message_images(context, message_parts)
