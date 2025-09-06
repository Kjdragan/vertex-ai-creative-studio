# Run Evaluation — ADK GenMedia i2v Flow

Date/time: 2025-09-05 22:04 (local)

## Summary
- The end-to-end image-to-video (i2v) flow succeeded using the uploaded image artifact.
- Artifact URIs were correctly converted from `artifact:user_image_0.jpg` to a valid `gs://...` GCS URI by the `before_tool_callback`.
- Veo 3 generated the video successfully and saved it to: `gs://supple-synapse-media/veo_outputs/17050361871438496886/sample_0.mp4`.
- Total generation time reported by the tool: ~1m1s.

## What Was Exercised
- ADK web started with GCS ArtifactService (`--artifact_service_uri`), ensuring deterministic GCS URIs.
- Uploaded image was saved as artifact `user_image_0.jpg` (indexing now starts at 0 and counts only image parts).
- `before_tool_callback` resolved `artifact:` to `gs://.../user_image_0.jpg/0` and Veo i2v consumed it.
- When the first attempt omitted `mime_type`, the tool returned a guidance error; a retry including `mime_type: image/jpeg` succeeded.

## Terminal Log Highlights (abridged)
```
Using provided and validated MIME type: image/jpeg
Handler: 'bucket' parameter not provided, using default constructed from GENMEDIA_BUCKET: gs://supple-synapse-media/veo_outputs/
Initiating GenerateVideos (i2v) ... ImageGCSURI: gs://supple-synapse-media/.../user_image_0.jpg/0, ... Duration: 8s, OutputGCS: gs://supple-synapse-media/veo_outputs/
GenerateVideos operation (i2v) ... completed. Total duration: 1m1s
Successfully generated 1 videos (i2v) ... available at GCS URI: gs://supple-synapse-media/veo_outputs/17050361871438496886/sample_0.mp4
```

## Observed Issues and Warnings
- Missing MIME type on first attempt (then auto-retry succeeded)
  - Tool initially returned: "MIME type ... could not be inferred ... Please specify 'image/jpeg' or 'image/png'."
  - Impact: extra round-trip; user still got video.
  - Recommendation: Auto-supply `mime_type` if missing in `before_tool_callback` by inferring from the artifact filename (e.g., `.jpg` → `image/jpeg`, `.png` → `image/png`).

- Logging format mismatch in Veo handler
  - Example line shows arguments printed in the wrong placeholders (e.g., `OutputDir='image/jpeg'`, `Model=...prompt...`, `%!d(string=...)`).
  - Impact: confusing logs; functionality unaffected.
  - Recommendation: Fix the `fmt` argument order/format string in `handlers.go` for i2v logging.

- MCP session cleanup warnings
  - Repeated warning: `Attempted to exit cancel scope in a different task than it was entered in` during MCP server restarts.
  - Impact: benign during server lifecycle, but noisy.
  - Recommendation: Investigate task/cancel-scope lifecycle; ensure cleanup happens within the same task scope. Low priority if functionality is unaffected.

- Vertex SDK warning about non-text parts
  - `Warning: there are non-text parts in the response: ['function_call'] ...`
  - Impact: expected when using tool calls; benign.
  - Recommendation: No action required.

## Performance Notes
- Veo i2v generation completed in ~61 seconds for a single 8-second 16:9 video, consistent with expectations.
- AFC (automatic function calling) enabled with max remote calls: 10; no signs of runaway calls.

## UX Improvement: Provide Direct HTTP Link to Output Video
Current output: a `gs://...` URI. Users often want a clickable HTTPS link.

Options to provide a validated HTTP URL:
- Signed URL (recommended)
  - Generate a short-lived V4 signed URL for the object (e.g., via Google Cloud Storage client libraries) and return it alongside the `gs://` in the tool response.
  - Pros: secure, no need to make the bucket public; works from any browser; expires automatically.
  - Where to implement:
    - In the Veo MCP server: after upload/availability, compute `public_url` using signed URL and include in the tool response JSON (e.g., `{ gs_uri, public_url, expires_at }`).
    - Alternatively, add a small ADK/MCP utility tool (e.g., `gcs_signed_url`) that returns a signed URL for any `gs://` path, which the agent can call after generation.

- Public bucket/object
  - Make the output prefix public and use `https://storage.googleapis.com/<bucket>/<object>`.
  - Pros: simple; Cons: public access management and potential data exposure risks. Less recommended for default.

- Cloud CDN / Website endpoint
  - Front the bucket with Cloud CDN or website hosting endpoint for stable HTTPS URLs.
  - Pros: cache, custom domain; Cons: extra setup/infra.

Recommendation: implement signed URL generation and return both `gs_uri` and `public_url` in the tool response.

## Overall Assessment
- The core objective—converting `artifact:` to `gs://` and successfully running Veo i2v—was achieved.
- The system behaved robustly: indexing is consistent (`user_image_0.jpg`), GCS artifact resolution works, and generation completed successfully.
- Minor polish items remain (auto MIME type, log formatting, and cleanup warnings), none of which block functionality.

## Proposed Follow-ups (No changes made yet)
- Auto-infer and inject `mime_type` in `before_tool_callback` if missing.
- Add a signed-URL feature and include `public_url` in tool responses so the UI can present a clickable link.
- Fix Veo i2v logging format string/argument order to eliminate `%!d(...)` artifacts.
- Investigate MCP cleanup warning; ensure cancel scopes are exited in the same task.

---
Prepared based on logs from the successful run on 2025-09-05 (local time) and current system configuration (`USE_GCS_ARTIFACTS=true`).
