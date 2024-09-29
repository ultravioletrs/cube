SNPGUEST_VERSION = main
SNPGUEST_SITE = $(call github,virtee,snpguest,$(SNPGUEST_VERSION))
SNPGUEST_LICENSE = Apache-2.0
SNPGUEST_LICENSE_FILES = LICENSE

SNPGUEST_DEPENDENCIES = host-rustc

define SNPGUEST_BUILD_CMDS
	$(TARGET_MAKE_ENV) $(TARGET_CONFIGURE_OPTS) \
		$(HOST_DIR)/bin/cargo build --release --manifest-path=$(@D)/Cargo.toml
endef

define SNPGUEST_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/target/release/snpguest $(TARGET_DIR)/usr/bin/snpguest
endef

$(eval $(generic-package))
