################################################################################
#
# ollama
#
################################################################################

OLLAMA_VERSION = 0.3.12
OLLAMA_SITE = $(call github,ollama,ollama,v$(OLLAMA_VERSION))
OLLAMA_LICENSE = MIT
OLLAMA_LICENSE_FILES = LICENSE
OLLAMA_CPE_ID_VENDOR = ollama

OLLAMA_LDFLAGS = -s -w

# GPU support
ifeq ($(BR2_PACKAGE_OLLAMA_GPU_NVIDIA),y)
OLLAMA_DEPENDENCIES += nvidia-driver
endif

ifeq ($(BR2_PACKAGE_OLLAMA_GPU_AMD),y)
OLLAMA_DEPENDENCIES += rocm
endif

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
	echo 'sleep 10' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
	$(if $(BR2_PACKAGE_OLLAMA_MODELS), \
		echo 'ollama pull tinyllama:1.1b' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh && \
		echo 'ollama pull starcoder2:3b' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh && \
		echo 'ollama pull nomic-embed-text:v1.5' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh)
	$(if $(call qstrip,$(BR2_PACKAGE_OLLAMA_CUSTOM_MODELS)), \
		$(foreach model,$(call qstrip,$(BR2_PACKAGE_OLLAMA_CUSTOM_MODELS)), \
			echo 'ollama pull $(model)' >> $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh;))
	chmod +x $(TARGET_DIR)/usr/libexec/ollama/pull-models.sh
endef

OLLAMA_POST_INSTALL_TARGET_HOOKS += OLLAMA_INSTALL_MODEL_SCRIPT

define OLLAMA_USERS
	ollama -1 ollama -1 * /var/lib/ollama - - Ollama Service
endef

$(eval $(golang-package))
