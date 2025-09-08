# Information: Text-to-Speech vs Native Veo Audio

Last updated: 2025-09-06 16:18 (local)
Applies to: `experiments/mcp-genmedia/sample-agents/adk/genmedia_agent/`

## Executive Summary
- The ADK agent generated a video from your uploaded image and produced a local TTS WAV that was then used in the combine step.
- The combine used `ffmpeg -map 0 -map 1:a`, which likely included Veo’s own audio stream along with your TTS track, letting players default to Veo’s audio instead of your TTS.
- The signed link tool (`gcs_signed_url`) failed in one later step because the runtime was still using token-only credentials; the code now supports IAM impersonation but the runtime must be restarted with the correct environment to take effect.
- If you prefer the Veo “native” alien voice yet want exact phrasing, current `veo_i2v` tools don’t expose a transcript input. You can prompt for speech, but it’s not guaranteed to be verbatim. TTS + combine remains the deterministic path, while Veo-native speech is a style-first path.

---

## What Happened in Your Run
- `chirp_tts()` was called with your exact text and saved a local file.
  - Example evidence:
    - "Speech synthesized successfully … Audio saved to: `chirp_audio-en-US-Chirp3-HD-Achernar-20250906-151648.wav`"
- `ffmpeg_combine_audio_and_video()` combined the Veo video and your TTS.
  - The tool logged the FFMpeg command: `ffmpeg -y -i sample_0.mp4 -i chirp_audio.wav -map 0 -map 1:a -c:v copy -shortest …`
  - `-map 0` includes all streams (video and possibly audio) from input 0 (Veo).
  - Result: multiple audio tracks in the final MP4, and media players often default to the first audio stream (commonly the source’s own audio), making it sound like your TTS wasn’t used.
- For sharing the audio, you uploaded it to GCS and attempted to create a signed URL.
  - Upload succeeded: `gs://supple-synapse-media/alien_happy_birthday_audio.wav`
  - Signed URL failed with token-only ADC. The agent code now supports impersonation, but the runtime must be restarted with `SIGNED_URL_SERVICE_ACCOUNT` (or `GOOGLE_IMPERSONATE_SERVICE_ACCOUNT`) and appropriate IAM (`Service Account Token Creator`) to succeed.

---

## Is Local TTS Accessible to the AV Tool?
- Yes. The Go AV tool accessed the WAV directly from the agent’s local workspace:
  - “Using local input file … for input_audio”.
- Uploading TTS to GCS is only required for user preview and shareable links, not for the internal combine step.

---

## Deterministic Words vs Native “Alien” Voice
- Current `veo_i2v` tool interface (`veo_i2v` parameters: `image_uri`, `prompt`, `model`, etc.) does not accept a transcript field.
- You can try to prompt Veo to say exact words, but consistency is not guaranteed by the API surface.
- For exact wording, TTS + combine is still the most reliable method. We can make it sound more “alien” via:
  - Enforcing specific `voice_name` in `chirp_tts` (and fixing the handler to always honor it).
  - Adding DSP filters at combine time (pitch/formant shifts, slight reverb) using FFMpeg filters.

---

## Recommended Approaches

- __A) Veo-native speech first, TTS as fallback__
  - Try to prompt Veo to speak the exact phrase. If generated audio matches, use it.
  - If not, fallback to TTS + combine.

- __B) Always TTS + combine (deterministic)__
  - Generate TTS with desired voice/style.
  - Combine with Veo video using strict mapping so only the TTS audio is included and set as default.

- __C) Hybrid (choose per request)__
  - Expose a user switch (e.g., “prefer_native_audio”: true/false) to control the path.

---

## Concrete Fixes (when you’re ready)

- __FFMpeg mapping (in AV tool)__
  - Replace `-map 0 -map 1:a` with a strict mapping to ensure your TTS is the only/default audio:

  ```bash
  ffmpeg -y -i input.mp4 -i tts.wav \
    -map 0:v:0 -map 1:a:0 \
    -c:v copy -c:a aac -shortest \
    -disposition:a:0 default \
    output.mp4
  ```

  - Optionally, run `ffmpeg_get_media_info` before/after to assert a single audio track and the expected codec/duration.

- __Signed URL generation (in agent runtime)__
  - `experiments/mcp-genmedia/sample-agents/adk/genmedia_agent/agent.py:gcs_signed_url` now prefers IAM impersonation.
  - Ensure runtime env and IAM are correct, then restart:
    - `SIGNED_URL_SERVICE_ACCOUNT=<sa>@<project>.iam.gserviceaccount.com` (or `GOOGLE_IMPERSONATE_SERVICE_ACCOUNT`)
    - Caller has `roles/iam.serviceAccountTokenCreator` on that service account
    - IAM Service Account Credentials API enabled

- __Voice selection enforcement (Chirp)__
  - Ensure `chirp_tts` honors `voice_name` consistently.
  - If needed, add filters for alienization via FFMpeg (e.g., as a follow-up tool step):

  ```bash
  ffmpeg -y -i tts.wav -af "asetrate=44100*0.9,aresample=44100,atempo=1.0,aphaser=type=t:speed=0.5" alienized.wav
  ```

---

## Files/Functions Relevant to This Topic
- `experiments/mcp-genmedia/sample-agents/adk/genmedia_agent/simple_callback.py`
  - `before_tool_callback()` — already enforces local TTS output (`output_directory='.'`) and does preflight checks for combine.
- `experiments/mcp-genmedia/sample-agents/adk/genmedia_agent/agent.py`
  - `gcs_signed_url()` — updated to use IAM impersonation for V4 signing.
- `experiments/mcp-genmedia/mcp-genmedia-go/` (AV tool)
  - `ffmpeg_commands.go` (or equivalent) — where the `-map` parameters are set.

---

## Quick Decision Guide
- __Need exact words as heard?__ Choose TTS + combine.
- __Prefer Veo’s alien style, can tolerate variability?__ Try Veo-native prompting first; fall back to TTS if it misses.
- __Need shareable links?__ Upload to GCS and use `gcs_signed_url` with impersonation.

---

## Next Steps for Validation
1. Run Veo i2v with your image; inspect with `ffmpeg_get_media_info` to see if Veo added an audio stream.
2. Combine using the strict mapping above; verify only one audio track and that the audible content matches your script.
3. Restart ADK so the impersonated `gcs_signed_url` path is active; confirm `{ gs_uri, public_url, expires_at }` is returned.
4. Optionally, test a Veo-native attempt to speak your exact phrase; if it misses, use TTS + combine.
