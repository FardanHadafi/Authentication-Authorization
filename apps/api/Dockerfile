# syntax=docker/dockerfile:1.7

# --- Build Stage ---
FROM golang:1.27-alpine AS builder

WORKDIR /build

# Install git and ca-certificates if dependencies require them
RUN apk add --no-cache ca-certificates git

# Cache Go modules layer
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY . .

# Build binary with compiler cache & stripped binary (-s -w) for minimal size
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /build/api ./cmd/main.go

# --- Minimal Runtime Stage ---
FROM alpine:3.21 AS runner

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=appuser:appgroup /build/api /app/api

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/api"]
