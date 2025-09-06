"""
Simple ADK callback for initializing session state and processing artifacts.

Based on the ADK callback documentation, this implements a simple function-based callback.
"""

import logging
import re
from typing import Optional, Any
import google.genai.types as types
from google.adk.agents.callback_context import CallbackContext
from google.adk.tools.tool_context import ToolContext
from google.adk.tools.base_tool import BaseTool
from .artifact_handler import ADKArtifactHandler

logger = logging.getLogger(__name__)

async def before_agent_callback(*, callback_context) -> Optional[types.Content]:
    """
    ADK before_agent_callback to initialize session state and process artifacts.
    
    This callback is called before the agent processes each request.
    It initializes the session state and processes any uploaded images as artifacts.
    """
    try:
        # Access session state using the state property
        session_state = callback_context.state
        
        # Initialize image artifacts list if not present
        if 'image_artifacts' not in session_state:
            session_state['image_artifacts'] = []
            logger.info("Initialized image_artifacts in session state")
        
        # Process uploaded images in the current user message
        artifact_handler = ADKArtifactHandler()
        
        # Use the user_content provided by ADK
        content = callback_context.user_content
        if content and hasattr(content, 'parts') and content.parts:
            # Process images and save as artifacts
            artifacts = await artifact_handler.process_message_images(
                callback_context, content.parts
            )
            
            # Add to session state
            for filename, artifact_ref in artifacts.items():
                if artifact_ref not in session_state['image_artifacts']:
                    session_state['image_artifacts'].append(artifact_ref)
                    logger.info(f"Added artifact to session: {artifact_ref}")
        
        # Log current state for debugging (State has no keys(); use to_dict())
        try:
            state_keys = list(session_state.to_dict().keys())
        except Exception:
            state_keys = []
        logger.info(
            f"Session state keys: {state_keys}, artifacts: {len(session_state.get('image_artifacts', []))}"
        )
        
        # Return None to continue with normal agent processing
        return None
        
    except Exception as e:
        logger.error(f"Error in before_agent_callback: {e}")
        # Return None to continue processing even if callback fails
        return None

async def before_tool_callback(tool: BaseTool, args: dict[str, Any], tool_context: ToolContext) -> Optional[dict]:
    """
    Convert artifact: URIs in tool arguments to GCS URIs before the tool executes.

    If conversion succeeds, mutate args in-place and return None to allow the
    tool to run normally with updated parameters.
    """
    try:
        # Only handle known tools that accept image_uri
        tool_name = getattr(tool, 'name', '') or ''
        if 'image_uri' in args and isinstance(args['image_uri'], str) and args['image_uri'].startswith('artifact:'):
            artifact_ref = args['image_uri']
            artifact_filename = artifact_ref.split(':', 1)[1]
            logger.info(f"Converting artifact reference for tool '{tool_name}': {artifact_filename}")

            handler = ADKArtifactHandler()
            gcs_uri = await handler.load_artifact_as_gcs_uri(tool_context, artifact_filename)
            if gcs_uri and gcs_uri.startswith('gs://'):
                args['image_uri'] = gcs_uri
                logger.info(f"Converted artifact '{artifact_ref}' -> '{gcs_uri}' for tool '{tool_name}'")
            else:
                logger.warning(
                    f"Failed to resolve artifact '{artifact_ref}' to GCS URI; attempting fallback"
                )
                # Fallback 1: if session has any image_artifacts, try the latest
                try:
                    image_artifacts = tool_context.state.get('image_artifacts', [])
                    if isinstance(image_artifacts, list) and image_artifacts:
                        fallback_entry = image_artifacts[-1]
                        # Entries in state may be stored as 'artifact:filename.ext'
                        fallback_filename = (
                            fallback_entry.split(':', 1)[1]
                            if isinstance(fallback_entry, str) and ':' in fallback_entry
                            else fallback_entry
                        )
                        logger.info(
                            f"Trying fallback artifact from session state: {fallback_filename}"
                        )
                        gcs_uri_fallback = await handler.load_artifact_as_gcs_uri(
                            tool_context, fallback_filename
                        )
                        if gcs_uri_fallback and gcs_uri_fallback.startswith('gs://'):
                            args['image_uri'] = gcs_uri_fallback
                            logger.info(
                                f"Fallback succeeded: '{artifact_ref}' -> '{gcs_uri_fallback}'"
                            )
                            return None
                except Exception as fe:
                    logger.warning(f"Fallback by session state failed: {fe}")

                # Fallback 2: fuzzy match any 'user_image_*.jpg/png' saved in state
                try:
                    if isinstance(image_artifacts, list):
                        # Normalize entries possibly stored as 'artifact:filename'
                        normalized_names = [
                            (x.split(':', 1)[1] if isinstance(x, str) and ':' in x else x)
                            for x in image_artifacts
                        ]
                        for fname in reversed(normalized_names):
                            if isinstance(fname, str) and fname.startswith('user_image_'):
                                gcs_uri_guess = await handler.load_artifact_as_gcs_uri(
                                    tool_context, fname
                                )
                                if gcs_uri_guess and gcs_uri_guess.startswith('gs://'):
                                    args['image_uri'] = gcs_uri_guess
                                    logger.info(
                                        f"Fuzzy fallback succeeded: '{artifact_ref}' -> '{gcs_uri_guess}'"
                                    )
                                    return None
                except Exception as fe2:
                    logger.warning(f"Fuzzy fallback failed: {fe2}")

        # Returning None lets the tool proceed
        return None
    except Exception as e:
        logger.error(f"Error in before_tool_callback: {e}")
        return None

