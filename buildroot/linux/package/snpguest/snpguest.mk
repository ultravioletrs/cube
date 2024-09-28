SNPGUEST_VERSION=v0.7.1
SNPGUEST_SITE = $(call github,virtee,snpguest,$(SNPGUEST_VERSION))

define SNPGUEST_CARGO_BUILD_OPTS
	--release
endef

define SNPGUEST_INSTALL_TARGET_CMDS
	$(INSTALL) -D -m 0755 $(@D)/target/release/snpguest $(TARGET_DIR)/bin
endef

$(eval $(cargo-package))
