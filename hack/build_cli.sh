#!/bin/bash
#This script invoked via a make target by the Dockerfile
#which builds a cli wrapper container that contains all release images
archs=(amd64 arm64)
oses=(darwin linux windows)

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
    GOOS="${os}" GOARCH="${arch}" go build -o "${tmpdir}/rosa${extension}" ./cmd/rosa
    tar -czf "releases/rosa_${os}_${arch}.tar.gz" -C "${tmpdir}" "rosa${extension}"
    rm -rf "${tmpdir}"
  done
done
}

build_release
