.PHONY: build-go build-docker-go build-docker-pt

build-go:
	@env GOOS=linux GOARCH=amd64 go build -o go-gtids-linux

build-docker-go: build-go
	@docker build -t go-gtids -f Dockerfile.go-gtids .
build-docker-pt:
	@docker build -t pt-slave-restart -f Dockerfile.pt-slave-restart .

all: build-docker-go build-docker-pt