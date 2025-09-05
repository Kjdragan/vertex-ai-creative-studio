
import streamlit as st
from io import StringIO, BytesIO
from dotenv import load_dotenv
import os
from PIL import Image
from google import genai
from google.generai import types

# Load environment variables (for GEMINI_API_KEY)
load_dotenv()

# Set Streamlit page configuration
st.set_page_config(page_title='Gemini Nano Banana Chatbot',
                    initial_sidebar_state='auto')

# Avatars for chat messages
avatars = {
    "assistant": "🤖",
    "user": "👤"
}

# Main page title and subtitle
st.markdown("<h2 style='text-align: center; color: #3184a0;'>Gemini Nano Banana</h2>", unsafe_allow_html=True)
st.markdown("<h3 style='text-align: center; color: #3184a0;'>Image generator chatbot</h3>", unsafe_allow_html=True)

# Sidebar for controls
with st.sidebar:
    st.markdown("### 🍌 Gemini Nano Banana")

    # Clear chat history button
    def clear_chat_history():
        st.session_state.messages = [
            {"role": "assistant", "content": "How may I assist you today?", "image": None}
        ]
        if "image" in st.session_state:
            del st.session_state["image"] # Also clear uploaded image

    st.button("Clear Chat History", on_click=clear_chat_history)

    # Image uploader
    uploaded_file = st.file_uploader("Upload an image", type=["jpg", "jpeg", "png"])
    if uploaded_file:
        image_bytes = Image.open(uploaded_file)
        st.session_state.image = image_bytes
        st.image(image_bytes, caption="Uploaded Image", use_container_width=True)

# Initialize session state for messages if not already present
if "messages" not in st.session_state.keys():
    st.session_state.messages = [
        {"role": "assistant", "content": "How may I assist you today?", "image": None}
    ]

# Display existing chat messages
for message in st.session_state.messages:
    with st.chat_message(message["role"],
                         avatar=avatars[message["role"]]):
        st.write(message["content"])
        if message["role"] == "assistant" and message["image"]:
            st.image(message["image"])

# Function to run the query against Gemini API
def run_query(input_text):
    try:
        GEMINI_API_KEY = os.getenv("GEMINI_API_KEY")
        if not GEMINI_API_KEY:
            st.error("GEMINI_API_KEY not found. Please set it in your .env file or as an environment variable.")
            return "Error: API Key Missing"

        client = genai.Client(api_key=GEMINI_API_KEY)

        system_prompt = """
        #INSTRUCTIONS
        Generate an image according to the instructions.
        Specify in the output text the changes made to the image.
        #OUTPUT
        A generated image and a short text.
        """

        contents = [system_prompt, input_text]
        if "image" in st.session_state and st.session_state.image:
            contents.append(st.session_state.image)

        response = client.models.generate_content(
            model="gemini-2.5-flash-image-preview",
            contents=contents,
            config=types.GenerateContentConfig(
                response_modalities=['Text', 'Image']
            )
        )

        if response:
            return response
        else:
            return "Error: No response from model"
    except Exception as e:
        return f"Error: {str(e)}"

# Function to unpack the response from Gemini
def unpack_response(prompt):
    response = run_query(prompt)

    full_response = ""
    generated_image = None

    # Handle error responses from run_query
    if isinstance(response, str) and "Error" in response:
        return response, None, None # Return error message, no placeholder, no image

    try:
        # Assuming the response structure based on the provided text
        # The text implies `response.candidates[0].content.parts` but `google.genai`
        # usually has content directly accessible via `text` and `images` attributes
        # or parts of `response.parts`. Adjusting based on common `genai` usage.

        # Check for text in the response
        if hasattr(response, 'text') and response.text is not None:
            full_response = response.text
        elif hasattr(response, 'parts') and response.parts is not None:
             for part in response.parts:
                if hasattr(part, 'text') and part.text is not None:
                    full_response += part.text
                if hasattr(part, 'inline_data') and part.inline_data is not None:
                    generated_image = Image.open(BytesIO(part.inline_data.data))

        # Check for images in the response (if available directly or via parts)
        if hasattr(response, 'images') and response.images:
            # Assuming the first image is the one to display
            # The actual API might return image data directly, not a list of PIL Images
            if isinstance(response.images[0], Image.Image):
                generated_image = response.images[0]
            elif hasattr(response.images[0], 'data') and response.images[0].data:
                generated_image = Image.open(BytesIO(response.images[0].data))

    except Exception as ex:
        full_response = f"ERROR in unpack response: {str(ex)}"
        # If an error occurs during unpacking, try to retain the uploaded image if any
        generated_image = st.session_state.image if "image" in st.session_state else None

    return full_response, None, generated_image # No placeholder used directly in this function's return

# Chat input at the bottom
if prompt := st.chat_input("Enter your prompt..."):
    # Add user message to chat history
    st.session_state.messages.append({"role": "user", "content": prompt, "image": None})
    with st.chat_message("user", avatar=avatars["user"]):
        st.write(prompt)

    # Generate assistant response
    with st.chat_message("assistant", avatar=avatars["assistant"]):
        with st.spinner("Thinking..."):
            full_response, _, generated_image = unpack_response(prompt) # Ignoring placeholder from unpack_response

            if full_response:
                st.write(full_response)
            if generated_image:
                st.image(generated_image)

    # Add assistant message to chat history
    message = {"role": "assistant",
               "content": full_response,
               "image": generated_image}
    st.session_state.messages.append(message)

```

**To use this script:**

1.  **Save it:** Save the code above as a Python file (e.g., `app.py`).
2.  **Create a `.env` file:** In the same directory as `app.py`, create a file named `.env` and add your Google Gemini API key:
    ```
    GEMINI_API_KEY="YOUR_GEMINI_API_KEY"
    ```
    (Replace `"YOUR_GEMINI_API_KEY"` with your actual key obtained from Google AI Studio).
3.  **Install dependencies:** Open your terminal or command prompt, navigate to the directory where you saved `app.py` and `.env`, and run:
    ```bash
    pip install streamlit google-generativeai python-dotenv Pillow
    ```
4.  **Run the app:**
    ```bash
    streamlit run app.py
    ```
