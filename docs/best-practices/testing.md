# Testing best practices

## Assertions
In Kas Fleet Manager, we make extensive usage of the [gomega](https://onsi.github.io/gomega/) framework.
Instead of writing custom `if then fail` code, use the provided `Expect` methods

### Examples
#### Checking for equality
Instead of this code:
```go
if reflect.DeepEqual(x, y) {
	t.Errorf(....)
} 
```
the following should be used:
```go
Expect(x).To(Equal(y), "optional error description")
```

#### Checking array size
Instead of this code:
```go
if len(array) != xx {
	t.Errorg("error message")
}
```
the following should be used:
```go
Expect(array).To(HaveLen(xx), "error message")
```
## Polling
If you need to poll to wait for some event to happen, avoid using directly the `wait.PollImmediate` function: Kas Fleet Manager provides a rich set of utilities that will handle for you all the caveats of polling when dealing with it.

Such utilities are:
* **The poller builder**. A builder interface that will guide you in configuring a custom poller
* **Cluster pollers**. A number of preconfigured pollers to be used to poll for a specific cluster event:
    * **WaitForClustersMatchCriteriaToBeGivenCount**. Awaits for the number of clusters with an assigned cluster id to be exactly `count`
    * **WaitForClusterIDToBeAssigned**. Awaits for clusterID to be assigned to the designed cluster
    * **WaitForClusterToBeDeleted**. Awaits for the specified cluster to be deleted
    * **WaitForClusterStatus**. Awaits for the cluster to reach the desired status
* **Kafka pollers**. A number of preconfigured pollers that will allow to easily poll for a specific kafka event:
    * **WaitForNumberOfKafkaToBeGivenCount**. Awaits for the number of kafkas to be exactly X
    * **WaitForKafkaCreateToBeAccepted**. Creates a kafka and awaits for the request to be accepted
    * **WaitForKafkaToReachStatus**. Awaits for a kafka to reach a specified status
    * **WaitForKafkaToBeDeleted**. Awaits for a kafka to be deleted

For more details, see the [poller documentation](../utilities/testing.md)
