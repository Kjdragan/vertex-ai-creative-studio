# Run Evaluation 4 — i2v + TTS + Combine + Link (Prompt-Fidelity Review)

Date/time: 2025-09-06 15:47 (local)
Session context: ADK GenMedia agent (`genmedia_agent`)

## Summary
- __Outcome__: The system successfully generated a video from the uploaded image and combined it with a TTS audio track. A functional HTTPS link was returned.
- __Prompt fidelity__: The trace shows `chirp_tts()` was called with your exact text. However, the final output may have prioritized a different audio track during playback (likely an audio stream present in the Veo video), causing a mismatch with what you intended to hear.
- __Signed link__: The response returned a direct `https://storage.googleapis.com/...` link rather than calling `gcs_signed_url`. The link worked, but this bypasses our new signed-URL behavior and may rely on public object access.

## Timeline of Key Events
- 15:18:10
  - AFC/LLM init logging; environment looks normal.
- 15:18:39
  - LLM issues function calls:
    - `chirp_tts(text="Happy birthday to you... happy birthday to you,,, happy birthday dear Kevin, Happy birthday to you!", output_directory=".")`
    - `veo_i2v(model="veo-3.0-generate-001", image_uri="gs://.../user_image_0.jpg/0", prompt="Alien Monster happily singing to the camera")`
- 15:18:39-15:18:41
  - Function responses:
    - `chirp_tts` → "Speech synthesized successfully ... Audio saved to: `chirp_audio-en-US-Chirp3-HD-Achernar-20250906-151648.wav`"
    - `veo_i2v` → "Generated 1 video ... GCS: `gs://supple-synapse-media/veo_outputs/12133573038466716787/sample_0.mp4`"
- 15:19:00-15:19:02
  - Combine: `ffmpeg_combine_audio_and_video(input_video_uri=gs://.../sample_0.mp4, input_audio_uri=chirp_audio-...-151648.wav, output_file_name=Alien_sings_happy_birthday.mp4)`
  - FFMpeg command:
    - `ffmpeg -y -i sample_0.mp4 -i chirp_audio.wav -map 0 -map 1:a -c:v copy -shortest ...`
  - Upload: `gs://supple-synapse-media/Alien_sings_happy_birthday.mp4`
- 15:19:31
  - Agent response includes:
    - `gs://supple-synapse-media/Alien_sings_happy_birthday.mp4`
    - `https://storage.googleapis.com/supple-synapse-media/Alien_sings_happy_birthday.mp4`
    - Mentions the TTS WAV filename.

## Evidence (from logs/traces)
- TTS call used your exact text:
```
functionCall: chirp_tts
args: {
  "text": "Happy birthday to you... happy birthday to you,,, happy birthday dear Kevin, Happy birthday to you!",
  "output_directory": "."
}
```
- TTS succeeded and saved audio locally:
```
"Speech synthesized successfully with voice en-US-Chirp3-HD-Achernar. Audio saved to: chirp_audio-en-US-Chirp3-HD-Achernar-20250906-151648.wav (245804 bytes)."
```
- Veo video generated:
```
"Generated 1 video(s) ... gs://supple-synapse-media/veo_outputs/12133573038466716787/sample_0.mp4"
```
- Combine command and result:
```
FFMpeg command:
ffmpeg -y -i /tmp/input_.../sample_0.mp4 -i chirp_audio-en-US-Chirp3-HD-Achernar-20250906-151648.wav \
  -map 0 -map 1:a -c:v copy -shortest /tmp/output_.../Alien_sings_happy_birthday.mp4
...
Output uploaded to GCS: gs://supple-synapse-media/Alien_sings_happy_birthday.mp4
```
- Final assistant message (no explicit `gcs_signed_url` tool call observed):
```
Here is the final video with audio:
  gs://supple-synapse-media/Alien_sings_happy_birthday.mp4
  https://storage.googleapis.com/supple-synapse-media/Alien_sings_happy_birthday.mp4
```
- CHIRP trace excerpt shows a `ListVoices` RPC (status OK), which is consistent with the TTS service probing voices. The main TTS synthesis also succeeded as shown above.

## Findings
- __TTS fidelity__: The agent passed your exact text to `chirp_tts` and received a WAV. There is no indication the model invented its own TTS text in this run.
- __Likely cause of perceived mismatch__: The combine command used `-map 0 -map 1:a`. The `-map 0` flag selects all streams from the video input, including any existing audio stream. If Veo’s output contained an audio stream (music/voice), the final MP4 now contains multiple audio tracks. Most players default to the first audio stream (from input 0), so you may have heard the video’s own audio and not the Chirp TTS (or heard a different track by default).
- __Signed link behavior__: The response included a direct `https://storage.googleapis.com/...` link without a `gcs_signed_url` tool call. The link was functional but may rely on the object being public or previously configured ACLs, and it bypasses our preferred signed-URL mechanism.
- __CHIRP MCP health__: `chirp_tts` succeeded (WAV created). The `ListVoices` trace is normal capability probing; it does not indicate a failure to synthesize.

## Root Causes
- __FFMpeg mapping included source audio__: Using `-map 0` instead of `-map 0:v:0` caused all streams (including possible source audio) to be included. This leads to multiple audio tracks and an unpredictable default audio selection at playback.
- __Agent link policy not enforced__: The assistant returned a direct public URL instead of invoking `gcs_signed_url`, contrary to agent instruction. This appears to be an LLM response gap rather than a tool failure.

## Recommendations
- __Combine mapping fix__: Update the combine to select only video from input 0 and audio from the TTS file, and set the Chirp audio as the default track:
  - Replace `-map 0 -map 1:a` with `-map 0:v:0 -map 1:a:0`.
  - Optionally add `-disposition:a:0 default` to ensure the TTS track is the one players pick by default.
  - Keep `-c:v copy -shortest` and transcode audio as needed (e.g., `-c:a aac`).
- __Preflight safeguards__ (agent callback or AV tool):
  - Run `ffmpeg_get_media_info` on the Veo video before combine; if an audio stream exists, either strip it or force video-only mapping with `-map 0:v:0`.
  - After combine, run `ffmpeg_get_media_info` to assert exactly 1 audio track and that it matches the desired codec/duration.
- __TTS determinism__:
  - Continue passing the exact `text` as done here. Optionally switch to SSML for precise pacing and punctuation handling.
  - Allow optional `voice_name` selection for reproducibility.
- __Signed URL enforcement__:
  - Ensure every final response calls the `gcs_signed_url` tool and returns `{gs_uri, public_url, expires_at}`. Avoid pasting raw `storage.googleapis.com` links unless explicitly configured for public access.

## Next Validation Steps
1. Re-run a short test with the same prompt and image.
2. Before combine, inspect the Veo video with `ffmpeg_get_media_info` to see if it has an audio stream.
3. Combine with `-map 0:v:0 -map 1:a:0 -c:v copy -c:a aac -shortest` and set the audio disposition default.
4. After combine, verify only one audio track (or that the TTS track is default) and confirm audible content matches your text.
5. Confirm the agent returns a signed `public_url` via `gcs_signed_url`.

## Direct Access Links
- Final combined video (from this run):
  - `gs://supple-synapse-media/Alien_sings_happy_birthday.mp4`
  - https://storage.googleapis.com/supple-synapse-media/Alien_sings_happy_birthday.mp4
- Silent Veo video:
  - `gs://supple-synapse-media/veo_outputs/12133573038466716787/sample_0.mp4`
- TTS audio file (local during run):
  - `chirp_audio-en-US-Chirp3-HD-Achernar-20250906-151648.wav`

## Trace Addendum (key excerpts)
- TTS call:
```
chirp_tts({
  "text": "Happy birthday to you... happy birthday to you,,, happy birthday dear Kevin, Happy birthday to you!",
  "output_directory": "."
})
```
- Combine call and FFMpeg mapping:
```
ffmpeg_combine_audio_and_video({
  "input_video_uri": "gs://supple-synapse-media/veo_outputs/12133573038466716787/sample_0.mp4",
  "input_audio_uri": "chirp_audio-en-US-Chirp3-HD-Achernar-20250906-151648.wav",
  "output_file_name": "Alien_sings_happy_birthday.mp4"
})

ffmpeg ... -map 0 -map 1:a -c:v copy -shortest ...
```
- Final response (no `gcs_signed_url` call logged):
```
... https://storage.googleapis.com/supple-synapse-media/Alien_sings_happy_birthday.mp4
```
