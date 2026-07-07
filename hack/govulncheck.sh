#!/usr/bin/env bash

# govulncheck.sh - Run source and binary vulnerability scans for the ROSA CLI.
#
# Source mode scans ./... for reachable dependency CVEs.
# Binary mode scans the compiled rosa binary for stdlib/toolchain CVEs.

set -euo pipefail

# renovate: datasource=github-releases depName=jqlang/jq extractVersion=^jq-(?<version>.+)$
readonly jq_version="1.8.2"

verify_jq_checksum() {
	local jq_dir=$1
	local jq_asset=$2
	local jq_bin=$3
	local sha256_file="${jq_dir}/sha256sum.txt"
	local expected actual

	curl -fsSL --retry 5 --retry-delay 2 \
		-o "$sha256_file" \
		"https://github.com/jqlang/jq/releases/download/jq-${jq_version}/sha256sum.txt"

	expected=$(grep -F " ${jq_asset}" "$sha256_file" | awk '{print $1}')
	if [[ -z "$expected" ]]; then
		echo "jq checksum not found in upstream sha256sum.txt for ${jq_asset}" >&2
		return 1
	fi

	actual=$(sha256sum "$jq_bin" | awk '{print $1}')
	if [[ "$expected" != "$actual" ]]; then
		echo "jq checksum mismatch for ${jq_asset}" >&2
		rm -f "$jq_bin"
		return 1
	fi
}

ensure_jq() {
	if command -v jq >/dev/null 2>&1; then
		return 0
	fi

	if [[ "$(uname -s)" != "Linux" ]]; then
		echo "jq not found; install jq locally (automatic bootstrap is Linux-only)" >&2
		return 1
	fi

	if ! command -v curl >/dev/null 2>&1; then
		echo "jq not found and curl is unavailable to download a static binary" >&2
		return 1
	fi

	local jq_dir jq_bin jq_asset arch
	jq_dir="${TMPDIR:-/tmp}/rosa-govulncheck-jq/${jq_version}"
	jq_bin="${jq_dir}/jq"
	mkdir -p "$jq_dir"

	case "$(uname -m)" in
	x86_64 | amd64) arch=amd64 ;;
	aarch64 | arm64) arch=arm64 ;;
	s390x) arch=s390x ;;
	ppc64le) arch=ppc64el ;;
	*)
		echo "unsupported architecture for jq bootstrap: $(uname -m)" >&2
		return 1
		;;
	esac

	jq_asset="jq-linux-${arch}"

	if [[ ! -x "$jq_bin" ]]; then
		echo "jq not found; downloading jq ${jq_version} for ${arch}..."
		curl -fsSL --retry 5 --retry-delay 2 \
			-o "$jq_bin" \
			"https://github.com/jqlang/jq/releases/download/jq-${jq_version}/${jq_asset}"
		chmod +x "$jq_bin"
		verify_jq_checksum "$jq_dir" "$jq_asset" "$jq_bin"
	fi

	export PATH="${jq_dir}:${PATH}"
}

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"

ensure_jq

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
