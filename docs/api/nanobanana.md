# Nano Banana Pro (via Fal.ai)

Generates images from text prompts using Nano Banana Pro, routed through the Fal.ai queue API.

Base URL: `http://localhost:8080/api/v1/nanobanana`

> This endpoint uses the async job system. See [jobs.md](jobs.md) for polling and SSE streaming.

---

## Required Configuration

| Variable | Description | Required |
|----------|-------------|----------|
| `NANOBANANA_API_KEY` | Fal.ai API key | Yes |

> The backend calls Fal.ai's queue API. Your API key must be a Fal.ai key.

---

## POST /nanobanana/generate

**Method + Path:** `POST /api/v1/nanobanana/generate`
**Description:** Submits an image generation request to Nano Banana Pro via Fal.ai and returns a job ID immediately.

### Request Body

| Field | Type | Required | Valid Values | Default |
|-------|------|----------|--------------|---------|
| `prompt` | string | Yes | Any text description | — |
| `negative_prompt` | string | No | Text describing what to avoid | — |
| `image_size` | string | No | `square`, `square_hd`, `portrait_4_3`, `portrait_16_9`, `landscape_4_3`, `landscape_16_9` | `square` |
| `num_images` | int | No | `1`–`4` | `1` |
| `num_inference_steps` | int | No | Integer | `28` |
| `guidance_scale` | float | No | Float | `3.5` |
| `seed` | int | No | Integer (for reproducibility) | — |

### Response (202 Accepted)

```json
{
  "success": true,
  "job_id": "a1b2c3d4-...",
  "message": "Nano Banana image generation job created"
}
```

### Example

```bash
curl -X POST http://localhost:8080/api/v1/nanobanana/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A clean minimal infographic on a dark background",
    "negative_prompt": "blurry, cartoon, stock photo",
    "image_size": "square_hd",
    "num_images": 2,
    "num_inference_steps": 28,
    "guidance_scale": 4.0
  }'
```

---

## Full Workflow

### Step 1 — Submit the request

```bash
curl -X POST http://localhost:8080/api/v1/nanobanana/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A clean minimal infographic on a dark background",
    "negative_prompt": "blurry, cartoon, stock photo",
    "image_size": "square_hd",
    "num_images": 2,
    "num_inference_steps": 28,
    "guidance_scale": 4.0
  }'
```

Response:
```json
{
  "success": true,
  "job_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "message": "Nano Banana image generation job created"
}
```

### Step 2 — Poll for status

Progress goes from 10% (submitted to Fal.ai) → 90% (downloading) → 100% (done).

```bash
curl http://localhost:8080/api/v1/jobs/a1b2c3d4-e5f6-7890-abcd-ef1234567890/status
```

Response while processing:
```json
{
  "success": true,
  "job": {
    "id": "a1b2c3d4-...",
    "provider": "nanobanana",
    "status": "processing",
    "progress": 45
  }
}
```

### Step 3 — Retrieve result (once status is `completed`)

```bash
curl http://localhost:8080/api/v1/jobs/a1b2c3d4-e5f6-7890-abcd-ef1234567890/result
```

Response:
```json
{
  "success": true,
  "job_id": "a1b2c3d4-...",
  "provider": "nanobanana",
  "result_url": "./outputs/nanobanana/image-abc123.png",
  "result": {
    "image_url": "https://fal.media/files/...",
    "image_path": "./outputs/nanobanana/image-abc123.png",
    "width": 1024,
    "height": 1024
  }
}
```

### Step 4 (Optional) — Stream real-time updates via SSE

```bash
curl -N http://localhost:8080/api/v1/jobs/a1b2c3d4-e5f6-7890-abcd-ef1234567890/sse
```

---

## Notes

- The API key in `.env` must be a **Fal.ai key**. Nano Banana Pro is accessed as a model hosted on Fal.ai.
- `num_images` supports up to 4 images per request.
- `guidance_scale` controls how closely the image follows the prompt — higher values = more literal.
- `num_inference_steps` affects quality vs speed — higher values = better quality, slower generation.
- `seed` is optional; set it for reproducible results across runs.
- Generated images are saved locally to `OUTPUT_DIR/nanobanana/` (defaults to `./outputs/nanobanana/`).
- The `image_url` from Fal.ai may expire — always use `image_path` for persistent access.
