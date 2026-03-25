package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	transport := flag.String("transport", "stdio", "transport to use: stdio or http")
	addr := flag.String("addr", ":8080", "HTTP listen address (only used with --transport http)")
	flag.Parse()

	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	srv, err := BuildServer(cfg)
	if err != nil {
		log.Fatalf("build server: %v", err)
	}

	switch *transport {
	case "stdio":
		if err := srv.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			log.Fatal(err)
		}
	case "http":
		handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
			return srv
		}, nil)
		http.Handle("/mcp", handler)
		log.Printf("listening on %s", *addr)
		log.Fatal(http.ListenAndServe(*addr, nil))
	default:
		log.Fatalf("unknown transport %q: must be stdio or http", *transport)
	}
}
