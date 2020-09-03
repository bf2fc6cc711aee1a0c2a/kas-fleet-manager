FROM centos:7

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
