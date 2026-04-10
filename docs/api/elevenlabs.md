# Sounds

Text-to-speech, sound effects, and music generation via the ElevenLabs API.

Base URL: `http://localhost:8080/api/v1/sounds`

> All generation endpoints use the async job system. See [jobs.md](jobs.md) for polling and SSE streaming.

---

## Required Configuration

| Variable | Description | Required |
|----------|-------------|----------|
| `ELEVENLABS_API_KEY` | ElevenLabs API key | Yes |

---

## POST /sounds/tts

**Method + Path:** `POST /api/v1/sounds/tts`
**Description:** Submits a text-to-speech request and returns a job ID immediately.

### Request Body

| Field | Type | Required | Valid Values | Default |
|-------|------|----------|--------------|---------|
| `text` | string | Yes | Any text to convert to speech | — |
| `voice_id` | string | Yes | ElevenLabs voice ID (use `/voices` to list) | — |
| `model_id` | string | No | ElevenLabs model ID (e.g. `eleven_multilingual_v2`) | Provider default |
| `voice_settings` | object | No | See voice settings below | Provider default |

**`voice_settings` object:**

| Field | Type | Description |
|-------|------|-------------|
| `stability` | float | 0.0–1.0. Higher = more consistent, less expressive |
| `similarity_boost` | float | 0.0–1.0. Higher = closer to original voice |

### Response (202 Accepted)

```json
{
  "success": true,
  "job_id": "a1b2c3d4-...",
  "message": "Speech generation job created"
}
```

### Example

```bash
curl -X POST http://localhost:8080/api/v1/sounds/tts \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Bienvenidos a Laguna Escondida, donde la naturaleza y el sabor se encuentran.",
    "voice_id": "21m00Tcm4TlvDq8ikWAM",
    "model_id": "eleven_multilingual_v2",
    "voice_settings": {
      "stability": 0.5,
      "similarity_boost": 0.75
    }
  }'
```

---

## POST /sounds/sound-effects

**Method + Path:** `POST /api/v1/sounds/sound-effects`
**Description:** Generates a sound effect from a text description and returns a job ID immediately.

### Request Body

| Field | Type | Required | Valid Values | Default |
|-------|------|----------|--------------|---------|
| `text` | string | Yes | Description of the sound effect to generate | — |
| `duration_seconds` | float | No | 0.5–22.0 | Provider default |
| `prompt_influence` | float | No | 0.0–1.0. Higher = closer to the prompt description | Provider default |

### Response (202 Accepted)

```json
{
  "success": true,
  "job_id": "a1b2c3d4-...",
  "message": "Sound effect generation job created"
}
```

### Example

```bash
curl -X POST http://localhost:8080/api/v1/sounds/sound-effects \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Water splashing on rocks in a mountain river, birds chirping in the background",
    "duration_seconds": 5.0,
    "prompt_influence": 0.3
  }'
```

---

## POST /sounds/music

**Method + Path:** `POST /api/v1/sounds/music`
**Description:** Generates music from a text prompt and returns a job ID immediately.

### Request Body

| Field | Type | Required | Valid Values | Default |
|-------|------|----------|--------------|---------|
| `text` | string | Yes | Description of the music to generate | — |
| `duration_seconds` | float | No | Duration of the generated music | Provider default |

### Response (202 Accepted)

```json
{
  "success": true,
  "job_id": "a1b2c3d4-...",
  "message": "Music generation job created"
}
```

### Example

```bash
curl -X POST http://localhost:8080/api/v1/sounds/music \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Relaxing Latin acoustic guitar, upbeat tropical feel, suitable for a nature restaurant",
    "duration_seconds": 30.0
  }'
```

---

## GET /sounds/voices

**Method + Path:** `GET /api/v1/sounds/voices`
**Description:** Returns all available voices for the configured API key. Use this to find a `voice_id` for TTS requests.

### Response (200 OK)

```json
{
  "success": true,
  "voices": [
    {
      "voice_id": "21m00Tcm4TlvDq8ikWAM",
      "name": "Rachel",
      "category": "premade"
    }
  ]
}
```

### Example

```bash
curl http://localhost:8080/api/v1/sounds/voices
```

---

## Full Workflow (TTS)

### Step 1 — Find a voice

```bash
curl http://localhost:8080/api/v1/sounds/voices
```

Note the `voice_id` you want to use.

### Step 2 — Submit TTS request

```bash
curl -X POST http://localhost:8080/api/v1/sounds/tts \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Bienvenidos a Laguna Escondida.",
    "voice_id": "21m00Tcm4TlvDq8ikWAM",
    "model_id": "eleven_multilingual_v2"
  }'
```

Response:
```json
{
  "success": true,
  "job_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "message": "Speech generation job created"
}
```

### Step 3 — Poll for status

```bash
curl http://localhost:8080/api/v1/jobs/a1b2c3d4-e5f6-7890-abcd-ef1234567890/status
```

### Step 4 — Retrieve result

```bash
curl http://localhost:8080/api/v1/jobs/a1b2c3d4-e5f6-7890-abcd-ef1234567890/result
```

Response:
```json
{
  "success": true,
  "job_id": "a1b2c3d4-...",
  "provider": "elevenlabs",
  "result_url": "https://...",
  "result": {
    "audio_url": "/path/to/output.mp3",
    "duration": 3.5
  }
}
```

### Step 5 (Optional) — Stream updates via SSE

```bash
curl -N http://localhost:8080/api/v1/jobs/a1b2c3d4-e5f6-7890-abcd-ef1234567890/sse
```

---

## Full Workflow (Sound Effects)

### Step 1 — Submit sound effect request

```bash
curl -X POST http://localhost:8080/api/v1/sounds/sound-effects \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Campfire crackling with gentle night insects in a tropical setting",
    "duration_seconds": 8.0,
    "prompt_influence": 0.5
  }'
```

Response:
```json
{
  "success": true,
  "job_id": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
  "message": "Sound effect generation job created"
}
```

### Step 2 — Poll and retrieve (same as TTS workflow, steps 3–5)

```bash
curl http://localhost:8080/api/v1/jobs/b2c3d4e5-f6a7-8901-bcde-f12345678901/result
```

---

## Notes

- Use `eleven_multilingual_v2` model for Spanish/multilingual TTS content.
- `voice_id` is required for TTS — there is no default. Always call `/sounds/voices` first if you don't have one saved.
- Generated audio is saved locally to `OUTPUT_DIR/elevenlabs/` (defaults to `./outputs/elevenlabs/`).
- `duration` in the result is the audio length in seconds.
- `prompt_influence` for sound effects controls how strictly the output follows the text description (0 = creative, 1 = literal).
