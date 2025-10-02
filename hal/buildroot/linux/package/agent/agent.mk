# Copyright (c) Ultraviolet
# SPDX-License-Identifier: Apache-2.0

################################################################################
#
# cube-agent
#
################################################################################

CUBE_AGENT_VERSION = main
CUBE_AGENT_SITE = $(call github,ultravioletrs,cube,$(CUBE_AGENT_VERSION))
CUBE_AGENT_LICENSE = Apache-2.0
CUBE_AGENT_LICENSE_FILES = LICENSE
CUBE_AGENT_CPE_ID_VENDOR = ultraviolet

CUBE_AGENT_LDFLAGS = \
	-s -w \
	-X 'github.com/absmach/supermq.BuildTime=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')' \
	-X 'github.com/absmach/supermq.Version=$(CUBE_AGENT_VERSION)' \
	-X 'github.com/absmach/supermq.Commit=$(CUBE_AGENT_VERSION)'

CUBE_AGENT_BUILD_TARGETS = cmd/agent

# Set target URL based on backend selection
ifeq ($(BR2_PACKAGE_CUBE_AGENT_BACKEND_OLLAMA),y)
CUBE_AGENT_TARGET_URL = http://localhost:11434
else ifeq ($(BR2_PACKAGE_CUBE_AGENT_BACKEND_VLLM),y)
CUBE_AGENT_TARGET_URL = http://localhost:8000
else ifeq ($(BR2_PACKAGE_CUBE_AGENT_BACKEND_CUSTOM),y)
CUBE_AGENT_TARGET_URL = $(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_TARGET_URL))
else
CUBE_AGENT_TARGET_URL = http://localhost:11434
endif

define CUBE_AGENT_INSTALL_INIT_SYSV
	$(INSTALL) -D -m 0755 $(BR2_EXTERNAL_CUBE_PATH)/package/cube-agent/S95cube-agent \
		$(TARGET_DIR)/etc/init.d/S95cube-agent
endef

define CUBE_AGENT_INSTALL_INIT_SYSTEMD
	$(INSTALL) -D -m 0644 $(BR2_EXTERNAL_CUBE_PATH)/package/cube-agent/cube-agent.service \
		$(TARGET_DIR)/usr/lib/systemd/system/cube-agent.service
endef

define CUBE_AGENT_INSTALL_CONFIG
	$(INSTALL) -d -m 0755 $(TARGET_DIR)/etc/cube
	echo "UV_CUBE_AGENT_LOG_LEVEL=$(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_LOG_LEVEL))" > $(TARGET_DIR)/etc/cube/agent.env
	echo "UV_CUBE_AGENT_HOST=$(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_HOST))" >> $(TARGET_DIR)/etc/cube/agent.env
	echo "UV_CUBE_AGENT_PORT=$(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_PORT))" >> $(TARGET_DIR)/etc/cube/agent.env
	echo "UV_CUBE_AGENT_INSTANCE_ID=$(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_INSTANCE_ID))" >> $(TARGET_DIR)/etc/cube/agent.env
	echo "UV_CUBE_AGENT_TARGET_URL=$(CUBE_AGENT_TARGET_URL)" >> $(TARGET_DIR)/etc/cube/agent.env
endef

CUBE_AGENT_POST_INSTALL_TARGET_HOOKS += CUBE_AGENT_INSTALL_CONFIG

$(eval $(golang-package))
