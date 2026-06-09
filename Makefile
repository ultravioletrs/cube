# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

CUBE_PROXY_IMAGE          ?= ghcr.io/ultravioletrs/cube/proxy
CUBE_AGENT_IMAGE          ?= ghcr.io/ultravioletrs/cube/agent
CUBE_EMBEDDER_IMAGE       ?= ghcr.io/ultravioletrs/cube/embedder
CUBE_GUARDRAILS_IMAGE     ?= ghcr.io/ultravioletrs/cube/guardrails
CUBE_IMAGE_EMBEDDER_IMAGE ?= ghcr.io/ultravioletrs/cube/image-embedder
CUBE_UI_IMAGE             ?= ghcr.io/ultravioletrs/cube/ui

CGO_ENABLED ?= 0
GOOS        ?= linux
GOARCH      ?= amd64
BUILD_DIR   = build
VERSION     ?= $(shell git describe --abbrev=0 --tags 2>/dev/null || echo 'v0.0.0')
COMMIT      ?= $(shell git rev-parse HEAD)

LOCAL_COMPOSE = docker/local/docker-compose.yaml
LOCAL_ENV     = docker/local/.env
PROD_COMPOSE  = docker/prod/docker-compose.yaml
PROD_ENV      = docker/prod/.env

SERVICES = proxy agent embedder

define compile_service
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	go build -ldflags "-s -w" -o $(BUILD_DIR)/cube-$(1) cmd/$(1)/main.go
endef

define make_docker
	docker build \
		--build-arg SVC=$(1) \
		--build-arg GOOS=$(GOOS) \
		--build-arg GOARCH=$(GOARCH) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--tag=$(2):$(VERSION) \
		--tag=$(2):latest \
		-f docker/Dockerfile .
endef

define docker_push
	docker push $(1):$(VERSION)
	docker push $(1):latest
endef

# ── Build ──────────────────────────────────────────────────────────────
.PHONY: all
all: build

.PHONY: build
build: $(addprefix build-,$(SERVICES))

build-%:
	$(call compile_service,$*)

# Per-service build aliases: `make proxy` == `make build-proxy`.
.PHONY: $(SERVICES)
$(SERVICES): %: build-%

# ── Docker images ──────────────────────────────────────────────────────
.PHONY: dockers
dockers: docker-proxy docker-agent docker-embedder docker-guardrails docker-image-embedder docker-ui

docker-proxy:
	$(call make_docker,proxy,$(CUBE_PROXY_IMAGE))

docker-agent:
	$(call make_docker,agent,$(CUBE_AGENT_IMAGE))

# Embedder needs a runtime with poppler-utils + tesseract (PDF/OCR text
# extraction), so it uses its own Dockerfile instead of the scratch image.
docker-embedder:
	docker build --tag=$(CUBE_EMBEDDER_IMAGE):$(VERSION) --tag=$(CUBE_EMBEDDER_IMAGE):latest \
		-f docker/Dockerfile.embedder .

docker-guardrails:
	docker build --tag=$(CUBE_GUARDRAILS_IMAGE):$(VERSION) --tag=$(CUBE_GUARDRAILS_IMAGE):latest \
		-f guardrails/Dockerfile ./guardrails

docker-image-embedder:
	docker build --tag=$(CUBE_IMAGE_EMBEDDER_IMAGE):$(VERSION) --tag=$(CUBE_IMAGE_EMBEDDER_IMAGE):latest \
		-f docker/Dockerfile.image-embedder .

# Build the UI container used by the local stack.
.PHONY: docker-ui
docker-ui:
	docker compose -f $(LOCAL_COMPOSE) --env-file $(LOCAL_ENV) build ui
	docker tag $(CUBE_UI_IMAGE):latest $(CUBE_UI_IMAGE):$(VERSION)

.PHONY: docker-push
docker-push:
	$(call docker_push,$(CUBE_PROXY_IMAGE))
	$(call docker_push,$(CUBE_AGENT_IMAGE))
	$(call docker_push,$(CUBE_EMBEDDER_IMAGE))
	$(call docker_push,$(CUBE_GUARDRAILS_IMAGE))
	$(call docker_push,$(CUBE_IMAGE_EMBEDDER_IMAGE))
	$(call docker_push,$(CUBE_UI_IMAGE))

# ── Local stack (minimal, no TEE) ──────────────────────────────────────
.PHONY: up down restart logs
up: docker-ui
	docker compose -f $(LOCAL_COMPOSE) --env-file $(LOCAL_ENV) up -d

down:
	docker compose -f $(LOCAL_COMPOSE) --env-file $(LOCAL_ENV) down

restart: down up

logs:
	docker compose -f $(LOCAL_COMPOSE) --env-file $(LOCAL_ENV) logs -f

# Stop and wipe volumes (databases, models, uploads).
.PHONY: clean-volumes
clean-volumes:
	docker compose -f $(LOCAL_COMPOSE) --env-file $(LOCAL_ENV) down -v

# ── Production stack ───────────────────────────────────────────────────
.PHONY: up-prod down-prod restart-prod logs-prod
up-prod:
	docker compose -f $(PROD_COMPOSE) --env-file $(PROD_ENV) up -d

down-prod:
	docker compose -f $(PROD_COMPOSE) --env-file $(PROD_ENV) down

restart-prod: down-prod up-prod

logs-prod:
	docker compose -f $(PROD_COMPOSE) --env-file $(PROD_ENV) logs -f

# ── Dev tooling ────────────────────────────────────────────────────────
.PHONY: lint mocks clean guardrails-venv
lint:
	golangci-lint run --config .golangci.yaml

mocks:
	mockery --config ./.mockery.yml

clean:
	rm -rf $(BUILD_DIR)

guardrails-venv:
	python -m venv .venv
	. .venv/bin/activate && pip install --upgrade pip && pip install -r guardrails/requirements.txt
	. .venv/bin/activate && python -m spacy download en_core_web_lg

# ── Help ───────────────────────────────────────────────────────────────
.PHONY: help
help:
	@echo "Cube — make targets"
	@echo ""
	@echo "Build:"
	@echo "  build              Compile proxy, agent, embedder binaries"
	@echo "  proxy|agent|embedder  Compile a single service binary"
	@echo "  dockers            Build all service docker images (:latest and :VERSION)"
	@echo "  docker-<service>   Build one service image: proxy, agent, embedder,"
	@echo "                     guardrails, image-embedder, ui"
	@echo "  docker-push        Push all Cube images"
	@echo ""
	@echo "Local stack (docker/local — minimal, no TEE):"
	@echo "  up                 Build UI and start the local stack (needs 'make dockers' first)"
	@echo "  down               Stop the local stack"
	@echo "  restart            Restart the local stack"
	@echo "  logs               Follow local stack logs"
	@echo "  clean-volumes      Stop and delete local volumes"
	@echo ""
	@echo "Production stack (docker/prod — full, edit docker/prod/.env first):"
	@echo "  up-prod            Start the production stack"
	@echo "  down-prod          Stop the production stack"
	@echo "  restart-prod       Restart the production stack"
	@echo "  logs-prod          Follow production stack logs"
	@echo ""
	@echo "Dev:"
	@echo "  lint  mocks  clean  guardrails-venv"

.DEFAULT_GOAL := help
