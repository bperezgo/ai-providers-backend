---
name: loki-investigate
description: Investigate backend logs in Loki via Grafana MCP. Pre-loads observability schema (labels, structured metadata, datasource UIDs) to avoid discovery overhead. Use when user asks to check logs, investigate errors, correlate user activity, or debug backend issues. Triggers on "check logs", "investigate errors", "loki", "what happened in the backend", "check user errors".
---

# Loki Investigation Skill — Grafana MCP

You are investigating backend logs stored in Loki using the Grafana MCP tools. This skill pre-loads your observability schema so you can skip discovery queries and go straight to useful LogQL queries.

## STEP 0: Load Observability Schema

**DO NOT run discovery queries** (`list_datasources`, `list_loki_label_names`, `list_loki_label_values`) unless the schema below doesn't match what you find. The schema is derived from the Alloy pipeline config at `backend/observability/alloy/config.alloy`.

### Datasources

| System | Type | UID | Purpose |
|--------|------|-----|---------|
| Loki | `loki` | `P8E80F9AEF21F6940` | Log aggregation (S3-backed) |
| Tempo | `tempo` | *(discover if needed)* | Distributed tracing |

### Loki Label Schema

**Stream labels** (low cardinality — usable in stream selectors and `by()` aggregations):

| Label | Values | Notes |
|-------|--------|-------|
| `service` | `ai-providers-backend` | Always use this as the base selector |
| `level` | `info`, `warn`, `error` | Log level from Zap logger |
| `component` | `http` | Currently only HTTP middleware logs |
| `container` | `/ai-backend-server`, `/ai-backend-loki`, etc. | Docker container name |

**Structured metadata** (high cardinality — indexed for fast lookups but CANNOT be used in `by()` aggregations or stream selectors):

| Field | Type | Notes |
|-------|------|-------|
| `user_id` | UUID | From `X-User-ID` header, set by middleware |
| `session_id` | string | From `X-Session-ID` header |
| `request_id` | UUID | Generated per-request by middleware |
| `trace_id` | hex string | OpenTelemetry trace ID, correlates with Tempo |

### Critical Rules

1. **Base selector**: Always start with `{service="ai-providers-backend"}` — never guess job labels
2. **Structured metadata is NOT a stream label**: You CANNOT do `{user_id="..."}` or `sum by (user_id)`. Instead:
   - Filter by content: `{service="ai-providers-backend"} |= "user-id-prefix"`
   - Parse then filter: `{service="ai-providers-backend"} | json | user_id = "full-uuid"`
3. **Log format**: Zap JSON — all fields accessible via `| json` parser
4. **Error levels**: Status >= 400 logs at `error` level with full validation messages in `errors` field. Status < 400 logs at `info`.

## STEP 1: Understand the Request

Ask yourself:
- Is the user asking about **specific users**? → Filter by user_id
- Is the user asking about **error patterns**? → Filter by level="error"
- Is the user asking about **a specific endpoint**? → Filter by path after JSON parse
- Is the user asking about **a time range**? → Set startRfc3339/endRfc3339
- Is the user asking about **traces**? → Extract trace_id, correlate with Tempo

## STEP 2: Query Loki

### Common Query Patterns

**All errors (start here):**
```logql
{service="ai-providers-backend", level="error"}
```

**Errors for a specific user (by UUID prefix):**
```logql
{service="ai-providers-backend", level="error"} |= "1eaad670"
```

**Formatted error summary:**
```logql
{service="ai-providers-backend", level="error"} | json | line_format "{{.user_id}} {{.status}} {{.method}} {{.path}} {{.errors}}"
```

**All activity for a specific user (all levels):**
```logql
{service="ai-providers-backend"} |= "1eaad670"
```

**Errors on a specific endpoint:**
```logql
{service="ai-providers-backend", level="error"} | json | path = "/api/v1/nanobanana/generate"
```

**Count errors by path (metric query):**
```logql
sum by (path) (count_over_time({service="ai-providers-backend", level="error"} | json [1h]))
```

**High latency requests (> 100ms):**
```logql
{service="ai-providers-backend"} | json | latency > 0.1
```

### Query Parameters

- **datasourceUid**: `P8E80F9AEF21F6940`
- **limit**: Start with 20-50 for investigation, use 100 for comprehensive pulls
- **direction**: `backward` (newest first, default) for recent issues
- **startRfc3339**: Always set to narrow the time window (e.g., last hour)
- **queryType**: Use `instant` for metric queries, `range` (default) for log queries

## STEP 3: Analyze & Report

When presenting findings:

1. **Group errors by category** (status code + endpoint + error message)
2. **Correlate with users** if user_id is present
3. **Note trace_ids** for errors that need deeper investigation via Tempo
4. **Highlight patterns** — is one user causing all errors? Is one endpoint failing?
5. **Include sample LogQL queries** in the report so the user can re-run them in Grafana

## STEP 4: Save Context (if requested)

If the user asks to save the investigation, append findings to `backend/tmp-stress-test-context.md` or create a new file in the appropriate location.

## Log Source Reference

The log pipeline is defined in `backend/observability/alloy/config.alloy`:
- **Collection**: Docker container logs via `loki.source.docker`
- **Processing**: JSON parsing → stream labels (low cardinality) → structured metadata (high cardinality)
- **Storage**: Loki at `http://loki:3100`
- **Traces**: OTLP HTTP at `0.0.0.0:4318` → Tempo at `http://tempo:4318`

## API Endpoints (for path filtering reference)

| Method | Path | Service |
|--------|------|---------|
| POST | `/api/v1/nanobanana/generate` | NanoBanana image generation |
| POST | `/api/v1/sounds/tts` | ElevenLabs text-to-speech |
| POST | `/api/v1/sounds/sound-effects` | ElevenLabs sound effects |
| POST | `/api/v1/sounds/music` | ElevenLabs music generation |
| GET | `/api/v1/jobs/:id/status` | Job status polling |
| GET | `/api/v1/health` | Health check |
