#!/bin/sh
# Run a Go command in the official Go container.
# Concurrency is capped: on some Docker Desktop/WSL2 hosts with many cores,
# the Go compiler crashes randomly under full parallelism.
set -e
ROOT=$(cd "$(dirname "$0")/.." && pwd -W 2>/dev/null || pwd)
exec env MSYS_NO_PATHCONV=1 docker run --rm --cpus 4 -e GOMAXPROCS=2 -e GOFLAGS=-p=1 \
  -e GOOS -e GOARCH -e GOARM -e CGO_ENABLED=0 \
  -v "$ROOT:/src" -v moodnas-gomod:/go/pkg/mod -v moodnas-gocache126:/root/.cache/go-build \
  -w /src golang:1.26 "$@"
