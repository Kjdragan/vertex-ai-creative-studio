#!/usr/bin/env python3
"""
Test script for ADK artifact workflow with uploaded images.
This script simulates the complete workflow of uploading an image,
saving it as an artifact, and converting it to a GCS URI for MCP tools.
"""

import asyncio
import base64
import os
import sys
import logging
from pathlib import Path

# Add the agent directory to path
sys.path.insert(0, str(Path(__file__).parent / "sample-agents/adk/genmedia_agent"))

import google.genai.types as types
from google.adk.agents import LlmAgent
from google.adk.artifacts import GcsArtifactService, InMemoryArtifactService
from google.adk.sessions import InMemorySessionService
from google.adk.runners import Runner

from artifact_handler import ADKArtifactHandler

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Sample base64 encoded PNG image (1x1 pixel red dot)
SAMPLE_IMAGE_BASE64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="
SAMPLE_IMAGE_MIME_TYPE = "image/png"

async def test_artifact_workflow():
    """Test the complete artifact workflow."""
    
    print("🧪 Testing ADK Artifact Workflow")
    print("=" * 50)
    
    # Step 1: Create artifact service
    print("1. Setting up artifact service...")
    try:
        # Try to use GCS artifact service
        bucket_name = os.getenv('GENMEDIA_ARTIFACT_BUCKET', 'supple-synapse-media')
        artifact_service = GcsArtifactService(bucket_name=bucket_name)
        print(f"✅ Using GcsArtifactService with bucket: {bucket_name}")
    except Exception as e:
        print(f"⚠️  GCS service failed ({e}), falling back to InMemoryArtifactService")
        artifact_service = InMemoryArtifactService()
    
    # Step 2: Create agent and runner
    print("2. Creating agent and runner...")
    session_service = InMemorySessionService()
    
    # Create a simple test agent
    test_agent = LlmAgent(
        name="test_artifact_agent",
        model="gemini-2.0-flash",
        instruction="Test agent for artifact workflow"
    )
    
    # Create runner with artifact service
    runner = Runner(
        agent=test_agent,
        app_name="test_artifact_app",
        session_service=session_service,
        artifact_service=artifact_service
    )
    print("✅ Runner created successfully")
    
    # Step 3: Create sample image part
    print("3. Creating sample image part...")
    image_bytes = base64.b64decode(SAMPLE_IMAGE_BASE64)
    image_part = types.Part(
        inline_data=types.Blob(
            data=image_bytes,
            mime_type=SAMPLE_IMAGE_MIME_TYPE
        )
    )
    print(f"✅ Created image part: {len(image_bytes)} bytes, MIME: {SAMPLE_IMAGE_MIME_TYPE}")
    
    # Step 4: Test artifact service directly
    print("4. Testing artifact service directly...")
    try:
        # Test saving artifact directly to service
        version = await artifact_service.save_artifact(
            app_name="test_artifact_app",
            user_id="test_user_456",
            session_id="test_session_123",
            filename="test_image.png",
            artifact=image_part
        )
        print(f"✅ Saved artifact directly: test_image.png version {version}")
        
        # Test loading artifact
        loaded_artifact = await artifact_service.load_artifact(
            app_name="test_artifact_app",
            user_id="test_user_456", 
            session_id="test_session_123",
            filename="test_image.png"
        )
        
        if loaded_artifact:
            print(f"✅ Loaded artifact: {type(loaded_artifact)}")
            if hasattr(loaded_artifact, 'inline_data') and loaded_artifact.inline_data:
                print(f"   - MIME type: {loaded_artifact.inline_data.mime_type}")
                print(f"   - Data size: {len(loaded_artifact.inline_data.data)} bytes")
            
            # If using GCS service, construct the expected GCS URI
            if hasattr(artifact_service, 'bucket_name'):
                bucket_name = artifact_service.bucket_name
                gcs_path = f"test_artifact_app/test_user_456/test_session_123/test_image.png/{version}"
                expected_gcs_uri = f"gs://{bucket_name}/{gcs_path}"
                print(f"✅ Expected GCS URI: {expected_gcs_uri}")
            else:
                print("✅ Using InMemoryArtifactService (no GCS URI)")
        else:
            print("❌ Failed to load artifact")
            return False
            
    except Exception as e:
        print(f"❌ Failed to test artifact service: {e}")
        return False
    
    print("\n🎉 Artifact workflow test completed successfully!")
    return True

if __name__ == "__main__":
    success = asyncio.run(test_artifact_workflow())
    sys.exit(0 if success else 1)
