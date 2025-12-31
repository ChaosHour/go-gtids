.PHONY: build build-macos build-linux build-go build-docker-go build-docker-pt all clean

build:
	@mkdir -p bin
	@echo "Building go-gtids for current platform..."
	@go build -o bin/go-gtids ./cmd/go-gtids

build-macos:
	@mkdir -p bin
	@echo "Building go-gtids for macOS..."
	@env GOOS=darwin GOARCH=amd64 go build -o bin/go-gtids-macos ./cmd/go-gtids

build-linux:
	@mkdir -p bin
	@echo "Building go-gtids for Linux..."
	@env GOOS=linux GOARCH=amd64 go build -o bin/go-gtids-linux ./cmd/go-gtids

build-go: build-macos build-linux

build-docker-go: build-go
	@docker build -t go-gtids -f docker_testing/Dockerfile.go-gtids .
build-docker-pt:
	@docker build -t pt-slave-restart -f docker_testing/Dockerfile.pt-slave-restart .
build-docker-test: build-linux
	@docker build -t go-gtids-test -f docker_testing/Dockerfile.test .

all: build-docker-go build-docker-pt

# Testing targets
test:
	@echo "Running unit tests..."
	@go test -v ./pkg/gtids

test-cover:
	@echo "Running tests with coverage..."
	@go test -cover ./pkg/gtids

test-integration: test-db-up
	@echo "Running integration tests..."
	@go test -v -tags=integration ./pkg/gtids || (make test-db-down; exit 1)
	@make test-db-down

test-db-up:
	@echo "Starting test databases..."
	@docker-compose up --wait mysql-source mysql-target

test-db-down:
	@echo "Stopping test databases..."
	@docker-compose down -v

test-db-logs:
	@docker-compose logs -f

test-db-shell-source:
	@./db-shell.sh source shell

test-db-shell-target:
	@./db-shell.sh target shell

test-db-ping-source:
	@./db-shell.sh source query

test-db-ping-target:
	@./db-shell.sh target query

test-gtids:
	@echo "Testing GTID detection..."
	@./bin/go-gtids -s 127.0.0.1 -source-port 3306 -t 127.0.0.1 -target-port 3307

test-gtids-fix:
	@echo "Testing GTID fixing..."
	@./bin/go-gtids -s 127.0.0.1 -source-port 3306 -t 127.0.0.1 -target-port 3307 -fix

test-gtids-fix-replica:
	@echo "Testing GTID fixing on replica..."
	@./bin/go-gtids -s 127.0.0.1 -source-port 3306 -t 127.0.0.1 -target-port 3307 -fix-replica

# Release targets
release-dry-run:
	@echo "Running GoReleaser dry run..."
	@goreleaser release --snapshot --clean

release-local:
	@echo "Building local release..."
	@goreleaser release --snapshot --clean --skip-publish

clean:
	@echo "Cleaning go binaries..."
	@rm -rf bin/go-gtids bin/go-gtids-macos bin/go-gtids-linux dist/