#!/usr/bin/env bash
# Copyright Red Hat
# SPDX-License-Identifier: Apache-2.0
#
# Install a pinned gitleaks release into a destination directory.
# Usage: install-gitleaks.sh <version> <dest_dir>
# Example: install-gitleaks.sh v8.30.1 ./bin

set -euo pipefail

version="${1:?version required (e.g. v8.30.1)}"
dest_dir="${2:?destination directory required}"

version_no_v="${version#v}"
arch="$(uname -m | sed 's/x86_64/x64/;s/aarch64/arm64/')"
os="$(uname | tr '[:upper:]' '[:lower:]')"
case "$os" in
  mingw*|msys*|cygwin*) os="windows" ;;
esac

case "${os}_${arch}" in
  linux_x64) platform_suffix="linux_x64" archive_ext="tar.gz" ;;
  linux_arm64) platform_suffix="linux_arm64" archive_ext="tar.gz" ;;
  darwin_x64) platform_suffix="darwin_x64" archive_ext="tar.gz" ;;
  darwin_arm64) platform_suffix="darwin_arm64" archive_ext="tar.gz" ;;
  windows_x64) platform_suffix="windows_x64" archive_ext="zip" ;;
  *)
    echo "Unsupported platform for gitleaks: ${os}_${arch}" >&2
    exit 1
    ;;
esac

sha256_verify() {
  local archive_path=$1
  local checksum_path=$2
  local archive_name
  archive_name=$(basename "$archive_path")

  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$(dirname "$archive_path")" && grep -F " ${archive_name}" "$checksum_path" | sha256sum -c -)
  elif command -v shasum >/dev/null 2>&1; then
    (cd "$(dirname "$archive_path")" && grep -F " ${archive_name}" "$checksum_path" | shasum -a 256 -c -)
  else
    echo "sha256sum or shasum is required to verify ${archive_name}" >&2
    return 1
  fi
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
mkdir -p "$dest_dir"

asset="gitleaks_${version_no_v}_${platform_suffix}.${archive_ext}"
url="https://github.com/gitleaks/gitleaks/releases/download/${version}/${asset}"
checksums_url="https://github.com/gitleaks/gitleaks/releases/download/${version}/gitleaks_${version_no_v}_checksums.txt"

curl -fsSL --retry 3 --connect-timeout 10 --max-time 120 -o "${tmp}/${asset}" "$url"
curl -fsSL --retry 3 --connect-timeout 10 --max-time 120 -o "${tmp}/checksums.txt" "$checksums_url"
sha256_verify "${tmp}/${asset}" "${tmp}/checksums.txt"

if [ "$archive_ext" = "zip" ]; then
  unzip -o "${tmp}/${asset}" -d "$tmp"
  dest_bin="${dest_dir}/gitleaks.exe"
  install -m 0755 "${tmp}/gitleaks.exe" "$dest_bin"
else
  tar -xzf "${tmp}/${asset}" -C "$tmp" gitleaks
  dest_bin="${dest_dir}/gitleaks"
  install -m 0755 "${tmp}/gitleaks" "$dest_bin"
fi

echo "Installed gitleaks ${version} at ${dest_bin}"
