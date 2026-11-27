FROM registry.access.redhat.com/ubi10/go-toolset:1.27.4-1795507689 AS builder
COPY --chown=1001:0 . .

RUN git config --global --add safe.directory /opt/app-root/src && \
    make release

# Runtime base is deliberately decoupled from the builder above: the Makefile exports
# CGO_ENABLED=0, so this stage only receives release zips of static binaries that it
# never executes.
FROM registry.access.redhat.com/ubi9/ubi-micro:latest
LABEL cpe="cpe:/a:redhat:openshift_service_on_aws:1.2::el9"
LABEL description="Red Hat OpenShift Service on AWS (ROSA) CLI"
LABEL io.k8s.description="Red Hat OpenShift Service on AWS (ROSA) CLI"
LABEL io.k8s.display-name="Red Hat OpenShift Service on AWS (ROSA) CLI"
LABEL summary="Red Hat OpenShift Service on AWS (ROSA) CLI"
LABEL com.redhat.component="rh-rosa-cli"
LABEL distribution-scope="release"
LABEL name="rh-rosa-cli" release="vX.Y" url="https://github.com/openshift/rosa"
LABEL vendor="Red Hat, Inc."
LABEL version="vX.Y"

COPY --from=builder /opt/app-root/src/releases /releases
