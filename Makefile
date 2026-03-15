# moneyclaw — mormoneyOS Go runtime build
# Use GOWORK=off if this repo is outside your go.work.
# Version and commit are injected at build time from git.

BINARY  := bin/moneyclaw
MAIN    := ./cmd/moneyclaw
PKG     := github.com/morpheumlabs/mormoneyos-go/cmd
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "")
BUILD   := $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo "")
LDFLAGS := -ldflags "-X $(PKG).version=$(VERSION) -X $(PKG).buildTime=$(BUILD) -X $(PKG).commit=$(COMMIT)"
GOBIN   := $(shell go env GOPATH)/bin
CGO_ENABLED ?= 0

# Cross-build matrix (override from command line if needed).
# Examples:
#   make build-all TARGETS="darwin/arm64 linux/amd64"
#   make build-all VERSION=v1.2.3
TARGETS ?= darwin/arm64 darwin/amd64 linux/arm64 linux/amd64 windows/amd64

WEBUI_DIR   := dashos
WEBUI_DIST  := $(WEBUI_DIR)/dist
WEBUI_STATIC := internal/web/static

.PHONY: all build build-webui run clean clean-all install build-all test test-coverage build-webui

all: build install

# Build dashos UI and replace embedded static files.
# Uses base /static/ so asset URLs work with the Go server's /static/ route.
web:
	cd $(WEBUI_DIR) && bun run build:embed
	@find $(WEBUI_STATIC) -mindepth 1 -delete 2>/dev/null || true
	@mkdir -p $(WEBUI_STATIC)
	@cp -r $(WEBUI_DIST)/* $(WEBUI_STATIC)/

build:
	@mkdir -p bin
	GOWORK=off go build $(LDFLAGS) -o $(BINARY) $(MAIN)

run: build
	MONEYCLAW_DEV_BYPASS=1 $(BINARY) run

install: build
	@mkdir -p $(GOBIN)
	cp $(BINARY) $(GOBIN)/moneyclaw

# Build gzipped binaries for all TARGETS into ./bin
# Output naming: bin/moneyclaw_<version>_<os>_<arch>[.exe].gz
build-all:
	@mkdir -p bin
	@set -e; \
	for t in $(TARGETS); do \
		os="$${t%/*}"; \
		arch="$${t#*/}"; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		out="bin/moneyclaw_$(VERSION)_$${os}_$${arch}$${ext}"; \
		echo "building $$os/$$arch -> $$out"; \
		GOWORK=off CGO_ENABLED=$(CGO_ENABLED) GOOS="$$os" GOARCH="$$arch" go build $(LDFLAGS) -o "$$out" $(MAIN); \
		echo "gzipping $$out -> $$out.gz"; \
		gzip -c "$$out" > "$$out.gz" && rm -f "$$out"; \
	done

test:
	go test ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -f $(BINARY) coverage.out coverage.html

clean-all:
	rm -f bin/moneyclaw
	rm -f bin/moneyclaw_*
	rm -f bin/moneyclaw_*.gz
	rm -f coverage.out coverage.html
