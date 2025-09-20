# syntax=docker/dockerfile:1.4

# Build on amd64 but cross-compile for ARMv6
FROM --platform=amd64 golang:1.24-bookworm AS builder

WORKDIR /app

# Install build dependencies for cgo + opus support
RUN dpkg --add-architecture armhf && \
    apt-get update && apt-get install -y --no-install-recommends \
    git pkg-config qemu-user-static \
    gcc-arm-linux-gnueabihf g++-arm-linux-gnueabihf \
    libopus-dev:armhf libopusfile-dev:armhf libasound2-dev:armhf && \
    rm -rf /var/lib/apt/lists/*

# Copy Go module files first
COPY go.mod go.sum ./

# Cache Go modules
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY . .

# Cross-compile for ARMv6 (arm32)
# - GOOS=linux: target OS
# - GOARCH=arm: ARM 32-bit
# - GOARM=6: target ARMv6 (e.g., Raspberry Pi Zero)
ENV GOOS=linux \
    GOARCH=arm \
    GOARM=6 \
    CGO_ENABLED=1 \
    CC=arm-linux-gnueabihf-gcc \
    PKG_CONFIG_LIBDIR=/usr/lib/arm-linux-gnueabihf/pkgconfig:/usr/share/pkgconfig \
    PKG_CONFIG_PATH=/usr/lib/arm-linux-gnueabihf/pkgconfig
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build  \
      -ldflags="-X 'golte/cmd.Version=$(git describe --tags --always --dirty 2>/dev/null || echo docker)' \
                -X 'golte/cmd.GitCommit=$(git rev-parse HEAD 2>/dev/null || echo unknown)' \
                -X 'golte/cmd.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" \
      -o golte

# Optional: you can define a runtime stage for the ARM binary if you want a smaller final image
# (use an ARM-compatible base image if running on ARM)
FROM --platform=linux/arm/v7 arm32v7/debian:bookworm-slim
COPY --from=builder /app/golte /usr/local/bin/golte
ENTRYPOINT ["golte"]
