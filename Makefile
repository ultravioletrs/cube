# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

CUBE_PROXY_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/proxy
CUBE_AGENT_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/agent
CGO_ENABLED ?= 0
GOOS ?= linux
GOARCH ?= amd64
BUILD_DIR = build
TIME=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
VERSION ?= $(shell git describe --abbrev=0 --tags 2>/dev/null || echo 'v0.0.0')
COMMIT ?= $(shell git rev-parse HEAD)

define compile_proxy
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build -ldflags "-s -w \
	-X 'github.com/absmach/supermq.BuildTime=$(TIME)' \
	-X 'github.com/absmach/supermq.Version=$(VERSION)' \
	-X 'github.com/absmach/supermq.Commit=$(COMMIT)'" \
	-o ${BUILD_DIR}/cube-proxy cmd/proxy/main.go
endef

define compile_agent
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build -ldflags "-s -w \
	-X 'github.com/absmach/supermq.BuildTime=$(TIME)' \
	-X 'github.com/absmach/supermq.Version=$(VERSION)' \
	-X 'github.com/absmach/supermq.Commit=$(COMMIT)'" \
	-o ${BUILD_DIR}/cube-agent cmd/agent/main.go
endef

define make_docker_proxy
	docker build \
		--no-cache \
		--build-arg GOOS=$(GOOS) \
		--build-arg GOARCH=$(GOARCH) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--tag=$(CUBE_PROXY_DOCKER_IMAGE_NAME):$(VERSION) \
		--tag=$(CUBE_PROXY_DOCKER_IMAGE_NAME):latest \
		-f docker/Dockerfile.proxy .
endef

define make_docker_agent
	docker build \
		--no-cache \
		--build-arg GOOS=$(GOOS) \
		--build-arg GOARCH=$(GOARCH) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--tag=$(CUBE_AGENT_DOCKER_IMAGE_NAME):$(VERSION) \
		--tag=$(CUBE_AGENT_DOCKER_IMAGE_NAME):latest \
		-f docker/Dockerfile.agent .
endef

define make_docker_proxy_dev
	docker build \
		--no-cache \
		--tag=$(CUBE_PROXY_DOCKER_IMAGE_NAME):$(VERSION) \
		--tag=$(CUBE_PROXY_DOCKER_IMAGE_NAME):latest \
		-f docker/Dockerfile.proxy.dev ./build
endef

define make_docker_agent_dev
	docker build \
		--no-cache \
		--tag=$(CUBE_AGENT_DOCKER_IMAGE_NAME):$(VERSION) \
		--tag=$(CUBE_AGENT_DOCKER_IMAGE_NAME):latest \
		-f docker/Dockerfile.agent.dev ./build
endef

define docker_push_proxy
	docker push $(CUBE_PROXY_DOCKER_IMAGE_NAME):$(VERSION)
	docker push $(CUBE_PROXY_DOCKER_IMAGE_NAME):latest
endef

define docker_push_agent
	docker push $(CUBE_AGENT_DOCKER_IMAGE_NAME):$(VERSION)
	docker push $(CUBE_AGENT_DOCKER_IMAGE_NAME):latest
endef

.PHONY: build
build: build-proxy build-agent

.PHONY: build-proxy
build-proxy:
	$(call compile_proxy)

.PHONY: build-agent
build-agent:
	$(call compile_agent)

.PHONY: docker
docker: docker-proxy docker-agent

.PHONY: docker-proxy
docker-proxy:
	$(call make_docker_proxy)

.PHONY: docker-agent
docker-agent:
	$(call make_docker_agent)

.PHONY: docker-dev
docker-dev: docker-proxy-dev docker-agent-dev

.PHONY: docker-proxy-dev
docker-proxy-dev:
	$(call make_docker_proxy_dev)

.PHONY: docker-agent-dev
docker-agent-dev:
	$(call make_docker_agent_dev)

all: build docker-dev

clean:
	rm -rf build

lint:
	golangci-lint run --config .golangci.yaml

.PHONY: latest
latest: docker docker-push

.PHONY: docker-push
docker-push: docker-push-proxy docker-push-agent

.PHONY: docker-push-proxy
docker-push-proxy:
	$(call docker_push_proxy)

.PHONY: docker-push-agent
docker-push-agent:
	$(call docker_push_agent)
