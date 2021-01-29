FROM quay.io/centos/centos:8

RUN \
    yum install -y \
    openssl \
    postgresql \
    && \
    yum clean all

COPY \
    kas-fleet-manager \
    /usr/local/bin/

EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/kas-fleet-manager", "serve"]

LABEL name="kas-fleet-manager" \
      vendor="Red Hat" \
      version="0.0.1" \
      summary="KasFleetManager" \
      description="Kafka Service Fleet Manager"
