# MoodNAS demo image: the real moodd + web UI running against the simulated
# WD My Cloud EX4. Build from the repository root:
#   docker build -f build/Dockerfile -t moodnas .
#   docker run --rm -p 8080:8080 moodnas

FROM node:24-alpine AS ui
WORKDIR /ui
COPY ui/package.json ui/package-lock.json ./
RUN npm ci
COPY ui/ ./
RUN npm run check-i18n && npm run build

FROM golang:1.26-alpine AS go
# Some Docker Desktop/WSL2 hosts with many cores crash the Go compiler under
# full parallelism; a modest cap costs little.
ENV GOMAXPROCS=2 GOFLAGS=-p=2
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /moodd ./cmd/moodd

FROM alpine:3
RUN adduser -D -H -u 10001 moodnas && mkdir -p /data && chown moodnas /data
COPY --from=go /moodd /usr/bin/moodd
COPY --from=ui /ui/dist /usr/share/moodnas/ui
USER moodnas
EXPOSE 8080
VOLUME /data
ENTRYPOINT ["/usr/bin/moodd", "-simulate", "-demo", "-insecure-http", "-listen", ":8080", "-http", "", "-data", "/data", "-ui", "/usr/share/moodnas/ui"]
