#!/usr/bin/env bash

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

while IFS= read -r go_file; do
  if [ -z "$go_file" ] || [ ! -f "$go_file" ]; then
    continue
  fi

  "$GCI_BIN" write -s standard -s default -s "prefix(k8s)" -s "prefix(sigs.k8s)" -s "prefix(github.com)" -s "prefix(gitlab)" -s "prefix(github.com/openshift/rosa)" --custom-order --skip-generated --skip-vendor "$go_file"
  gofmt -s -w "$go_file"
  git add -- "$go_file"
done <<< "$staged_go_files"
