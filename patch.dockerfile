FROM registry.ci.openshift.org/openshift/release:golang-1.14

COPY . /src
WORKDIR /src/internal/patch
RUN go build

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
LABEL io.openshift.managed.name="managed-velero-plugin-status-patch" \
      io.openshift.managed.description="Velero sidekick to apply status after creation"

COPY --from=0 /src/internal/patch/patch /bin/patch

ENTRYPOINT ["bin/patch"]
