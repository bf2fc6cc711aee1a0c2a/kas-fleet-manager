# Customizing the Fleet Manager OpenAPI Specification
This document outlines what needs to be changed in order to customize the Fleet Manager OpenAPI specifications to your own requirements.

  - [Private Agent Endpoints](#agent-cluster-private-endpoints)

##  Agent Cluster Private Endpoints
The [fleet-manager-private](../openapi/fleet-manager-private.yaml) OpenAPI spec defines the endpoints used by the agent for data plane communications.

### General Customization
The following are some general things that you can update or add to this template to meet your requirements.
- This template currently has references to `Dinosaur(s)` as an example service. Please change references of this to the name of your service.
- Any endpoints that should only be used by an agent or by other internal components should be added here.

Further customizations for existing endpoints are described in the following sections below.

### (PUT) /api/dinosaurs_mgmt/v1/agent-clusters/{id}/status
This endpoint is used to update the control plane of the status of a data plane cluster.

#### Request
The **DataPlaneClusterUpdateStatusRequest** is the schema used for the request body for this endpoint. This defines multiple fields relating to the status of a cluster as well as additional information like the cluster's capacity, available service and service operator versions.

##### Customization
- `.total`: This field should define a list of data plane cluster resources and their total capacity, currently used and available, that can be consumed by an instance or cluster deployed by your service (e.g. storage, cpu or memory).

    For example:
    ```
        total:
            ...
            properties: {
                storage:
                    type: string # example value may be '100Gi'.
            }
    ```
- `.remaining`: This field should define a list of data plane cluster resources that states the available/remaining capacity that can still be consumed by an instance or cluster deployed by your service. The list of resources you define here should match the resources you defined in **.total**.

    For example:
    ```
        remaining:
            ...
            properties: {
                storage:
                    type: string # example value may be '50Gi'. This means that the dataplane cluster only has 50/100Gi storage left available that can be consumed by 'Dinosaur' clusters.
            }
    ```
- `.resizeInfo`: This field contains information needed by the control plane on how to scale the data plane cluster. The schema `DatePlaneClusterUpdateStatusRequestResizeInfo` is used to define the properties for this field.

    The property `delta` in this schema should be updated. This should define a list of data plane resources, consumed by 'Dinosaur' clusters, and the amount that they would be increased or decreased when applying the `nodeDelta`. The list of resources you define here should match the resources you defined in **.total**.

    For example:
    ```
        resizeInfo:
            nodeDelta:
                type: integer # example value may be '3'
            delta:
                properties: {
                    storage:
                        type: string # example value may be '20Gi'. This means that storage will increase by 20Gi if the data plane cluster was scaled up by 3 nodes (new total for storage will then be '120Gi'). If the data plane cluster was scaled down by 3 nodes, then the storage will decrease by 20Gi (new total will then be '80Gi').
                }
    ```

#### Response
##### Customization
The template currently has three example responses defined. The following changes can be done here:
- Addition of any other responses can be added here to suit your requirements.
- Update any references of 'Dinosaur' to the name of your service.

#### (PUT) /api/dinosaurs_mgmt/v1/agent-clusters/{id}/dinosaurs/status
This endpoint is used to update the control plane of the statuses of instances or clusters managed by the agent (i.e. 'Dinosaur' clusters).

#### Request
The **DataPlaneDinosaurStatusUpdateRequest** is the schema used for the request body for this endpoint. The fields in each 'Dinosaur' status is defined by the **DataPlaneDinosaurStatus** schema.

##### Customization
- `.capacity`: This field should define a list of data plane resources consumed by a 'Dinosaur' cluster. The list of resources you define here should be the same resources defined in **DataPlaneClusterUpdateStatusRequest.total**.

    For example:
    ```
        capacity:
            ...
            properties: {
                storage:
                    type: string # example value here could be '5Gi'. This means that this 'Dinosaur' cluster is currently consuming 5Gi of storage.
            }
    ```

#### Response
##### Customization
The template currently has three example responses defined. The following changes can be done here:
- Addition of any other responses can be added here to suit your requirements.
- Update any references of 'Dinosaur' to the name of your service.

#### (GET) /api/dinosaurs_mgmt/v1/agent-clusters/{id}/dinosaurs
This endpoint is used to get a list of existing or new custom resources for instances/clusters (i.e. 'ManagedDinosaur') that is currently or will be managed by the agent on the data plane cluster.

#### Response
##### Customization
The template currently has three example responses defined. The following changes can be done here:
- Addition of any other responses can be added here to suit your requirements.
- Update any references of 'Dinosaur' to the name of your service.

Further customizations for specific responses can be seen in the sections below.

###### 200
**ManagedDinosaurList** is the schema used to define the response body for this endpoint. The fields for each 'Managed Dinosaur' custom resource is defined by the schema **ManagedDinosaur**.

###### Customization
- `.capacity`: Properties for this field is defined by the schema **ManagedDinosaurCapacity**. This should be updated to define a list of data plane resources that can be consumed by a 'Dinosaur' cluster. The list of resources you define here should be the same resources defined in **DataPlaneClusterUpdateStatusRequest.total**.

    For example:
    ```
        capacity:
            ...
            properties: {
                storage:
                    type: string # example value here could be '10Gi'. This means that this 'Dinosaur' cluster can consume up to 10Gi of storage.
            }
    ```

#### (GET) ​/api​/dinosaurs_mgmt​/v1​/agent-clusters​/{id}
This endpoint is used to get the configuration for the data plane cluster agent which is defined in the `.spec` of the agent custom resource (i.e. 'ManagedDinosaurAgent').

##### Customizations
Any additional configuration for the data plane cluster agent can be added here to suit your requirements.



