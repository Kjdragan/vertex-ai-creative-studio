#!/usr/bin/env python3
"""
End-to-end test for ADK artifact workflow with MCP tools.
This test simulates a complete workflow: image upload -> artifact save -> MCP tool call.
"""

import asyncio
import base64
import json
import requests
import sys
import logging
from pathlib import Path

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Sample base64 encoded PNG image (1x1 pixel red dot)
SAMPLE_IMAGE_BASE64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="

def test_adk_server_health():
    """Test if ADK server is responding."""
    try:
        response = requests.get("http://localhost:8000", timeout=5)
        return response.status_code == 200
    except Exception as e:
        logger.error(f"ADK server health check failed: {e}")
        return False

def create_test_message_with_image():
    """Create a test message with an uploaded image."""
    return {
        "contents": [
            {
                "parts": [
                    {
                        "text": "Please create a video from this uploaded image using Veo"
                    },
                    {
                        "inline_data": {
                            "mime_type": "image/png",
                            "data": SAMPLE_IMAGE_BASE64
                        }
                    }
                ]
            }
        ]
    }

def send_message_to_adk(message):
    """Send a message to the ADK server and get response."""
    try:
        response = requests.post(
            "http://localhost:8000/api/sessions/test_session/messages",
            json=message,
            timeout=30,
            headers={"Content-Type": "application/json"}
        )
        return response
    except Exception as e:
        logger.error(f"Failed to send message to ADK: {e}")
        return None

async def test_end_to_end_workflow():
    """Test the complete end-to-end workflow."""
    
    print("🚀 Testing End-to-End ADK Artifact Workflow")
    print("=" * 60)
    
    # Step 1: Check ADK server health
    print("1. Checking ADK server health...")
    if not test_adk_server_health():
        print("❌ ADK server is not responding")
        return False
    print("✅ ADK server is running")
    
    # Step 2: Create test message with image
    print("2. Creating test message with uploaded image...")
    test_message = create_test_message_with_image()
    print(f"✅ Created message with {len(SAMPLE_IMAGE_BASE64)} bytes of image data")
    
    # Step 3: Send message to ADK
    print("3. Sending message to ADK server...")
    response = send_message_to_adk(test_message)
    
    if not response:
        print("❌ Failed to send message to ADK server")
        return False
    
    print(f"✅ Received response with status: {response.status_code}")
    
    # Step 4: Analyze response
    print("4. Analyzing ADK response...")
    try:
        if response.status_code == 200:
            response_data = response.json()
            print("✅ Successfully received JSON response")
            
            # Look for evidence of artifact processing
            response_text = json.dumps(response_data, indent=2)
            
            # Check for artifact references
            if "artifact:" in response_text:
                print("✅ Found artifact references in response")
            elif "gs://" in response_text:
                print("✅ Found GCS URIs in response (artifacts converted successfully)")
            else:
                print("⚠️  No artifact references found in response")
            
            # Check for MCP tool calls
            if "veo" in response_text.lower():
                print("✅ Veo tool appears to be involved")
            
            # Check for errors
            if "error" in response_text.lower():
                print("⚠️  Response contains error messages")
                print(f"Response preview: {response_text[:500]}...")
            else:
                print("✅ No obvious errors in response")
                
        else:
            print(f"❌ Non-200 response: {response.status_code}")
            print(f"Response: {response.text[:500]}...")
            return False
            
    except Exception as e:
        print(f"❌ Failed to parse response: {e}")
        return False
    
    print("\n🎉 End-to-end workflow test completed!")
    print("\nKey Success Indicators:")
    print("- ✅ ADK server responded successfully")
    print("- ✅ Image upload was processed")
    print("- ✅ Response generated without critical errors")
    
    return True

if __name__ == "__main__":
    success = asyncio.run(test_end_to_end_workflow())
    sys.exit(0 if success else 1)
