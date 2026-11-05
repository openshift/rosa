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


# Label selection shared by every Seaplane validated phase.
SEAPLANE_LABEL_BASE := seaplane-validated && !seaplane-na && !seaplane-deferred

# e2e_test: run the E2E suite.
#
# Two modes, chosen automatically:
#
#   OCM       the historical single ginkgo invocation, filtered by LabelFilter.
#   Seaplane  an ordered, phased run. Used when the rosa config has a
#             seaplane_url and LabelFilter is either unset or
#             seaplane-validated. Any other LabelFilter (seaplane-sanity, a
#             compound expression, an OCM filter) takes the OCM path, so
#             existing callers are unaffected.
#
# The Seaplane run is phased because ginkgo randomizes top-level containers,
# so a single invocation can tear the cluster down before it is created:
#
#   1 day1            cluster preparation by profile
#   2 day1-readiness  wait for the cluster to be ready
#   3 tests           everything else — day1-post/-supplemental/-negative, day2,
#                     and specs carrying no day label — selected by negation
#   4 destroy         profile teardown
#   5 destroy-post    verify deletion
#
# Phase 4 always runs, even when an earlier phase fails, so a failed test never
# leaks a cluster. Phase 5 only runs when setup and tests succeeded, since with
# no cluster there is nothing whose deletion can be verified. The exit code is
# the test failure when there is one, otherwise the teardown failure.
#
# Seaplane mode reads the endpoint from the rosa config written by
# `rosa login --seaplane-url`; that is the value the CLI itself uses.
#
# Optional:
#   LabelFilter        — ginkgo label expression; selects the mode as above
#   TEST_PROFILE       — profile in tests/ci/data/profiles/rosa-seaplane.yaml
#                        (defaults to rosa-seaplane-basic)
#   CLUSTER_NAME       — defaults to sp-e2e-<unix timestamp> (≤18 chars)
#   SEAPLANE_URL       — optional override; must match the rosa config
#   AWS_DEFAULT_REGION — fallback when region cannot be derived from the URL
#   GINKGO_SEED        — pin the seed to reproduce a failing run
#   CLUSTER_TIMEOUT    — minutes to wait for the cluster (default 60)
#   NODEPOOL_TIMEOUT   — minutes to wait for node pools (default 45)
#
# Two validated specs are gated to their own profiles and report as skipped
# unless selected: rosa-seaplane-nocni and rosa-seaplane-disable-capabilities.
.PHONY: e2e_test
e2e_test: install
	@filter='$(LabelFilter)'; \
	cfg="$${OCM_CONFIG}"; \
	if [ -z "$$cfg" ] && [ -f "$$HOME/.ocm.json" ]; then cfg="$$HOME/.ocm.json"; fi; \
	if [ -z "$$cfg" ]; then \
		case "$$(uname -s)" in \
			Darwin) cfg="$$HOME/Library/Application Support/ocm/ocm.json" ;; \
			*) cfg="$${XDG_CONFIG_HOME:-$$HOME/.config}/ocm/ocm.json" ;; \
		esac; \
	fi; \
	cfgurl=$$(sed -n 's/.*"seaplane_url"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
		"$$cfg" 2>/dev/null | head -1); \
	if [ "$$filter" = "seaplane-validated" ] && [ -z "$$cfgurl" ]; then \
		echo "LabelFilter=seaplane-validated, but the rosa config ($$cfg) has no"; \
		echo "seaplane_url. The CLI reads the endpoint from its config, so the run"; \
		echo "would silently take the OCM path and fail somewhere unrelated. Run:"; \
		echo "  rosa login --seaplane-url \"$${SEAPLANE_URL:-<endpoint>}\""; \
		exit 1; \
	fi; \
	if [ -z "$$cfgurl" ] || { [ -n "$$filter" ] && [ "$$filter" != "seaplane-validated" ]; }; then \
		exec ginkgo run \
			--label-filter "$$filter" \
			--timeout 5h \
			-r \
			--focus-file 'tests/e2e/.*'; \
	fi; \
	name=$${CLUSTER_NAME:-sp-e2e-$$(date +%s)}; \
	profile='$(TEST_PROFILE)'; profile=$${profile:-rosa-seaplane-basic}; \
	seed=''; test -z "$${GINKGO_SEED}" || seed="--seed=$${GINKGO_SEED}"; \
	gobin=$$(go env GOBIN); test -n "$$gobin" || gobin=$$(go env GOPATH)/bin; \
	PATH="$$gobin:$$PATH"; export PATH; \
	if ! grep -qE "^ *- as: *$$profile *$$" tests/ci/data/profiles/*.yaml; then \
		echo "TEST_PROFILE '$$profile' is not a known profile. Valid seaplane profiles:"; \
		grep -rhoE "^ *- as: *rosa-seaplane-[a-z-]+" tests/ci/data/profiles/*.yaml \
			| sed 's/.*as: */  /' | sort; \
		exit 1; \
	fi; \
	if [ -n "$$SEAPLANE_URL" ] && [ "$$SEAPLANE_URL" != "$$cfgurl" ]; then \
		echo "SEAPLANE_URL and the rosa config disagree on the endpoint:"; \
		echo "  SEAPLANE_URL : $$SEAPLANE_URL"; \
		echo "  rosa config  : $$cfgurl"; \
		echo "The specs read SEAPLANE_URL but the CLI reads the config, so the run would"; \
		echo "target two different endpoints and fail against whichever one does not know"; \
		echo "this account. Unset SEAPLANE_URL to use the config, or re-run rosa login."; \
		exit 1; \
	fi; \
	SEAPLANE_URL="$$cfgurl"; \
	export SEAPLANE_URL; \
	export TEST_PROFILE="$$profile"; \
	export CLUSTER_NAME="$$name"; \
	export OPERATOR_ROLES_PREFIX="$$name"; \
	export AWS_DEFAULT_REGION; \
	run_phase() { \
		echo ""; \
		echo "=== seaplane phase $$1 ==="; \
		ginkgo run \
			--label-filter "$(SEAPLANE_LABEL_BASE) && ($$2)" \
			--timeout 3h \
			-v \
			$$seed \
			$(SEAPLANE_GINKGO_EXTRA) \
			./tests/e2e/...; \
	}; \
	echo "seaplane validated run: cluster=$$name profile=$$profile"; \
	echo "  endpoint: $$SEAPLANE_URL"; \
	echo "  rosa:     $$(command -v rosa)"; \
	rc=0; td=0; \
	run_phase "1/5 setup" "day1" || rc=$$?; \
	if [ $$rc -eq 0 ]; then \
		run_phase "2/5 readiness" "day1-readiness" || rc=$$?; \
	fi; \
	if [ $$rc -eq 0 ]; then \
		run_phase "3/5 tests" \
			"!day1 && !day1-readiness && !destroy && !destroy-post" || rc=$$?; \
	fi; \
	run_phase "4/5 destroy" "destroy" || td=$$?; \
	if [ $$rc -eq 0 ]; then \
		run_phase "5/5 destroy-post" "destroy-post" || td=$$?; \
	else \
		echo ""; \
		echo "=== seaplane phase 5/5 destroy-post: skipped ==="; \
		echo "setup or tests failed, so there is no cluster whose deletion can be"; \
		echo "verified; running it would only bury the real failure above."; \
	fi; \
	if [ $$rc -ne 0 ]; then \
		echo "seaplane validated run FAILED (tests rc=$$rc, teardown rc=$$td)"; \
		exit $$rc; \
	fi; \
	if [ $$td -ne 0 ]; then \
		echo "seaplane validated run: tests passed but teardown rc=$$td"; \
		exit $$td; \
	fi; \
	echo "seaplane validated run OK"

# clean-seaplane-cluster: delete one Seaplane cluster by name.
#
# For recovering from a run that left a cluster behind. The next run otherwise
# fails with "a cluster named X already exists in this account", and because
# that happens during setup it leaves no usable state for the normal teardown
# phases to work from.
#
# Required:
#   CLUSTER_NAME — cluster to delete
#
# Optional:
#   DELETE_OPERATOR_ROLES — set to any value to also delete the operator roles
#                           sharing the cluster-name prefix. Implies
#                           DELETE_WATCH: the API refuses to remove roles while
#                           any cluster still references the prefix, reporting
#                           "There are clusters using Operator Roles Prefix",
#                           so the cluster deletion has to finish first.
#   DELETE_WATCH          — set to any value to wait for uninstall to complete
#                           and stream its logs
.PHONY: clean-seaplane-cluster
clean-seaplane-cluster: install
	@test -n "$${CLUSTER_NAME}" || { \
		echo "CLUSTER_NAME is required, e.g.:"; \
		echo "  make clean-seaplane-cluster CLUSTER_NAME=hf-e2e-1793816856"; \
		exit 1; \
	}; \
	gobin=$$(go env GOBIN); test -n "$$gobin" || gobin=$$(go env GOPATH)/bin; \
	PATH="$$gobin:$$PATH"; export PATH; \
	watch=''; test -z "$${DELETE_WATCH}" || watch='--watch'; \
	if [ -n "$${DELETE_OPERATOR_ROLES}" ]; then watch='--watch'; fi; \
	echo "deleting cluster '$${CLUSTER_NAME}'"; \
	echo "  rosa: $$(command -v rosa)"; \
	rosa delete cluster --cluster "$${CLUSTER_NAME}" --yes $$watch || exit $$?; \
	if [ -n "$${DELETE_OPERATOR_ROLES}" ]; then \
		echo "deleting operator roles with prefix '$${CLUSTER_NAME}'"; \
		rosa delete operator-roles --prefix "$${CLUSTER_NAME}" --yes || exit $$?; \
	fi; \
	echo "deleted '$${CLUSTER_NAME}'"

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
