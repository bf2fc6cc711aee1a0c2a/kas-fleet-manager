# Implementation

This document is intended to be an overview of the implementation details of the service.

The system is comprised of three main components:

- REST API
- Kafka Worker
- Cluster Worker

## REST API

The system has a REST API for managing Kafka resources. This is the primary interface used by
end-users to communicate with the service.

The OpenAPI spec for this API can be seen by running:

```
make run/docs
```

This will serve the OpenAPI spec on `localhost:80`

It's important to note that the system is asynchronous, meaning that once a Kafka resource is
created via the REST API, there won't be a running Kafka instance created by the time the HTTP
response is sent by the service. A client will need to continue watching or polling their Kafka
resource to determine whether it has a `complete` status or not.

The REST API requires a valid `Authorization` OCM Bearer token header to be provided with all
requests, to obtain a short-lived token run:

```
ocm token
```

## Kafka Worker

The Kafka Worker is responsible for reconciling Kafka resources requested by an end-user on to a
cluster and updating the status of the Kafka resource to reflect it's current progress. It will
periodically reconcile on all pending Kafka resources, attempt to find a valid OpenShift cluster to
fit it's requirements (cloud provider, region, etc.) and provision a Kafka instance to the cluster.

Once the Kafka Worker has set up a Kafka resource, it will mark the Kafka request status as
`complete`.

The end-user has no way to directly interact with the Kafka worker, management of Kafka resources
should be handled through the REST API.

## Cluster Worker

The Cluster Worker is responsible for reconciling OpenShift clusters and ensuring they are in a
state capable of hosting Kafka instances, this process is referred to as terraforming in this
service.

Once a cluster has been provisioned and terraformed, it is considered capable of hosting Kafka
resources as it's status will be marked as `ready`. This cluster will then be visible to the Kafka
Worker when it is searching for clusters.

The end-user has no way to directly interact with the Cluster Worker.