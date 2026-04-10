# Jobs

Shared system for tracking async operations across all providers.
Every endpoint that generates content (DALL-E, ElevenLabs, Luma, Kling, Pika) returns a `job_id` — use these endpoints to track progress and retrieve results.

Base URL: `http://localhost:8080/api/v1/jobs`

---

## Job Status Values

| Status | Description |
|--------|-------------|
| `pending` | Job created, waiting to start |
| `processing` | Job is running (check `progress` field, 0–100) |
| `completed` | Job finished successfully — result is available |
| `failed` | Job encountered an error — check `error` field |

---

## GET /jobs/:job_id/status

**Method + Path:** `GET /api/v1/jobs/:job_id/status`
**Description:** Returns the current status and progress of a job.

### Response (200 OK)

```json
{
  "success": true,
  "job": {
    "id": "a1b2c3d4-...",
    "provider": "dalle",
    "status": "processing",
    "progress": 50,
    "created_at": "2026-02-23T10:00:00Z",
    "updated_at": "2026-02-23T10:00:05Z"
  }
}
```

### Response (404 Not Found)

```json
{
  "success": false,
  "error": "Job not found"
}
```

### Example

```bash
curl http://localhost:8080/api/v1/jobs/<job_id>/status
```

---

## GET /jobs/:job_id/result

**Method + Path:** `GET /api/v1/jobs/:job_id/result`
**Description:** Returns the output of a completed job. Returns an error if the job is not yet finished.

### Response (200 OK — completed job)

```json
{
  "success": true,
  "job_id": "a1b2c3d4-...",
  "provider": "dalle",
  "result_url": "./outputs/dalle/image-abc123.png",
  "result": { ... }
}
```

The shape of `result` depends on the provider. See each provider's doc for the exact fields.

### Response (400 Bad Request — job not finished)

```json
{
  "success": false,
  "error": "Job is not completed (status: processing)"
}
```

### Example

```bash
curl http://localhost:8080/api/v1/jobs/<job_id>/result
```

---

## GET /jobs/:job_id/sse

**Method + Path:** `GET /api/v1/jobs/:job_id/sse`
**Description:** Opens a Server-Sent Events stream that pushes updates in real time until the job reaches a terminal state (`completed` or `failed`). Automatically closes the connection when done.

### SSE Event Types

| Event | When |
|-------|------|
| `status` | Sent immediately on connect with current state |
| `status` | Sent on each status/progress change |
| `timeout` | Sent after 10 minutes if still running |
| `error` | Sent if an internal error occurs |

### Example

```bash
# -N disables buffering so events print as they arrive
curl -N http://localhost:8080/api/v1/jobs/<job_id>/sse
```

Sample output:
```
event: status
data: {"id":"a1b2c3d4-...","provider":"dalle","status":"processing","progress":50}

event: status
data: {"id":"a1b2c3d4-...","provider":"dalle","status":"completed","progress":100}
```

---

## GET /jobs

**Method + Path:** `GET /api/v1/jobs`
**Description:** Returns a list of jobs. Supports optional filtering by provider or status.

### Query Parameters

| Param | Type | Required | Valid Values |
|-------|------|----------|--------------|
| `provider` | string | No | `dalle`, `elevenlabs`, `luma`, `kling`, `pika` |
| `status` | string | No | `pending`, `processing`, `completed`, `failed` |

### Response (200 OK)

```json
{
  "success": true,
  "count": 3,
  "jobs": [
    {
      "id": "a1b2c3d4-...",
      "provider": "dalle",
      "status": "completed",
      "progress": 100
    }
  ]
}
```

### Examples

```bash
# All jobs
curl http://localhost:8080/api/v1/jobs

# Filter by provider
curl "http://localhost:8080/api/v1/jobs?provider=dalle"

# Filter by status
curl "http://localhost:8080/api/v1/jobs?status=completed"

# Filter by both
curl "http://localhost:8080/api/v1/jobs?provider=elevenlabs&status=processing"
```

---

## Notes

- Jobs expire after `JOB_EXPIRATION_HOURS` (default: 24h). After expiry, `/status` returns 404.
- Default list limit is 100 jobs. Pagination is not yet supported.
- SSE stream has a hard timeout of 10 minutes — re-connect if you need to wait longer.
- Always check `/status` before calling `/result` to avoid a 400 error.
