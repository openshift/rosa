#!/bin/bash
set -euo pipefail

#This script invoked via a make target by the Dockerfile
#which builds a cli wrapper container that contains all release images
archs=(amd64 arm64)
oses=(darwin linux windows)

mkdir -p releases

build_release() {
# Resolve exact tag on HEAD (stable or prerelease). Prerelease builds
# intentionally report the prerelease version (e.g. 1.2.66-rc1).
release_version=$(git describe --tags --exact-match 2>/dev/null || true)
release_version=${release_version#v}

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

release_version=$(git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD)
release_version=${release_version#v}

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
