FROM registry.access.redhat.com/ubi8/ubi-minimal:8.4

COPY \
    fleet-manager \
    /usr/local/bin/

EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/fleet-manager", "serve"]

LABEL name="fleet-manager" \
      vendor="Red Hat" \
      version="0.0.1" \
      summary="FleetManager" \
      description="Dinosaur Service Fleet Manager"
