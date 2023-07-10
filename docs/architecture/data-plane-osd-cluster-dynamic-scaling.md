# Data Plane OSD Cluster Dynamic Scaling functionality

## Description

This documents explains how Data Plane OSD Cluster Dynamic Scaling functionality
has been implemented.

Data Plane OSD Cluster Dynamic Scaling functionality currently deals with:
* Autoscaling of worker nodes of an OSD cluster
* Prewarming of worker nodes 
* OSD Cluster creation and deletion

## Autoscaling of worker nodes of an OSD cluster

Autoscaling of worker nodes of an OSD cluster is done by leveraging the [Cluster Autoscaler](https://docs.openshift.com/container-platform/4.9/machine_management/applying-autoscaling.html) as described in [AP-15: Dynamic Scaling of Data Plane Clusters](https://architecture.bf2.dev/ap/15/). For Manual clusters, this has to be enabled manually. Worker node autoscaling is enabled by default for all clusters that are created dynamically by the Fleet manager

## Prewarming of worker nodes 

Data Plane worker node prewarming helps speed up the allocation of Kafka instances in the Data Planes managed by the Control Plane in some scenarios. It
is achieved by reserving resources in the Data Planes. The reservation of resources is specified in the form of reserved Kafka instances of a given
size and it is done on a per Kafka instance type basis.

When enabling and configuring Data Plane worker node prewarming for a given Kafka instance type, each Data Plane managed by the Control Plane maintains the
specified reserved capacity for that instance type at all times. In this way, if the available resources in the existing worker nodes of a Data Plane are not
enough to contain the reserved capacity then the number of worker nodes is automatically scaled up to be able to maintain the reserved capacity. By doing
so, the worker nodes can potentially be automatically scaled up ahead of time. In that case, when a Kafka instance is created there is no need to scale up
additional worker nodes and wait for them to be created.

>NOTE: Worker node horizontal scale up within a Data Plane cluster is limited to the maximum number of supported worker nodes supported by an OSD cluster in OCM.

Data Plane worker node prewarming is used to mitigate the wait time of Kafka instances provisioning due to new worker nodes being created for a given data
plane. How effectively it mitigates that wait time depends on the amount of reserved capacity. The more reserved capacity, the more effective
Data Plane Worker Node prewarming will be against actions like bursts of Kafka instance creations.

Thus, the decision on how much reserved capacity is configured as a tradeoff between:

* Cost: The more reserved capacity the more resources are consumed and thus have to be paid for
* The desired likelihood of new Kafka Instances creation having to wait for new worker nodes to be available so they can be allocated

### Configuration

Fleet manager exposes the [node prewarming configuration file](../../config/node-prewarming-configuration.yaml) that has the following structure.

```yaml
<instance-type>:
  num_reserved_instances: <desired-num-reserved-kafka-instances>
  base_streaming_unit_size: <desired-base-streaming-unit-size>
```

1. `<instance-type>`: is the name of the Kafka instance type that Data Plane worker node prewarming is to be configured for. 
Only instance types defined in the SUPPORTED_INSTANCE_TYPES template parameter can be configured. 
2. `<desired-num-reserved-kafka-instances>` is the number of reserved Kafka instances to be reserved in each Data Plane Cluster managed by KAS Fleet Manager
information_source
3. `<desired-base-streaming-unit-size>` is the name of the Kafka Instance Size of the defined <instance-type>. Only Kafka Instance sizes defined in the [supported kafka instance types](../../config/kafka-instance-types-configuration.yaml) for the <instance-type> template can be configured. The reserved Kafka instances will be of this Kafka Instance Size for the <instance-type>

For example, if I want that that at any moment in each Data Plane cluster there is a pre-warmed available capacity of 7 Kafka instances with Kafka Size x1 for standard instances the prewarming configuration would look like:

```yaml
  standard:
    num_reserved_instances: 7
    base_streaming_unit_size: x1
```

>NOTE: To disable pre-warming for a given instance type, set the `num_reserved_instances` to `0`. 
>NOTE: To disable pre-warming completely, set the configuration as empty `{}` or omit it completely. 

### Reserved ManagedKafka 

For each Data Plane cluster, the Fleet manager returns a list of reserved managed Kafkas depending on the instance type that's supported by the cluster. 
The number of generated reserved managed Kafkas in the cluster is the sum of the specified number of reserved instances among all instance types supported by the cluster. If the cluster is not in `ready` status the result is an empty list. Generated reserved Kafka names have the following naming schema:
`reserved-Kafka-<instance_type>-<Kafka_number>` where Kafka_number goes from `1..<num_reserved_instances_for_the_given_instance_type>`
Each generated reserved Kafka has a namespace equal to its name and it contains the `bf2.org/deployment: reserved` label which identifies it as a reserved deployment. The Strimzi, Kafka and Kafka IBP versions are always set to the latest version.  

## OSD Cluster creation and deletion

### OSD cluster capacity information

For each cluster, the fleet manager stores the dynamic capacity information per instance type. These information are useful for evaluation of the Kafka placement strategy onto a cluster and for cluster auto creation / deletion evaluation logic.

The struct is of the form:
```yaml
instance_type:
  max_nodes: <integer> # The maximum number of worker nodes for this instance type. This config is read from the dynamic_scaling_config.go and initialised once the cluster is in provisioned stage after when the machine pool has been created.
  max_units: <integer> # The maximum number of streaming units that can be provisioned on the given max_nodes. The information is static for a given number of max_nodes. It is computed and sent by the Fleetshard sync
  remaining_units: <integer> # The number of remaining streaming units that can be provisioned. This field is highly dynamic depending on the number of Kafkas provisioned in the Data Plane clusters. This is sent by the Fleetshard sync
```

For example:

For a cluster that supports both standard and developer instances, we could have the following information stored in the database. 
```yaml
{
 standard: {
   max_nodes: 30
   max_units: 10
   remaining_units: 4
 },
 developer: {
   max_nodes: 1,
   max_units: 10,
   remaining_units: 8
 }
}
```

1. To retrieve the information for a given cluster:
```go
cluster.RetrieveDynamicCapacityInfo()
```

the above method has to be called every time you want to check the information.

2. To set the infomation
```go
cluster.SetDynamicCapacityInfo(map[string]DynamicCapacityInfo{})
```

### OSD Cluster creation
#### OSD cluster creation evaluation
A new data plane cluster should be created for a given instance type in the given provider and region when the following conditions happen.
 1. If specified, the streaming units limit for the given instance type in
    the provider's region has not been reached
 2. There is no scale up action ongoing. A scale up action is ongoing
    if there is at least one cluster in the following states: `provisioning`,
    `provisioned`, `accepted`, `waiting_for_kas_fleetshard_operator`
 3. At least one of the two following conditions are true:
      * No cluster in the provider's region has enough capacity to allocate the biggest instance size of the given instance type
      * The free capacity (in streaming units) for the given instance type in the provider's region is smaller or equal than the defined slack
        capacity (also in streaming units) of the given instance type. Free capacity is defined as max(total) capacity - consumed capacity. 
        For the calculation of the max capacity:
        * Clusters in `deprovisioning` and `cleanup` state are excluded, as
          clusters into those states don't accept kafka instances anymore.
        * Clusters that are still not `ready` to accept kafka instance but that
          should eventually accept them (like accepted state for example)
          are included
          
>NOTE: cluster in `failed` state are not counted in capacity and limit calculations.
>NOTE: Region's limit and capacity slack are defined in the [supported cloud providers configuration](../../config/provider-configuration.yaml)

#### OSD cluster creation and terraforming

Once the fleet manager has evaluated that there is a need to create a cluster in a given region that supports a given instance type, it will proceed on creating a new cluster that has the following characteristics:
- `supported_instance_type` will be set to the Streaming unit instance type as evaluated in the scale up detection logic
- `cloud_provider` and `region` will be set to the cloud provider and region that's needs scaling up
- `provider_type` is harcoded to `ocm` as it is the only cluster provider that's support auto creation
- `status` is set to `accepted`
- `multi_az` will be set to `true` if the supported instance type is `standard`. Otherwise, it is set to `false` for developer supporting clusters
 
During provisioning phase, the default machine pool will be autoscaled and the `max` number of worker nodes is hardcoded to `18` while the minimum is hardcoded to `6`. The compute machine type is read from the configuration value. 

Once the cluster has been provisioned, dedicated machine pools will be created par supported instance type.
The created machine pools will be autoscaled: the minimum number of worker nodes is set to `1` if the cluster is single az. Otherwise it is set to `3`. 
The maximum number of worker nodes is the one given by the [dynamic_scaling_config.go](../../internal/kafka/internal/config/dynamic_scaling_config.go)
Once the machine pools have been created, the `max_nodes` for each supported instance type will be stored in the database.

Once the cluster has been terraformed, Fleetshard sync will send back the capacity information and they'll be stored in the database.

### OSD cluster deletion

#### OSD cluster deletion evaluation

A data plane cluter can be removed if the following conditions happen.

 1. If it is empty i.e does not contain any Kafka workload of any instance type.
 2. All the following conditions are true:
      * There is at least a cluster in the provider's region that has enough capacity to allocate
        the biggest instance size of the instance types supported by the empty cluster
      * There will be free capacity (in streaming units) for each instance type supported by the cluster in
        the provider's region that is smaller or equal than the defined slack capacity (also in streaming units) of the given instance type. 
        Free capacity is defined as max(total) capacity - consumed capacity.
        For the calculation of the max capacity:
        * Clusters in `deprovisioning` and `cleanup` state are excluded, as
          clusters into those states don't accept kafka instances anymore.
        * Clusters that are still not `ready` to accept kafka instance are also excluded from the capacity calculation
          
>NOTE: cluster in `failed` state are not counted in capacity and limit calculations.
>NOTE: Region's limit and capacity slack are defined in the [supported cloud providers configuration](../../config/provider-configuration.yaml)

#### OSD cluster deletion

Once the fleet manager has successfully detected that a cluster can be deleted, it'll mark the cluster as `deprovisioning`.
At this stage, it'll perform again the checks to see if the cluster is still empty and it can be safely deleted i.e without causing the cloud provider's region 
to be under capacity. If all the conditions are satisfying then the fleet manager will delete the cluster from the cluster provider, deletes the external resources held by the cluster and soft delete it from the database.
 