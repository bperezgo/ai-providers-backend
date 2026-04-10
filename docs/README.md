# Backend API Docs

This folder documents all HTTP endpoints in the backend.

Before writing or updating any documentation, read [RULES.md](RULES.md) for the required structure and formatting standards.

---

## Documented Endpoints

| File | Provider | Endpoints |
|------|----------|-----------|
| [health.md](api/health.md) | — | `GET /health` |
| [jobs.md](api/jobs.md) | All providers | `GET /jobs`, `GET /jobs/:id/status`, `GET /jobs/:id/result`, `GET /jobs/:id/sse` |
| [dalle.md](api/dalle.md) | OpenAI / DALL-E | `POST /dalle/generate` |
| [elevenlabs.md](api/elevenlabs.md) | ElevenLabs | `POST /elevenlabs/tts`, `GET /elevenlabs/voices` |
| [kling.md](api/kling.md) | Kling AI | `POST /kling/generate` |
| [pika.md](api/pika.md) | Pika via Fal.ai | `POST /pika/generate` |
| [nanobanana.md](api/nanobanana.md) | Nano Banana Pro via Fal.ai | `POST /nanobanana/generate` |

## Undocumented Endpoints (pending)

| Provider | Endpoints | Handler |
|----------|-----------|---------|
| Luma Labs | `POST /luma/generate` | `handlers/luma.go` |

---

## How the Async System Works

Most generation endpoints are async:

```
POST /api/v1/<provider>/generate
  → 202 Accepted { job_id }
  → GET /api/v1/jobs/:job_id/status   (poll)
  → GET /api/v1/jobs/:job_id/result   (when completed)
  → GET /api/v1/jobs/:job_id/sse      (optional real-time stream)
```

See [jobs.md](api/jobs.md) for the complete reference.

---

## Server Defaults

| Setting | Default |
|---------|---------|
| Port | `8080` |
| Base URL | `http://localhost:8080/api/v1` |
| Output dir | `./outputs` |
| Job expiry | 24 hours |
