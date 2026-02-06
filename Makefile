# Skill Hub Makefile

.PHONY: build clean test install

# Build the binary
build:
	go build -o bin/skill-hub ./cmd/skill-hub

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

# Create release binaries
release: clean
	GOOS=linux GOARCH=amd64 go build -o dist/skill-hub-linux-amd64 ./cmd/skill-hub
	GOOS=darwin GOARCH=amd64 go build -o dist/skill-hub-darwin-amd64 ./cmd/skill-hub
	GOOS=windows GOARCH=amd64 go build -o dist/skill-hub-windows-amd64.exe ./cmd/skill-hub

# Run linting
lint:
	gofmt -d .
	go vet ./...

# Update dependencies
deps:
	go mod tidy
	go mod verify