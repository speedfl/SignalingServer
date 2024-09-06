# Makefile for signaling-server

# Variables
BINARY_NAME=signaling-server
DOCKER_IMAGE=speedfl/signaling-server:latest

# Targets
.PHONY: all build run docker-build docker-run docker-push

all: build

mod:
	go mod tidy
	go mod vendor

build: mod
	go build -o $(BINARY_NAME) ./cmd/

run: build
	./$(BINARY_NAME)

docker-build:
	docker build -t $(DOCKER_IMAGE) .

docker-run: docker-build
	docker run --name $(BINARY_NAME) -p 4444:4444 $(DOCKER_IMAGE)

docker-push:
	docker push $(DOCKER_IMAGE)