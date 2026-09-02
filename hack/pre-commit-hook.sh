#!/usr/bin/env bash

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

if git diff --cached --quiet --exit-code; then
  exit 0
fi

if ! command -v pre-commit >/dev/null 2>&1; then
  echo "Commit blocked: pre-commit is required for gitleaks (see CONTRIBUTING.md)"
  exit 1
fi

# pre-commit honors SKIP; never allow inherited SKIP=gitleaks to bypass this scan.
gitleaks_pre_commit_env=()
if [ -n "${SKIP:-}" ]; then
  cleaned_skip=""
  IFS=',' read -r -a skip_hooks <<< "$SKIP"
  for hook in "${skip_hooks[@]}"; do
    hook="${hook#"${hook%%[![:space:]]*}"}"
    hook="${hook%"${hook##*[![:space:]]}"}"
    if [ "$hook" = "gitleaks" ]; then
      continue
    fi
    if [ -n "$cleaned_skip" ]; then
      cleaned_skip+=",$hook"
    else
      cleaned_skip="$hook"
    fi
  done
  if [ -n "$cleaned_skip" ]; then
    gitleaks_pre_commit_env=(env "SKIP=$cleaned_skip")
  else
    gitleaks_pre_commit_env=(env -u SKIP)
  fi
else
  gitleaks_pre_commit_env=(env -u SKIP)
fi

set +e
"${gitleaks_pre_commit_env[@]}" pre-commit run gitleaks --config .pre-commit-config.yaml
gitleaks_exit_code=$?
set -e

if [ "$gitleaks_exit_code" -ne 0 ]; then
  echo
  if [ "$gitleaks_exit_code" -eq 130 ] || [ "$gitleaks_exit_code" -eq 143 ]; then
    echo "Commit blocked: gitleaks pre-commit check interrupted"
  else
    echo "Commit blocked: gitleaks pre-commit check failed"
  fi
  exit 1
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
