#syntax=docker/dockerfile:1

FROM dhi.io/golang:1.26-alpine3.22-dev AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o append-log-mcp .

FROM dhi.io/alpine-base:3.23
COPY --from=builder /app/append-log-mcp /append-log-mcp
LABEL io.modelcontextprotocol.server.name="io.github.bobbyhouse/append-log-mcp"

# Create /data owned by nonroot before VOLUME declaration.
# chown after VOLUME is a no-op in Docker, so this must come first.
USER 0
RUN mkdir -p /data && chown nonroot:nonroot /data
USER nonroot

VOLUME ["/data"]
ENTRYPOINT ["/append-log-mcp"]
CMD ["--transport", "stdio"]
