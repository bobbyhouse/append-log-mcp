package main

import (
	"testing"
)

// INVARIANT: APPEND_LOG_FILE is required; missing returns error.
func TestLoadConfigMissingLogFile(t *testing.T) {
	t.Setenv("APPEND_LOG_FILE", "")
	t.Setenv("APPEND_LOG_TOOLS", "append,query")

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error when APPEND_LOG_FILE is not set")
	}
}

// INVARIANT: APPEND_LOG_FILE sets LogFile.
func TestLoadConfigLogFile(t *testing.T) {
	t.Setenv("APPEND_LOG_FILE", "/data/log.jsonl")
	t.Setenv("APPEND_LOG_TOOLS", "append")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.LogFile != "/data/log.jsonl" {
		t.Errorf("LogFile = %q, want %q", cfg.LogFile, "/data/log.jsonl")
	}
}

// INVARIANT: APPEND_LOG_TOOLS is required; missing returns error.
func TestLoadConfigMissingTools(t *testing.T) {
	t.Setenv("APPEND_LOG_FILE", "/data/log.jsonl")
	t.Setenv("APPEND_LOG_TOOLS", "")

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error when APPEND_LOG_TOOLS is not set")
	}
}

// INVARIANT: APPEND_LOG_TOOLS is parsed as comma-separated list, trimming whitespace.
func TestLoadConfigToolsParsed(t *testing.T) {
	t.Setenv("APPEND_LOG_FILE", "/data/log.jsonl")
	t.Setenv("APPEND_LOG_TOOLS", "append, query")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	want := []string{"append", "query"}
	if len(cfg.Tools) != len(want) {
		t.Fatalf("Tools = %v, want %v", cfg.Tools, want)
	}
	for i, w := range want {
		if cfg.Tools[i] != w {
			t.Errorf("Tools[%d] = %q, want %q", i, cfg.Tools[i], w)
		}
	}
}

// INVARIANT: single tool in APPEND_LOG_TOOLS is valid.
func TestLoadConfigSingleTool(t *testing.T) {
	t.Setenv("APPEND_LOG_FILE", "/data/log.jsonl")
	t.Setenv("APPEND_LOG_TOOLS", "append")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Tools) != 1 || cfg.Tools[0] != "append" {
		t.Errorf("Tools = %v, want [append]", cfg.Tools)
	}
}
