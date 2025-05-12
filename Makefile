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

.DEFAULT_GOAL := rosa

# Ensure go modules are enabled:
export GO111MODULE=on
export GOPROXY=https://proxy.golang.org

# Disable CGO so that we always generate static binaries:
export CGO_ENABLED=0

# Unset GOFLAG for CI and ensure we've got nothing accidently set
unexport GOFLAGS

.PHONY: rosa
rosa:
	go build -ldflags="-X github.com/openshift/rosa/pkg/info.Build=$(shell git rev-parse --short HEAD)" ./cmd/rosa

.PHONY: test
test:
	go test $(shell go list ./... | grep -v /tests/)

.PHONY: coverage
coverage:
	go test -coverprofile=cover.out -covermode=atomic -p 4 $(shell go list ./... | grep -v /tests/)

.PHONY: install
install:
	go install ./cmd/rosa

.PHONY: fmt
fmt: fmt-imports
	gofmt -s -l -w cmd pkg tests

.PHONY: fmt-imports
fmt-imports: $(GCI)
	find . -name '*.go' -not -path './vendor/*' | xargs $(GCI) write -s standard -s default -s "prefix(k8s)" -s "prefix(sigs.k8s)" -s "prefix(github.com)" -s "prefix(gitlab)" -s "prefix(github.com/openshift/rosa)" --custom-order --skip-generated

.PHONY: lint
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --timeout 5m0s ./...

.PHONY: commits/check
commits/check:
	@./hack/commit-msg-verify.sh

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

.PHONY: generate
generate: $(GO_BINDATA) mocks
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


.PHONY: e2e_test
e2e_test: install
	ginkgo run \
        --label-filter $(LabelFilter)\
        --timeout 5h \
        -r \
        --focus-file tests/e2e/.* \
		$(NULL)

# verifies changes to .goreleaser.yaml
.PHONY: check-release-config
check-release-config:
	$(GORELEASER) check

# builds a snap shot release in /dist
.PHONY: local-release
local-release:
	$(GORELEASER) release --snapshot --clean

# builds and publishes a latest release from tag set as env-ver GORELEASER_PREVIOUS_TAG
# leave GORELEASER_PREVIOUS_TAG empty to build from last release tag
.PHONY: release
release:
	bash ./hack/build_cli.sh