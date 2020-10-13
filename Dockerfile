FROM registry.access.redhat.com/ubi7/ubi

RUN \
    yum install -y \
    openssl \
    postgresql \
    && \
    yum clean all

COPY \
    managed-services-api \
    /usr/local/bin/

EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/managed-services-api", "serve"]

LABEL name="managed-services-api" \
      vendor="Red Hat" \
      version="0.0.1" \
      summary="Managed Service API" \
      description="Managed Service API"
