# Pika (via Fal.ai)

Text-to-video generation using Pika v2.2, routed through the Fal.ai queue API.

Base URL: `http://localhost:8080/api/v1/pika`

> This endpoint uses the async job system. See [jobs.md](jobs.md) for polling and SSE streaming.

---

## Required Configuration

| Variable | Description | Required |
|----------|-------------|----------|
| `PIKA_API_KEY` | Fal.ai API key (not Pika directly) | Yes |

> The backend calls Fal.ai's queue API using model `fal-ai/pika-v2.2`. Your API key must be a Fal.ai key, not a Pika key.

---

## POST /pika/generate

**Method + Path:** `POST /api/v1/pika/generate`
**Description:** Submits a video generation request to Pika via Fal.ai and returns a job ID immediately. The backend queues the job on Fal.ai, polls until complete, then downloads and saves the video locally.

### Request Body

| Field | Type | Required | Valid Values | Default |
|-------|------|----------|--------------|---------|
| `prompt` | string | Yes | Any text description of the video | — |
| `duration` | int | No | Seconds (integer) | `3` |

### Response (202 Accepted)

```json
{
  "success": true,
  "job_id": "a1b2c3d4-...",
  "message": "Pika video generation job created"
}
```

### Example

```bash
curl -X POST http://localhost:8080/api/v1/pika/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A serene lake with fish jumping out of the water at sunrise, cinematic slow motion",
    "duration": 5
  }'
```

---

## Full Workflow

### Step 1 — Submit the request

```bash
curl -X POST http://localhost:8080/api/v1/pika/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A serene lake with fish jumping out of the water at sunrise, cinematic slow motion",
    "duration": 3
  }'
```

Response:
```json
{
  "success": true,
  "job_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "message": "Pika video generation job created"
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
    "provider": "pika",
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
  "provider": "pika",
  "result_url": "./outputs/pika/video-abc123.mp4",
  "result": {
    "video_url": "https://fal.media/files/...",
    "video_path": "./outputs/pika/video-abc123.mp4",
    "file_name": "video-abc123.mp4"
  }
}
```

### Step 4 (Optional) — Stream real-time updates via SSE

```bash
curl -N http://localhost:8080/api/v1/jobs/a1b2c3d4-e5f6-7890-abcd-ef1234567890/sse
```

---

## Fal.ai Queue Status Values

The backend maps these internal Fal.ai statuses to job progress:

| Fal.ai Status | Meaning |
|---------------|---------|
| `IN_QUEUE` | Waiting for a worker on Fal.ai |
| `IN_PROGRESS` | Video is being generated |
| `COMPLETED` | Video is ready to download |
| `FAILED` | Generation failed |

---

## Notes

- `duration` is an **integer** (e.g. `3`), unlike Kling which uses a string.
- Default duration is **3 seconds** — omit the field if that works for you.
- The API key in `.env` must be a **Fal.ai key**, not a Pika key. Pika is accessed as a model hosted on Fal.ai (`fal-ai/pika-v2.2`).
- Generated videos are saved locally to `OUTPUT_DIR/pika/` (defaults to `./outputs/pika/`).
- The `video_url` from Fal.ai may expire — always use `video_path` for persistent access.
- Pika is simpler than Kling but has fewer controls (no aspect ratio, mode, or cfg_scale options).
