"""
ADK Image Artifact Callback

Function-based callback for ADK that processes uploaded images and saves them as artifacts.
"""

import logging
from typing import Optional
import google.genai.types as types
from google.adk.agents.callback_context import CallbackContext
from .artifact_handler import artifact_handler

logger = logging.getLogger(__name__)

def image_artifact_before_agent_callback(context: CallbackContext) -> Optional[types.Content]:
    """
    ADK before_agent_callback function that processes uploaded images and saves them as artifacts.
    
    This callback inspects the user's message for uploaded images, saves them as artifacts,
    and updates the session state with artifact references for MCP tools to use.
    
    Args:
        context: The callback context containing session state and other information
        
    Returns:
        Optional[types.Content]: None (allows normal agent processing to continue)
    """
    try:
        # Initialize session state for image artifacts if not present
        session_state = context.get_session_state()
        if 'image_artifacts' not in session_state:
            session_state['image_artifacts'] = {}
            context.set_session_state(session_state)
            logger.info("Initialized image artifacts in session state")
        
        # Note: In ADK, we can't directly access the user message from before_agent_callback
        # The image processing will need to be handled differently, possibly through
        # the agent's instruction or through a different callback type
        
        logger.info("Image artifact callback executed - session state initialized")
        return None  # Allow normal agent processing to continue
        
    except Exception as e:
        logger.error(f"Error in image artifact callback: {e}")
        return None
            # Modify the last text part or add a new one
            modified_parts = list(response.parts)
            if modified_parts and hasattr(modified_parts[-1], 'text'):
                modified_parts[-1] = types.Part(text=modified_parts[-1].text + artifact_info)
            else:
                modified_parts.append(types.Part(text=artifact_info))
            
            return types.Content(parts=modified_parts)
        
        return None
    
    def get_available_artifacts(self) -> Dict[str, str]:
        """
        Get currently available image artifacts.
        
        Returns:
            Dict[str, str]: Mapping of artifact filename to MIME type
        """
        return self.processed_images.copy()
    
    def clear_artifacts(self):
        """Clear the processed artifacts cache."""
        self.processed_images.clear()

# Global instance for the agent
image_artifact_callback = ImageArtifactCallback()
