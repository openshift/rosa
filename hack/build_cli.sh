#!/bin/bash
set -euo pipefail

#This script invoked via a make target by the Dockerfile
#which builds a cli wrapper container that contains all release images
archs=(amd64 arm64)
oses=(darwin linux windows)
release_version=$(git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD)
release_version=${release_version#v}

mkdir -p releases

# CDN (Content Gateway) uses the basename of source paths from the RPA.
# Legacy mirror names must be produced at build time; filename: in the RPA
# is ignored for CGW pushes.
cdn_archive_name() {
  case "${1}/${2}" in
    linux/amd64) echo "rosa-linux.tar.gz" ;;
    linux/arm64) echo "rosa-linux-arm64.tar.gz" ;;
    darwin/amd64) echo "rosa-macosx.tar.gz" ;;
    darwin/arm64) echo "rosa-macos-arm64.tar.gz" ;;
    windows/amd64) echo "rosa-windows.tar.gz" ;;
    windows/arm64) echo "rosa-windows-arm64.tar.gz" ;;
    *) echo "rosa_${1}_${2}.tar.gz" ;;
  esac
}

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
    GOOS="${os}" GOARCH="${arch}" go build -ldflags="-X github.com/openshift/rosa/pkg/info.Build=$(git rev-parse --short HEAD)${release_version:+ -X github.com/openshift/rosa/pkg/info.DefaultVersion=${release_version}}" -o "${tmpdir}/rosa${extension}" ./cmd/rosa
    archive_name=$(cdn_archive_name "${os}" "${arch}")
    tar -czf "releases/${archive_name}" -C "${tmpdir}" "rosa${extension}"
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

if [[ ! "$release_version" =~ ^[a-zA-Z0-9._-]+$ ]]; then
  echo "ERROR: unexpected release_version '${release_version}'" >&2
  exit 1
fi

cat > "releases/rosa_${release_version}_metadata.json" <<METADATA
{
  "product": "Red Hat OpenShift Service on AWS (ROSA) CLI",
  "version": "${release_version}",
  "commit": "$(git rev-parse HEAD)",
  "platforms": [
    "linux/amd64", "linux/arm64",
    "darwin/amd64", "darwin/arm64",
    "windows/amd64", "windows/arm64"
  ],
  "formats": ["tar.gz", "zip"]
}
METADATA

(cd releases && sha256sum -- *.tar.gz *.zip *.json > "rosa_${release_version}_SHA256SUMS")
