#!/usr/bin/env bash

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

if [ "$#" -ne 1 ]; then
  echo "Commit blocked: commit message file path argument is required."
  exit 1
fi

message_file=$1
if [ ! -f "$message_file" ]; then
  echo "Commit blocked: unable to read commit message file: $message_file"
  exit 1
fi

set +e
"$repo_root/hack/commit-msg-verify.sh" "$message_file"
checks_exit_code=$?
set -e

if [ "$checks_exit_code" -ne 0 ]; then
  echo
  if [ "$checks_exit_code" -eq 130 ] || [ "$checks_exit_code" -eq 143 ]; then
    echo "Commit blocked: commit message check interrupted"
  else
    echo "Commit blocked: commit message check failed"
  fi
  exit 1
fi

echo
echo "Commit message is valid."
