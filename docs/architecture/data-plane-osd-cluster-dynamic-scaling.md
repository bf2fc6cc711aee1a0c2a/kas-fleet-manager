# Data Plane OSD Cluster Dynamic Scaling functionality

## Description

This documents explains how Data Plane OSD Cluster Dynamic Scaling functionality
has been implemented.

Data Plane OSD Cluster Dynamic Scaling functionality currently deals with:
* Dynamic scaling of OSD Compute Nodes
* OSD Cluster Status capacity calculation
* Dynamic scaling of OSD Clusters

Dynamic scaling evaluation/trigger for a given Data Plane OSD cluster is
initiated and performed by KAS Fleet Shard Operator on the data plane, calling
the `/agent-clusters/{cluster_id}/status` endpoint provided by KAS Fleet Manager.
The information provided when this call is performed is defined in the private
OpenAPI spec file in the `DataPlaneClusterUpdateStatusRequest` data type.

The implementation assumes that the `.status.resizeInfo.nodeDelta` attribute
provided as part of the `DataPlaneClusterUpdateStatusRequest` information
will be always 3 and that resizeInfo.delta capacity attributes will not change
in value. In case that assumption changes then the implementation will need
adjustment in the future.

The number of nodes to scale-up and scale-down when a scaling action is performed
is taken from `.status.resizeInfo.nodeDelta`.

In order for the dynamic scaling evaluation to be processed the following conditions
need to happen:
* The agent must be in a 'ready' state
* The OSD cluster must be in one of the states accepted by the dynamic scaling
  functionality (implementation details on code)

## OSD Cluster Capacity Calculation

To calculate whether the cluster being evaluated has available capacity
Kas Fleet Manager will check whether the number of remaining Kafka Connections
or remaining Kafka Partitions (provided in `.status.remaining`) is less than
what a single Kafka Cluster of Model T consumes correspondingly.

If there's available capacity then the cluster will be marked as `ready` if it
wasn't already.

If there's no available capacity:
* If `.status.nodeInfo.current` is less than the restricted ceiling
  value (see [Scale-Up criteria](#scale-up-criteria)) then the cluster will be
  marked as `compute_node_scaling_up` if it wasn't already. The
  `compute_node_scaling_up` state means that the data plane cluster has no
  available capacity with the current number of compute nodes but more compute
  nodes can be added and a scale-up of compute nodes is in progress as a
  consequence
* If `status.nodeInfo.current` is higher or equal than the restricted ceiling
  value then the cluster will be marked as `full` if it wasn't already. The
  `full` states means that the data plane cluster has no available capacity with
  the current number of compute nodes it has and no more compute nodes can be
  added

The capacity calculation will be performed from the data received by KAS Fleet
Shard Operator, BEFORE performing the scaling actions.

## Compute nodes Scale-Up criteria

KAS Fleet Manager will scale up compute nodes of a data plane cluster if
all the following conditions are true:

* **At least one** of the reported Kafka attribute values has crossed its
  corresponding Scale-Up threshold
* The current number of nodes is smaller than the restricted
  ceiling value (see below for a definition of restricted ceiling)
* The number of nodes that would result after performing scale-up is
  still less or equal than the restricted ceiling value.
* There's no scaling action already in progress

Restricted Ceiling: The provided ceiling in the
status (`status.nodeInfo.ceiling`) but rounded down to  multi-az node
requirements (3) ONLY in case the cluster is multi-az. In case
cluster isn't multi-az then restricted ceiling is the same as reported ceiling

Scale-Up thresholds:
  * Kafka Connections: The number of connections required by a single Kafka Cluster of Model T
  * Kafka Partitions: The number of partitions required by a single Kafka Cluster of Model T

### Compute nodes Scale-Up value calculation

Once KAS Fleet Manager has decided to perform scale up of a data plane's cluster
compute nodes, it will try to scale up by the value specified
in `.status.resizeInfo.nodeDelta`. If the cluster being
scaled is a Multi-AZ cluster and `.status.resizeInfo.nodeDelta` is not a multiple
of the Multi-AZ required number of nodes (3) then the value to increase will be
round-up to that nearest multiple.

(At least) for the initial version, when performing the scale-up increase, if
that increment does not reach the ScaleUpThreshold we will still increase only
that value. Then, on subsequent calls to the agent status report endpoint it
will continue scaling step by step until the threshold is reached. In this way
it will scale in a more controlled way.

## Compute nodes Scale-Down criteria

KAS Fleet Manager will scale down compute nodes of a data plane cluster if all
the following conditions are true:

* **All** (notice the difference with scale-up criteria) of the reported Kafka
  attribute values have crossed their corresponding Scale-Down threshold
* The current number of nodes is strictly higher than
  the `.status.nodeInfo.currentWorkLoadMinimum` value
* The number of nodes that would remain after performing the Scale-Down value
  calculation (see subsection below) is higher
  than `.status.nodeInfo.currentWorkLoadMinimum` , higher than restricted floor,
  higher than multi-az multiple requirement (3) and higher
  than 0.
* There's no scaling action already in progress

Restricted Floor: The provided floor in the
status (`status.nodeInfo.floor`) but rounded up to multi-az node
requirements (3) ONLY in case the cluster is multi-az. In case
cluster isn't multi-az then restricted floor is the same as reported floor

Scale-Down thresholds:
  * Kafka Connections: The connections specified in `resizeInfo.delta.connections`.
    Due to it is currently being assumed `resizeInfo.nodeDelta` will always be
    3 this means that it is a value equivalent to 3 full OSD Compute nodes worth
    of connections
  * Kafka Partitions: The partitions specified in `resizseInfo.delta.partitions`.
    Due to it is currently being assumed `resizeInfo.nodeDelta` will always be 3
    this means that it is a value equivalent to 3 full OSD Compute nodes worth
    of partitions

### Compute nodes Scale-Down value calculation

Once KAS Fleet Manager has decided to perform scale down of a data plane's compute
nodes, it will try to scale down by the value specified
in `.status.resizeInfo.nodeDelta`. If the cluster
being scaled is a Multi-AZ cluster and `.status.resizeInfo.nodeDelta` is not a
multiple of the Multi-AZ required number of nodes (3) then the value to
decrease will be round-up to that nearest multiple.

(At least) for the initial version, when performing the nodeDelta decrease, if
that decrement does not reach the ScaleDownThreshold we will still decrease
only the specified nodeDelta  value anyway. Then, on subsequent calls to the
agent status report endpoint it will continue scaling down until the threshold
or currentWorkLoadMin is reached. In this way it will scale in a more
controlled way

## OSD Cluster Scale-Down criteria and value calculation

The system will delete an OSD cluster, let's call it `clusterA`, from the pool of 
available clusters when the following conditions are met:

  * `clusterA` is empty i.e there are no longer any Kafka instances on `clusterA`
  * `clusterA` is in a ready state
  * `clusterA` has at least one sibling cluster that satisfies these conditions:
    * Has the same Cloud Provider as `clusterA`
    * In the same region as `clusterA`
    * That sibling is also in a 'ready' state

Once all of these conditions are met, it is safe to assume that deletion 
of `clusterA` shall not lead to any disruption of service and subsequently it's 
status is changed to *deprovisioning*. 

A call is then made to OCM to delete it. 
`clusterA` is then *soft deleted* from from the database.
`clusterA`'s external dependencies are then removed: 
This step consists of removing the keycloak client created for this cluster's IDP along with removing the kas-fleetshard-operator service account.

## OSD Cluster Scale-Up criteria and value calculation

The cluster reconciler periodically evaluates the state of all data plane
clusters stored in the KAS Fleet Manager database.

KAS Fleet Manager will evaluate each defined provider's regions. For each
region it will create a new OSD cluster in it (a scale-up will be triggered at
OSD cluster level) if all the following conditions are true:
* All of the clusters in provider's region are in state `full` or `failed`

The number of OSD clusters that will be created when a scale-up is triggered
will be `1`