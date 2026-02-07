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

# Create release packages for all platforms (tar.gz + sha256)
release-all: clean
	@echo "Building release packages for version $(VERSION)..."
	
	# 创建dist目录
	mkdir -p dist
	
	# Linux amd64
	@echo "Building linux-amd64..."
	@mkdir -p dist/tmp-linux-amd64
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/tmp-linux-amd64/skill-hub ./cmd/skill-hub
	cp README.md dist/tmp-linux-amd64/
	cp LICENSE dist/tmp-linux-amd64/
	cd dist/tmp-linux-amd64 && tar -czf ../skill-hub-linux-amd64.tar.gz .
	cd dist && sha256sum skill-hub-linux-amd64.tar.gz > skill-hub-linux-amd64.sha256
	rm -rf dist/tmp-linux-amd64
	@echo "  Created: skill-hub-linux-amd64.tar.gz + .sha256"
	
	# Linux arm64
	@echo "Building linux-arm64..."
	@mkdir -p dist/tmp-linux-arm64
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/tmp-linux-arm64/skill-hub ./cmd/skill-hub
	cp README.md dist/tmp-linux-arm64/
	cp LICENSE dist/tmp-linux-arm64/
	cd dist/tmp-linux-arm64 && tar -czf ../skill-hub-linux-arm64.tar.gz .
	cd dist && sha256sum skill-hub-linux-arm64.tar.gz > skill-hub-linux-arm64.sha256
	rm -rf dist/tmp-linux-arm64
	@echo "  Created: skill-hub-linux-arm64.tar.gz + .sha256"
	
	# macOS amd64
	@echo "Building darwin-amd64..."
	@mkdir -p dist/tmp-darwin-amd64
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/tmp-darwin-amd64/skill-hub ./cmd/skill-hub
	cp README.md dist/tmp-darwin-amd64/
	cp LICENSE dist/tmp-darwin-amd64/
	cd dist/tmp-darwin-amd64 && tar -czf ../skill-hub-darwin-amd64.tar.gz .
	cd dist && sha256sum skill-hub-darwin-amd64.tar.gz > skill-hub-darwin-amd64.sha256
	rm -rf dist/tmp-darwin-amd64
	@echo "  Created: skill-hub-darwin-amd64.tar.gz + .sha256"
	
	# macOS arm64
	@echo "Building darwin-arm64..."
	@mkdir -p dist/tmp-darwin-arm64
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/tmp-darwin-arm64/skill-hub ./cmd/skill-hub
	cp README.md dist/tmp-darwin-arm64/
	cp LICENSE dist/tmp-darwin-arm64/
	cd dist/tmp-darwin-arm64 && tar -czf ../skill-hub-darwin-arm64.tar.gz .
	cd dist && sha256sum skill-hub-darwin-arm64.tar.gz > skill-hub-darwin-arm64.sha256
	rm -rf dist/tmp-darwin-arm64
	@echo "  Created: skill-hub-darwin-arm64.tar.gz + .sha256"
	
	# Windows amd64
	@echo "Building windows-amd64..."
	@mkdir -p dist/tmp-windows-amd64
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o dist/tmp-windows-amd64/skill-hub.exe ./cmd/skill-hub
	cp README.md dist/tmp-windows-amd64/
	cp LICENSE dist/tmp-windows-amd64/
	cd dist/tmp-windows-amd64 && tar -czf ../skill-hub-windows-amd64.tar.gz .
	cd dist && sha256sum skill-hub-windows-amd64.tar.gz > skill-hub-windows-amd64.sha256
	rm -rf dist/tmp-windows-amd64
	@echo "  Created: skill-hub-windows-amd64.tar.gz + .sha256"
	
	# Windows arm64
	@echo "Building windows-arm64..."
	@mkdir -p dist/tmp-windows-arm64
	GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o dist/tmp-windows-arm64/skill-hub.exe ./cmd/skill-hub
	cp README.md dist/tmp-windows-arm64/
	cp LICENSE dist/tmp-windows-arm64/
	cd dist/tmp-windows-arm64 && tar -czf ../skill-hub-windows-arm64.tar.gz .
	cd dist && sha256sum skill-hub-windows-arm64.tar.gz > skill-hub-windows-arm64.sha256
	rm -rf dist/tmp-windows-arm64
	@echo "  Created: skill-hub-windows-arm64.tar.gz + .sha256"
	
	@echo ""
	@echo "Release packages created in dist/ directory:"
	@cd dist && ls -la *.tar.gz *.sha256

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