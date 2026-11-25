# MoodNAS developer tasks. Go runs inside the official container (see
# build/go.sh), so only Docker, Node and Python are needed on the host.
GO      := ./build/go.sh
PY      := python3 -I
FW      ?= My_Cloud_LT4A_2.13.108.bin

.PHONY: all test test-go test-py ui moodd-arm demo i18n firmware-info clean

all: test ui moodd-arm

test: test-go test-py i18n

test-go:
	$(GO) sh -c 'go vet ./... && go test ./...'

test-py:
	$(PY) -m unittest discover -s tools/wdfw/tests

i18n:
	cd ui && node scripts/check-i18n.mjs

ui:
	cd ui && npm ci && npm run build

# Static ARMv5TE binary for the EX4's Marvell Kirkwood 88F6282.
moodd-arm:
	mkdir -p dist
	GOOS=linux GOARCH=arm GOARM=5 $(GO) go build -trimpath -ldflags="-s -w" -o dist/moodd-linux-armv5 ./cmd/moodd

# Simulated EX4 + web UI on http://localhost:8080
demo:
	docker compose up --build

firmware-info:
	$(PY) tools/wdfw/wdfw.py info $(FW)

clean:
	rm -rf dist ui/dist
