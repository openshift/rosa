#!/usr/bin/env bash

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

if [ ! -f ".githooks/commit-msg" ] || [ ! -f ".githooks/pre-commit" ] || [ ! -f ".githooks/pre-push" ]; then
  echo "Missing expected hook files under .githooks/"
  exit 1
fi

git config --local core.hooksPath .githooks
chmod +x .githooks/commit-msg
chmod +x .githooks/pre-commit
chmod +x .githooks/pre-push

echo "Installed local git hooks (.githooks/commit-msg, .githooks/pre-commit, .githooks/pre-push). YOU MUST RUN THESE HOOKS ON EVERY COMMIT AND PUSH."
