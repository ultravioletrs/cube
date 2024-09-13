CUBE_UI_DOCKER_IMAGE_NAME ?= ghcr.io/ultravioletrs/cube/ui
VERSION ?= $(shell git describe --abbrev=0 --tags 2>/dev/null || echo 'v0.0.0')

define make_docker
	docker build \
		--no-cache \
		--tag=$(CUBE_UI_DOCKER_IMAGE_NAME):$(VERSION) \
		--tag=$(CUBE_UI_DOCKER_IMAGE_NAME):latest \
		-f ./docker/ui.prod.Dockerfile .
endef

define make_docker_dev
	docker build \
		--no-cache \
		--tag=$(CUBE_UI_DOCKER_IMAGE_NAME):$(VERSION) \
		--tag=$(CUBE_UI_DOCKER_IMAGE_NAME):latest \
		-f ./docker/ui.dev.Dockerfile .
endef

define docker_push
	docker push $(CUBE_UI_DOCKER_IMAGE_NAME):$(VERSION)
	docker push $(CUBE_UI_DOCKER_IMAGE_NAME):latest
endef

.PHONY: docker
docker:
	$(call make_docker)

.PHONY: docker-dev
docker_dev:
	$(call make_docker_dev)

latest: docker
	$(call docker_push)
