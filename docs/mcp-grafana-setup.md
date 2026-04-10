# MCP Grafana Setup

This guide explains how to set up [mcp-grafana](https://github.com/grafana/mcp-grafana), the official Grafana MCP server that provides Claude Code with direct access to Loki logs, Prometheus metrics, Tempo traces, dashboards, and alerts.

## Prerequisites

- Observability stack running (`make obs-up` or `make up`)
- [uv](https://docs.astral.sh/uv/getting-started/installation/) installed (`brew install uv` on macOS)

## 1. Create a Grafana Service Account

Open a terminal and run the following commands. The observability stack must be running on `localhost:3000`.

### Create the service account

```bash
curl -s -u admin:changeme -X POST http://localhost:3000/api/serviceaccounts \
  -H "Content-Type: application/json" \
  -d '{"name":"mcp-grafana","role":"Editor","isDisabled":false}'
```

Expected response:

```json
{
  "id": 2,
  "name": "mcp-grafana",
  "login": "sa-1-mcp-grafana",
  "role": "Editor"
}
```

> **Note:** The `Editor` role provides access to dashboards, datasources, and queries. Use `Viewer` if you only need read access.

### Generate an API token

Replace `<SERVICE_ACCOUNT_ID>` with the `id` from the previous response:

```bash
curl -s -u admin:changeme -X POST http://localhost:3000/api/serviceaccounts/<SERVICE_ACCOUNT_ID>/tokens \
  -H "Content-Type: application/json" \
  -d '{"name":"mcp-grafana-token"}'
```

Expected response:

```json
{
  "id": 1,
  "name": "mcp-grafana-token",
  "key": "glsa_xxxxxxxxxxxxxxxxxxxxxxxxxxxx_xxxxxxxx"
}
```

**Save the `key` value** — it cannot be retrieved again after creation.

### Verify the token works

```bash
curl -s -H "Authorization: Bearer <YOUR_TOKEN>" http://localhost:3000/api/datasources | python3 -m json.tool
```

You should see Loki, Prometheus, and Tempo datasources in the response.

## 2. Configure the MCP Server

Add the following entry to `.mcp.json` in the project root:

```json
{
  "mcpServers": {
    "mcp-grafana": {
      "command": "uvx",
      "args": [
        "mcp-grafana",
        "--enable-tool", "search_dashboards",
        "--enable-tool", "get_dashboard_by_uid",
        "--enable-tool", "list_datasources",
        "--enable-tool", "query_prometheus",
        "--enable-tool", "query_loki_logs",
        "--enable-tool", "list_alert_rules",
        "--enable-tool", "searchlogs"
      ],
      "env": {
        "GRAFANA_URL": "http://localhost:3000",
        "GRAFANA_SERVICE_ACCOUNT_TOKEN": "<YOUR_TOKEN>"
      }
    }
  }
}
```

Replace `<YOUR_TOKEN>` with the key from step 1.

## 3. Enabled Tools

| Tool | Purpose |
|------|---------|
| `search_dashboards` | Find dashboards by name or tag |
| `get_dashboard_by_uid` | Retrieve full dashboard definition |
| `list_datasources` | List all configured datasources |
| `query_prometheus` | Execute PromQL queries for metrics |
| `query_loki_logs` | Execute LogQL queries for logs |
| `list_alert_rules` | View configured alerting rules |
| `searchlogs` | Search across all log sources |

To enable additional tools, see the [mcp-grafana documentation](https://github.com/grafana/mcp-grafana#tools).

## 4. Restart Claude Code

After modifying `.mcp.json`, restart Claude Code (or start a new conversation) for the MCP server to load.

## Troubleshooting

### Token expired or revoked

Create a new token using step 1. Tokens persist as long as Grafana's database volume exists. If you ran `make obs-reset`, you need to recreate the service account and token.

### Connection refused

Make sure the observability stack is running:

```bash
make obs-up
```

Verify Grafana is accessible:

```bash
curl -s http://localhost:3000/api/health
```

### Permission denied

The service account needs the `Editor` role. Check the account:

```bash
curl -s -u admin:changeme http://localhost:3000/api/serviceaccounts/search | python3 -m json.tool
```
