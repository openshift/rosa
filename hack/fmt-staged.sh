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

staged_go_files=$(git diff --cached --name-only --diff-filter=ACMR -- '*.go')
if [ -z "$staged_go_files" ]; then
  exit 0
fi

partially_staged_file_detected=0
while IFS= read -r go_file; do
  if [ -z "$go_file" ] || [ ! -f "$go_file" ]; then
    continue
  fi

  if ! git diff --quiet -- "$go_file"; then
    echo "Commit blocked: staged Go file has unstaged changes: $go_file"
    echo "Stage all changes for this file (or stash them) before committing."
    partially_staged_file_detected=1
  fi
done <<< "$staged_go_files"

if [ "$partially_staged_file_detected" -ne 0 ]; then
  exit 1
fi

formatting_updated_files=""
while IFS= read -r go_file; do
  if [ -z "$go_file" ] || [ ! -f "$go_file" ]; then
    continue
  fi

  before_formatting=$(mktemp)
  cp "$go_file" "$before_formatting"

  "$GCI_BIN" write -s standard -s default -s "prefix(k8s)" -s "prefix(sigs.k8s)" -s "prefix(github.com)" -s "prefix(gitlab)" -s "prefix(github.com/openshift/rosa)" --custom-order --skip-generated --skip-vendor "$go_file"
  gofmt -s -w "$go_file"

  if ! cmp -s "$before_formatting" "$go_file"; then
    formatting_updated_files+="$go_file"$'\n'
  fi

  rm -f "$before_formatting"
done <<< "$staged_go_files"

if [ -n "$formatting_updated_files" ]; then
  echo "Commit blocked: formatting updates were applied to staged Go files:"
  printf '%s' "$formatting_updated_files"
  echo "Review the changes, stage them, and commit again."
  exit 1
fi
