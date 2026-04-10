# Kling AI

Text-to-video generation via the Kling AI API.

Base URL: `http://localhost:8080/api/v1/kling`

> This endpoint uses the async job system. See [jobs.md](jobs.md) for polling and SSE streaming.

---

## Required Configuration

| Variable | Description | Required |
|----------|-------------|----------|
| `KLING_API_KEY` | Kling AI API key | Yes |

---

## POST /kling/generate

**Method + Path:** `POST /api/v1/kling/generate`
**Description:** Submits a video generation request and returns a job ID immediately. The backend submits to Kling, polls until the video is ready, then downloads and saves it locally.

### Request Body

| Field | Type | Required | Valid Values | Default |
|-------|------|----------|--------------|---------|
| `prompt` | string | Yes | Any text description of the video | — |
| `negative_prompt` | string | No | What to avoid in the video | — |
| `mode` | string | No | `std`, `pro` | `std` |
| `duration` | string | No | `"5"`, `"10"` | `"5"` |
| `aspect_ratio` | string | No | `16:9`, `9:16`, `1:1` | `16:9` |
| `cfg_scale` | float | No | `0.0` – `1.0` | Provider default |

**Mode notes:**
- `std` — Standard quality, faster and cheaper.
- `pro` — Higher quality, slower and more expensive.

**`cfg_scale` note:** Controls how strictly the video follows the prompt. Higher = more faithful, lower = more creative.

### Response (202 Accepted)

```json
{
  "success": true,
  "job_id": "a1b2c3d4-...",
  "message": "Kling video generation job created"
}
```

### Example

```bash
curl -X POST http://localhost:8080/api/v1/kling/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A fisherman catching a large fish in a crystal clear lake surrounded by mountains, cinematic, golden hour",
    "negative_prompt": "blurry, low quality, shaky camera",
    "mode": "std",
    "duration": "5",
    "aspect_ratio": "16:9",
    "cfg_scale": 0.5
  }'
```

---

## Full Workflow

### Step 1 — Submit the request

```bash
curl -X POST http://localhost:8080/api/v1/kling/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A fisherman catching a large fish in a crystal clear lake surrounded by mountains, cinematic, golden hour",
    "mode": "std",
    "duration": "5",
    "aspect_ratio": "16:9"
  }'
```

Response:
```json
{
  "success": true,
  "job_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "message": "Kling video generation job created"
}
```

### Step 2 — Poll for status

Video generation takes time (typically 1–3 minutes). Progress goes from 10% (submitted to Kling) → 90% (downloading) → 100% (done).

```bash
curl http://localhost:8080/api/v1/jobs/a1b2c3d4-e5f6-7890-abcd-ef1234567890/status
```

Response while processing:
```json
{
  "success": true,
  "job": {
    "id": "a1b2c3d4-...",
    "provider": "kling",
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
  "provider": "kling",
  "result_url": "./outputs/kling/video-abc123.mp4",
  "result": {
    "video_url": "https://kling.ai/cdn/...",
    "video_path": "./outputs/kling/video-abc123.mp4",
    "duration": "5"
  }
}
```

### Step 4 (Optional) — Stream real-time updates via SSE

```bash
curl -N http://localhost:8080/api/v1/jobs/a1b2c3d4-e5f6-7890-abcd-ef1234567890/sse
```

---

## Notes

- `duration` is a **string**, not an integer — send `"5"` not `5`.
- `pro` mode produces higher quality but takes significantly longer to generate.
- `9:16` is the correct aspect ratio for TikTok/Reels vertical format.
- The backend polls Kling until the task reaches `succeed` or `failed` status, then downloads the video.
- Generated videos are saved locally to `OUTPUT_DIR/kling/` (defaults to `./outputs/kling/`).
- The `video_url` from Kling may expire — always use `video_path` for persistent access.
- If the job fails, check the server logs for the Kling task status message.
