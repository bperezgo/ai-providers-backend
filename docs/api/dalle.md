# DALL-E (OpenAI)

Generates images from text prompts using OpenAI's DALL-E 3 model.

Base URL: `http://localhost:8080/api/v1/dalle`

> This endpoint uses the async job system. See [jobs.md](jobs.md) for polling and SSE streaming.

---

## Required Configuration

| Variable | Description | Required |
|----------|-------------|----------|
| `OPENAI_API_KEY` | OpenAI API key | Yes |

---

## POST /dalle/generate

**Method + Path:** `POST /api/v1/dalle/generate`
**Description:** Submits an image generation request and returns a job ID immediately.

### Request Body

| Field | Type | Required | Valid Values | Default |
|-------|------|----------|--------------|---------|
| `prompt` | string | Yes | Any text description | — |
| `size` | string | No | `1024x1024`, `1792x1024`, `1024x1792` | `1024x1024` |
| `quality` | string | No | `standard`, `hd` | `standard` |
| `n` | int | No | `1` (DALL-E 3 only supports 1) | `1` |
| `style` | string | No | `vivid`, `natural` | `vivid` |

### Response (202 Accepted)

```json
{
  "success": true,
  "job_id": "a1b2c3d4-...",
  "message": "Image generation job created"
}
```

### Example

```bash
curl -X POST http://localhost:8080/api/v1/dalle/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A beautiful mountain lake at sunset with fishing boats",
    "size": "1024x1024",
    "quality": "standard",
    "n": 1,
    "style": "vivid"
  }'
```

---

## Full Workflow

### Step 1 — Submit the request

```bash
curl -X POST http://localhost:8080/api/v1/dalle/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A beautiful mountain lake at sunset with fishing boats",
    "size": "1024x1024",
    "quality": "hd",
    "style": "natural"
  }'
```

Response:
```json
{
  "success": true,
  "job_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "message": "Image generation job created"
}
```

### Step 2 — Poll for status

```bash
curl http://localhost:8080/api/v1/jobs/a1b2c3d4-e5f6-7890-abcd-ef1234567890/status
```

Response while processing:
```json
{
  "success": true,
  "job": {
    "id": "a1b2c3d4-...",
    "provider": "dalle",
    "status": "processing",
    "progress": 50
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
  "provider": "dalle",
  "result_url": "./outputs/dalle/image-abc123.png",
  "result": {
    "image_url": "https://oaidalleapiprodscus.blob.core.windows.net/...",
    "image_path": "./outputs/dalle/image-abc123.png",
    "revised_prompt": "A serene mountain lake at golden hour..."
  }
}
```

### Step 4 (Optional) — Stream real-time updates via SSE

```bash
curl -N http://localhost:8080/api/v1/jobs/a1b2c3d4-e5f6-7890-abcd-ef1234567890/sse
```

---

## Notes

- DALL-E 3 only supports `n=1`; setting a higher value will likely be rejected by OpenAI.
- `hd` quality costs more tokens and takes longer than `standard`.
- `vivid` style produces more dramatic/hyper-real images; `natural` is more photographic.
- Generated images are downloaded and saved locally to `OUTPUT_DIR/dalle/` (defaults to `./outputs/dalle/`).
- OpenAI returns a `revised_prompt` with safety-adjusted wording — always check this in the result.
- The `image_url` from OpenAI expires after ~1 hour; use `image_path` for persistent access.
