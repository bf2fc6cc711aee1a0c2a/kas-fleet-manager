FROM registry.ci.openshift.org/openshift/release:golang-1.17 AS builder

WORKDIR /workspace

COPY . ./

RUN make binary

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4

COPY --from=builder /workspace/kas-fleet-manager /usr/local/bin/

EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/kas-fleet-manager", "serve"]

LABEL name="kas-fleet-manager" \
      vendor="Red Hat" \
      version="0.0.1" \
      summary="KasFleetManager" \
      description="Kafka Service Fleet Manager"
