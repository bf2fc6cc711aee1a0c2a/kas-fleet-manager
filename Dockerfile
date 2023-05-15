FROM registry.access.redhat.com/ubi9-minimal:9.2-484 AS builder
 
RUN microdnf install -y tar gzip make which git

# install go 1.19.6
RUN curl -O -J https://dl.google.com/go/go1.19.6.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.19.6.linux-amd64.tar.gz
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go

WORKDIR /workspace

# Copy Go module related files. Done as a separate
# step to invalidate the next layers in case of a change
# in the dependency specifications
COPY go.mod go.sum ./
RUN go mod download

# Copy rest of the files containing
# the source code
COPY . ./

RUN make binary

FROM registry.access.redhat.com/ubi9-minimal:9.2-484

COPY --from=builder /workspace/kas-fleet-manager /usr/local/bin/

EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/kas-fleet-manager", "serve"]

LABEL name="kas-fleet-manager" \
      vendor="Red Hat" \
      version="0.0.1" \
      summary="KasFleetManager" \
      description="Kafka Service Fleet Manager"
