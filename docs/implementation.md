# Implementation

This document is intended to be an overview of the implementation details of the service.

The system is comprised of three main components:

- REST API
- Dinosaur Worker
- Cluster Worker

![Component Architecture Diagram](images/kas-fleet-manager-component-architecture.png)

## REST API

The system has a REST API for managing Dinosaur resources. This is the primary interface used by
end-users to communicate with the service.

The OpenAPI spec for this API can be seen by running:

```
make run/docs
```

This will serve the OpenAPI spec on `localhost:80`

It's important to note that the system is asynchronous, meaning that once a Dinosaur resource is
created via the REST API, there won't be a running Dinosaur instance created by the time the HTTP
response is sent by the service. A client will need to continue watching or polling their Dinosaur
resource to determine whether it has a `ready` status or not.

The REST API requires a valid `Authorization` OCM Bearer token header to be provided with all
requests, to obtain a short-lived token run:

```
ocm token
```

## Dinosaur Workers

The Dinosaur Workers are responsible for reconciling Dinosaurs as requested by an end-user. 
There are currently six dinosaur workers:
- `dinosaur_mgr.go` responsible for reconciling dinosaur metrics and performing cleanup of trial dinosaurs, and cleanup of dinosaurs of denied owners. 
- `deleting_dinosaur_mgr.go` responsible for handling the deletion of dinosaurs e.g removing resources like AWS Route53 entry, Keycloak secrets client
- `accepted_dinosaur_mgr.go` responsible for checking if user is within Quota before provisioning a dinosaur. Afterwards, it will periodically reconcile on all pending Dinosaur resources, attempt to find a valid OpenShift cluster to fit it's requirements (cloud provider, region, etc.) and provision a Dinosaur instance to the cluster. Once a suitable Dataplane cluster has been found, we'll update the status of the Dinosaur resource to reflect it's current progress. 
- `preparing_dinosaur_mgr.go` responsible for creating external resources e.g AWS Route53 DNS, Keycloak authentication secrets 
- `provisioned_dinosaur_mgr.go` responsible for checking if a provisioned dinosaur is ready as reported by the fleetshard-operator
- `ready_dinosaur_mgr` responsible for reconciling external resources of a ready dinosaur e.g keycloak client and secret

Once the Dinosaur Workers have set up a Dinosaur resource, the status of the Dinosaur request will be `ready`.
If provisioning of a dinosaur fails, the status will be `failed` and a failed reason will be capture in the database. 
A deleted dinosaur has a final state of `deleting`, and it will appear in the database as a soft deleted record with a `deleted_at` timestamp different from `NULL`. 

The end-user has no way to directly interact with the Dinosaur worker, management of Dinosaur resources should be handled through the REST API.
## Cluster Worker

The Cluster Worker is responsible for reconciling OpenShift clusters and ensuring they are in a
state capable of hosting Dinosaur instances, this process is referred to as terraforming in this
service.

Once a cluster has been provisioned and terraformed, it is considered capable of hosting Dinosaur
resources as it's status will be marked as `ready`. This cluster will then be visible to the Dinosaur
Worker when it is searching for clusters.

The end-user has no way to directly interact with the Cluster Worker.
