# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

################################################################################
#
# ollama
#
################################################################################

OLLAMA_VERSION = 0.12.3
OLLAMA_SITE = https://github.com/ollama/ollama/releases/download/v$(OLLAMA_VERSION)
OLLAMA_LICENSE = MIT
OLLAMA_LICENSE_FILES = LICENSE
OLLAMA_CPE_ID_VENDOR = ollama

# Set architecture
ifeq ($(BR2_aarch64),y)
OLLAMA_ARCH = arm64
else ifeq ($(BR2_x86_64),y)
OLLAMA_ARCH = amd64
endif

OLLAMA_SOURCE = ollama-linux-$(OLLAMA_ARCH).tgz

# GPU support
ifeq ($(BR2_PACKAGE_OLLAMA_GPU_NVIDIA),y)
OLLAMA_DEPENDENCIES += nvidia-driver
endif

ifeq ($(BR2_PACKAGE_OLLAMA_GPU_AMD),y)
OLLAMA_DEPENDENCIES += rocm
endif

define OLLAMA_INSTALL_TARGET_CMDS
	cp -dpfr $(@D)/* $(TARGET_DIR)/usr/
endef

define OLLAMA_INSTALL_INIT_SYSV
	$(INSTALL) -D -m 0755 $(BR2_EXTERNAL_CUBE_PATH)/package/ollama/S96ollama \
		$(TARGET_DIR)/etc/init.d/S96ollama
endef

define OLLAMA_INSTALL_INIT_SYSTEMD
	$(INSTALL) -D -m 0644 $(BR2_EXTERNAL_CUBE_PATH)/package/ollama/ollama.service \
		$(TARGET_DIR)/usr/lib/systemd/system/ollama.service
endef

define OLLAMA_INSTALL_MODEL_SCRIPT
	$(INSTALL) -d -m 0755 $(TARGET_DIR)/usr/libexec/ollama
	echo '#!/bin/sh' > $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
	echo 'pull_model() {' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
	echo '  retries=0' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
	echo '  while [ $$retries -lt 20 ]; do' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
	echo '    if ollama pull "$$1"; then' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
	echo '      return 0' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
	echo '    fi' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
	echo '    sleep 5' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
	echo '    retries=$$((retries + 1))' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
	echo '  done' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
	echo '  return 1' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
	echo '}' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
	echo 'sleep 10' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
	$(if $(BR2_PACKAGE_OLLAMA_MODELS), \
		echo 'pull_model llama3.2:3b' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh && \
		echo 'pull_model starcoder2:3b' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh && \
		echo 'pull_model nomic-embed-text:v1.5' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh)
	$(if $(call qstrip,$(BR2_PACKAGE_OLLAMA_CUSTOM_MODELS)), \
		$(foreach model,$(call qstrip,$(BR2_PACKAGE_OLLAMA_CUSTOM_MODELS)), \
			echo 'pull_model $(model)' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh;))
	chmod +x $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
endef

OLLAMA_POST_INSTALL_TARGET_HOOKS += OLLAMA_INSTALL_MODEL_SCRIPT

define OLLAMA_USERS
	ollama -1 ollama -1 * /var/lib/ollama - - Ollama Service
endef

$(eval $(generic-package))
