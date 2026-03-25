package main

import (
	"fmt"
	"os"
	"strings"
)

// Config holds the server configuration, read from environment variables:
//
//	APPEND_LOG_FILE    Path to the JSONL log file (default: append-log.jsonl in cwd)
//	APPEND_LOG_TOOLS   Comma-separated list of tools to expose (required)
type Config struct {
	LogFile string
	Tools   []string
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() (Config, error) {
	var cfg Config

	cfg.LogFile = os.Getenv("APPEND_LOG_FILE")
	if cfg.LogFile == "" {
		cfg.LogFile = "append-log.jsonl"
	}

	if v := os.Getenv("APPEND_LOG_TOOLS"); v != "" {
		for t := range strings.SplitSeq(v, ",") {
			if t := strings.TrimSpace(t); t != "" {
				cfg.Tools = append(cfg.Tools, t)
			}
		}
	}
	if len(cfg.Tools) == 0 {
		return Config{}, fmt.Errorf("APPEND_LOG_TOOLS is required; specify at least one tool")
	}

	return cfg, nil
}
