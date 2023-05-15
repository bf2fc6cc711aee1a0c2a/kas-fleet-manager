FROM registry.access.redhat.com/ubi9-minimal:9.2-484

RUN curl -Lo /usr/local/bin/yq https://github.com/mikefarah/yq/releases/download/v4.18.1/yq_linux_amd64 && \
    chmod +x /usr/local/bin/yq

RUN microdnf install -y git

COPY ./create_release.sh /src/

WORKDIR /src

CMD ["./create_release.sh"]
