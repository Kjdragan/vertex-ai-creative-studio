"""
ADK Runner Configuration with Artifact Support

This module configures the ADK Runner with proper artifact service support
for handling uploaded images and other media files.
"""

import os
import logging
from google.adk.runners import Runner
from google.adk.artifacts import GcsArtifactService, InMemoryArtifactService
from google.adk.sessions import InMemorySessionService
from .agent import root_agent

logger = logging.getLogger(__name__)

def create_runner_with_artifacts(app_name: str = "genmedia_agent") -> Runner:
    """
    Create an ADK Runner with proper artifact service configuration.
    
    Args:
        app_name: Name of the application
        
    Returns:
        Runner: Configured ADK Runner with artifact support
    """
    # Configure session service
    session_service = InMemorySessionService()
    
    # Configure artifact service based on environment
    artifact_bucket = os.getenv("GENMEDIA_ARTIFACT_BUCKET", "supple-synapse-media")
    use_gcs_artifacts = os.getenv("USE_GCS_ARTIFACTS", "true").lower() == "true"
    
    if use_gcs_artifacts and artifact_bucket:
        try:
            logger.info(f"Configuring GCS artifact service with bucket: {artifact_bucket}")
            artifact_service = GcsArtifactService(bucket_name=artifact_bucket)
        except Exception as e:
            logger.warning(f"Failed to initialize GCS artifact service: {e}")
            logger.info("Falling back to in-memory artifact service")
            artifact_service = InMemoryArtifactService()
    else:
        logger.info("Using in-memory artifact service")
        artifact_service = InMemoryArtifactService()
    
    # Create and configure the runner
    runner = Runner(
        agent=root_agent,
        app_name=app_name,
        session_service=session_service,
        artifact_service=artifact_service
    )
    
    logger.info(f"ADK Runner configured with artifact service: {type(artifact_service).__name__}")
    return runner

def get_artifact_service_info() -> dict:
    """
    Get information about the configured artifact service.
    
    Returns:
        dict: Information about artifact service configuration
    """
    artifact_bucket = os.getenv("GENMEDIA_ARTIFACT_BUCKET", "supple-synapse-media")
    use_gcs_artifacts = os.getenv("USE_GCS_ARTIFACTS", "true").lower() == "true"
    
    return {
        "use_gcs_artifacts": use_gcs_artifacts,
        "artifact_bucket": artifact_bucket,
        "service_type": "GcsArtifactService" if use_gcs_artifacts else "InMemoryArtifactService"
    }

# Create the configured runner instance
configured_runner = create_runner_with_artifacts()
