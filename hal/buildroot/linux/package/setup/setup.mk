define SETUP_INSTALL_TARGET_CMDS
	mkdir -p $(TARGET_DIR)/cube/
endef

define SETUP_INSTALL_INIT_SYSTEMD
	cp ../cube/hal/buildroot/linux/systemd/cube.service $(TARGET_DIR)/usr/lib/systemd/system/cube.service
	cp ../cube/hal/buildroot/linux/systemd/setup-cube.sh $(TARGET_DIR)/cube/setup-cube.sh
	cp ../cube/hal/buildroot/linux/systemd/attest.sh $(TARGET_DIR)/cube/attest.sh
endef

$(eval $(generic-package))
