# Copyright 2017, 2019, 2020 the Velero contributors.
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

PKG := github.com/vmware-tanzu/velero-plugin-example
BIN := velero-plugin-example

REGISTRY ?= quay.io/rbutter
IMAGE    ?= $(REGISTRY)/velero-restore-plugin
VERSION  ?= latest 
CONTAINER_ENGINE ?= podman

GOOS   ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# local builds the binary using 'go build' in the local environment.
.PHONY: local
local: build-dirs
	CGO_ENABLED=0 go build -v -o _output/bin/$(GOOS)/$(GOARCH) .

# test runs unit tests using 'go test' in the local environment.
.PHONY: test
test:
	CGO_ENABLED=0 go test -v -timeout 60s ./...

# ci is a convenience target for CI builds.
.PHONY: ci
ci: verify-modules local test

# container builds a Docker image containing the binary.
.PHONY: container
container:
	$(CONTAINER_ENGINE) build -t $(IMAGE):$(VERSION) .

# push pushes the Docker image to its registry.
.PHONY: push
push:
	@$(CONTAINER_ENGINE) push $(IMAGE):$(VERSION)
ifeq ($(TAG_LATEST), true)
	$(CONTAINER_ENGINE) tag $(IMAGE):$(VERSION) $(IMAGE):latest
	$(CONTAINER_ENGINE) push $(IMAGE):latest
endif

# modules updates Go module files
.PHONY: modules
modules:
	go mod tidy

# verify-modules ensures Go module files are up to date
.PHONY: verify-modules
verify-modules: modules
	@if !(git diff --quiet HEAD -- go.sum go.mod); then \
		echo "go module files are out of date, please commit the changes to go.mod and go.sum"; exit 1; \
	fi

# build-dirs creates the necessary directories for a build in the local environment.
.PHONY: build-dirs
build-dirs:
	@mkdir -p _output/bin/$(GOOS)/$(GOARCH)

# clean removes build artifacts from the local environment.
.PHONY: clean
clean:
	@echo "cleaning"
	rm -rf _output

.PHONY: build-push-patch
build-push-patch:
	$(CONTAINER_ENGINE) build -f patch.dockerfile . -t $(REGISTRY)/managed-velero-plugin-status-patch
	$(CONTAINER_ENGINE) push $(REGISTRY)/managed-velero-plugin-status-patch

.PHONY: all
all: build-push-patch container push


install:
	velero plugin add $(IMAGE):$(VERSION) -n openshift-velero

remove: uninstall
uninstall:
	velero plugin remove $(IMAGE):$(VERSION) -n openshift-velero