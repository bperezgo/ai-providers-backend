# Health

Checks the operational status of the server and its dependencies.

Available at both the root level and under the API prefix.

---

## GET /health

**Method + Path:** `GET /health` or `GET /api/v1/health`
**Description:** Returns the health status of the server and database connection.

### Response (200 OK — healthy)

```json
{
  "status": "healthy",
  "timestamp": "2026-02-23T10:00:00Z",
  "services": {
    "database": "healthy"
  }
}
```

### Response (503 Service Unavailable — unhealthy)

```json
{
  "status": "unhealthy",
  "timestamp": "2026-02-23T10:00:00Z",
  "services": {
    "database": "unhealthy: connection refused"
  }
}
```

### Examples

```bash
# Root level (no API prefix)
curl http://localhost:8080/health

# Under API prefix
curl http://localhost:8080/api/v1/health
```

---

## Notes

- Use this as your first check when the server is not behaving — it will tell you if the database is down.
- A 503 response means at least one service is unhealthy. Check `services` for details.
- Both paths are identical; prefer `/health` for simplicity in monitoring scripts.
