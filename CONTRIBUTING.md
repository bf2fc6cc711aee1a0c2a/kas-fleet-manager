# Contributing

## Modifying the API definition

All OpenAPI spec modifications must be done through [Apicurio Studo](https://studio.apicur.io/apis/35337) first and manually copied into the repo.

## Mocking

We use the [moq](https://github.com/matryer/moq) tool to automate the generation of interface mocks.

In order to generate a mock on a new interface, simply place the following line above your interface
definition:

```
//go:generate moq -out output_file.go . InterfaceName
```

For example:

```
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
- The IDGenerator [interface](pkg/ocm/id.go) and [mock](pkg/ocm/idgenerator_moq_test.go) in this repository 

## Unit Tests

We follow the Go [table driven test](https://github.com/golang/go/wiki/TableDrivenTests) approach.

This means that test cases for a function are defined as a set of structs and executed in a for
loop one by one. For an example of writing a table driven test, see:

- [The table driven test documentation](https://github.com/golang/go/wiki/TableDrivenTests) which has examples
- [The Test_kafkaService_Get test in this repository](pkg/services/kafka_test.go)


## Integration Tests

Integration tests in this service can take advantage of running in an "emulated OCM API". This
essentially means a configurable mock OCM API server can be used in place of a "real" OCM API for
testing how the system responds to different failure scenarios in OCM.

The emulated OCM API will be used if the `OCM_ENV` environment variable is set to `integration`.

The emulated OCM API can also be set manually in non-`integration` environments by setting the
`ocm-mock-mode` flag to `emulate-server`.

When handling OCM error scenarios in integration tests, decide which ServiceError you'd like an
endpoint to return, for a full list of ServiceError types, see [this file](./pkg/errors/errors.go).
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
_, resp, err := client.DefaultApi.ApiManagedServicesApiV1KafkasPost(ctx, k)
Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
```

When adding new integration tests, use the following files as guidelines:
- [TestKafkaPost integration test](./test/integration/kafkas_test.go)
- [Mock OCM API server](./test/mocks/api_server.go)