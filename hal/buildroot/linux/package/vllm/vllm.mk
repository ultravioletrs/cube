# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

################################################################################
#
# vllm
#
################################################################################

VLLM_VERSION = 0.10.2
VLLM_SITE = $(call github,vllm-project,vllm,v$(VLLM_VERSION))
VLLM_LICENSE = Apache-2.0
VLLM_LICENSE_FILES = LICENSE
VLLM_SETUP_TYPE = setuptools
VLLM_DEPENDENCIES = python-pytorch python-transformers python-numpy

# Copy custom model if specified
ifneq ($(call qstrip,$(BR2_PACKAGE_VLLM_CUSTOM_MODEL_PATH)),)
define VLLM_INSTALL_CUSTOM_MODEL
	$(INSTALL) -d -m 0755 $(TARGET_DIR)/var/lib/vllm/models
	cp -r $(call qstrip,$(BR2_PACKAGE_VLLM_CUSTOM_MODEL_PATH))/* \
		$(TARGET_DIR)/var/lib/vllm/models/
	chown -R vllm:vllm $(TARGET_DIR)/var/lib/vllm/models
endef
VLLM_POST_INSTALL_TARGET_HOOKS += VLLM_INSTALL_CUSTOM_MODEL
endif

define VLLM_INSTALL_INIT_SYSV
	$(INSTALL) -D -m 0755 $(BR2_EXTERNAL_CUBE_PATH)/package/vllm/S96vllm \
		$(TARGET_DIR)/etc/init.d/S96vllm
endef

define VLLM_INSTALL_INIT_SYSTEMD
	$(INSTALL) -D -m 0644 $(BR2_EXTERNAL_CUBE_PATH)/package/vllm/vllm.service \
		$(TARGET_DIR)/usr/lib/systemd/system/vllm.service
endef

define VLLM_INSTALL_CONFIG
	$(INSTALL) -d -m 0755 $(TARGET_DIR)/etc/vllm
	echo "VLLM_MODEL=$(call qstrip,$(BR2_PACKAGE_VLLM_MODEL))" > $(TARGET_DIR)/etc/vllm/vllm.env
	echo "VLLM_GPU_MEMORY_UTILIZATION=$(call qstrip,$(BR2_PACKAGE_VLLM_GPU_MEMORY))" >> $(TARGET_DIR)/etc/vllm/vllm.env
	echo "VLLM_MAX_MODEL_LEN=$(call qstrip,$(BR2_PACKAGE_VLLM_MAX_MODEL_LEN))" >> $(TARGET_DIR)/etc/vllm/vllm.env
	$(if $(call qstrip,$(BR2_PACKAGE_VLLM_CUSTOM_MODEL_PATH)), \
		echo "VLLM_MODEL=/var/lib/vllm/models" >> $(TARGET_DIR)/etc/vllm/vllm.env)
endef

VLLM_POST_INSTALL_TARGET_HOOKS += VLLM_INSTALL_CONFIG

define VLLM_USERS
	vllm -1 vllm -1 * /var/lib/vllm - - vLLM Service
endef

$(eval $(python-package))
