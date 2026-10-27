#!/usr/bin/env bash
# Copyright Red Hat
# SPDX-License-Identifier: Apache-2.0

# check-version-env-isolation.sh - Regression guard for ROSAENG-67768.
#
# Base images such as ubi9/go-toolset export a VERSION environment variable
# (their own toolset version). This check asserts that an inherited VERSION
# environment variable, or an inherited ROSA_VERSION environment variable,
# never changes the linker flags the build computes. It only inspects
# `make print-ldflags` output, so it never invokes `go build` and leaves the
# repo's normal `./rosa` binary and build cache untouched.

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

injected_version="99.99.99-leaked-env-version"

baseline_ldflags=$(make --no-print-directory print-ldflags)

version_env_ldflags=$(VERSION="$injected_version" make --no-print-directory print-ldflags)
if [[ "$version_env_ldflags" != "$baseline_ldflags" || "$version_env_ldflags" == *"$injected_version"* ]]; then
  echo "FAIL: an inherited VERSION environment variable changed the computed linker flags" >&2
  echo "Baseline: $baseline_ldflags" >&2
  echo "With VERSION set: $version_env_ldflags" >&2
  exit 1
fi

rosa_version_env_ldflags=$(ROSA_VERSION="$injected_version" make --no-print-directory print-ldflags 2>/dev/null)
if [[ "$rosa_version_env_ldflags" != "$baseline_ldflags" || "$rosa_version_env_ldflags" == *"$injected_version"* ]]; then
  echo "FAIL: an inherited ROSA_VERSION environment variable changed the computed linker flags" >&2
  echo "Baseline: $baseline_ldflags" >&2
  echo "With ROSA_VERSION set: $rosa_version_env_ldflags" >&2
  exit 1
fi

explicit_override_ldflags=$(make --no-print-directory ROSA_VERSION="$injected_version" print-ldflags)
if [[ "$explicit_override_ldflags" != *"$injected_version"* ]]; then
  echo "FAIL: an explicit ROSA_VERSION=... command-line override was not honored" >&2
  echo "Got: $explicit_override_ldflags" >&2
  exit 1
fi

echo "PASS: linker flags are isolated from inherited VERSION/ROSA_VERSION environment variables, and explicit command-line overrides still work"
