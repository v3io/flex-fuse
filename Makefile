RPM_PATH = "iguazio_yum"
DEB_PATH = "iguazio_deb"
BINARY_NAME = "igz-fuse"
RELEASE_VERSION = "0.4.0"

.PHONY: build
build:
	docker build --tag iguaziodocker/flex-fuse:unstable .

.PHONY: download
download:
	@rm -rf hack/libs/${BINARY_NAME}*
	@cd hack/libs && wget --quiet $(MIRROR)/$(RPM_PATH)/$(IGUAZIO_VERSION)/$(BINARY_NAME).rpm
	@cd hack/libs && wget --quiet $(MIRROR)/$(DEB_PATH)/$(IGUAZIO_VERSION)/$(BINARY_NAME).deb

.PHONY: release
release: check-req download build
	docker tag iguaziodocker/flex-fuse:unstable iguaziodocker/flex-fuse:$(IGUAZIO_VERSION)-$(RELEASE_VERSION)

check-req:
ifndef MIRROR
	$(error MIRROR must be set)
endif
ifndef IGUAZIO_VERSION
	$(error IGUAZIO_VERSION must be set)
endif
ifndef RELEASE_VERSION
	$(error RELEASE_VERSION must be set)
endif