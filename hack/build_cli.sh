#!/bin/bash
#This script invoked via a make target by the Dockerfile
#which builds a cli wrapper container that contains all release images
archs=(amd64)
oses=(darwin linux windows)

mkdir -p releases

build_release() {
for os in ${oses[@]}
do
  for arch in ${archs[@]}
  do
    if [[ $os == "windows" ]]; then
        extension=".exe"
    fi
    GOOS=${os} GOARCH=${arch} go build -o /tmp/ocm_${os}_${arch} ./cmd/ocm
    mv /tmp/ocm_${os}_${arch} ocm${extension}
    zip releases/ocm_${os}_${arch}.zip ocm${extension}
    rm ocm${extension}
  done
done
cd releases
}

build_release