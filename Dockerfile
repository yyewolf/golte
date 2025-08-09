# syntax=docker/dockerfile:1.4

FROM arm32v7/golang:1.24-bookworm AS builder

WORKDIR /app

# Required for version info
RUN apt-get update && apt-get install -y git pkg-config libopus-dev libopusfile-dev libasound2-dev

# Copy go mod files
COPY go.mod go.sum ./

# Mount mod cache and download dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source
COPY . .

# Mount both mod and build cache
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 GOOS=linux go build -x \
    -ldflags="-X 'golte/cmd.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'docker')' \
              -X 'golte/cmd.GitCommit=$(git rev-parse HEAD 2>/dev/null || echo 'unknown')' \
              -X 'golte/cmd.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
    -o golte

FROM scratch
COPY --from=builder /app/golte /
