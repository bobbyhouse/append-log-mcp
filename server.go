package main

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// appendSchema is the input schema for the append tool.
var appendSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"data": map[string]any{
			"description": "JSON payload to log (any valid JSON value)",
		},
	},
	"required":             []string{"data"},
	"additionalProperties": false,
}

// querySchema is the input schema for the query tool.
var querySchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"since_days": map[string]any{
			"type":        "integer",
			"description": "Only return entries logged within the last N days; omit for all entries",
		},
	},
	"additionalProperties": false,
}

// BuildServer creates the MCP server and registers exactly the tools listed in cfg.Tools.
// Returns an error if the tools list is empty or contains an unknown tool name.
func BuildServer(cfg Config) (*mcp.Server, error) {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "append-log-mcp",
		Version: "v1.0.0",
	}, nil)

	h := &toolHandlers{logFile: cfg.LogFile}

	for _, name := range cfg.Tools {
		switch name {
		case "append":
			mcp.AddTool(srv, &mcp.Tool{
				Name:        "append",
				Description: "Append a JSON entry to the log. The server adds a logged_at timestamp.",
				InputSchema: appendSchema,
			}, h.appendEntry)
		case "query":
			mcp.AddTool(srv, &mcp.Tool{
				Name:        "query",
				Description: "Return logged entries, optionally filtered to the last N days.",
				InputSchema: querySchema,
			}, h.queryEntries)
		default:
			return nil, fmt.Errorf("unknown tool %q; valid tools: append, query", name)
		}
	}

	return srv, nil
}
