#!/bin/bash -ex
#
# Copyright (c) 2018 Red Hat, Inc.
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

# This script is executed by a Jenkins job for each change request. If it
# doesn't succeed the change won't be merged.

# Set the `GOBIN` environment variable so that dependencies will be installed
# always in the same place, regardless of the value of `GOPATH`:
export GOBIN="${PWD}/.gobin"
export PATH="${GOBIN}:${PATH}"

# Install Go tools:
go get github.com/onsi/ginkgo/ginkgo@v1.8.0
go get golang.org/x/tools/cmd/goimports@v0.0.0-20200518194103-259583f2d8a9

# Check that running `make generate` doesn't introduce any change in the
# generated code:
make generate
git diff --exit-code clustersmgmt accountsmgmt
if [ $? = 1 ]; then
  echo "Generated code isn't in sync with model and metamodel"
  exit 1
fi

# Remove the temporary model and metamodel directories, as otherwise `ginkgo`
# will try to run the tests inside them:
rm -rf model metamodel

# Run the checks:
make \
  examples \
  test \
  lint
