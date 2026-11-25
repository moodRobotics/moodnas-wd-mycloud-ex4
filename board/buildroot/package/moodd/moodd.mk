################################################################################
#
# moodd — built from this repository (the Go module at the repo root).
# The web UI is built on the host beforehand (`make ui`) and installed as
# static files.
#
################################################################################

MOODD_VERSION = local
MOODD_SITE = $(BR2_EXTERNAL_MOODNAS_PATH)/../..
MOODD_SITE_METHOD = local
MOODD_LICENSE = GPL-3.0-or-later
MOODD_LICENSE_FILES = LICENSE
MOODD_BUILD_TARGETS = cmd/moodd
MOODD_LDFLAGS = -s -w -X main.version=$(MOODD_VERSION)

define MOODD_INSTALL_UI
	mkdir -p $(TARGET_DIR)/usr/share/moodnas
	cp -a $(@D)/ui/dist $(TARGET_DIR)/usr/share/moodnas/ui
	$(INSTALL) -D -m 0755 $(BR2_EXTERNAL_MOODNAS_PATH)/package/moodd/S90moodd \
		$(TARGET_DIR)/etc/init.d/S90moodd
endef
MOODD_POST_INSTALL_TARGET_HOOKS += MOODD_INSTALL_UI

$(eval $(golang-package))
