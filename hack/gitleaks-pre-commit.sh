#!/usr/bin/env bash
# Copyright Red Hat
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

if [[ ",${SKIP:-}," == *,gitleaks,* ]]; then
  echo "Commit blocked: SKIP=gitleaks is not permitted." >&2
  exit 1
fi

env -u SKIP pre-commit run --hook-stage pre-commit gitleaks
