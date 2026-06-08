# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

CUBE_PROXY_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/proxy
CUBE_AGENT_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/agent
CUBE_EMBEDDER_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/embedder
CUBE_GUARDRAILS_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/guardrails
CUBE_IMAGE_EMBEDDER_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/image-embedder
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
COMPOSE_PROFILE = $(if $(filter vllm,$(AI_BACKEND)),vllm,default)

ENV_FILE = ./docker/.env
CONFIG_FILE = ./docker/config.json
COMPOSE_FILE = ./docker/compose.yaml
COMPOSE = docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE)
LOCAL_UI_URL ?= https://localhost
LOCAL_ADMIN_IDENTITY ?= admin
LOCAL_ADMIN_PASSWORD ?= m2N2Lfno
LOCAL_READY_RETRIES ?= 60
LOCAL_READY_SLEEP ?= 5
OLLAMA_MODELS ?= llama3.2:3b starcoder2:3b nomic-embed-text:v1.5

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
build: build-proxy build-agent build-embedder

.PHONY: build-proxy
build-proxy:
	$(call compile_service,proxy)

.PHONY: build-agent
build-agent:
	$(call compile_service,agent)

.PHONY: build-embedder
build-embedder:
	$(call compile_service,embedder)

.PHONY: docker
docker: docker-proxy docker-agent docker-embedder docker-guardrails docker-image-embedder

.PHONY: docker-proxy
docker-proxy:
	$(call make_docker,proxy,$(CUBE_PROXY_DOCKER_IMAGE_NAME))

.PHONY: docker-agent
docker-agent:
	$(call make_docker,agent,$(CUBE_AGENT_DOCKER_IMAGE_NAME))

.PHONY: docker-embedder
docker-embedder:
	$(call make_docker,embedder,$(CUBE_EMBEDDER_DOCKER_IMAGE_NAME))

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

.PHONY: docker-image-embedder
docker-image-embedder:
	docker build \
		--tag=$(CUBE_IMAGE_EMBEDDER_DOCKER_IMAGE_NAME):$(VERSION) \
		--tag=$(CUBE_IMAGE_EMBEDDER_DOCKER_IMAGE_NAME):latest \
		-f docker/Dockerfile.image-embedder .

.PHONY: guardrails-venv
guardrails-venv:
	@echo "Setting up guardrails virtual environment in root .venv..."
	python -m venv .venv
	. .venv/bin/activate && pip install --upgrade pip && pip install -r guardrails/requirements.txt
	. .venv/bin/activate && python -m spacy download en_core_web_lg
	@echo "Guardrails venv created successfully at .venv"

.PHONY: docker-dev
docker-dev: docker-proxy-dev docker-agent-dev docker-embedder-dev docker-guardrails-dev

.PHONY: docker-proxy-dev
docker-proxy-dev:
	$(call make_docker_dev,proxy,$(CUBE_PROXY_DOCKER_IMAGE_NAME))

.PHONY: docker-agent-dev
docker-agent-dev:
	$(call make_docker_dev,agent,$(CUBE_AGENT_DOCKER_IMAGE_NAME))

.PHONY: docker-embedder-dev
docker-embedder-dev:
	$(call make_docker_dev,embedder,$(CUBE_EMBEDDER_DOCKER_IMAGE_NAME))

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
	$(COMPOSE) --profile default up -d
	@$(MAKE) wait-local
	@$(MAKE) wait-ollama-models

.PHONY: up-vllm
up-vllm: config-vllm
	@echo "Starting Cube with vLLM backend..."
	$(COMPOSE) --profile vllm up -d
	@$(MAKE) wait-local

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

.PHONY: disable-atls
disable-atls:
	@if grep -q '^UV_CUBE_AGENT_ATTESTED_TLS=true' docker/.env; then \
		echo "Disabling attested TLS for local development..."; \
		sed -i 's|^UV_CUBE_AGENT_CLIENT_CERT=.*|UV_CUBE_AGENT_CLIENT_CERT=|' docker/.env; \
		sed -i 's|^UV_CUBE_AGENT_CLIENT_KEY=.*|UV_CUBE_AGENT_CLIENT_KEY=|' docker/.env; \
		sed -i 's|^UV_CUBE_AGENT_SERVER_CA_CERTS=.*|UV_CUBE_AGENT_SERVER_CA_CERTS=|' docker/.env; \
		sed -i 's|^UV_CUBE_AGENT_ATTESTED_TLS=.*|UV_CUBE_AGENT_ATTESTED_TLS=false|' docker/.env; \
		sed -i 's|^UV_CUBE_AGENT_ATTESTATION_POLICY=.*|UV_CUBE_AGENT_ATTESTATION_POLICY=|' docker/.env; \
		echo "✓ Attested TLS disabled"; \
	else \
		echo "✓ Attested TLS already configured, skipping"; \
	fi

.PHONY: up
up: check-local config-local enable-guardrails config-backend disable-atls
ifeq ($(AI_BACKEND),vllm)
	@$(MAKE) up-vllm
else
	@$(MAKE) up-ollama
endif

.PHONY: up-disable-guardrails
up-disable-guardrails: check-local config-cloud-local disable-guardrails config-backend disable-atls
ifeq ($(AI_BACKEND),vllm)
	@$(MAKE) up-vllm
else
	@$(MAKE) up-ollama
endif

.PHONY: check-local
check-local:
	@command -v docker >/dev/null 2>&1 || { echo "Docker is required."; exit 1; }
	@docker compose version >/dev/null 2>&1 || { echo "Docker Compose v2 is required."; exit 1; }
	@docker info >/dev/null 2>&1 || { echo "Docker daemon is not running or is not accessible."; exit 1; }
	@command -v curl >/dev/null 2>&1 || { echo "curl is required for startup readiness checks."; exit 1; }
	@test -f $(ENV_FILE) || { echo "Missing $(ENV_FILE)."; exit 1; }
	@test -f $(COMPOSE_FILE) || { echo "Missing $(COMPOSE_FILE)."; exit 1; }
	@echo "✓ Local prerequisites are available"

.PHONY: wait-local
wait-local:
	@echo "Waiting for Cube local services..."
	@i=1; \
	while [ $$i -le $(LOCAL_READY_RETRIES) ]; do \
		code=$$(curl -k -sS -o /dev/null -w "%{http_code}" -X POST "$(LOCAL_UI_URL)/users/tokens/issue" \
			-H "Content-Type: application/json" \
			-d '{"username":"$(LOCAL_ADMIN_IDENTITY)","password":"$(LOCAL_ADMIN_PASSWORD)"}' 2>/dev/null || true); \
		if [ "$$code" = "201" ]; then \
			echo "✓ Cube is ready at $(LOCAL_UI_URL)"; \
			exit 0; \
		fi; \
		if [ $$i -eq 12 ]; then \
			echo "Restarting dependent services while auth settles..."; \
			$(COMPOSE) --profile $(COMPOSE_PROFILE) restart users magistrala-backend ui >/dev/null 2>&1 || true; \
		fi; \
		echo "Waiting for services ($$i/$(LOCAL_READY_RETRIES), login status: $${code:-unreachable})..."; \
		sleep $(LOCAL_READY_SLEEP); \
		i=$$((i + 1)); \
	done; \
	echo "Cube did not become ready. Run 'make ps' or 'make logs' to inspect services."; \
	exit 1

.PHONY: wait-ollama-models
wait-ollama-models:
	@echo "Waiting for Ollama models: $(OLLAMA_MODELS)"
	@i=1; \
	while [ $$i -le $(LOCAL_READY_RETRIES) ]; do \
		models="$$(docker exec ollama ollama list 2>/dev/null || true)"; \
		missing=""; \
		for model in $(OLLAMA_MODELS); do \
			echo "$$models" | awk 'NR > 1 { print $$1 }' | grep -Fxq "$$model" || missing="$$missing $$model"; \
		done; \
		if [ -z "$$missing" ]; then \
			echo "✓ Required Ollama models are ready"; \
			exit 0; \
		fi; \
		echo "Waiting for models ($$i/$(LOCAL_READY_RETRIES)):$$missing"; \
		sleep $(LOCAL_READY_SLEEP); \
		i=$$((i + 1)); \
	done; \
	echo "Required Ollama models were not downloaded. Run 'docker logs pull-llama',"; \
	echo "'docker logs pull-starcoder2', and 'docker logs pull-nomic-embed-text'."; \
	exit 1

.PHONY: config-local
config-local:
	@echo "Configuring for local development..."
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
	@sed -i 's|__MG_MAILCHIMP_API_KEY__||g' docker/.env
	@sed -i 's|__MG_MAILCHIMP_SERVER_PREFIX__||g' docker/.env
	@sed -i 's|__MG_MAILCHIMP_AUDIENCE_ID__||g' docker/.env
	@sed -i 's|__CUBE_PUBLIC_URL__|localhost|g' docker/.env
	@sed -i 's|__TRAEFIK_HTTP_PORT__|80|g' docker/.env
	@sed -i 's|__TRAEFIK_HTTPS_PORT__|443|g' docker/.env
	@sed -i 's|__TRAEFIK_DASHBOARD_PORT__|8080|g' docker/.env
	@sed -i 's|__TUNNEL_TOKEN__||g' docker/.env
	@sed -i 's|__CUBE_AGENT_CERTS_TOKEN__|localdevtoken12we12we12we12we12we|g' docker/.env
	@sed -i "s|__NEXTAUTH_SECRET__|$(shell python3 -c 'import secrets; print(secrets.token_urlsafe(37))')|g" docker/.env
	@echo "✓ Configured with local defaults"

.PHONY: restore-config
restore-config:
	@echo "Restoring configuration placeholders..."
	@git checkout -- docker/.env docker/traefik/dynamic.toml docker/config.json $(GUARDRAILS_CONFIG_FILE) 2>/dev/null && \
		echo "✓ Restored from git" || echo "⚠ git restore failed, files may not be tracked"

.PHONY: down
down:
	@echo "Stopping all Cube services..."
	$(COMPOSE) --profile $(COMPOSE_PROFILE) down

.PHONY: down-volumes
down-volumes:
	@echo "Stopping all Cube services and removing volumes..."
	$(COMPOSE) --profile $(COMPOSE_PROFILE) down -v

.PHONY: restart
restart: down up

.PHONY: restart-ollama
restart-ollama: down up-ollama

.PHONY: restart-vllm
restart-vllm: down up-vllm

.PHONY: logs
logs:
	$(COMPOSE) --profile $(COMPOSE_PROFILE) logs -f

.PHONY: ps
ps:
	$(COMPOSE) --profile $(COMPOSE_PROFILE) ps

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
	@echo "  docker-image-embedder Build image embedding sidecar Docker image"
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
	@echo "  check-local             Validate local Docker prerequisites"
	@echo "  wait-local              Wait until the local UI/login API is ready"
	@echo "  wait-ollama-models      Wait until required local models are available"
	@echo "  ps                      Show local Docker service status"
	@echo ""
	@echo "Cloud Configuration Commands:"
	@echo "  config-cloud-local Configure cloud deployment with localhost defaults"
	@echo "  restore-config       Restore placeholder values in config files"
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
docker-push: docker-push-proxy docker-push-agent docker-push-guardrails docker-push-image-embedder

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

.PHONY: docker-push-image-embedder
docker-push-image-embedder:
	$(call docker_push,$(CUBE_IMAGE_EMBEDDER_DOCKER_IMAGE_NAME))

.DEFAULT_GOAL := help
