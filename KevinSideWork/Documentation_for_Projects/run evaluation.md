# Run Evaluation — ADK GenMedia i2v + TTS + Signed URL

Date/time: 2025-09-06 13:56 (local)
Session ID: `0a4f0145-5ea0-48b2-afc7-b60274850f98`

## Summary
- __Success__: Veo i2v generation succeeded using the uploaded image artifact.
  - Video saved to `gs://supple-synapse-media/veo_outputs/11029419343277766449/sample_0.mp4`.
- __Partial__: Agent attempted to combine TTS audio with the video via `ffmpeg_combine_audio_and_video`, but reported the audio object did not exist.
- __Hiccup__: Signed URL generation failed even after adding Token Creator on a signer service account and setting `SIGNED_URL_SERVICE_ACCOUNT`.

## Evidence (Terminal & Tracing)
- Veo i2v success:
  - "Generated 1 video(s) using model veo-3.0-generate-001. This took about 1m1s. Videos saved to GCS: gs://supple-synapse-media/veo_outputs/11029419343277766449/sample_0.mp4."
- Combine failure:
  - `ffmpeg_combine_audio_and_video` → "Failed to prepare input audio: failed to download gs://supple-synapse-media/chirp_outputs/11029419343277766449_0.wav from GCS: Object("chirp_outputs/11029419343277766449_0.wav").NewReader: storage: object doesn't exist"
- Signed URL failure (tool `gcs_signed_url`):
  - "Failed to generate signed URL: you need a private key to sign credentials. the credentials you are currently using <class 'google.oauth2.credentials.Credentials'> just contains a token... Ensure your ADK runtime uses sign-capable credentials... or set SIGNED_URL_SERVICE_ACCOUNT=<service-account-email> (or GOOGLE_IMPERSONATE_SERVICE_ACCOUNT) and grant your current principal 'Service Account Token Creator'."
- Agent response fallback:
  - "I am unable to provide a direct HTTP link... The tool I use to do this is currently unavailable... You can still access the video using the following gs:// URI..."

## Observations
- __Inline audio present in trace__: The request context includes an `audio/wav` blob snippet (base64 truncated), suggesting TTS audio was produced inline (not saved to GCS).
- __Agent attempted GCS combine path__: The combine tool used a presumed GCS path for TTS (`gs://supple-synapse-media/chirp_outputs/11029419343277766449_0.wav`) that did not exist.
- __User experience__: Despite the combine error, the clicked video contained correct audio and voice. This implies either:
  - Veo output already included audio (no combine needed), or
  - A separate combine path succeeded outside the `ffmpeg_combine_audio_and_video` tool invocation tracked here.
- __Environment__: `.env` shows `SIGNED_URL_SERVICE_ACCOUNT=genmedia-sa@supple-synapse-470916-a2.iam.gserviceaccount.com` and ADK startup script sources `.env`.
- __Go workspace__: `experiments/mcp-genmedia/mcp-genmedia-go/go.work` requires `go 1.24.3`.

## Root Cause Analysis
- __Signed URL (primary)__
  - The ADK process is running under user ADC (`GOOGLE_APPLICATION_CREDENTIALS` → ADC token file), which cannot sign URLs locally.
  - The `gcs_signed_url` implementation attempted IAM-based signing using `service_account_email` with ADC, but the installed `google-cloud-storage` likely defaulted to local signing and rejected the user token, or IAMCredentials API was not used by the library version in the ADK runtime.
  - Result: Error indicating lack of private key on the credential, despite Token Creator being granted on the signer SA.

- __Audio/Video combine (secondary)__
  - The agent attempted to combine with an audio file at a pre-computed GCS path that never got written, while the TTS output in this run appears inline (returned in the response) rather than persisted to GCS.
  - There are missing guards:
    - No existence check before calling `ffmpeg_combine_audio_and_video`.
    - No fallback to upload inline audio to GCS if TTS doesn’t store directly.
    - No detection to skip combine when the video already contains an audio track.

- __Messaging UX__
  - When `gcs_signed_url` fails, the agent falls back to a generic apology message. We should return the `gs://` URI plus a clear “signed-link disabled” reason and a remediation hint, or attempt alternate signing strategies.

## Plan to Fix (No code changes yet)

### Phase 1 — Environment & Access Validation
- __Enable IAM Service Account Credentials API__ (`iamcredentials.googleapis.com`) on project `supple-synapse-470916-a2`.
- __Confirm IAM bindings__:
  - The caller identity (your current ADC principal) has `roles/iam.serviceAccountTokenCreator` on `genmedia-sa@supple-synapse-470916-a2.iam.gserviceaccount.com`.
- __Verify `.env` is being sourced__ by `start_adk.sh` (it is) and that `SIGNED_URL_SERVICE_ACCOUNT` remains set.

### Phase 2 — Signed URL Tool Robustness
- __Prefer explicit impersonation credentials__ in `gcs_signed_url`:
  - Use `google.auth.impersonated_credentials.Credentials` with `target_principal=<SIGNED_URL_SERVICE_ACCOUNT>` and `target_scopes=['https://www.googleapis.com/auth/cloud-platform']`, then call `blob.generate_signed_url(credentials=impersonated_creds)`.
  - This avoids reliance on library auto-detection of IAM-based signing and works with user ADC.
- __Runtime dependency__:
  - Ensure ADK environment has a modern `google-cloud-storage` and `google-auth` supporting impersonation. Pin a known-good version in the ADK env (separate from repo `requirements.txt`).
- __Graceful fallback__:
  - On failure, return the `gs://` URI and include a short remediation ("Signed URL service not configured: enable IAM Credentials API / check Token Creator role / confirm signer SA") and optionally a Console object URL hint.

### Phase 3 — Audio + Video Combine Reliability
- __TTS output contract__:
  - Always call `chirp_tts` with `output_gcs_bucket` (default to `GENMEDIA_BUCKET` or `CHIRP3_BUCKET_PATH`) so it saves to GCS and returns a `gs://` URI.
  - If `chirp_tts` returns inline audio only, upload it to `gs://supple-synapse-media/chirp_outputs/<session>/<id>.wav` using our artifact/GCS handler, then proceed.
- __Existence gating__:
  - Before `ffmpeg_combine_audio_and_video`, verify both `gs://` objects exist via `storage.Client().bucket(...).blob(...).exists()` and emit a helpful error if missing.
- __Skip when not needed__:
  - Detect audio track in the video (via `ffprobe`/`ffmpeg -i`) and skip combine if audio is already present.

### Phase 4 — Agent Messaging Improvements
- When `gcs_signed_url` errors, reply with:
  - The `gs://` URI
  - A one-line reason (e.g., "Signed URL requires IAM impersonation")
  - A button/instruction to retry after configuration (no generic "tool unavailable").

### Phase 5 — Go Toolchain & Rebuild
- Upgrade to Go 1.24.3+ to match `go.work` and `go.mod` directives.
- Rebuild all MCP servers under `experiments/mcp-genmedia/mcp-genmedia-go/` so fixes (e.g., Veo log formatting) take effect.
- Smoke test: Veo (t2v/i2v), Imagen, Chirp3, Lyria, AVTool.

### Phase 6 — Observability
- Extend Arize tracing to capture tool error fields for `gcs_signed_url` and combine failures (e.g., which URI failed existence check) for faster triage.

## Open Questions
- Does Veo i2v already include an audio track in some modes? If yes, we should add an explicit detection to avoid unnecessary combine.
- Should we create an explicit "audio artifact" path similar to images to unify handling of inline audio?

## Action Items (for next iteration)
- __Signed URL__: implement explicit impersonated credentials in `gcs_signed_url`; verify IAM API enabled; re-run.
- __Audio pipeline__: force TTS to write to GCS or upload inline audio; add existence checks; skip combine if audio already in video.
- __Messaging__: improve fallback text and include console object URL.
- __Go upgrade__: move to 1.24.3+, rebuild MCP servers, smoke test.
- __Docs__: update `lessonslearned.md` and `currentstatus.md` with the above once implemented.

## References
- `.env`: `experiments/mcp-genmedia/sample-agents/adk/genmedia_agent/.env`
- Start script: `experiments/mcp-genmedia/sample-agents/adk/start_adk.sh`
- Signed URL tool: `experiments/mcp-genmedia/sample-agents/adk/genmedia_agent/agent.py` (function `gcs_signed_url`)
- MCP Go workspace: `experiments/mcp-genmedia/mcp-genmedia-go/go.work`
