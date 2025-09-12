# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

CUBE_PROXY_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/proxy
CUBE_AGENT_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/agent
CUBE_GUARDRAILS_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/guardrails
NEMO_GUARDRAILS_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/nemo-guardrails
CGO_ENABLED ?= 0
GOOS ?= linux
GOARCH ?= amd64
BUILD_DIR = build

# Setup automation variables
CUBE_URL ?= https://localhost:6193
ADMIN_EMAIL ?= 
ADMIN_PASSWORD ?=
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
build: build-proxy build-agent build-guardrails

.PHONY: build-proxy
build-proxy:
	$(call compile_service,proxy)

.PHONY: build-agent
build-agent:
	$(call compile_service,agent)

.PHONY: build-guardrails
build-guardrails:
	$(call compile_service,guardrails)

.PHONY: run
run:
	@echo "Starting Cube AI application stack..."
	docker compose -f docker/compose.yaml up

# Docker targets - builds both Cube services and NeMo Guardrails
.PHONY: docker
docker: docker-proxy docker-agent docker-guardrails docker-nemo-guardrails-dev

.PHONY: docker-proxy
docker-proxy:
	$(call make_docker,proxy,$(CUBE_PROXY_DOCKER_IMAGE_NAME))

.PHONY: docker-agent
docker-agent:
	$(call make_docker,agent,$(CUBE_AGENT_DOCKER_IMAGE_NAME))

.PHONY: docker-guardrails
docker-guardrails:
	$(call make_docker,guardrails,$(CUBE_GUARDRAILS_DOCKER_IMAGE_NAME))

.PHONY: docker-nemo-guardrails-dev
docker-nemo-guardrails-dev:
	docker build \
		--no-cache \
		--tag=$(NEMO_GUARDRAILS_DOCKER_IMAGE_NAME):$(VERSION) \
		--tag=$(NEMO_GUARDRAILS_DOCKER_IMAGE_NAME):latest \
		-f docker/guardrails/Dockerfile .

.PHONY: docker-dev
docker-dev: docker-proxy-dev docker-agent-dev docker-guardrails-dev docker-nemo-guardrails-dev

.PHONY: docker-proxy-dev
docker-proxy-dev:
	$(call make_docker_dev,proxy,$(CUBE_PROXY_DOCKER_IMAGE_NAME))

.PHONY: docker-agent-dev
docker-agent-dev:
	$(call make_docker_dev,agent,$(CUBE_AGENT_DOCKER_IMAGE_NAME))

.PHONY: docker-guardrails-dev
docker-guardrails-dev:
	$(call make_docker_dev,guardrails,$(CUBE_GUARDRAILS_DOCKER_IMAGE_NAME))

all: build docker-dev

clean:
	rm -rf build
	rm -f cube_ai_users.json test_report_*.json test_summary_*.txt

lint:
	golangci-lint run --config .golangci.yaml

.PHONY: latest
latest: docker docker-push

.PHONY: docker-push
docker-push: docker-push-proxy docker-push-agent docker-push-guardrails docker-push-nemo-guardrails

.PHONY: docker-push-proxy
docker-push-proxy:
	$(call docker_push,$(CUBE_PROXY_DOCKER_IMAGE_NAME))

.PHONY: docker-push-agent
docker-push-agent:
	$(call docker_push,$(CUBE_AGENT_DOCKER_IMAGE_NAME))

.PHONY: docker-push-guardrails
docker-push-guardrails:
	$(call docker_push,$(CUBE_GUARDRAILS_DOCKER_IMAGE_NAME))

.PHONY: docker-push-nemo-guardrails
docker-push-nemo-guardrails:
	$(call docker_push,$(NEMO_GUARDRAILS_DOCKER_IMAGE_NAME))

