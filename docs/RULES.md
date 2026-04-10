# Backend API Documentation Rules

Read this file before writing any endpoint documentation.
It defines the standard structure every doc must follow.

---

## File Naming

- One file per endpoint group (handler file = one doc file)
- Location: `backend/docs/api/<provider>.md`
- Use lowercase kebab-case: `dalle.md`, `elevenlabs.md`, `jobs.md`

---

## Required Sections (in this order)

### 1. Title + One-liner
```
# Provider Name

One sentence describing what this group of endpoints does.
```

### 2. Base URL
```
Base URL: http://localhost:8080/api/v1/<provider>
```

### 3. Required Configuration
List the env vars needed. Reference `.env` keys exactly.
```
| Variable | Description | Required |
```

### 4. Async Pattern (if applicable)
If the endpoint creates a job (async), add this note once at the top:
> This endpoint uses the async job system. See [jobs.md](jobs.md) for polling and SSE streaming.

### 5. Endpoints
One `##` section per HTTP endpoint. Format:

```
## Endpoint Name

**Method + Path:** `POST /api/v1/<provider>/action`
**Description:** What it does in one sentence.

### Request Body

| Field | Type | Required | Valid Values | Default |
|-------|------|----------|--------------|---------|

### Response (202 Accepted)

\`\`\`json
{
  "success": true,
  "job_id": "<uuid>",
  "message": "..."
}
\`\`\`

### Example

\`\`\`bash
curl ...
\`\`\`
```

### 6. Full Workflow (for async endpoints)
Show the complete happy path as numbered curl steps. Always include:
1. Submit → get `job_id`
2. Poll status
3. Get result
4. (Optional) SSE stream

### 7. Notes
Bullet list of gotchas, limits, or edge cases specific to this provider.

---

## Style Rules

- Use backtick blocks for all curl commands
- Always show `-H "Content-Type: application/json"` in POST curls
- Replace real API keys with `$OPENAI_API_KEY` style env var references
- Mark optional fields clearly in the table (`No` in Required column)
- Show the actual JSON response shape, not just field names
- For enum fields, list ALL valid values in the "Valid Values" column

---

## When to Update Docs

- New endpoint added → create or update the relevant `.md` file
- Request/response shape changes → update the doc in the same PR
- New env var required → update the configuration table

---

## Index

Keep `backend/docs/README.md` updated with every new documented endpoint.
