#!/usr/bin/env bash

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

if ! git diff --quiet --exit-code; then
  echo "Push blocked: unstaged changes detected. Commit or stash changes before pushing."
  exit 1
fi

if ! git diff --cached --quiet --exit-code; then
  echo "Push blocked: staged but uncommitted changes detected. Commit or stash changes before pushing."
  exit 1
fi

set +e
make --no-print-directory pre-push-checks
checks_exit_code=$?
set -e

if [ "$checks_exit_code" -ne 0 ]; then
  echo
  if [ "$checks_exit_code" -eq 130 ] || [ "$checks_exit_code" -eq 143 ]; then
    echo "Push blocked: pre-push checks interrupted"
  else
    echo "Push blocked: pre-push checks failed"
  fi
  exit 1
fi

echo
echo "Pre-push checks passed."
