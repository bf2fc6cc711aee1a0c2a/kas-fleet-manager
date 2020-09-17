# Contributing

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