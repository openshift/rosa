#!/usr/bin/env bash
# Copyright Red Hat
# SPDX-License-Identifier: Apache-2.0


set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

if [ -z "${GCI_BIN:-}" ]; then
  echo "GCI_BIN is required"
  exit 1
fi

gofmt_output=$(gofmt -s -l cmd pkg tests)
gci_output=$("$GCI_BIN" list -s standard -s default -s "prefix(k8s)" -s "prefix(sigs.k8s)" -s "prefix(github.com)" -s "prefix(gitlab)" -s "prefix(github.com/openshift/rosa)" --custom-order --skip-generated --skip-vendor cmd pkg tests)

if [ -z "$gofmt_output" ] && [ -z "$gci_output" ]; then
  exit 0
fi

"$GCI_BIN" write -s standard -s default -s "prefix(k8s)" -s "prefix(sigs.k8s)" -s "prefix(github.com)" -s "prefix(gitlab)" -s "prefix(github.com/openshift/rosa)" --custom-order --skip-generated --skip-vendor cmd pkg tests
gofmt -s -w cmd pkg tests

echo "Formatting updates were applied. Command failed so you can review and stage changes."
if [ -n "$gofmt_output" ]; then
  echo "Files that needed gofmt:"
  echo "$gofmt_output"
fi
if [ -n "$gci_output" ]; then
  echo "Files that needed import formatting (gci):"
  echo "$gci_output"
fi
echo "Run your commit flow again after reviewing/staging the updates."
exit 1
