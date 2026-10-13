# Copyright (C) 2024 Enflame Technologies, Inc.

WITH_ISGCU   ?= yes

ifeq ($(WITH_ISGCU), yes)
##### Read Release Version #####
VERSION_TXT ?= $(shell if [ -f "../../g_version.txt" ]; then echo "../../g_version.txt"; fi)
RELEASE_VERSION ?= $(shell if [ -n "$(VERSION_TXT)" ]; then grep -w 'GCU_FEATURE_DISCOVERY_VERSION' $(VERSION_TXT) | awk -F '=' '{print $$2}'; fi)
VERSION = $(RELEASE_VERSION)

ifeq ($(VERSION),)
    VERSION := 0.0.0
endif

all: binaries

GCU_FEATURE_DISCOVERY        := gcu-feature-discovery

clean:
	$(info Cleaning up build artifacts...)
	@if [ -f ./${GCU_FEATURE_DISCOVERY} ]; then $(RM) ${GCU_FEATURE_DISCOVERY}; fi

gomod:
	go list -mod=readonly -m -json all

endif

DOCKER   ?= docker
MKDIR    ?= mkdir
DIST_DIR ?= $(CURDIR)/dist

include $(CURDIR)/versions.mk

MODULE := enflame/gcu-feature-discovery

ifeq ($(IMAGE_NAME),)
REGISTRY ?= enflame
IMAGE_NAME = $(REGISTRY)/gcu-feature-discovery
endif

BUILDIMAGE_TAG ?= golang$(GOLANG_VERSION)
BUILDIMAGE ?= $(IMAGE_NAME)-build:$(BUILDIMAGE_TAG)

CMDS := $(patsubst ./cmd/%/,%,$(sort $(dir $(wildcard ./cmd/*/))))
CMD_TARGETS := $(patsubst %,cmd-%, $(CMDS))

$(info CMD_TARGETS=$(CMD_TARGETS))

CHECK_TARGETS := assert-fmt vet lint misspell
MAKE_TARGETS := binaries build check fmt lint-internal cmds coverage generate $(CHECK_TARGETS)

TARGETS := $(MAKE_TARGETS) $(CMD_TARGETS)

DOCKER_TARGETS := $(patsubst %,docker-%, $(TARGETS))
.PHONY: $(TARGETS) $(DOCKER_TARGETS)

GOOS ?= linux

binaries: cmds
ifneq ($(PREFIX),)
cmd-%: COMMAND_BUILD_OPTIONS = -o $(PREFIX)/$(*)
endif
cmds: $(CMD_TARGETS)
$(CMD_TARGETS): cmd-%:
	GOOS=$(GOOS) go build -ldflags "-s -w -X main.Version=$(VERSION)" $(COMMAND_BUILD_OPTIONS) $(MODULE)/cmd/$(*)

build:
	GOOS=$(GOOS) go build ./...

ifeq ($(WITH_ISGCU), no)
all: check build binaries
endif
check: $(CHECK_TARGETS)

# Apply go fmt to the codebase
fmt:
	go list -f '{{.Dir}}' $(MODULE)/... \
		| xargs gofmt -s -l -w

assert-fmt:
	go list -f '{{.Dir}}' $(MODULE)/... \
		| xargs gofmt -s -l > fmt.out
	@if [ -s fmt.out ]; then \
		echo "\nERROR: The following files are not formatted:\n"; \
		cat fmt.out; \
		rm fmt.out; \
		exit 1; \
	else \
		rm fmt.out; \
	fi

lint:
# We use `go list -f '{{.Dir}}' $(MODULE)/...` to skip the `vendor` folder.
	go list -f '{{.Dir}}' $(MODULE)/... | grep -v /internal/ | xargs golint -set_exit_status

lint-internal:
# We use `go list -f '{{.Dir}}' $(MODULE)/...` to skip the `vendor` folder.
	go list -f '{{.Dir}}' $(MODULE)/internal/... | xargs golint -set_exit_status

misspell:
	misspell $(MODULE)/...

vet:
	go vet $(MODULE)/...

generate:
	go generate $(MODULE)/...

# Generate an image for containerized builds
# Note: This image is local only
.PHONY: .build-image .pull-build-image .push-build-image
.build-image: docker/Dockerfile.devel
	if [ x"$(SKIP_IMAGE_BUILD)" = x"" ]; then \
		$(DOCKER) build \
			--progress=plain \
			--build-arg GOLANG_VERSION="$(GOLANG_VERSION)" \
			--tag $(BUILDIMAGE) \
			-f $(^) \
			docker; \
	fi

.pull-build-image:
	$(DOCKER) pull $(BUILDIMAGE)

.push-build-image:
	$(DOCKER) push $(BUILDIMAGE)

$(DOCKER_TARGETS): docker-%: .build-image
	@echo "Running 'make $(*)' in docker container $(BUILDIMAGE)"
	$(DOCKER) run \
		--rm \
		-e GOCACHE=/tmp/.cache \
		-v $(PWD):$(PWD) \
		-w $(PWD) \
		--user $$(id -u):$$(id -g) \
		$(BUILDIMAGE) \
			make $(*)
