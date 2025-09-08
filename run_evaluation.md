Run Evaluation — GenMedia Creative Studio (experiments/veo-app)

Date: 2025-09-08
Environment: Local dev (uv run main.py), Mesop + FastAPI, Vertex AI (us-central1)
Project: supple-synapse-470916-a2

Summary
- Goal: Diagnose why generated images were not appearing in the Imagen page and verify end-to-end flow.
- Outcome: Fixed browser display by rendering signed HTTPS URLs for GCS objects. Images now display; critique text renders; metadata persists to Firestore.

Observed Logs (key excerpts)
- Duplicate init lines at startup:
  - "[FirebaseClient] - initiating firebase client" (twice)
  - "Initiating Gemini client for project ..." (twice)
  - Cause: Uvicorn reloader spawns a parent and a worker; module-level inits execute in both. Not an error.

- Imagen generation request:
  - Request: imagen-4.0-generate-001, 4 images, output_gcs_uri=gs://.../images/generated_images
  - Response: 200 OK; 4 generated_images
  - Each image has gcs_uri populated; image_bytes absent (expected when output_gcs_uri is provided).

- Critique generation:
  - Model: gemini-2.5-flash; AFC enabled
  - Response: 200 OK; critique text returned
  - Timing reported: ~35.6s for critique

- Metadata persistence:
  - Firestore document created, user_email=anonymous@google.com, gcs_uris=[gs://.../sample_0.png, ...], aspect=1:1, num_images=4, seed=0, critique=<text>

UI Result
- Four images render in the Output grid.
- SynthID watermark notice displays.
- Critique text renders under “Magazine Editor’s Critique”.

Root Cause of Missing Images (prior to fix)
- Browser cannot fetch gs:// URIs. Previous code replaced gs:// with https://storage.mtls.cloud.google.com/, which still fails in browsers (auth/CORS/redirect issues).
- Solution implemented: generate V4 signed URLs server-side and render those in the UI; keep gs:// URIs for metadata and critique inputs.

Current Behavior After Fix
- Image display uses time-limited signed URLs (15 min).
- Firestore continues to receive gs:// URIs and other metadata.
- CSP allows images from storage.googleapis.com (configured in app middleware), so signed URLs load successfully.

Notable Observations (non-blocking)
- Startup log duplication is expected with uvicorn reload.
- Reported generation_time in metadata (e.g., ~150s) includes both image generation and critique time due to measurement window; not incorrect, but label could be interpreted as total pipeline time rather than pure generation.
- Critique takes ~35s; acceptable for Flash with image parts, but noticeable in UX. Currently the critique call is synchronous after images generate.

Errors or Warnings
- None fatal observed. AFC info logs and httpx request logs are informational. No exception traces during the reviewed run.

Operational Checks
- Service account used for signing: genmedia-viewer@… Must have roles/storage.objectViewer and roles/iam.serviceAccountTokenCreator (for the user performing local signing via ADC).
- CSP header includes storage domains; OK.
- FastAPI endpoint /api/get_signed_url present and working; also added internal helper for signing during render.

Optional Considerations (no action taken)
- Make uvicorn reload conditional on DEBUG to reduce duplicate init logs.
- Split timing into generation_time and critique_time to clarify metrics.
- If desired, perform critique asynchronously to avoid blocking UI after images appear.

Conclusion
The run is healthy. Image generation, signed-URL display, critique, and Firestore logging work end-to-end. No blocking issues detected; noted items are informational or optional improvements only.

