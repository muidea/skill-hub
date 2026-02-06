# Skill Hub Makefile

.PHONY: build clean test install release release-all lint deps

# Version variables (can be overridden)
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS = -X 'skill-hub/internal/cli.version=$(VERSION)' \
          -X 'skill-hub/internal/cli.commit=$(COMMIT)' \
          -X 'skill-hub/internal/cli.date=$(DATE)'

# Build the binary
build:
	go build -ldflags="$(LDFLAGS)" -o bin/skill-hub ./cmd/skill-hub

# Clean build artifacts
clean:
	rm -f bin/skill-hub
	rm -rf dist/

# Run tests
test:
	go test ./...

# Install to /usr/local/bin
install: build
	sudo cp bin/skill-hub /usr/local/bin/

# Create release binaries for all platforms
release-all: clean
	@echo "Building release binaries for version $(VERSION)..."
	
	# Linux
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/skill-hub-linux-amd64 ./cmd/skill-hub
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/skill-hub-linux-arm64 ./cmd/skill-hub
	
	# macOS
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/skill-hub-darwin-amd64 ./cmd/skill-hub
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/skill-hub-darwin-arm64 ./cmd/skill-hub
	
	# Windows
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/skill-hub-windows-amd64.exe ./cmd/skill-hub
	GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/skill-hub-windows-arm64.exe ./cmd/skill-hub
	
	# Create checksums
	cd dist && sha256sum * > checksums.txt
	
	@echo "Release binaries created in dist/ directory"

# Create release binaries (backward compatibility)
release: release-all

# Run linting
lint:
	gofmt -d .
	go vet ./...

# Update dependencies
deps:
	go mod tidy
	go mod verify

# Help
help:
	@echo "Available targets:"
	@echo "  build       - Build binary for current platform"
	@echo "  release-all - Build release binaries for all platforms"
	@echo "  test        - Run tests"
	@echo "  lint        - Run linting checks"
	@echo "  install     - Install to /usr/local/bin"
	@echo "  clean       - Clean build artifacts"
	@echo "  deps        - Update dependencies"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION    - Version string (default: dev)"
	@echo "  COMMIT     - Git commit hash (default: auto-detected)"
	@echo "  DATE       - Build date (default: current UTC time)"