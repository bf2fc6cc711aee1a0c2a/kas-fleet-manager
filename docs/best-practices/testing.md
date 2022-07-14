# Table of Contents

<!-- toc -->

- [Testing best practices](#testing-best-practices)
  * [Assertions](#assertions)
    + [Examples](#examples)
      - [Checking for equality](#checking-for-equality)
      - [Checking array size](#checking-array-size)
  * [Polling](#polling)

<!-- tocstop -->

# Testing best practices

## Importing the package

Import the package using
```go
import (
  "github.com/onsi/gomega"
)
```

Don't import the gomega's package symbols in the file you are working on.
This is, don't do the following:
```go
import (
  . "github.com/onsi/gomega"
)
```

The reason for that is that although it might be convenient (less typing needed
for some of gomega's methods) it could also introduce issues like using
gomega's package level `Expect` method which is deprecated. In this case we favour
safety over convenience.

## Using Expectations

To use gomega's Expect in your tests, get a gomega's `WithT` instance by calling
the NewWithT(t) method and saving the returned result in a variable. Then call
the Expect method by using the obtained variable:
```go
import (
  "github.com/onsi/gomega"
)
...

g := NewWithT(t)
...
g.Expect(x).To(gomega.Equal(y))
```

If the tests you are writing are table-driven tests with subtests make sure
that you get a new instance of `WithT` for every subtest. For example:
```go
for _, test := range tests {
  t.Run(test.name, func(t *testing.T) {
    g := NewWithT(t)
    ...
    g.Expect(gotErr).To(gomega.Equal(test.wantErr))
    g.Expect(res).To(gomega.Equal(test.want))
  })
```

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
g := gomega.NewWithT(t)
...
g.Expect(x).To(gomega.Equal(y), "optional error description")
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
g := gomega.NewWithT(t)
...
g.Expect(array).To(gomega.HaveLen(xx), "error message")
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
