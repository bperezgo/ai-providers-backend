# AI Providers Backend

A Go backend that wraps multiple AI providers (image generation, TTS, music, sound effects) behind a unified async API — with a full observability stack for experimenting with **cold observability** using Loki + S3 + Claude MCP.

> **Companion to the blog post:** [Cold Observability: Using Loki + MinIO + Claude MCP for Cheap, Full-Story Log Investigation](https://www.linkedin.com/feed/update/urn:li:activity:7438642310336311296/?originTrackingId=%2BZ0Na8OHjkEVU7ZlNcW40Q%3D%3D)
>
> This repository contains the complete stack referenced in the article. Follow the steps below to reproduce the setup on your machine.

---

## What's Inside

| Component | Purpose |
|-----------|---------|
| **Go HTTP server** (Gin) | Async job-based API for AI providers |
| **Structured logging** (Zap) | JSON logs with entity IDs + OpenTelemetry trace context |
| **Loki** | Log aggregation with S3 backend |
| **MinIO** | S3-compatible object storage (local substitute for AWS S3) |
| **Alloy** | Log collector — Docker log discovery, label extraction, structured metadata |
| **Tempo** | Distributed tracing (OTLP) |
| **Prometheus** | Metrics collection |
| **Grafana** | Visualization + MCP server integration |
| **Stress test CLI** | Generates mixed traffic across simulated users |

---

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- [Go 1.22+](https://go.dev/dl/)
- [Make](https://www.gnu.org/software/make/)
- (Optional) [Grafana MCP Server](https://github.com/grafana/mcp-grafana) — for AI-assisted log investigation

---

## Quick Start

### 1. Clone and configure

```bash
git clone https://github.com/bperezgo/ai-providers-backend.git
cd ai-providers-backend

cp .env.example .env
```

Edit `.env` and add your API keys. For testing the observability stack without API keys, skip this step and use **Option A** in step 4 — it loads `.env.mock` automatically.

### 2. Start everything

```bash
make up
```

This builds and starts the full stack in Docker: PostgreSQL, the Go backend, a mock AI provider server, and the complete observability pipeline (MinIO, Loki, Tempo, Alloy, Prometheus, Grafana, AlertManager).

By default, the backend uses `.env.mock` — no API keys needed.

Verify everything is running:

| Service | URL |
|---------|-----|
| Backend | http://localhost:8080 |
| Mock Server | http://localhost:9999 |
| Grafana | http://localhost:3000 (admin / changeme) |
| MinIO Console | http://localhost:9001 (loki / supersecret) |
| Loki | http://localhost:3100/ready |
| Prometheus | http://localhost:9090 |
| Alloy UI | http://localhost:12345 |

> **Using real providers?** Edit `.env`, replace `.env.mock` with `.env` in the backend's `env_file` in `docker-compose.yml`, and update the `*_BASE_URL` variables to point to the real provider endpoints.

### 3. Run the stress test

```bash
make stress-test
```

This sends 50 requests across 5 simulated users with a 20% error rate. You'll see output like:

```
=== Simulated Users ===
  User 1: 1eaad670-6fd2-466f-8150-e22a79993eb3
  User 2: 7385eb89-4c1f-49f5-8c36-3678b1d1bdd5
  ...
```

Customize the test:

```bash
# 100 requests, 10 concurrent workers, 10 users, 30% error rate
make stress-test STRESS_ARGS="-n 100 -c 10 -users 10 -error-rate 0.3"
```

### 4. Query logs in Grafana

Open Grafana at http://localhost:3000 → **Explore** → select **Loki** datasource.

**See all errors:**

```logql
{service="ai-providers-backend", level="error"} | json
```

**See a specific user's activity:**

```logql
{service="ai-providers-backend"} |= "1eaad670" | json
```

**Filter by structured metadata:**

```logql
{service="ai-providers-backend"} | json | user_id = "1eaad670-6fd2-466f-8150-e22a79993eb3"
```

---

## Architecture: How the Observability Stack Works

```
┌─────────────┐     JSON logs      ┌─────────┐     push      ┌──────┐     chunks      ┌───────┐
│  Go Backend │ ──── stdout ─────▸ │  Alloy  │ ────────────▸ │ Loki │ ──────────────▸ │ MinIO │
│  (Zap)      │                    │         │               │      │                 │ (S3)  │
└──────┬──────┘                    └─────────┘               └──────┘                 └───────┘
       │ OTLP                          │
       │                               │ OTLP relay
       ▼                               ▼
   ┌───────┐                      ┌─────────┐
   │ Tempo │ ◂────────────────────│  Alloy  │
   └───────┘                      └─────────┘
       │
       ▼
   ┌───────────┐        ┌────────────┐
   │ Prometheus│ ◂──────│  Grafana   │──── MCP Server ──── Claude
   └───────────┘        └────────────┘
```

### The Label vs Structured Metadata Split

This is the key design decision. Alloy processes every log line and splits fields into two categories:

| Category | Fields | Cardinality | Usage in LogQL |
|----------|--------|-------------|----------------|
| **Stream labels** | `level`, `service`, `component` | Low (few values) | `{service="ai-providers-backend", level="error"}` |
| **Structured metadata** | `user_id`, `trace_id`, `session_id`, `request_id` | High (unique per request) | `\| json \| user_id = "..."` |

Why? Using high-cardinality fields as stream labels creates millions of streams and kills Loki's performance. Structured metadata keeps them indexed without the cardinality cost.

See [observability/alloy/config.alloy](observability/alloy/config.alloy) for the full pipeline.

---

## Using Claude MCP for Log Investigation

The [Grafana MCP Server](https://github.com/grafana/mcp-grafana) lets Claude query Loki programmatically. The key insight from the blog post: **giving Claude schema context upfront eliminates discovery overhead**.

### Setup: Grafana MCP Server

#### 1. Generate a Grafana Service Account Token

With Grafana running locally (`make up` or `make obs-up`):

1. Open Grafana at http://localhost:3000 and log in (`admin` / `changeme`).
2. Go to **Administration → Service accounts** (left sidebar → gear icon → Service accounts).
3. Click **Add service account**.
4. Give it a name (e.g., `mcp-server`), set the role to **Admin** (needed for datasource and log queries), and click **Create**.
5. On the service account page, click **Add service account token**.
6. Give the token a name (e.g., `claude-code`), then click **Generate token**.
7. Copy the token — it starts with `glsa_`. You won't be able to see it again.

#### 2. Configure `.mcp.json`

Add the following to your `.mcp.json` file (create it in the project root if it doesn't exist):

```json
{
  "mcpServers": {
    "mcp-grafana": {
      "command": "uvx",
      "args": [
        "mcp-grafana",
        "-enabled-tools",
        "search,dashboard,datasource,prometheus,loki,alerting,searchlogs"
      ],
      "env": {
        "GRAFANA_URL": "http://localhost:3000",
        "GRAFANA_SERVICE_ACCOUNT_TOKEN": "<your-glsa-token-here>"
      }
    }
  }
}
```

Replace `<your-glsa-token-here>` with the token you generated in the previous step.

> **Note:** `uvx` requires [uv](https://docs.astral.sh/uv/) to be installed. It will automatically download and run `mcp-grafana` without a manual install.

### Without context: 9 MCP calls, 2 failures, ~5 minutes

Claude has to discover datasources, labels, and learn what's a stream label vs structured metadata through trial and error.

### With a skill file: 2 MCP calls, 0 failures, seconds

A skill file pre-loads:
- Datasource UID (`P8E80F9AEF21F6940`)
- Stream labels and their values
- Structured metadata fields and how to filter them
- Critical rules (never use structured metadata in `by()` or selectors)

See the blog post for the full comparison.

### Example prompts to try

After running `make stress-test`, open Claude Code in this project and try these prompts to see the skill in action:

| Prompt | What it tests |
|--------|---------------|
| "Check the logs for any errors in the last hour" | Basic error lookup — should go straight to `query_loki_logs` |
| "What errors did user `1eaad670-...` hit in the last 30 minutes?" | User-scoped filtering via structured metadata |
| "Which endpoint has the most errors? Show me a breakdown by path" | Metric query with `count_over_time` + `sum by` |
| "Find requests that took longer than 200ms in the last hour" | Latency filtering via JSON parsed fields |
| "Investigate the backend — any error patterns? Group by endpoint and user, give me trace IDs for Tempo" | Full investigation: multiple queries, correlation, trace handoff |

> **Tip:** The best end-to-end test is: run `make stress-test`, then ask Claude to summarize what happened. With the skill file, it should resolve in 2 MCP calls with zero failures.

---

## Project Structure

```
.
├── cmd/
│   ├── server/          # Main HTTP server entry point
│   ├── stresstest/      # Stress test CLI
│   ├── mockserver/      # Mock AI provider server
│   └── mcp-remote/      # MCP server (HTTP transport)
├── internal/
│   ├── handlers/        # HTTP handlers per provider
│   ├── providers/       # AI provider clients (ElevenLabs, Luma, Kling, etc.)
│   ├── middleware/      # Logger, CORS, OIDC
│   ├── jobs/            # Async job manager
│   ├── database/        # PostgreSQL models
│   └── mockserver/      # Mock provider implementations
├── observability/
│   ├── alloy/           # Log collection pipeline config
│   ├── loki/            # Loki config (S3 backend)
│   ├── tempo/           # Distributed tracing config
│   ├── prometheus/      # Metrics + alert rules
│   ├── alertmanager/    # Alert routing
│   └── grafana/         # Dashboard + datasource provisioning
├── migrations/          # PostgreSQL migrations
├── docs/                # API endpoint documentation
├── docker-compose.yml   # Full stack
├── Makefile             # All commands
└── .env.example         # Configuration template
```

---

## Make Commands Reference

### Build

| Command | Description |
|---------|-------------|
| `make build` | Build HTTP server → `bin/server` |
| `make build-mock` | Build mock server → `bin/mockserver` |
| `make build-stresstest` | Build stress test → `bin/stresstest` |

### Run

| Command | Description |
|---------|-------------|
| `make up` | Start full stack in Docker (backend + mock + DB + observability) |
| `make down` | Stop all services |
| `make run` | Start server locally (real providers) |
| `make run-mock` | Start mock server locally on :9999 |
| `make run-with-mock` | Start server locally pointing to mock providers |
| `make stress-test` | Run stress test (default: 50 req, 5 workers) |

### Database

| Command | Description |
|---------|-------------|
| `make db-up` | Start PostgreSQL only |
| `make db-down` | Stop PostgreSQL |
| `make db-reset` | Drop and recreate |

### Observability

| Command | Description |
|---------|-------------|
| `make obs-up` | Start observability stack only (no backend) |
| `make obs-down` | Stop observability stack |
| `make obs-logs` | Tail observability container logs |
| `make obs-reset` | Drop volumes and restart |

### Testing

| Command | Description |
|---------|-------------|
| `make test` | Run all tests |
| `make test-coverage` | Generate coverage report |

---

## Key Files for the Blog Post

If you're following along with the article, these are the files referenced in each step:

| Blog Step | File |
|-----------|------|
| Step 1: Infrastructure | [docker-compose.yml](docker-compose.yml) |
| Step 2: Loki config | [observability/loki/loki-config.yml](observability/loki/loki-config.yml) |
| Step 3: Alloy pipeline | [observability/alloy/config.alloy](observability/alloy/config.alloy) |
| Step 4: Go logger | [internal/middleware/logger.go](internal/middleware/logger.go) |
| Step 5: Stress test | [cmd/stresstest/main.go](cmd/stresstest/main.go) |
| Step 6: Grafana MCP | [Grafana MCP Server](https://github.com/grafana/mcp-grafana) (external) |

---

## License

MIT
