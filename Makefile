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
	go test ./...

.PHONY: coverage
coverage:
	go test -coverprofile=cover.out -covermode=atomic -p 4 ./...

.PHONY: install
install:
	go install ./cmd/rosa

.PHONY: fmt
fmt: fmt-imports
	gofmt -s -l -w cmd pkg

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
		*-darwin-amd64 \
		*-linux-amd64 \
		*-windows-amd64 \
		*.sha256 \
		$(NULL)

.PHONY: generate
generate: $(GO_BINDATA)
	$(GO_BINDATA) -nometadata -nocompress -pkg assets -o ./assets/bindata.go ./templates/...

.PHONY: codecov
codecov: coverage
	@./hack/codecov.sh

mocks:
	mockgen --build_flags=--mod=mod -package mocks -destination=cmd/create/idp/mocks/identityprovider.go -source=cmd/create/idp/cmd.go IdentityProvider
	mockgen -source=pkg/aws/api_interface/iam_api_client.go -package=mocks -destination=pkg/aws/mocks/mock_iam_api_client.go
	mockgen -source=pkg/aws/api_interface/organizations_api_client.go -package=mocks -destination=pkg/aws/mocks/mock_organizations_api_client.go
	mockgen -source=pkg/aws/api_interface/sts_api_client.go -package=mocks -destination=pkg/aws/mocks/mock_sts_api_client.go
	mockgen -source=pkg/aws/api_interface/cloudformation_api_client.go -package=mocks -destination=pkg/aws/mocks/mock_cloudformation_api_client.go
	mockgen -source=pkg/aws/api_interface/servicequotas_api_client.go -package=mocks -destination=pkg/aws/mocks/mock_servicequotas_api_client.go
	mockgen -source=pkg/aws/api_interface/ec2_api_client.go -package=mocks -destination=pkg/aws/mocks/mock_ec2_api_client.go
	mockgen -source=pkg/aws/api_interface/s3_api_client.go -package=mocks -destination=pkg/aws/mocks/mock_s3_api_client.go
	mockgen -source=pkg/aws/api_interface/secretsmanager_api_client.go -package=mocks -destination=pkg/aws/mocks/mock_secretsmanager_api_client.go

	mockgen --build_flags=--mod=mod -package mocks -destination=pkg/aws/mocks/iamapi.go github.com/aws/aws-sdk-go/service/iam/iamiface IAMAPI
	mockgen --build_flags=--mod=mod -package mocks -destination=pkg/aws/mocks/organaztionsapi.go github.com/aws/aws-sdk-go/service/organizations/organizationsiface OrganizationsAPI
	mockgen --build_flags=--mod=mod -package mocks -destination=pkg/aws/mocks/stsapi.go github.com/aws/aws-sdk-go/service/sts/stsiface STSAPI
	mockgen --build_flags=--mod=mod -package mocks -destination=pkg/aws/mocks/cloudformationapi.go github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface CloudFormationAPI
	mockgen --build_flags=--mod=mod -package mocks -destination=pkg/aws/mocks/ec2api.go github.com/aws/aws-sdk-go/service/ec2/ec2iface EC2API
	mockgen --build_flags=--mod=mod -package mocks -destination=pkg/aws/mocks/servicequotasapi.go github.com/aws/aws-sdk-go/service/servicequotas/servicequotasiface ServiceQuotasAPI
	mockgen --build_flags=--mod=mod -package mocks -destination=pkg/aws/mocks/s3api.go github.com/aws/aws-sdk-go/service/s3/s3iface S3API
	mockgen --build_flags=--mod=mod -package mocks -destination=pkg/aws/mocks/secretsmanagerapi.go github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface SecretsManagerAPI
	
