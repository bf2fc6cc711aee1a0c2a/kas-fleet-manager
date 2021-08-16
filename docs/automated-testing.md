# Testing kas-fleet-manager

## Mocking

We use the [moq](https://github.com/matryer/moq) tool to automate the generation of interface mocks.

In order to generate a mock on a new interface, simply place the following line above your interface
definition:

```go
//go:generate moq -out output_file.go . InterfaceName
```

For example:

```go
//go:generate moq -out output_file.go . InterfaceName
// IDGenerator interface for string ID generators.
type IDGenerator interface {
	// Generate create a new string ID.
	Generate() string
}
```

A new `InterfaceNameMock` struct will be generated and can be used in tests.

***NOTE: Unless a mock is required outside of it's current package, add a "_test.go" suffix to the file name to keep it isolated***

For more information on using `moq`, see:

- [The moq repository](https://github.com/matryer/moq)
- The IDGenerator [interface](../pkg/client/ocm/id.go) and [mock](../pkg/client/ocm/idgenerator_moq_test.go) in this repository 

For mocking the OCM client, [this wrapper interface](../pkg/client/ocm/client.go). If using the OCM SDK in
any component of the system, please ensure the OCM SDK logic is placed inside this mock-able
wrapper.

## Unit Tests

We follow the Go [table driven test](https://github.com/golang/go/wiki/TableDrivenTests) approach.

This means that test cases for a function are defined as a set of structs and executed in a for
loop one by one. For an example of writing a table driven test, see:

- [The table driven test documentation](https://github.com/golang/go/wiki/TableDrivenTests) which has examples
- [The Test_kafkaService_Get test in this repository](../internal/kafka/internal/services/kafka_test.go)


## Integration Tests

Before every integration test, `RegisterIntegration()` must be invoked. This will ensure that the
API server and background workers are running and that the database has been reset to a clean
state. `RegisterIntegration()` also returns a teardown function that should be invoked at the end
of the test. This will stop the API server and background workers. For example:

```go
helper, httpClient, teardown := test.RegisterIntegration(t, mockOCMServer)
defer teardown()
```

See [TestKafkaPost integration test](../internal/kafka/test/integration/kafkas_test.go) as an example of this.

Integration tests in this service can take advantage of running in an "emulated OCM API". This
essentially means a configurable mock OCM API server can be used in place of a "real" OCM API for
testing how the system responds to different failure scenarios in OCM.

The emulated OCM API will be used if the `OCM_ENV` environment variable is set to `integration`.

The emulated OCM API can also be set manually in non-`integration` environments by setting the
`ocm-mock-mode` flag to `emulate-server`.

When handling OCM error scenarios in integration tests, decide which ServiceError you'd like an
endpoint to return, for a full list of ServiceError types, see [this file](../pkg/errors/errors.go).
Ensure when mocking an endpoint the correct ServiceError type is returned via the mock function e.g.

```go
ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
ocmServerBuilder.SetClustersPostResponse(nil, errors.Validation("test failed validation"))
ocmServer := ocmServerBuilder.Build()
defer ocmServer.Close()
```

In the integration test itself you can then test the expected HTTP response/error from this
service based on the OCM failure e.g.

```go
_, resp, err = client.DefaultApi.CreateKafka(ctx, true, k)
Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
```

When adding new integration tests, use the following files as guidelines:
- [TestKafkaPost integration test](../internal/kafka/test/integration/kafkas_test.go)
- [Mock OCM API server](../test/mocks/api_server.go)

When adding new features to the service that interact with OCM endpoints that have not currently
been emulated, you must add them. If you do not add them you begin seeing errors in integration
tests around `text/plain` responses being provided by OCM.

Adding a new endpoint involves modifying [the mock OCM API server](../test/mocks/api_server.go) and
adding the new endpoint. Using the OCM `GET /api/clusters_mgmt/v1/clusters` endpoint as an example:

- Add a new path variable, `EndpointPathClusters`
- Add a new endpoint variable, including the HTTP method, `EndpointClusterGet`
- Add a new default return value, `MockCluster`, and initialise it in `init()`
- If the data type the endpoint returns (e.g. `*clustersmgmtv1.Cluster`) is new, ensure the mock
  API server knows how to marshal it to JSON in `marshalOCMType()`. If the new type is a `*List`
  type (e.g. `*clustersmgmtv1.IngressList`) then follow the pattern used by `IngressList` where
  `*clustersmgmtv1.IngressList`, `[]*clustersmgmtv1.Ingress`, and `*clustersmgmtv1.Ingress` are all
  handled.
- Register the default return handler in `getDefaultHandlerRegister()`
- Allow overrides to be provided by adding an override function, `SetClusterGetResponse()`

### Polling
When you need to poll to wait for an event to happen (for example, when you have to wait for a cluster to be ready), a
poller object is provided [here](../internal/kafka/test/common/util.go).
The simplest way to use it would be:
```go
package mytest
import (
    "testing"
    "time"

    utils "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
)

func MyTestFunction(t *testing.T) {
    err := utils.NewPollerBuilder().
        // test output should go to `t.Logf` instead of `fmt.Printf`
        OutputFunction(t.Logf). 
        // sets number of retries and interval between each retry
        IntervalAndRetries(10 * time.Second, 10).
        OnRetry(func(attempt int, maxAttempts int) (done bool, err error) { // function executed on each retry
            // put your logic here 
            return true, nil
        }).
        Build().Poll()
    if err != nil { 
        // ...
    }
    // ...
}
```
> :warning: By default the logging is set to `short-verbose`. To be able to see the polling logs you must set the
> `TEST_SUMMARY_FORMAT` environment variable to `default-verbose`

### Mock KAS Fleetshard Sync
The mock [KAS Fleetshard Sync](../internal/kafka/test/mocks/kasfleetshardsync/kas-fleetshard-sync.go) is used to reconcile data plane and Kafka cluster status during integration tests. This also needs to be used even when running integration tests against a real OCM environment as the KAS Fleetshard Sync in the data plane cluster cannot
communicate to the KAS Fleet Manager during integration test runs. 

The mock KAS Fleetshard Sync needs to be started at the start of a test that requires updates to a data plane or Kafka cluster status. 

**Example Usage:**
```go
package mytest
import (
    "testing"
    "time"

    "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
    "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks"
    "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"
)

func MyTestFunction(t *testing.T) {
    // ...
    ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	  defer ocmServer.Close()

    h, _, teardown := test.RegisterIntegration(t, ocmServer)
  	defer teardown()

    mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
    mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
    mockKasfFleetshardSync.Start()
    defer mockKasfFleetshardSync.Stop()
    // ...
}
```

The default behaviour of how the status of a data plane and a Kafka cluster are updated are as follows:
- **Data plane cluster update**: Retrieves all clusters in the database in a `waiting_for_kas_fleetshard_operator` state and updates it to `ready` once all of the addons are installed.
- **Kafka cluster update**: Retrieves all Kafkas that can be processed by the KAS Fleetshard Operator (i.e. Kafkas not in an `accepted` or `preparing` state). Any Kafkas marked for deletion are updated to `deleting`. Kafkas with any other status are updated to `ready`.

These default behaviours can be changed by setting the update status implementation. For example, the implementation for updating a data plane cluster status can be changed by calling the `SetUpdateDataplaneClusterStatusFunc` of the Mock KAS Fleetshard Sync builder:

```go
//...
import(
    //...

    privateopenapi "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
)

func MyTestFunction(t *testing.T) {
    // ...
    mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
    mockKasFleetshardSyncBuilder.SetUpdateDataplaneClusterStatusFunc(func(helper *test.Helper, privateClient *privateopenapi.APIClient, ocmClient ocm.Client) error {
      // Custom data plane cluster status update implementation
    })
    mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
    mockKasfFleetshardSync.Start()
    defer mockKasfFleetshardSync.Stop()
    // ...
}
```

### Utilizing API clients in the integration tests
When testing specific API endpoints (e.g. public, private or admin-private API), it is required to create an appropriate client. To do so, use a client creation function from the [helper](../internal/kafka/test/helper.go):

```go
// create private admin API client
ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
mockedGetClusterResponse, err := mockedClusterWithMetricsInfo(mocks.MockClusterComputeNodes)
if err != nil {
    t.Fatalf(err.Error())
}
ocmServerBuilder.SetClusterGetResponse(mockedGetClusterResponse, nil)

ocmServer := ocmServerBuilder.Build()
defer ocmServer.Close()

h, _, tearDown := test.NewKafkaHelper(t, ocmServer)
defer tearDown()
ctx := tt.args.ctx(h)
client := test.NewAdminPrivateAPIClient(h)
```

### Testing privileged permissions
Some endpoints will act differently depending on privileges of the entity calling them, e.g. `org_admin` can have CRUD access to resources within an organisation not owned by them, while regular users can usually only access their own resources. To find out more about various claims used by the kas-fleet-manager endpoints, go to [this document](../docs/jwt-claims.md)
