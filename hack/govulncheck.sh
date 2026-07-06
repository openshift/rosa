#!/usr/bin/env bash

# govulncheck.sh - Run source and binary vulnerability scans for the ROSA CLI.
#
# Source mode scans ./... for reachable dependency CVEs.
# Binary mode scans the compiled rosa binary for stdlib/toolchain CVEs.

set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

wrapper="$repo_root/hack/govulncheck-wrapper.sh"
rosa_binary="$repo_root/rosa"

if [[ ! -x "$wrapper" ]]; then
	echo "govulncheck wrapper not executable: $wrapper" >&2
	exit 1
fi

echo "Running govulncheck source scan (./...)..."
"$wrapper" --mode source ./...

if [[ ! -f "$rosa_binary" ]]; then
	echo "rosa binary not found at $rosa_binary; build it with 'make rosa' before running govulncheck" >&2
	exit 1
fi

echo "Running govulncheck binary scan ($rosa_binary)..."
"$wrapper" --mode binary "$rosa_binary"

echo "govulncheck passed (source and binary scans)"
