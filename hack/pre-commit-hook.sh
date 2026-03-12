#!/usr/bin/env bash

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

if git diff --cached --quiet --exit-code; then
  exit 0
fi

set +e
make --no-print-directory pre-commit-checks
checks_exit_code=$?
set -e

if [ "$checks_exit_code" -ne 0 ]; then
  echo
  if [ "$checks_exit_code" -eq 130 ] || [ "$checks_exit_code" -eq 143 ]; then
    echo "Commit blocked: pre-commit checks interrupted"
  else
    echo "Commit blocked: pre-commit checks failed"
  fi
  exit 1
fi

echo
echo "Pre-commit checks passed."
