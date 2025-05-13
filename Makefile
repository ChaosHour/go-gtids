.PHONY: build-go build-docker-go build-docker-pt all clean

build-go:
	@mkdir -p bin
	@echo "Building go-gtids..."
	@echo "Building for macOS..."
	@env GOOS=darwin GOARCH=amd64 go build -o bin/go-gtids-macos ./cmd/go-gtids
	@echo "Building for Linux..."
	@env GOOS=linux GOARCH=amd64 go build -o bin/go-gtids-linux ./cmd/go-gtids

build-docker-go: build-go
	@docker build -t go-gtids -f Dockerfile.go-gtids .
build-docker-pt:
	@docker build -t pt-slave-restart -f Dockerfile.pt-slave-restart .

all: build-docker-go build-docker-pt

clean:
	@echo "Cleaning go binaries..."
	@rm -rf bin/go-gtids-macos bin/go-gtids-linux