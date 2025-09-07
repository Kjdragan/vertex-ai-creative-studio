# Run Evaluation 2 — ADK GenMedia i2v + TTS + Signed URL

Date/time: 2025-09-06 14:41 (local)
Session context: ADK GenMedia agent (user-initiated second run)

## Summary
- __Observed__: The run initially failed to combine video and audio due to a non-existent local "artifact:" audio path. A retry synthesized audio locally and the `ffmpeg` combine step succeeded, uploading the final video to GCS.
- __Status__: Final combined video exists at `gs://supple-synapse-media/xenomorph_singing_happy_birthday.mp4`.
- __Signed URL__: Generation failed again due to token-only ADC credentials. IAM Service Account Credentials API is now enabled, so impersonation-based signing should work after restart and/or minor config adjustments.

## Timeline of Key Events
- 14:37-14:39
  - __Attempted combine__: `ffmpeg_combine_audio_and_video` called with `input_audio_uri="artifact:chirp_tts_response.result.content[1]"` and `input_video_uri="gs://supple-synapse-media/veo_outputs/18318113552497408356/sample_0.mp4"`.
  - __Error__: `Failed to prepare input audio: local input file artifact:chirp_tts_response.result.content[1] does not exist`.
- 14:37:44
  - __TTS retry__: `chirp_tts` invoked with local output (no GCS), returned `chirp_audio-en-US-Chirp3-HD-Achernar-20250906-143744.wav`.
- Immediately after
  - __Combine succeeds__: `ffmpeg_combine_audio_and_video` with the local WAV and the Veo video completed in ~2s.
  - __Upload__: Output uploaded to `gs://supple-synapse-media/xenomorph_singing_happy_birthday.mp4`.
- 14:39-14:40
  - __Signed URL attempt__: `gcs_signed_url(gs_uri=gs://supple-synapse-media/xenomorph_singing_happy_birthday.mp4)` → failure with error:
    - "you need a private key to sign credentials ... credentials are <class 'google.oauth2.credentials.Credentials'> token-only ... set SIGNED_URL_SERVICE_ACCOUNT=... and grant Token Creator ..."
  - __Agent fallback__: Returned only the `gs://` URI.

## Evidence (from logs)
- Combine error (first attempt):
  - `Failed to prepare input audio: local input file artifact:chirp_tts_response.result.content[1] does not exist`
- TTS success:
  - `Speech synthesized successfully ... Audio saved to: chirp_audio-en-US-Chirp3-HD-Achernar-20250906-143744.wav`
- Combine success:
  - `Audio and video combination completed in 1.979s ... Output uploaded to GCS: gs://supple-synapse-media/xenomorph_singing_happy_birthday.mp4`
- Signed URL error:
  - `Failed to generate signed URL: you need a private key to sign credentials ... token-only credentials ...`

## Direct Access Links
- __gs:// URI__:
  - `gs://supple-synapse-media/xenomorph_singing_happy_birthday.mp4`
- __Console details (clickable)__:
  - https://console.cloud.google.com/storage/browser/_details/supple-synapse-media/xenomorph_singing_happy_birthday.mp4?project=supple-synapse-470916-a2
- __Authenticated viewer (requires Google login)__:
  - https://storage.cloud.google.com/supple-synapse-media/xenomorph_singing_happy_birthday.mp4

## Findings
- __Artifact audio placeholder was not a real file__: The first combine attempt used a pseudo path `artifact:chirp_tts_response.result.content[1]` which does not map to a local or GCS file, causing the failure.
- __Retry saved TTS locally, enabling combine__: Once `chirp_tts` produced a local WAV path, `ffmpeg_combine_audio_and_video` worked and uploaded the final video to GCS.
- __Signed URL still blocked by auth method__: Despite `SIGNED_URL_SERVICE_ACCOUNT` being set and IAM Service Account Credentials API now enabled, the code path still encountered a token-only credential context, so signing failed. A restart and/or using explicit impersonated credentials is likely required.
- __Perceived "timeout" likely from final link step__: The core media pipeline completed, but the agent could not produce a clickable HTTPS link, which may present as an incomplete/"timed out" experience.

## Root Causes
- __Audio persistence contract gap__:
  - TTS sometimes returns inline/local output rather than a GCS URI. Combine logic assumed a persisted GCS object or valid local path. When presented with the `artifact:` token, the combine step failed.
- __Signed URL credential mismatch__:
  - ADK runtime uses ADC token credentials; library attempted local signing instead of IAM impersonation. Without explicit impersonated credentials, signing fails even with Token Creator role.

## Recommendations (no code changes applied yet)
- __TTS → GCS enforcement__:
  - Always request TTS to save to GCS (e.g., `GENMEDIA_BUCKET` or `CHIRP3_BUCKET_PATH`) so combine receives a `gs://` URI. If only inline/local data is returned, upload it to `gs://supple-synapse-media/chirp_outputs/...` before combine.
  - Add preflight existence checks for both audio and video before invoking `ffmpeg_combine_audio_and_video`.
  - Optionally detect if the target video already has an audio track and skip combine.
- __Signed URL via impersonation__:
  - Now that `iamcredentials.googleapis.com` is enabled, restart the ADK process so env is fresh, then re-run the `gcs_signed_url` tool.
  - If it still fails, update the signing path to use explicit `impersonated_credentials.Credentials(target_principal=SIGNED_URL_SERVICE_ACCOUNT)` and pass `credentials=...` to `blob.generate_signed_url(...)`.
- __User messaging__:
  - When signing fails, include the `gs://` URI plus a clickable Console details link in the reply so the user has an immediate path to the asset.

## Next Validation Steps
1. Restart ADK (`experiments/mcp-genmedia/sample-agents/adk/start_adk.sh`) so the enabled IAM Credentials API and env vars are active in the runtime.
2. Trigger `gcs_signed_url(gs_uri=gs://supple-synapse-media/xenomorph_singing_happy_birthday.mp4)` again from the UI.
3. If signing still fails, switch to explicit impersonation credentials in `gcs_signed_url` and re-test.
4. For future i2v+TTS runs, ensure TTS output is persisted to GCS or uploaded before the combine step.

## Trace Addendum
- __Session ID__: `c60b1549-d001-4c18-b145-3eb0723a0914`
- __Initial plan__: The agent stated it would generate a silent video and separate audio, then combine them.
- __veo_i2v call__: `veo_i2v(model="veo-3.0-generate-001", image_uri="artifact:user_image_0.jpg", prompt="A funny video of this monster happily singing to the camera")`.
- __First chirp_tts call__: `chirp_tts(text="Happy birthday to you, ...")` (no explicit GCS output), implying audio returned inline/locally rather than persisted to GCS.
- __First combine attempt__: `ffmpeg_combine_audio_and_video(input_audio_uri="artifact:chirp_tts_response.result.content[1]", input_video_uri="gs://supple-synapse-media/veo_outputs/18318113552497408356/sample_0.mp4", output_file_name="xenomorph_singing_happy_birthday.mp4")` →
  - Error: `Failed to prepare input audio: local input file artifact:chirp_tts_response.result.content[1] does not exist for input_audio`.
- __TTS retry with local output__: `chirp_tts(text="Happy birthday to you, ...", output_directory=".")` →
  - Success: `Audio saved to: chirp_audio-en-US-Chirp3-HD-Achernar-20250906-143744.wav (274604 bytes)`.
- __Second combine attempt__: `ffmpeg_combine_audio_and_video(input_video_uri="gs://supple-synapse-media/veo_outputs/18318113552497408356/sample_0.mp4", input_audio_uri="chirp_audio-en-US-Chirp3-HD-Achernar-20250906-143744.wav", output_file_name="xenomorph_singing_happy_birthday.mp4")` →
  - Success in ~1.98s; uploaded to `gs://supple-synapse-media/xenomorph_singing_happy_birthday.mp4`.
- __Signed URL attempt__: `gcs_signed_url(gs_uri="gs://supple-synapse-media/xenomorph_singing_happy_birthday.mp4")` →
  - Error: `Failed to generate signed URL: you need a private key to sign credentials. the credentials you are currently using <class 'google.oauth2.credentials.Credentials'> just contains a token ...`.
- __Agent fallback__: Returned only the `gs://` URI to the user (no clickable HTTPS link).

## References
- Env: `experiments/mcp-genmedia/sample-agents/adk/genmedia_agent/.env`
- Start script: `experiments/mcp-genmedia/sample-agents/adk/start_adk.sh`
- Tools: `ffmpeg_combine_audio_and_video`, `chirp_tts`, `gcs_signed_url`
- Output object: `gs://supple-synapse-media/xenomorph_singing_happy_birthday.mp4`
