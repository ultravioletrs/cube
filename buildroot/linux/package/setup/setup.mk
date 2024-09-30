define SETUP_INSTALL_TARGET_CMDS
	mkdir -p $(TARGET_DIR)/cube/
endef

define SETUP_INSTALL_INIT_SYSTEMD
	cp ../cube/buildroot/linux/systemd/cube.service $(TARGET_DIR)/usr/lib/systemd/system/cube.service
	cp ../cube/buildroot/linux/systemd/setup-cube.sh $(TARGET_DIR)/cube/setup-cube.sh
endef

$(eval $(generic-package))
