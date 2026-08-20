#!/bin/bash
set -euo pipefail

#This script invoked via a make target by the Dockerfile
#which builds a cli wrapper container that contains all release images
archs=(amd64 arm64)
oses=(darwin linux windows)
release_version=$(git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD)
release_version=${release_version#v}

mkdir -p releases

build_release() {
for os in "${oses[@]}"
do
  for arch in "${archs[@]}"
  do
    extension=""
    if [[ "$os" == "windows" ]]; then
        extension=".exe"
    fi
    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT
    GOOS="${os}" GOARCH="${arch}" go build -ldflags="-X github.com/openshift/rosa/pkg/info.Build=$(git rev-parse --short HEAD)" -o "${tmpdir}/rosa${extension}" ./cmd/rosa
    tar -czf "releases/rosa_${os}_${arch}.tar.gz" -C "${tmpdir}" "rosa${extension}"
    (
      cd "${tmpdir}" && \
      zip -r "${OLDPWD}/releases/rosa_${os}_${arch}.zip" "rosa${extension}"
    )
    rm -rf "${tmpdir}"
    trap - EXIT
  done
done
}

build_release

# Konflux release-to-github expects a single *_SHA256SUMS manifest.
(
  cd releases || exit 1
  sha256sum rosa_*.tar.gz > "rosa_${release_version}_SHA256SUMS"
)
