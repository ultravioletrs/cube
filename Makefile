CUBE_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/proxy
CGO_ENABLED ?= 0
GOOS ?= linux
GOARCH ?= amd64
BUILD_DIR = build
TIME=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
VERSION ?= $(shell git describe --abbrev=0 --tags 2>/dev/null || echo 'v0.0.0')
COMMIT ?= $(shell git rev-parse HEAD)

define compile_service
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build -ldflags "-s -w \
	-X 'github.com/absmach/magistrala.BuildTime=$(TIME)' \
	-X 'github.com/absmach/magistrala.Version=$(VERSION)' \
	-X 'github.com/absmach/magistrala.Commit=$(COMMIT)'" \
	-o ${BUILD_DIR}/cube-proxy cmd/main.go
endef

define make_docker
	docker build \
		--no-cache \
		--build-arg GOOS=$(GOOS) \
		--build-arg GOARCH=$(GOARCH) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--tag=$(CUBE_DOCKER_IMAGE_NAME):$(VERSION) \
		--tag=$(CUBE_DOCKER_IMAGE_NAME):latest \
		-f docker/Dockerfile .
endef

define make_docker_dev
	docker build \
		--no-cache \
		--tag=$(CUBE_DOCKER_IMAGE_NAME):$(VERSION) \
		--tag=$(CUBE_DOCKER_IMAGE_NAME):latest \
		-f docker/Dockerfile.dev ./build
endef

define docker_push
	docker push $(CUBE_DOCKER_IMAGE_NAME):$(VERSION)
	docker push $(CUBE_DOCKER_IMAGE_NAME):latest
endef

.PHONY: build
build:
	$(call compile_service)

.PHONY: docker
docker:
	$(call make_docker)

.PHONY: docker-dev
docker-dev:
	$(call make_docker_dev)

all: build docker-dev

clean:
	rm -rf build

lint:
	golangci-lint run  --config .golangci.yaml

latest: docker
	$(call docker_push)
