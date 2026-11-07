# SPDX-FileCopyrightText: 2026 The provider-opencve authors
#
# SPDX-License-Identifier: Apache-2.0

# ====================================================================================
# Setup Project

PROJECT_NAME ?= provider-opencve
PROJECT_REPO ?= github.com/Cliftonz/$(PROJECT_NAME)

PLATFORMS ?= linux_amd64 linux_arm64

-include build/makelib/common.mk

# ====================================================================================
# Setup Output

-include build/makelib/output.mk

# ====================================================================================
# Setup Go

GO_TEST_PARALLEL := $(shell echo $$(( $(NPROCS) / 2 )))

GO_REQUIRED_VERSION ?= 1.27
GOLANGCILINT_VERSION ?= 2.14.0
GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/provider
GO_LDFLAGS += -X $(GO_PROJECT)/internal/version.Version=$(VERSION)
GO_SUBDIRS += cmd internal apis
-include build/makelib/golang.mk

# ====================================================================================
# Setup Kubernetes tools

KIND_VERSION = v0.32.0
CROSSPLANE_CLI_VERSION = v2.2.1
-include build/makelib/k8s_tools.mk

# ====================================================================================
# Setup Images

REGISTRY_ORGS ?= xpkg.upbound.io/cliftonz
IMAGES = $(PROJECT_NAME)
-include build/makelib/imagelight.mk

# ====================================================================================
# Setup XPKG

XPKG_REG_ORGS ?= xpkg.upbound.io/cliftonz
XPKG_REG_ORGS_NO_PROMOTE ?= xpkg.upbound.io/cliftonz
XPKGS = $(PROJECT_NAME)
-include build/makelib/xpkg.mk

# ====================================================================================
# Setup E2E

CROSSPLANE_VERSION = 2.2.1
-include build/makelib/controlplane.mk
-include build/makelib/local.xpkg.mk

# uptest v2.2.0 mangles JSON update parameters; the fix is on main but unreleased.
UPTEST_COMMIT = f85cc370e3dab6365919e58a4607584d4b260224
UPTEST := $(TOOLS_HOST_DIR)/uptest-$(UPTEST_COMMIT)
$(UPTEST):
	@$(INFO) installing uptest $(UPTEST_COMMIT)
	@rm -rf $(WORK_DIR)/uptest
	@git clone -q https://github.com/crossplane/uptest $(WORK_DIR)/uptest
	@git -C $(WORK_DIR)/uptest checkout -q $(UPTEST_COMMIT)
	@cd $(WORK_DIR)/uptest && $(GOHOST) build -o $(UPTEST) ./cmd/uptest
	@$(OK) installing uptest $(UPTEST_COMMIT)

UPTEST_LOCAL_DEPLOY_TARGET = local.xpkg.deploy.provider.$(PROJECT_NAME)
UPTEST_SETUP_SCRIPT = test/harness/setup.sh
-include build/makelib/uptest.mk

e2e.provider: $(YQ) local.xpkg.deploy.provider.$(PROJECT_NAME)

e2e-local:
	@test/harness/e2e.sh

e2e-release:
	$(if $(filter command line environment,$(origin VERSION)),,$(error VERSION is required, e.g. VERSION=v0.1.0))
	@E2E_PACKAGE=xpkg.upbound.io/cliftonz/provider-opencve:$(VERSION) test/harness/e2e.sh

.PHONY: e2e.provider e2e-local e2e-release

# ====================================================================================
# Fallthrough

fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

xpkg.build.provider-opencve: do.build.images

build.init: $(UP) $(CROSSPLANE_CLI)

# ====================================================================================
# Targets

submodules:
	@git submodule sync
	@git submodule update --init --recursive

run: go.build
	@$(INFO) Running the provider locally out-of-cluster . . .
	$(GO_OUT_DIR)/provider --debug

BATS_VERSION = v1.11.1
BATS := $(TOOLS_HOST_DIR)/bats-$(BATS_VERSION)/bin/bats
$(BATS):
	@$(INFO) installing bats $(BATS_VERSION)
	@mkdir -p $(TOOLS_HOST_DIR)/bats-$(BATS_VERSION)
	@curl -fsSLo $(TOOLS_HOST_DIR)/bats-$(BATS_VERSION).tar.gz https://github.com/bats-core/bats-core/archive/refs/tags/$(BATS_VERSION).tar.gz
	@echo "5c57ed9616b78f7fd8c553b9bae3c7c9870119edd727ec17dbd1185c599f79d9  $(TOOLS_HOST_DIR)/bats-$(BATS_VERSION).tar.gz" | sha256sum -c -
	@tar -xzf $(TOOLS_HOST_DIR)/bats-$(BATS_VERSION).tar.gz --strip-components=1 -C $(TOOLS_HOST_DIR)/bats-$(BATS_VERSION)
	@$(OK) installing bats $(BATS_VERSION)

CRDDIFF_VERSION = v0.12.1
CRDDIFF := $(TOOLS_HOST_DIR)/crddiff-$(CRDDIFF_VERSION)
$(CRDDIFF):
	@$(INFO) installing crddiff $(CRDDIFF_VERSION)
	@mkdir -p $(TOOLS_HOST_DIR)/tmp-crddiff
	@GOBIN=$(TOOLS_HOST_DIR)/tmp-crddiff $(GOHOST) install github.com/upbound/uptest/cmd/crddiff@$(CRDDIFF_VERSION)
	@mv $(TOOLS_HOST_DIR)/tmp-crddiff/crddiff $(CRDDIFF)
	@rm -fr $(TOOLS_HOST_DIR)/tmp-crddiff
	@$(OK) installing crddiff $(CRDDIFF_VERSION)

crd-breaking-changes: $(CRDDIFF)
	@hack/breaking-changes.sh $(CRDDIFF)

test-scripts: $(BATS) $(CRDDIFF)
	@git ls-files '*.sh' | xargs shellcheck
	@CRDDIFF=$(CRDDIFF) $(BATS) test/scripts

SETUP_ENVTEST_VERSION = v0.24.1
SETUP_ENVTEST := $(TOOLS_HOST_DIR)/setup-envtest-$(SETUP_ENVTEST_VERSION)
ENVTEST_K8S_VERSION = 1.36.x
$(SETUP_ENVTEST):
	@$(INFO) installing setup-envtest $(SETUP_ENVTEST_VERSION)
	@mkdir -p $(TOOLS_HOST_DIR)/tmp-setup-envtest
	@GOBIN=$(TOOLS_HOST_DIR)/tmp-setup-envtest $(GOHOST) install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(SETUP_ENVTEST_VERSION)
	@mv $(TOOLS_HOST_DIR)/tmp-setup-envtest/setup-envtest $(SETUP_ENVTEST)
	@rm -fr $(TOOLS_HOST_DIR)/tmp-setup-envtest
	@$(OK) installing setup-envtest $(SETUP_ENVTEST_VERSION)

test-envtest: $(SETUP_ENVTEST)
	@KUBEBUILDER_ASSETS="$$($(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(TOOLS_HOST_DIR)/envtest -p path)" $(GOHOST) test -tags envtest ./...

.PHONY: submodules fallthrough run crd-breaking-changes test-scripts test-envtest

# ====================================================================================
# Special Targets

define CROSSPLANE_MAKE_HELP
Crossplane Targets:
    submodules            Update the submodules, such as the common build scripts.
    run                   Run the provider locally, out-of-cluster. Useful for development.

endef
export CROSSPLANE_MAKE_HELP

crossplane.help:
	@echo "$$CROSSPLANE_MAKE_HELP"

help-special: crossplane.help

.PHONY: crossplane.help help-special

vendor: modules.download
vendor.check: modules.check
