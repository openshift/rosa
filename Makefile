#
# Copyright (c) 2020 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

include .bingo/Variables.mk

VALE_VERSION ?= v3.21.0
ADDLICENSE_VERSION ?= v1.2.0
VALE := $(GOBIN)/vale
ADDLICENSE := $(GOBIN)/addlicense

.DEFAULT_GOAL := rosa

# Wrapper script entry points used by make convenience targets.
RUN_CHECKS_SCRIPT := ./hack/run-checks.sh

# Ensure go modules are enabled:
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org
# Use Go's documented toolchain switching, since some repackaged toolchains default to local-only.
export GOTOOLCHAIN=auto
# Let the selected Go binary provide its own GOROOT to avoid mixed-toolchain builds.
unexport GOROOT
# Use the selected toolchain binary for Bingo-managed tool builds.
GO := $(shell go env GOROOT)/bin/go

# Disable CGO so that we always generate static binaries:
export CGO_ENABLED=0

# Unset GOFLAG for CI and ensure we've got nothing accidently set
unexport GOFLAGS

# Version from the exact tag on HEAD; empty (uses the fallback in pkg/info/info.go)
# when HEAD is not tagged.
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null | sed 's/^v//')
GINKGO_BIN ?= ginkgo
GOBIN_DIR := $(patsubst %/,%,$(GOBIN))
ROSA_BIN ?= $(GOBIN_DIR)/rosa

.PHONY: rosa
rosa:
	go build -ldflags="-X github.com/openshift/rosa/pkg/info.Build=$(shell git rev-parse --short HEAD) $(if $(VERSION),-X github.com/openshift/rosa/pkg/info.DefaultVersion=$(VERSION),)" ./cmd/rosa

.PHONY: test
test:
	go test $(GO_TEST_FLAGS) $(shell go list ./... | grep -v /tests/)

.PHONY: coverage
coverage:
	go test -coverprofile=cover.out -covermode=atomic -p 1 $(GO_TEST_FLAGS) $(shell go list ./... | grep -v /tests/)

.PHONY: coverage-changed-files
coverage-changed-files:
	./hack/coverage-changed-files.sh

.PHONY: install
install:
	go install -ldflags="-X github.com/openshift/rosa/pkg/info.Build=$(shell git rev-parse --short HEAD) $(if $(VERSION),-X github.com/openshift/rosa/pkg/info.DefaultVersion=$(VERSION),)" ./cmd/rosa

.PHONY: fmt
fmt: $(GCI)
	GCI_BIN="$(GCI)" ./hack/fmt.sh

.PHONY: fmt-staged
fmt-staged: $(GCI)
	GCI_BIN="$(GCI)" ./hack/fmt-staged.sh

.PHONY: fmt-check
fmt-check: $(GCI)
	@test -z "$$(gofmt -s -l cmd pkg tests)"
	@test -z "$$($(GCI) list -s standard -s default -s "prefix(k8s)" -s "prefix(sigs.k8s)" -s "prefix(github.com)" -s "prefix(gitlab)" -s "prefix(github.com/openshift/rosa)" --custom-order --skip-generated --skip-vendor cmd pkg tests)"

.PHONY: lint
LINT_NEW_FROM_REV ?= $(PULL_BASE_SHA)
LINT_NEW_FROM_REV_FLAG := $(if $(LINT_NEW_FROM_REV),--new-from-rev=$(LINT_NEW_FROM_REV))
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --timeout 15m0s $(LINT_OUTPUT_FLAGS) $(LINT_NEW_FROM_REV_FLAG) ./...

$(VALE):
	CGO_ENABLED=1 GOBIN="$(GOBIN)" go install github.com/vale-cli/vale/v3/cmd/vale@$(VALE_VERSION)

$(ADDLICENSE):
	GOBIN="$(GOBIN)" go install github.com/google/addlicense@$(ADDLICENSE_VERSION)

.PHONY: vale
vale: $(VALE)

.PHONY: docs-lint
docs-lint: $(VALE)
	"$(VALE)" --minAlertLevel=error --glob='!.vale.ini' --glob='!vendor/**' .

.PHONY: govulncheck
govulncheck: $(GOVULNCHECK) rosa
	GOVULNCHECK_BIN="$(GOVULNCHECK)" ./hack/govulncheck.sh

.PHONY: commits/check
commits/check:
	@./hack/commit-msg-verify.sh

.PHONY: install-hooks
install-hooks:
	@hooks_path=$$(git config --local core.hooksPath 2>/dev/null || true); \
	legacy_hooks_path=; \
	if [ "$$hooks_path" = ".githooks" ]; then \
		if ! command -v pre-commit >/dev/null 2>&1; then \
			echo "Error: pre-commit is not installed; preserving legacy hooks."; \
			exit 1; \
		fi; \
		legacy_hooks_path=$$hooks_path; \
		echo "Detected legacy core.hooksPath=.githooks (from the old hook system)."; \
		echo "Unsetting it so pre-commit hooks are picked up correctly..."; \
		git config --local --unset core.hooksPath || exit $$?; \
	fi; \
	hooks_path=$$(git config --get core.hooksPath 2>/dev/null || true); \
	if [ -n "$$hooks_path" ]; then \
		echo "Hook installation skipped: core.hooksPath is configured; preserving existing hooks."; \
		echo "All checks remain mandatory. Follow the explicit stage commands in CONTRIBUTING.md."; \
		exit 0; \
	fi; \
	if ! command -v pre-commit >/dev/null 2>&1; then \
		echo "Error: pre-commit is not installed."; \
		echo "See https://pre-commit.com/#install for installation instructions"; \
		exit 1; \
	fi; \
	if pre-commit install --install-hooks --hook-type pre-commit --hook-type commit-msg --hook-type pre-push; then \
		echo "Installed pre-commit hooks (pre-commit, commit-msg, pre-push). All checks remain mandatory."; \
	else \
		status=$$?; \
		if [ "$$legacy_hooks_path" = ".githooks" ]; then \
			git config --local core.hooksPath .githooks || exit $$?; \
		fi; \
		exit $$status; \
	fi

.PHONY: license-check
license-check: $(ADDLICENSE)
	@echo "Checking for missing license headers..."
	@ADDLICENSE="$(ADDLICENSE)" bash scripts/add-license-header.sh -check

.PHONY: license-add
license-add: $(ADDLICENSE)
	@echo "Adding license headers to files..."
	@ADDLICENSE="$(ADDLICENSE)" bash scripts/add-license-header.sh

.PHONY: license-add-staged
license-add-staged: $(ADDLICENSE)
	@ADDLICENSE="$(ADDLICENSE)" ./hack/license-add-staged.sh

.PHONY: basic-checks
basic-checks:
	@$(RUN_CHECKS_SCRIPT) basic

.PHONY: pre-commit-checks
pre-commit-checks:
	@$(MAKE) --no-print-directory fmt-staged
	@$(MAKE) --no-print-directory license-add-staged

.PHONY: pre-push-checks
pre-push-checks:
	@$(RUN_CHECKS_SCRIPT) pre-push

.PHONY: run-checks
run-checks:
	@$(RUN_CHECKS_SCRIPT) $(filter-out $@,$(MAKECMDGOALS))

PASSTHROUGH_MAKE_TARGETS := run-checks
ifneq (,$(filter $(PASSTHROUGH_MAKE_TARGETS),$(MAKECMDGOALS)))
PASSTHROUGH_ARG_GOALS := $(filter-out $(PASSTHROUGH_MAKE_TARGETS),$(MAKECMDGOALS))
$(foreach goal,$(PASSTHROUGH_ARG_GOALS),$(eval .PHONY: $(goal)))
$(foreach goal,$(PASSTHROUGH_ARG_GOALS),$(eval $(goal):;@:))
endif

.PHONY: diff
diff:
	git diff --exit-code

.PHONY: verify
verify: fmt
	go mod tidy
	go mod vendor
	$(MAKE) diff

.PHONY: clean
clean:
	rm -rf \
		./cover.out \
		rosa \
		rosa-darwin-amd64 \
		rosa-darwin-arm64 \
		rosa-linux-amd64 \
		rosa-linux-arm64 \
		rosa-windows-amd64.exe \
		*.sha256 \
		$(NULL)

PATHBIND_GEN_PKG    := github.com/openshift-online/rosa-seaplane-api/clientset/cmd/pathbind-gen
PATHBIND_GEN_BIN    := /tmp/pathbind-gen-$(shell git rev-parse --short HEAD)
PATHBIND_DRAFT      := vendor/github.com/openshift-online/rosa-seaplane-api/clientset/pathbind/pathbind-draft.yaml
PATHBIND_OVERRIDES  := pkg/seaplane/pathbind-overrides.yaml
PATHBIND_OUT        := pkg/seaplane/pathbind

$(PATHBIND_GEN_BIN):
	GOFLAGS="-mod=mod" go build -o $(PATHBIND_GEN_BIN) $(PATHBIND_GEN_PKG)

.PHONY: generate-seaplane
generate-seaplane: $(PATHBIND_GEN_BIN)
	$(PATHBIND_GEN_BIN) \
		--mode=cobra \
		--draft=$(PATHBIND_DRAFT) \
		--overrides=$(PATHBIND_OVERRIDES) \
		--output-dir=$(PATHBIND_OUT)
	./scripts/add-license-header.sh $(PATHBIND_OUT)/*.go

.PHONY: generate
generate: $(GO_BINDATA) mocks generate-seaplane
	$(GO_BINDATA) -nometadata -nocompress -pkg assets -o ./assets/bindata.go ./templates/...
	go generate ./...

.PHONY: codecov
codecov: coverage
	@./hack/codecov.sh

mocks: $(MOCKGEN)
	$(MOCKGEN) -source=pkg/aws/api_interface/iam_api_client.go -package=mocks -destination=pkg/aws/mocks/iam_api_client_mock.go
	$(MOCKGEN) -source=pkg/aws/api_interface/organizations_api_client.go -package=mocks -destination=pkg/aws/mocks/organizations_api_client_mock.go
	$(MOCKGEN) -source=pkg/aws/api_interface/sts_api_client.go -package=mocks -destination=pkg/aws/mocks/sts_api_client_mock.go
	$(MOCKGEN) -source=pkg/aws/api_interface/cloudformation_api_client.go -package=mocks -destination=pkg/aws/mocks/cloudformation_api_client_mock.go
	$(MOCKGEN) -source=pkg/aws/api_interface/servicequotas_api_client.go -package=mocks -destination=pkg/aws/mocks/servicequotas_api_client_mock.go
	$(MOCKGEN) -source=pkg/aws/api_interface/ec2_api_client.go -package=mocks -destination=pkg/aws/mocks/ec2_api_client_mock.go
	$(MOCKGEN) -source=pkg/aws/api_interface/s3_api_client.go -package=mocks -destination=pkg/aws/mocks/s3_api_client_mock.go
	$(MOCKGEN) -source=pkg/aws/api_interface/secretsmanager_api_client.go -package=mocks -destination=pkg/aws/mocks/secretsmanager_api_client_mock.go
	$(MOCKGEN) -source=pkg/aws/client.go -package=aws -destination=pkg/aws/client_mock.go
	$(MOCKGEN) -source=pkg/version/version.go -package=version -destination=pkg/version/version_mock.go
	$(MOCKGEN) -source=pkg/version/retriever.go -package=version -destination=pkg/version/retriever_mock.go
	$(MOCKGEN) -source=cmd/verify/rosa/cmd.go -package=rosa -destination=cmd/verify/rosa/cmd_mock.go
	$(MOCKGEN) -source=pkg/clients/http.go -package=clients -destination=pkg/clients/http_mock.go
	$(MOCKGEN) -source=pkg/cache/service.go -package=cache -destination=pkg/cache/service_mock.go
	$(MOCKGEN) -source=pkg/cache/cache.go -package=cache -destination=pkg/cache/cache_mock.go
	$(MOCKGEN) -source=pkg/network/network.go -package=network -destination=pkg/network/network_mock.go
	$(MOCKGEN) -source=pkg/machinepool/machinepool.go -package=machinepool -destination=pkg/machinepool/machinepool_mock.go
	$(MOCKGEN) -source=pkg/kubeletconfig/config.go -package=kubeletconfig -destination=pkg/kubeletconfig/capability_checker_mock.go
	$(MOCKGEN) -source=cmd/create/idp/cmd.go -package=mocks -destination=cmd/create/idp/mocks/identityprovider.go
	$(MOCKGEN) -source=vendor/github.com/openshift-online/rosa-seaplane-api/clientset/seaplane.go -package=mocks -destination=pkg/seaplane/mocks/seaplane_mock.go
	$(MOCKGEN) -source=vendor/github.com/openshift-online/rosa-seaplane-api/clientset/platform/bridge_wrappers_generated.go -package=mocks -destination=pkg/seaplane/mocks/wrappers_mock.go
	perl -pi -e 's|github\.com/openshift/rosa/vendor/github\.com/openshift-online/rosa-seaplane-api/clientset/platform|github.com/openshift-online/rosa-seaplane-api/clientset/platform|g' pkg/seaplane/mocks/wrappers_mock.go


.PHONY: e2e_test
e2e_test: install
	ginkgo run \
        --label-filter $(LabelFilter)\
        --timeout 5h \
        -r \
        --focus-file tests/e2e/.* \
		$(NULL)

# e2e-seaplane: seaplane e2e via ginkgo --label-filter.
#
# Labels (see tests/ci/labels/seaplane.go):
#   seaplane-validated — day1 FVT (profile-driven cluster prep; needs TEST_PROFILE)
#   seaplane-sanity    — full CLI/SDK lifecycle spec (self-contained; no TEST_PROFILE)
#
# Required:
#   SEAPLANE_URL  — Platform API v2 base URL
#   TEST_PROFILE    — e.g. rosa-seaplane-basic (for seaplane-validated only)
#
# Optional:
#   CLUSTER_NAME       — defaults to sp-e2e-<unix timestamp> (≤18 chars)
#   AWS_DEFAULT_REGION — fallback when region cannot be derived from SEAPLANE_URL
#   LABEL_FILTER       — defaults to seaplane-validated; use seaplane-sanity for sanity
.PHONY: e2e-seaplane
e2e-seaplane: install
	@test -n "$${SEAPLANE_URL}" || { \
		echo "SEAPLANE_URL is required (see guidelines/seaplane-guidelines.md)"; \
		exit 1; \
	}; \
	name=$${CLUSTER_NAME:-sp-e2e-$$(date +%s)}; \
	filter=$${LABEL_FILTER:-seaplane-validated}; \
	SEAPLANE_URL="$${SEAPLANE_URL}" \
	TEST_PROFILE="$(TEST_PROFILE)" \
	CLUSTER_NAME="$$name" \
	OPERATOR_ROLES_PREFIX="$$name" \
	AWS_DEFAULT_REGION="$${AWS_DEFAULT_REGION}" \
	ginkgo run --label-filter "($$filter) && !seaplane-na && !seaplane-deferred" \
		--timeout 3h \
		-v \
		./tests/e2e/... \
		$(NULL)

# e2e-seaplane-day1-day2: run the ordered, profile-backed Seaplane lifecycle,
# reusing one cluster and output directory for a single TEST_PROFILE.
#
# Phases: day1, day1-readiness, day1-post, day1-supplemental, day1-negative,
# then day2. The profile-managed cluster is destroyed on exit unless
# E2E_SEAPLANE_KEEP_CLUSTER=true or E2E_SEAPLANE_DESTROY=false.
#
# Optional:
#   E2E_SEAPLANE_PHASES       — space-separated phase override; include
#                              "destroy" for a destroy-only CI step
#   E2E_SEAPLANE_TIMEOUT      — per-phase ginkgo timeout (default: 4h)
#   E2E_SEAPLANE_DESTROY      — set false to skip the final destroy phase
#   E2E_SEAPLANE_KEEP_CLUSTER — set true to retain the cluster
#   E2E_SEAPLANE_CONTINUE_ON_FAILURE — set true to run remaining phases after
#                                      a phase fails; the script still exits 1
#   GINKGO_BIN                — ginkgo binary (default: ginkgo)
.PHONY: e2e-seaplane-day1-day2
e2e-seaplane-day1-day2: install
	@SEAPLANE_URL="$${SEAPLANE_URL}" \
	TEST_PROFILE="$(TEST_PROFILE)" \
	E2E_SEAPLANE_CONTINUE_ON_FAILURE="$(E2E_SEAPLANE_CONTINUE_ON_FAILURE)" \
	GINKGO_BIN="$(GINKGO_BIN)" \
	ROSA_BIN="$(ROSA_BIN)" \
	./hack/e2e-seaplane-day1-day2.sh

E2E_SEAPLANE_LIFECYCLE_PROFILES ?= \
	rosa-seaplane-basic \
	rosa-seaplane-fips \
	rosa-seaplane-advanced \
	rosa-seaplane-private
E2E_SEAPLANE_OUTPUT_ROOT ?= $(CURDIR)/tests/output
E2E_SEAPLANE_TIMEOUT ?= 4h
E2E_SEAPLANE_DESTROY ?= true
E2E_SEAPLANE_KEEP_CLUSTER ?= false
E2E_SEAPLANE_CONTINUE_ON_FAILURE ?= false

define run_seaplane_focused
	@test -n "$(SEAPLANE_URL)" || { \
		echo "SEAPLANE_URL is required (see tests/e2e/seaplane-guidelines.md)"; \
		exit 1; \
	}; \
	profile="$(1)"; \
	focus_file="$(2)"; \
	focus="$(3)"; \
	suffix="$(4)"; \
	junit_report="$(5)"; \
	output_dir="$(E2E_SEAPLANE_OUTPUT_ROOT)/$$profile"; \
	mkdir -p "$$output_dir"; \
	export PATH="$(GOBIN_DIR):$$PATH"; \
	config_file="$${OCM_CONFIG:-$$output_dir/rosa-config.json}"; \
	OCM_CONFIG="$$config_file" "$(ROSA_BIN)" login --seaplane-url="$(SEAPLANE_URL)" >/dev/null; \
	cluster_name="sp-e2e-$$(date +%s | cut -c5-)-$$suffix"; \
	OCM_CONFIG="$$config_file" \
	SEAPLANE_URL="$(SEAPLANE_URL)" \
	TEST_PROFILE="$$profile" \
	CLUSTER_NAME="$$cluster_name" \
	OPERATOR_ROLES_PREFIX="$$cluster_name" \
	SHARED_DIR="$$output_dir" \
	ARTIFACT_DIR="$$output_dir" \
	OCP_IMAGE="$(OCP_IMAGE)" \
	SEAPLANE_VERSION="$(SEAPLANE_VERSION)" \
	"$(GINKGO_BIN)" run \
		--focus-file="$$focus_file" \
		--focus="$$focus" \
		--label-filter='(day1) && seaplane-validated && !seaplane-na && !seaplane-deferred' \
		--timeout="$(E2E_SEAPLANE_TIMEOUT)" \
		--junit-report="$$junit_report" \
		--output-dir="$$output_dir" \
		-v \
		./tests/e2e/...
endef

# e2e-seaplane-profiles: run the ordered lifecycle once per profile.
#
# TEST_PROFILE is read once per Ginkgo process, so each profile must run in a
# separate process. This target also isolates SHARED_DIR, ARTIFACT_DIR, and the
# cluster name for every profile.
#
# Optional:
#   E2E_SEAPLANE_LIFECYCLE_PROFILES — space-separated profile list
#   E2E_SEAPLANE_OUTPUT_ROOT        — artifacts root (default: tests/output)
#   E2E_SEAPLANE_PHASES             — phase override, applied to every profile
#   E2E_SEAPLANE_TIMEOUT            — per-phase Ginkgo timeout (default: 4h)
#   E2E_SEAPLANE_CONTINUE_ON_FAILURE — set true to run remaining phases after
#                                     a phase fails; the target still exits 1
#   OCP_IMAGE or SEAPLANE_VERSION   — required when a selected profile's version
#                                     is not a release image pullspec
#                                     (advanced and private currently use "latest")
.PHONY: e2e-seaplane-profiles
e2e-seaplane-profiles: install
	@test -n "$(SEAPLANE_URL)" || { \
		echo "SEAPLANE_URL is required (see tests/e2e/seaplane-guidelines.md)"; \
		exit 1; \
	}; \
	for profile in $(E2E_SEAPLANE_LIFECYCLE_PROFILES); do \
		case "$$profile" in \
		rosa-seaplane-advanced | rosa-seaplane-private) \
			if [ -z "$(OCP_IMAGE)" ] && [ -z "$(SEAPLANE_VERSION)" ]; then \
				echo "OCP_IMAGE or SEAPLANE_VERSION is required for $$profile"; \
				exit 1; \
			fi; \
			;; \
		esac; \
	done; \
	status=0; \
	run_id=$$(date +%s | cut -c5-); \
	index=0; \
	for profile in $(E2E_SEAPLANE_LIFECYCLE_PROFILES); do \
		index=$$((index + 1)); \
		cluster_name="sp-e2e-$${run_id}-$${index}"; \
		echo; \
		echo "=== Seaplane E2E profile: $$profile (cluster: $$cluster_name) ==="; \
		SEAPLANE_URL="$(SEAPLANE_URL)" \
		TEST_PROFILE="$$profile" \
		CLUSTER_NAME="$$cluster_name" \
		OPERATOR_ROLES_PREFIX="$$cluster_name" \
		SHARED_DIR="$(E2E_SEAPLANE_OUTPUT_ROOT)/$$profile" \
		ARTIFACT_DIR="$(E2E_SEAPLANE_OUTPUT_ROOT)/$$profile" \
		E2E_SEAPLANE_PHASES="$(E2E_SEAPLANE_PHASES)" \
		E2E_SEAPLANE_TIMEOUT="$(E2E_SEAPLANE_TIMEOUT)" \
		E2E_SEAPLANE_DESTROY="$(E2E_SEAPLANE_DESTROY)" \
		E2E_SEAPLANE_KEEP_CLUSTER="$(E2E_SEAPLANE_KEEP_CLUSTER)" \
		E2E_SEAPLANE_CONTINUE_ON_FAILURE="$(E2E_SEAPLANE_CONTINUE_ON_FAILURE)" \
		OCP_IMAGE="$(OCP_IMAGE)" \
		SEAPLANE_VERSION="$(SEAPLANE_VERSION)" \
		GINKGO_BIN="$(GINKGO_BIN)" \
		ROSA_BIN="$(ROSA_BIN)" \
		./hack/e2e-seaplane-day1-day2.sh || status=1; \
	done; \
	exit $$status

.PHONY: e2e-seaplane-nocni
e2e-seaplane-nocni: install
	$(call run_seaplane_focused,rosa-seaplane-nocni,tests/e2e/test_rosacli_cluster.go,creates an HCP cluster with no CNI,n,junit-day1-nocni.xml)

.PHONY: e2e-seaplane-disable-capabilities
e2e-seaplane-disable-capabilities: install
	$(call run_seaplane_focused,rosa-seaplane-disable-capabilities,tests/e2e/seaplane_capabilities_test.go,ImageRegistry capability disabled,c,junit-day1-disable-capabilities.xml)

.PHONY: e2e-seaplane-profile-tests
e2e-seaplane-profile-tests: e2e-seaplane-nocni e2e-seaplane-disable-capabilities

.PHONY: generate-docs
generate-docs:
	go run ./tools/gendocs

# verifies changes to .goreleaser.yaml
.PHONY: check-release-config
check-release-config:
	$(GORELEASER) check

.PHONY: changelog-bootstrap
changelog-bootstrap:
	./hack/changelog-generate.sh --bootstrap

.PHONY: changelog-generate
changelog-generate:
	@test -n "$(TAG)"
	./hack/changelog-generate.sh --tag "$(TAG)" $(if $(PREVIOUS_TAG),--previous-tag "$(PREVIOUS_TAG)",)

.PHONY: changelog-pr
changelog-pr:
	./hack/changelog-pr.sh $(if $(TAG),--tag "$(TAG)",) $(if $(PREVIOUS_TAG),--previous-tag "$(PREVIOUS_TAG)",)

# builds a snap shot release in /dist
.PHONY: local-release
local-release:
	$(GORELEASER) release --snapshot --clean

# builds and publishes a latest release from tag set as env-ver GORELEASER_PREVIOUS_TAG
# leave GORELEASER_PREVIOUS_TAG empty to build from last release tag
.PHONY: release
release:
	bash ./hack/build_cli.sh
