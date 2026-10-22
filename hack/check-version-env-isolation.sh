#!/usr/bin/env bash
# Copyright Red Hat
# SPDX-License-Identifier: Apache-2.0

# check-version-env-isolation.sh - Regression guard for ROSAENG-67768.
#
# Base images such as ubi9/go-toolset export a VERSION environment variable
# (their own toolset version). Make's `?=` treats an inherited environment
# variable as already set, so a Makefile variable literally named VERSION
# would silently adopt that value instead of the git-derived one. This check
# builds with VERSION pre-set in the environment and asserts the resulting
# ROSA CLI version does not leak that value. It builds into a temporary
# location so the repo's normal `./rosa` binary is left untouched.

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

injected_version="99.99.99-leaked-env-version"

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/rosa-version-env-isolation.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT
tmp_binary="$tmp_dir/rosa"

ldflags=$(VERSION="$injected_version" make --no-print-directory print-ldflags)

echo "Building rosa with VERSION pre-set in the environment (simulating ubi9/go-toolset)..."
go build -ldflags="$ldflags" -o "$tmp_binary" ./cmd/rosa

reported_version=$("$tmp_binary" version --client)

if [[ "$reported_version" == *"$injected_version"* ]]; then
  echo "FAIL: ROSA CLI reported the environment-inherited VERSION ($injected_version) instead of the git-derived version" >&2
  echo "Reported: $reported_version" >&2
  exit 1
fi

echo "PASS: ROSA CLI version ($reported_version) is isolated from an inherited VERSION environment variable"
