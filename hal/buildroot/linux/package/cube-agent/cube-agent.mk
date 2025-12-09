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

define CUBE_AGENT_BUILD_CMDS
	$(MAKE) -C $(@D) build-agent
endef

define CUBE_AGENT_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/build/cube-agent $(TARGET_DIR)/bin
	$(INSTALL) -d -m 0755 $(TARGET_DIR)/var/lib/cube
endef

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
	echo "AGENT_OS_BUILD=$(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_OS_BUILD))" >> $(TARGET_DIR)/etc/cube/agent.env
	echo "AGENT_OS_DISTRO=$(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_OS_DISTRO))" >> $(TARGET_DIR)/etc/cube/agent.env
	echo "AGENT_OS_TYPE=$(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_OS_TYPE))" >> $(TARGET_DIR)/etc/cube/agent.env
	echo "AGENT_VMPL=$(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_VMPL))" >> $(TARGET_DIR)/etc/cube/agent.env
	echo "UV_CUBE_AGENT_CA_URL=$(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_CA_URL))" >> $(TARGET_DIR)/etc/cube/agent.env
	echo "UV_CUBE_AGENT_ATTESTED_TLS=$(if $(BR2_PACKAGE_CUBE_AGENT_ATTESTED_TLS),true,false)" >> $(TARGET_DIR)/etc/cube/agent.env
	$(if $(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_SERVER_CA_CERTS)), \
		$(INSTALL) -D -m 0644 $(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_SERVER_CA_CERTS)) $(TARGET_DIR)/etc/cube/certs/server_ca.pem && \
		echo "UV_CUBE_AGENT_SERVER_CA_CERTS=/etc/cube/certs/server_ca.pem" >> $(TARGET_DIR)/etc/cube/agent.env, \
		echo "UV_CUBE_AGENT_SERVER_CA_CERTS=" >> $(TARGET_DIR)/etc/cube/agent.env)
	$(if $(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_SERVER_CERT)), \
		$(INSTALL) -D -m 0644 $(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_SERVER_CERT)) $(TARGET_DIR)/etc/cube/certs/server.crt && \
		echo "UV_CUBE_AGENT_SERVER_CERT=/etc/cube/certs/server.crt" >> $(TARGET_DIR)/etc/cube/agent.env, \
		echo "UV_CUBE_AGENT_SERVER_CERT=" >> $(TARGET_DIR)/etc/cube/agent.env)
	$(if $(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_SERVER_KEY)), \
		$(INSTALL) -D -m 0600 $(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_SERVER_KEY)) $(TARGET_DIR)/etc/cube/certs/server.key && \
		echo "UV_CUBE_AGENT_SERVER_KEY=/etc/cube/certs/server.key" >> $(TARGET_DIR)/etc/cube/agent.env, \
		echo "UV_CUBE_AGENT_SERVER_KEY=" >> $(TARGET_DIR)/etc/cube/agent.env)
	$(if $(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_CLIENT_CA_CERTS)), \
		$(INSTALL) -D -m 0644 $(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_CLIENT_CA_CERTS)) $(TARGET_DIR)/etc/cube/certs/client_ca.pem && \
		echo "UV_CUBE_AGENT_CLIENT_CA_CERTS=/etc/cube/certs/client_ca.pem" >> $(TARGET_DIR)/etc/cube/agent.env, \
		echo "UV_CUBE_AGENT_CLIENT_CA_CERTS=" >> $(TARGET_DIR)/etc/cube/agent.env)
	echo "UV_CUBE_AGENT_CERTS_TOKEN=$(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_CERTS_TOKEN))" >> $(TARGET_DIR)/etc/cube/agent.env
	echo "UV_CUBE_AGENT_CVM_ID=$(call qstrip,$(BR2_PACKAGE_CUBE_AGENT_CVM_ID))" >> $(TARGET_DIR)/etc/cube/agent.env
endef

CUBE_AGENT_POST_INSTALL_TARGET_HOOKS += CUBE_AGENT_INSTALL_CONFIG

$(eval $(generic-package))
