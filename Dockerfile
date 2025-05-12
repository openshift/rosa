FROM registry.access.redhat.com/ubi9/go-toolset:latest AS builder
COPY . .

ENV GOFLAGS=-buildvcs=false
RUN git config --global --add safe.directory /opt/app-root/src && \
    make release

FROM registry.access.redhat.com/ubi9/ubi-micro:latest
LABEL description="ROSA CLI"
LABEL io.k8s.description="ROSA CLI"
LABEL com.redhat.component="rh-rosa-cli"
LABEL distribution-scope="release"
LABEL name="rh-rosa-cli" release="vX.Y" url="https://github.com/openshift/rosa"
LABEL vendor="Red Hat, Inc."
LABEL version="vX.Y"

COPY --from=builder /opt/app-root/src/releases /releases
