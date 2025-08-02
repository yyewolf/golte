# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git (needed for version info)
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-X 'golte/cmd.Version=$(git describe --tags --always --dirty 2>/dev/null || echo 'docker')' \
              -X 'golte/cmd.GitCommit=$(git rev-parse HEAD 2>/dev/null || echo 'unknown')' \
              -X 'golte/cmd.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
    -o golte

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/golte .

# Copy example config
COPY config.yaml.example /etc/golte/config.yaml.example

# Create non-root user
RUN adduser -D -s /bin/sh golte

# Switch to non-root user
USER golte

# Expose any ports if needed (none for this app)
# EXPOSE 8080

# Command to run
ENTRYPOINT ["./golte"]
