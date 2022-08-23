# Copyright 2018 Iguazio
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
SRC_BINARY_NAME ?= igz-fuse
DST_BINARY_NAME ?= igz-fuse
FETCH_METHOD ?= download
MIRROR ?=
IGUAZIO_VERSION ?=
NAS_IP ?=
NAS_PASSWORD ?=

RPM_PATH = iguazio_yum
DEB_PATH = iguazio_deb

.PHONY: build
build:
	docker build --progress=plain --tag flex-fuse:unstable .

.PHONY: download
download:
	rm -rf hack/libs/${DST_BINARY_NAME}*
	wget --quiet $(MIRROR)/$(RPM_PATH)/$(IGUAZIO_VERSION)/$(SRC_BINARY_NAME).rpm -O hack/libs/$(DST_BINARY_NAME).rpm
	#wget --quiet $(MIRROR)/$(DEB_PATH)/$(IGUAZIO_VERSION)/$(SRC_BINARY_NAME).deb -O hack/libs/$(DST_BINARY_NAME).deb
	touch hack/libs/$(IGUAZIO_VERSION)

.PHONY: download-rsync
download-rsync:
	rm -rf hack/libs/${DST_BINARY_NAME}*
	rsync -avz \
		--rsh="sshpass -p $(NAS_PASSWORD) ssh -o StrictHostKeyChecking=no -l iguazio" \
 		iguazio@$(NAS_IP):/mnt/ztank01/nfs/offline_versions/$(IGUAZIO_VERSION)/deploy/artifacts/zeek-packages/$(SRC_BINARY_NAME).rpm \
 		hack/libs/$(DST_BINARY_NAME).rpm
	touch hack/libs/$(IGUAZIO_VERSION)

.PHONY: copy
copy:
	rm -rf hack/libs/${DST_BINARY_NAME}*
	cp $(MIRROR)/$(SRC_BINARY_NAME).rpm hack/libs/$(DST_BINARY_NAME).rpm
	#cp $(MIRROR)/$(SRC_BINARY_NAME).deb hack/libs/$(DST_BINARY_NAME).deb
	touch hack/libs/$(IGUAZIO_VERSION)

.PHONY: release
release: check-req $(FETCH_METHOD) build
	docker tag flex-fuse:unstable iguazio/flex-fuse:$(IGUAZIO_VERSION)

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


check-req:
ifndef MIRROR
	$(error MIRROR must be set)
endif
ifndef IGUAZIO_VERSION
	$(error IGUAZIO_VERSION must be set)
endif

ensure-gopath:
ifndef GOPATH
	$(error GOPATH must be set)
endif
