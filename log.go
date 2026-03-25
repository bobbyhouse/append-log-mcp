package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Entry is a single record in the JSONL log file.
type Entry struct {
	Data     json.RawMessage `json:"data"`
	LoggedAt time.Time       `json:"logged_at"`
}

// AppendInput is the input for the append tool.
type AppendInput struct {
	Data json.RawMessage `json:"data"`
}

// QueryInput is the input for the query tool.
type QueryInput struct {
	// SinceDays, when non-nil, filters entries to those logged within the last N days.
	SinceDays *int `json:"since_days,omitempty"`
}

// toolHandlers holds the dependencies for all MCP tool handlers.
type toolHandlers struct {
	logFile string
	mu      sync.Mutex // guards all file I/O
}

// appendEntry implements the append tool.
// It acquires the mutex, opens the log file in append mode, and writes one JSON line.
func (h *toolHandlers) appendEntry(
	_ context.Context,
	_ *mcp.CallToolRequest,
	in AppendInput,
) (*mcp.CallToolResult, any, error) {
	if len(in.Data) == 0 {
		return errResult("data is required"), nil, nil
	}

	entry := Entry{
		Data:     in.Data,
		LoggedAt: time.Now().UTC(),
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return errResult("marshal entry: " + err.Error()), nil, nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	f, err := os.OpenFile(h.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errResult("open log file: " + err.Error()), nil, nil
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "%s\n", line); err != nil {
		return errResult("write log file: " + err.Error()), nil, nil
	}

	return textResult("logged"), nil, nil
}

// queryEntries implements the query tool.
// It reads the JSONL file and returns entries filtered by the optional since_days parameter.
func (h *toolHandlers) queryEntries(
	_ context.Context,
	_ *mcp.CallToolRequest,
	in QueryInput,
) (*mcp.CallToolResult, any, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	f, err := os.Open(h.logFile)
	if errors.Is(err, os.ErrNotExist) {
		return textResult("[]"), nil, nil
	}
	if err != nil {
		return errResult("open log file: " + err.Error()), nil, nil
	}
	defer f.Close()

	var cutoff time.Time
	if in.SinceDays != nil {
		cutoff = time.Now().UTC().AddDate(0, 0, -*in.SinceDays)
	}

	entries := []Entry{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1<<20), 1<<20) // 1 MiB max line
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue // skip malformed lines
		}
		if !cutoff.IsZero() && e.LoggedAt.Before(cutoff) {
			continue
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return errResult("read log file: " + err.Error()), nil, nil
	}

	data, _ := json.Marshal(entries)
	return textResult(string(data)), nil, nil
}

// --- result helpers ---

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

func errResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
