package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// contentText extracts the text from the first TextContent in a CallToolResult.
func contentText(r *mcp.CallToolResult) string {
	if len(r.Content) == 0 {
		return ""
	}
	if tc, ok := r.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

// newTestHandlers creates a toolHandlers writing to a temp file.
func newTestHandlers(t *testing.T) *toolHandlers {
	t.Helper()
	return &toolHandlers{logFile: filepath.Join(t.TempDir(), "log.jsonl")}
}

// --- append ---

// INVARIANT: a successful append writes exactly one newline-terminated JSON line.
func TestAppendWritesOneLine(t *testing.T) {
	h := newTestHandlers(t)

	result, _, err := h.appendEntry(context.Background(), nil, AppendInput{Data: json.RawMessage(`{"key":"value"}`)})
	if err != nil {
		t.Fatalf("returned Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("IsError=true: %s", contentText(result))
	}

	data, err := os.ReadFile(h.logFile)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1", len(lines))
	}
}

// INVARIANT: appended entry contains the original data and a logged_at timestamp.
func TestAppendEntryFields(t *testing.T) {
	h := newTestHandlers(t)
	before := time.Now().UTC().Truncate(time.Second)

	_, _, _ = h.appendEntry(context.Background(), nil, AppendInput{Data: json.RawMessage(`{"passage_key":"2680:42"}`)})

	data, _ := os.ReadFile(h.logFile)
	var e Entry
	if err := json.Unmarshal(data[:len(data)-1], &e); err != nil {
		t.Fatalf("unmarshal entry: %v", err)
	}
	if string(e.Data) != `{"passage_key":"2680:42"}` {
		t.Errorf("Data = %s, want %s", e.Data, `{"passage_key":"2680:42"}`)
	}
	if e.LoggedAt.Before(before) {
		t.Errorf("LoggedAt %v is before test start %v", e.LoggedAt, before)
	}
	// INVARIANT: logged_at is UTC.
	if e.LoggedAt.Location() != time.UTC {
		t.Errorf("LoggedAt is not UTC: %v", e.LoggedAt.Location())
	}
}

// INVARIANT: data is stored as raw JSON without re-encoding.
func TestAppendRawJSONPreserved(t *testing.T) {
	payloads := []string{
		`"hello"`,
		`42`,
		`null`,
		`[1,2,3]`,
		`{"nested":{"a":1}}`,
		`true`,
	}
	for _, payload := range payloads {
		t.Run(payload, func(t *testing.T) {
			h := newTestHandlers(t)
			_, _, _ = h.appendEntry(context.Background(), nil, AppendInput{Data: json.RawMessage(payload)})

			data, _ := os.ReadFile(h.logFile)
			var e Entry
			if err := json.Unmarshal(data[:len(data)-1], &e); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if string(e.Data) != payload {
				t.Errorf("Data = %s, want %s", e.Data, payload)
			}
		})
	}
}

// INVARIANT: append with nil/empty data returns IsError=true, not a Go error.
func TestAppendEmptyDataReturnsError(t *testing.T) {
	h := newTestHandlers(t)

	result, _, err := h.appendEntry(context.Background(), nil, AppendInput{Data: nil})
	if err != nil {
		t.Fatalf("returned Go error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError=true for nil data")
	}
}

// --- query ---

// INVARIANT: query on missing file returns empty array, not error.
func TestQueryMissingFileReturnsEmpty(t *testing.T) {
	h := &toolHandlers{logFile: filepath.Join(t.TempDir(), "nonexistent.jsonl")}

	result, _, err := h.queryEntries(context.Background(), nil, QueryInput{})
	if err != nil {
		t.Fatalf("returned Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("IsError=true: %s", contentText(result))
	}
	if contentText(result) != "[]" {
		t.Errorf("result = %q, want %q", contentText(result), "[]")
	}
}

// INVARIANT: query on empty file returns empty array.
func TestQueryEmptyFileReturnsEmpty(t *testing.T) {
	h := newTestHandlers(t)
	os.WriteFile(h.logFile, nil, 0644)

	result, _, err := h.queryEntries(context.Background(), nil, QueryInput{})
	if err != nil {
		t.Fatalf("returned Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("IsError=true: %s", contentText(result))
	}
	if contentText(result) != "[]" {
		t.Errorf("result = %q, want %q", contentText(result), "[]")
	}
}

// INVARIANT: query without since_days returns all entries.
func TestQueryReturnsAllEntries(t *testing.T) {
	h := newTestHandlers(t)

	for i := range 3 {
		_, _, _ = h.appendEntry(context.Background(), nil, AppendInput{Data: json.RawMessage(fmt.Sprintf(`{"i":%d}`, i))})
	}

	result, _, err := h.queryEntries(context.Background(), nil, QueryInput{})
	if err != nil {
		t.Fatalf("returned Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("IsError=true: %s", contentText(result))
	}
	var entries []Entry
	if err := json.Unmarshal([]byte(contentText(result)), &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("got %d entries, want 3", len(entries))
	}
}

// INVARIANT: query with since_days filters out entries older than N days.
func TestQuerySinceDaysFiltersOldEntries(t *testing.T) {
	h := newTestHandlers(t)

	// Write one old entry directly (40 days ago) and one recent entry via append.
	oldEntry := Entry{
		Data:     json.RawMessage(`{"old":true}`),
		LoggedAt: time.Now().UTC().AddDate(0, 0, -40),
	}
	oldLine, _ := json.Marshal(oldEntry)
	os.WriteFile(h.logFile, append(oldLine, '\n'), 0644)

	_, _, _ = h.appendEntry(context.Background(), nil, AppendInput{Data: json.RawMessage(`{"new":true}`)})

	n := 30
	result, _, err := h.queryEntries(context.Background(), nil, QueryInput{SinceDays: &n})
	if err != nil {
		t.Fatalf("returned Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("IsError=true: %s", contentText(result))
	}
	var entries []Entry
	if err := json.Unmarshal([]byte(contentText(result)), &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("got %d entries, want 1 (old entry should be filtered)", len(entries))
	}
	if string(entries[0].Data) != `{"new":true}` {
		t.Errorf("unexpected entry: %s", entries[0].Data)
	}
}

// INVARIANT: query without since_days returns both old and new entries.
func TestQueryNilSinceDaysReturnsAll(t *testing.T) {
	h := newTestHandlers(t)

	oldEntry := Entry{
		Data:     json.RawMessage(`{"old":true}`),
		LoggedAt: time.Now().UTC().AddDate(0, 0, -60),
	}
	oldLine, _ := json.Marshal(oldEntry)
	os.WriteFile(h.logFile, append(oldLine, '\n'), 0644)

	_, _, _ = h.appendEntry(context.Background(), nil, AppendInput{Data: json.RawMessage(`{"new":true}`)})

	result, _, err := h.queryEntries(context.Background(), nil, QueryInput{})
	if err != nil {
		t.Fatalf("returned Go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("IsError=true: %s", contentText(result))
	}
	var entries []Entry
	if err := json.Unmarshal([]byte(contentText(result)), &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2", len(entries))
	}
}

// INVARIANT: concurrent appends do not corrupt the file.
func TestConcurrentAppends(t *testing.T) {
	h := newTestHandlers(t)

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			_, _, _ = h.appendEntry(context.Background(), nil, AppendInput{
				Data: json.RawMessage(fmt.Sprintf(`{"i":%d}`, i)),
			})
		}(i)
	}
	wg.Wait()

	// Each line must be valid JSON.
	f, err := os.Open(h.logFile)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Errorf("line %d is not valid JSON: %v — line: %s", count+1, err, line)
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if count != n {
		t.Errorf("got %d lines, want %d", count, n)
	}
}

// --- fuzz ---

// FuzzAppendData verifies that any valid JSON payload can be appended and round-tripped.
func FuzzAppendData(f *testing.F) {
	f.Add([]byte(`"hello"`))
	f.Add([]byte(`42`))
	f.Add([]byte(`{"key":"value"}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[1,2,3]`))
	f.Add([]byte(`true`))

	f.Fuzz(func(t *testing.T, data []byte) {
		if !json.Valid(data) {
			return // only fuzz valid JSON
		}
		h := &toolHandlers{logFile: filepath.Join(t.TempDir(), "log.jsonl")}

		result, _, err := h.appendEntry(context.Background(), nil, AppendInput{Data: json.RawMessage(data)})
		if err != nil {
			t.Fatalf("returned Go error: %v", err)
		}
		if result.IsError {
			t.Fatalf("IsError=true: %s", contentText(result))
		}

		// Query back and verify round-trip.
		qResult, _, err := h.queryEntries(context.Background(), nil, QueryInput{})
		if err != nil {
			t.Fatalf("query returned Go error: %v", err)
		}
		if qResult.IsError {
			t.Fatalf("query IsError=true: %s", contentText(qResult))
		}
		var entries []Entry
		if err := json.Unmarshal([]byte(contentText(qResult)), &entries); err != nil {
			t.Fatalf("unmarshal query result: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
		// INVARIANT: data is preserved exactly.
		if string(entries[0].Data) != string(data) {
			t.Errorf("Data = %s, want %s", entries[0].Data, data)
		}
	})
}
