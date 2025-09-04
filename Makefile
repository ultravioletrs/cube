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

define compile_service
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build -ldflags "-s -w \
	-X 'github.com/absmach/supermq.BuildTime=$(TIME)' \
	-X 'github.com/absmach/supermq.Version=$(VERSION)' \
	-X 'github.com/absmach/supermq.Commit=$(COMMIT)'" \
	-o ${BUILD_DIR}/cube-$(1) cmd/$(1)/main.go
endef

define make_docker
	docker build \
		--no-cache \
		--build-arg SVC=$(1) \
		--build-arg GOOS=$(GOOS) \
		--build-arg GOARCH=$(GOARCH) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--tag=$(2):$(VERSION) \
		--tag=$(2):latest \
		-f docker/Dockerfile .
endef

define make_docker_dev
	docker build \
		--no-cache \
		--build-arg SVC=$(1) \
		--tag=$(2):$(VERSION) \
		--tag=$(2):latest \
		-f docker/Dockerfile.dev ./build
endef

define docker_push
	docker push $(1):$(VERSION)
	docker push $(1):latest
endef

.PHONY: build
build: build-proxy build-agent

.PHONY: build-proxy
build-proxy:
	$(call compile_service,proxy)

.PHONY: build-agent
build-agent:
	$(call compile_service,agent)

.PHONY: docker
docker: docker-proxy docker-agent

.PHONY: docker-proxy
docker-proxy:
	$(call make_docker,proxy,$(CUBE_PROXY_DOCKER_IMAGE_NAME))

.PHONY: docker-agent
docker-agent:
	$(call make_docker,agent,$(CUBE_AGENT_DOCKER_IMAGE_NAME))

.PHONY: docker-dev
docker-dev: docker-proxy-dev docker-agent-dev

.PHONY: docker-proxy-dev
docker-proxy-dev:
	$(call make_docker_dev,proxy,$(CUBE_PROXY_DOCKER_IMAGE_NAME))

.PHONY: docker-agent-dev
docker-agent-dev:
	$(call make_docker_dev,agent,$(CUBE_AGENT_DOCKER_IMAGE_NAME))

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
	$(call docker_push,$(CUBE_PROXY_DOCKER_IMAGE_NAME))

.PHONY: docker-push-agent
docker-push-agent:
	$(call docker_push,$(CUBE_AGENT_DOCKER_IMAGE_NAME))
