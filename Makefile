# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

CUBE_PROXY_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/proxy
CUBE_AGENT_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/agent
CUBE_GUARDRAILS_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/guardrails
CGO_ENABLED ?= 0
GOOS ?= linux
GOARCH ?= amd64
BUILD_DIR = build
TIME=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
VERSION ?= $(shell git describe --abbrev=0 --tags 2>/dev/null || echo 'v0.0.0')
COMMIT ?= $(shell git rev-parse HEAD)

AI_BACKEND ?= ollama
OLLAMA_TARGET_URL = http://ollama:11434
VLLM_TARGET_URL = http://vllm:8000

ENV_FILE = ./docker/.env
CONFIG_FILE = ./docker/config.json

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

define update_env_var
	@if [ -f $(ENV_FILE) ]; then \
		if grep -q "^$(1)=" $(ENV_FILE); then \
			sed -i 's|^$(1)=.*|$(1)=$(2)|' $(ENV_FILE); \
		else \
			echo "$(1)=$(2)" >> $(ENV_FILE); \
		fi; \
	else \
		echo "$(1)=$(2)" > $(ENV_FILE); \
	fi
	@echo "Updated $(1) to $(2)"
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
docker: docker-proxy docker-agent docker-guardrails

.PHONY: docker-proxy
docker-proxy:
	$(call make_docker,proxy,$(CUBE_PROXY_DOCKER_IMAGE_NAME))

.PHONY: docker-agent
docker-agent:
	$(call make_docker,agent,$(CUBE_AGENT_DOCKER_IMAGE_NAME))

.PHONY: docker-guardrails
docker-guardrails:
	docker build \
		--no-cache \
		--tag=$(CUBE_GUARDRAILS_DOCKER_IMAGE_NAME):$(VERSION) \
		--tag=$(CUBE_GUARDRAILS_DOCKER_IMAGE_NAME):latest \
		-f guardrails/Dockerfile ./guardrails

.PHONY: docker-guardrails-dev
docker-guardrails-dev:
	docker build \
		--tag=$(CUBE_GUARDRAILS_DOCKER_IMAGE_NAME):$(VERSION) \
		--tag=$(CUBE_GUARDRAILS_DOCKER_IMAGE_NAME):latest \
		-f guardrails/Dockerfile.dev .

.PHONY: guardrails-venv
guardrails-venv:
	@echo "Setting up guardrails virtual environment in root .venv..."
	python -m venv .venv
	. .venv/bin/activate && pip install --upgrade pip && pip install -r guardrails/requirements.txt
	. .venv/bin/activate && python -m spacy download en_core_web_lg
	@echo "Guardrails venv created successfully at .venv"

.PHONY: docker-dev
docker-dev: docker-proxy-dev docker-agent-dev docker-guardrails-dev

.PHONY: docker-proxy-dev
docker-proxy-dev:
	$(call make_docker_dev,proxy,$(CUBE_PROXY_DOCKER_IMAGE_NAME))

.PHONY: docker-agent-dev
docker-agent-dev:
	$(call make_docker_dev,agent,$(CUBE_AGENT_DOCKER_IMAGE_NAME))

.PHONY: config-ollama
config-ollama:
	$(call update_env_var,UV_CUBE_AGENT_TARGET_URL,$(OLLAMA_TARGET_URL))
	@echo "Configured for Ollama backend"

.PHONY: config-vllm
config-vllm:
	$(call update_env_var,UV_CUBE_AGENT_TARGET_URL,$(VLLM_TARGET_URL))
	@echo "Configured for vLLM backend"

.PHONY: config-backend
config-backend:
ifeq ($(AI_BACKEND),vllm)
	@$(MAKE) config-vllm
else ifeq ($(AI_BACKEND),ollama)
	@$(MAKE) config-ollama
else
	@echo "Invalid AI_BACKEND: $(AI_BACKEND). Use 'ollama' or 'vllm'"
	@exit 1
endif

.PHONY: up-ollama
up-ollama: config-ollama
	@echo "Starting Cube with Ollama backend..."
	docker compose -f docker/compose.yaml --profile default up -d

.PHONY: up-vllm
up-vllm: config-vllm
	@echo "Starting Cube with vLLM backend..."
	docker compose -f docker/compose.yaml --profile vllm up -d

GUARDRAILS_CONFIG_FILE = ./guardrails/rails/config.yml

.PHONY: config-guardrails-vllm
config-guardrails-vllm:
	@echo "Configuring guardrails for vLLM backend..."
	@sed -i '/^models:/,/X-Guardrails-Request:/{s/engine: CubeLLM/engine: CubeVLLM/; s/model: llama3.2:3b/model: microsoft\/DialoGPT-medium/}' $(GUARDRAILS_CONFIG_FILE)
	@echo "Guardrails configured for vLLM"

.PHONY: config-guardrails-ollama
config-guardrails-ollama:
	@echo "Configuring guardrails for Ollama backend..."
	@sed -i '/^models:/,/X-Guardrails-Request:/{s/engine: CubeVLLM/engine: CubeLLM/; s/model: microsoft\/DialoGPT-medium/model: llama3.2:3b/}' $(GUARDRAILS_CONFIG_FILE)
	@echo "Guardrails configured for Ollama"

.PHONY: up-vllm-guardrails
up-vllm-guardrails: enable-guardrails config-guardrails-vllm up-vllm

.PHONY: up
up: enable-guardrails config-backend config-cloud-local
ifeq ($(AI_BACKEND),vllm)
	@$(MAKE) up-vllm
else
	@$(MAKE) up-ollama
endif

.PHONY: up-disable-guardrails
up-disable-guardrails: disable-guardrails config-backend config-cloud-local
ifeq ($(AI_BACKEND),vllm)
	@$(MAKE) up-vllm
else
	@$(MAKE) up-ollama
endif

.PHONY: config-cloud-local
config-cloud-local:
	@echo "Configuring cloud deployment for local environment..."
	@cp docker/.env docker/.env.backup 2>/dev/null || true
	@cp docker/traefik/dynamic.toml docker/traefik/dynamic.toml.backup 2>/dev/null || true
	@cp docker/config.json docker/config.json.backup 2>/dev/null || true
	@sed -i 's|__SMQ_EMAIL_HOST__|localhost|g' docker/.env
	@sed -i 's|__SMQ_EMAIL_PORT__|1025|g' docker/.env
	@sed -i 's|__SMQ_EMAIL_USERNAME__|test|g' docker/.env
	@sed -i 's|__SMQ_EMAIL_PASSWORD__|test|g' docker/.env
	@sed -i 's|__SMQ_EMAIL_FROM_ADDRESS__|noreply@localhost|g' docker/.env
	@sed -i 's|__CUBE_INTERNAL_AGENT_URL__|http://cube-agent:8901|g' docker/.env
	@sed -i 's|__CUBE_INTERNAL_AGENT_URL__|http://cube-agent:8901|g' docker/config.json
	@sed -i 's|__CUBE_DOMAIN__|localhost|g' docker/traefik/dynamic.toml
	@sed -i 's|__SMQ_GOOGLE_CLIENT_ID__||g' docker/.env
	@sed -i 's|__SMQ_GOOGLE_CLIENT_SECRET__||g' docker/.env
	@sed -i 's|__SMQ_GOOGLE_STATE__||g' docker/.env
	@sed -i 's|__CUBE_PUBLIC_URL__|localhost|g' docker/.env
	@sed -i 's|^TRAEFIK_HTTP_PORT=.*|TRAEFIK_HTTP_PORT=49210|g' docker/.env
	@sed -i 's|^TRAEFIK_HTTPS_PORT=.*|TRAEFIK_HTTPS_PORT=49211|g' docker/.env
	@sed -i 's|^TRAEFIK_DASHBOARD_PORT=.*|TRAEFIK_DASHBOARD_PORT=49212|g' docker/.env
	@echo "✓ Configured with local defaults"

.PHONY: restore-cloud-config
restore-cloud-config:
	@echo "Restoring cloud deployment placeholders..."
	@if [ -f docker/.env.backup ]; then \
		mv docker/.env.backup docker/.env; \
		echo "✓ Restored .env"; \
	fi
	@if [ -f docker/traefik/dynamic.toml.backup ]; then \
		mv docker/traefik/dynamic.toml.backup docker/traefik/dynamic.toml; \
		echo "✓ Restored dynamic.toml"; \
	fi
	@if [ -f docker/config.json.backup ]; then \
		mv docker/config.json.backup docker/config.json; \
		echo "✓ Restored config.json"; \
	fi

.PHONY: up-cloud
up-cloud: config-cloud-local
	@echo "Starting Cube Cloud services with local configuration..."
	@mkdir -p docker/traefik/ssl/certs docker/traefik/letsencrypt
	@if [ ! -f docker/traefik/ssl/certs/acme.json ]; then \
		printf '{}' > docker/traefik/ssl/certs/acme.json; \
		chmod 600 docker/traefik/ssl/certs/acme.json; \
		echo "✓ Created acme.json"; \
	fi
	docker compose -f docker/compose.yaml --profile cloud up -d
	@echo ""
	@echo "=== Cube Cloud Services Started ==="
	@echo "  - UI: http://localhost:49210/"
	@echo "  - Proxy API: http://localhost:49210/proxy"
	@echo "  - Traefik Dashboard: http://localhost:49212"
	@echo ""
	@echo "Note: Run 'make restore-cloud-config' to restore placeholders after stopping"

.PHONY: down
down:
	@echo "Stopping all Cube services..."
	docker compose -f docker/compose.yaml down

.PHONY: down-cloud
down-cloud:
	@echo "Stopping Cube Cloud services..."
	docker compose -f docker/compose.yaml --profile cloud down
	@$(MAKE) restore-cloud-config

.PHONY: down-volumes
down-volumes:
	@echo "Stopping all Cube services and removing volumes..."
	docker compose -f docker/compose.yaml down -v

.PHONY: down-cloud-volumes
down-cloud-volumes:
	@echo "Stopping Cube Cloud services and removing volumes..."
	docker compose -f docker/compose.yaml --profile cloud down -v
	@$(MAKE) restore-cloud-config

.PHONY: restart
restart: down up

.PHONY: restart-ollama
restart-ollama: down up-ollama

.PHONY: restart-vllm
restart-vllm: down up-vllm

.PHONY: restart-cloud
restart-cloud: down-cloud up-cloud

.PHONY: logs
logs:
	docker compose -f docker/compose.yaml logs -f

.PHONY: logs-cloud
logs-cloud:
	docker compose -f docker/compose.yaml --profile cloud logs -f

.PHONY: dev-setup
dev-setup: build docker-dev

.PHONY: show-config
show-config:
	@echo "=== Current Configuration ==="
	@echo "AI Backend: $(AI_BACKEND)"
	@echo "Ollama Target: $(OLLAMA_TARGET_URL)"
	@echo "vLLM Target: $(VLLM_TARGET_URL)"
	@echo ""
	@if [ -f $(ENV_FILE) ]; then \
		echo "=== Environment Variables ==="; \
		grep -E "(UV_CUBE_AGENT_TARGET_URL|VLLM_MODEL)" $(ENV_FILE) 2>/dev/null || echo "No AI backend variables configured"; \
	fi

.PHONY: clean-env
clean-env:
	@echo "Cleaning environment configuration..."
	@if [ -f $(ENV_FILE) ]; then \
		sed -i '/^UV_CUBE_AGENT_TARGET_URL=/d' $(ENV_FILE); \
		echo "Removed UV_CUBE_AGENT_TARGET_URL from $(ENV_FILE)"; \
	fi

.PHONY: enable-guardrails
enable-guardrails:
	@echo "Enabling guardrails in config.json..."
	@sed -i '/"name": "guardrails-agent"/,/"enabled":/{s/"enabled": false/"enabled": true/}' $(CONFIG_FILE)
	@sed -i '/"name": "forward-to-guardrails"/,/"enabled":/{s/"enabled": false/"enabled": true/}' $(CONFIG_FILE)
	@sed -i '/"name": "guardrails-admin"/,/"enabled":/{s/"enabled": false/"enabled": true/}' $(CONFIG_FILE)
	@echo "Guardrails enabled"

.PHONY: disable-guardrails
disable-guardrails:
	@echo "Disabling guardrails in config.json..."
	@sed -i '/"name": "guardrails-agent"/,/"enabled":/{s/"enabled": true/"enabled": false/}' $(CONFIG_FILE)
	@sed -i '/"name": "forward-to-guardrails"/,/"enabled":/{s/"enabled": true/"enabled": false/}' $(CONFIG_FILE)
	@sed -i '/"name": "guardrails-admin"/,/"enabled":/{s/"enabled": true/"enabled": false/}' $(CONFIG_FILE)
	@echo "Guardrails disabled"

# Help
.PHONY: help
help:
	@echo "Cube AI - Available Commands:"
	@echo ""
	@echo "Build Commands:"
	@echo "  build              Build all services"
	@echo "  build-proxy        Build proxy service"
	@echo "  build-agent        Build agent service"
	@echo "  docker             Build Docker images"
	@echo "  docker-guardrails  Build Nemo Guardrails Docker image"
	@echo "  docker-dev         Build development Docker images"
	@echo ""
	@echo "Configuration Commands:"
	@echo "  config-ollama           Configure for Ollama backend"
	@echo "  config-vllm             Configure for vLLM backend"
	@echo "  config-guardrails-vllm  Configure guardrails config.yml for vLLM"
	@echo "  config-guardrails-ollama Configure guardrails config.yml for Ollama"
	@echo "  enable-guardrails       Enable guardrails routes in config.json"
	@echo "  disable-guardrails      Disable guardrails routes in config.json"
	@echo "  show-config             Show current configuration"
	@echo "  clean-env               Clean environment configuration"
	@echo ""
	@echo "Deployment Commands:"
	@echo "  up                      Start with guardrails enabled (default)"
	@echo "  up-disable-guardrails   Start without guardrails"
	@echo "  up-ollama               Start with Ollama backend (pulls models automatically)"
	@echo "  up-vllm                 Start with vLLM backend"
	@echo "  up-vllm-guardrails      Start with vLLM backend and guardrails enabled"
	@echo "  up-cloud                Start cloud deployment using cloud-compose.yaml"
	@echo "  down                    Stop all services"
	@echo "  down-cloud              Stop cloud services and restore config"
	@echo "  down-volumes            Stop all services and remove volumes"
	@echo "  down-cloud-volumes      Stop cloud services, remove volumes, and restore config"
	@echo "  restart                 Restart with configured backend"
	@echo "  restart-cloud           Restart cloud deployment"
	@echo ""
	@echo "Cloud Configuration Commands:"
	@echo "  config-cloud-local Configure cloud deployment with localhost defaults"
	@echo "  restore-cloud-config Restore placeholder values in cloud config files"
	@echo ""
	@echo "Logs:"
	@echo "  logs               Show all logs"
	@echo "  logs-cloud         Show cloud deployment logs"
	@echo ""
	@echo "Examples:"
	@echo "  make up                              # Start with guardrails (default)"
	@echo "  make up-disable-guardrails           # Start without guardrails"
	@echo "  make up AI_BACKEND=vllm              # Start with vLLM + guardrails"
	@echo "  make up-ollama                       # Start with Ollama (pulls models)"
	@echo "  make up-cloud                        # Start cloud deployment locally"



all: build docker-dev

clean:
	rm -rf build

lint:
	golangci-lint run --config .golangci.yaml

.PHONY: latest
latest: docker docker-push

.PHONY: docker-push
docker-push: docker-push-proxy docker-push-agent docker-push-guardrails

.PHONY: docker-push-proxy
docker-push-proxy:
	$(call docker_push,$(CUBE_PROXY_DOCKER_IMAGE_NAME))

.PHONY: docker-push-agent
docker-push-agent:
	$(call docker_push,$(CUBE_AGENT_DOCKER_IMAGE_NAME))

.PHONY: mocks
mocks:
	mockery --config ./.mockery.yml

.PHONY: docker-push-guardrails
docker-push-guardrails:
	$(call docker_push,$(CUBE_GUARDRAILS_DOCKER_IMAGE_NAME))

.DEFAULT_GOAL := help
