RPM_PATH = "iguazio_yum"
DEB_PATH = "iguazio_deb"
BINARY_NAME = "igz-fuse"
RELEASE_VERSION = "0.6.1"
DOCKER_HUB_USER = "iguaziodocker"
QUAY_IO_USER = "quay.io/iguazio"
FULL_VERSION = $(IGUAZIO_VERSION:igz_%=%)-$(RELEASE_VERSION)

.PHONY: build
build: lint vet
	docker build --tag $(DOCKER_HUB_USER)/flex-fuse:unstable .

.PHONY: download
download:
	@rm -rf hack/libs/${BINARY_NAME}*
	@cd hack/libs && wget --quiet $(MIRROR)/$(RPM_PATH)/$(IGUAZIO_VERSION)/$(BINARY_NAME).rpm
	@cd hack/libs && wget --quiet $(MIRROR)/$(DEB_PATH)/$(IGUAZIO_VERSION)/$(BINARY_NAME).deb
	@touch hack/libs/$(IGUAZIO_VERSION)

.PHONY: release
release: check-req download build
	docker tag $(DOCKER_HUB_USER)/flex-fuse:unstable $(DOCKER_HUB_USER)/flex-fuse:$(FULL_VERSION)
	docker tag $(DOCKER_HUB_USER)/flex-fuse:unstable $(QUAY_IO_USER)/flex-fuse:$(FULL_VERSION)
	echo $(FULL_VERSION) > VERSION

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

.PHONY: lint
lint: ensure-gopath
	@echo Installing linters...
	go get -u gopkg.in/alecthomas/gometalinter.v2
	@$(GOPATH)/bin/gometalinter.v2 --install

	@echo Linting...
	@$(GOPATH)/bin/gometalinter.v2 \
		--deadline=300s \
		--disable-all \
		--enable-gc \
		--enable=deadcode \
		--enable=goconst \
		--enable=gofmt \
		--enable=golint \
		--enable=gosimple \
		--enable=ineffassign \
		--enable=interfacer \
		--enable=misspell \
		--enable=staticcheck \
		--enable=unconvert \
		--enable=varcheck \
		--enable=vet \
		--enable=vetshadow \
		--enable=errcheck \
		--exclude="_test.go" \
		--exclude="comment on" \
		--exclude="error should be the last" \
		--exclude="should have comment" \
		./cmd/... ./pkg/...

	@echo Done.

.PHONY: vet
vet:
	go vet ./cmd/...
	go vet ./pkg/...

.PHONY: test
test:
	go test -v ./cmd/...
	go test -v ./pkg/...

.PHONY: ensure-gopath
check-gopath:
ifndef GOPATH
    $(error GOPATH must be set)
endif
