# append-log-mcp

A generic append-only log MCP server. Agents write timestamped JSON entries to a JSONL file and query them back by recency.

## Tools

| Tool | Description |
|---|---|
| `append` | Append a JSON entry to the log. The server adds a `logged_at` timestamp. |
| `query` | Return logged entries, optionally filtered to the last N days. |

## Configuration

| Variable | Required | Description |
|---|---|---|
| `APPEND_LOG_FILE` | No | Path to the JSONL file (default: `append-log.jsonl` in cwd) |
| `APPEND_LOG_TOOLS` | Yes | Comma-separated tools to expose: `append`, `query` |

## Running

### Binary (stdio)

```bash
task build  # outputs to build/append-log-mcp

APPEND_LOG_FILE=/tmp/log.jsonl \
APPEND_LOG_TOOLS=append,query \
./build/append-log-mcp --transport stdio
```

### Binary (HTTP)

```bash
APPEND_LOG_FILE=/tmp/log.jsonl \
APPEND_LOG_TOOLS=append,query \
./build/append-log-mcp --transport http --addr :8080
```

### Docker

The Dockerfile uses [Docker Hardened Images (DHI)](https://docs.docker.com/dhi/). Building requires Docker Hub authentication:

```bash
docker login
docker build -t append-log-mcp .
```

Run via stdio:

```bash
docker run -i --rm \
  -e APPEND_LOG_FILE=/data/log.jsonl \
  -e APPEND_LOG_TOOLS=append,query \
  -v append-log-data:/data \
  append-log-mcp
```

### Docker Compose

```bash
docker compose up
```

The compose file mounts a named volume at `/data` and sets both env vars.

## MCP client config (stdio)

```json
{
  "mcpServers": {
    "append-log": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-v", "append-log-data:/data",
        "append-log-mcp"
      ],
      "env": {
        "APPEND_LOG_FILE": "/data/log.jsonl",
        "APPEND_LOG_TOOLS": "append,query"
      }
    }
  }
}
```

## Log file format

Each line is a JSON object:

```json
{"data": {"passage_key": "2680:42"}, "logged_at": "2026-03-25T08:00:00Z"}
```

`data` accepts any valid JSON value. `logged_at` is always RFC3339 UTC.

## Development

```bash
task test          # go test ./...
task build         # outputs to build/append-log-mcp
go vet ./...
staticcheck ./...
```
