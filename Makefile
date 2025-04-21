# Makefile for cfddns - Cloudflare Dynamic DNS Updater

# Go parameters
BINARY_NAME=cfddns
GO=go
GOFMT=gofmt
GOFILES=$(shell find . -type f -name "*.go")
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0")
LDFLAGS=-ldflags "-X main.version=${VERSION}"

# Docker parameters
DOCKER_REPO=lhaig/cfddns
DOCKER_TAG?=latest
PLATFORMS=linux/amd64,linux/arm64,linux/arm/v7

.PHONY: all build clean fmt lint test run docker docker-buildx push help

all: fmt lint test build

# Build the binary
build:
	@echo "Building..."
	${GO} build ${LDFLAGS} -o ${BINARY_NAME} ./cmd/main.go

# Clean the project
clean:
	@echo "Cleaning..."
	${GO} clean
	rm -f ${BINARY_NAME}

# Format the code
fmt:
	@echo "Formatting..."
	${GOFMT} -w ${GOFILES}

# Lint the code
lint:
	@echo "Linting..."
	${GO} vet ./...
	@if command -v golint &> /dev/null; then \
		golint ./...; \
	else \
		echo "golint not installed. Run: go install golang.org/x/lint/golint@latest"; \
	fi

# Run tests
test:
	@echo "Testing..."
	${GO} test -v ./...

# Run the application
run:
	@echo "Running..."
	${GO} run ${LDFLAGS} ./cmd/main.go

# Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t ${DOCKER_REPO}:${DOCKER_TAG} .

# Build and push multi-architecture Docker images
docker-buildx:
	@echo "Building multi-arch Docker images..."
	docker buildx create --name cfddns-builder --use || true
	docker buildx build --platform ${PLATFORMS} \
		-t ${DOCKER_REPO}:${DOCKER_TAG} \
		-t ${DOCKER_REPO}:${VERSION} \
		--push .

# Push Docker image to registry
push: docker
	@echo "Pushing Docker image..."
	docker push ${DOCKER_REPO}:${DOCKER_TAG}
	@if [ "${DOCKER_TAG}" != "${VERSION}" ] && [ "${DOCKER_TAG}" = "latest" ]; then \
		docker tag ${DOCKER_REPO}:${DOCKER_TAG} ${DOCKER_REPO}:${VERSION}; \
		docker push ${DOCKER_REPO}:${VERSION}; \
	fi

# Release versioned Docker image
release:
	@echo "Releasing version ${VERSION}..."
	$(MAKE) docker DOCKER_TAG=${VERSION}
	$(MAKE) push DOCKER_TAG=${VERSION}
	$(MAKE) push DOCKER_TAG=latest

# Install the binary
install: build
	@echo "Installing..."
	cp ${BINARY_NAME} /usr/local/bin/${BINARY_NAME}

# Help output
help:
	@echo "Make targets:"
	@echo "  all        - Format, lint, test, and build"
	@echo "  build      - Build the binary"
	@echo "  clean      - Remove build artifacts"
	@echo "  fmt        - Format the code"
	@echo "  lint       - Lint the code"
	@echo "  test       - Run tests"
	@echo "  run        - Run the application"
	@echo "  docker     - Build Docker image"
	@echo "  push       - Push Docker image to registry"
	@echo "  docker-buildx - Build and push multi-arch Docker images"
	@echo "  release    - Release versioned Docker image"
	@echo "  install    - Install the binary"
	@echo "  help       - This help output"
	@echo
	@echo "Variables:"
	@echo "  VERSION    - Version tag (default: from git or v0.1.0)"
	@echo "  DOCKER_TAG - Docker image tag (default: latest)"