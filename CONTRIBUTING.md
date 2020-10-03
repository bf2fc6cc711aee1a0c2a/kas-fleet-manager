# Contributing

## Install Go

### On OSX

```sh
brew install go@1.13

export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin  # our binaries compile here
```

### On Other Linux Platforms

```sh
wget https://golang.org/dl/go1.13.15.linux-amd64.tar.gz # for latest versions check https://golang.org/dl/
tar -xzf go1.13.15.linux-amd64.tar.gz
mv go /usr/local

export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin
```

## Project source

Project source is to be found under `$GOPATH/src` by a distinct directory path.

Fork managed-services-api to your own gitlab repository: https://gitlab.cee.redhat.com/service/managed-services-api/forks/new

```sh
cd $GOPATH
git clone git@gitlab.cee.redhat.com:{username}/managed-services-api.git src/gitlab.cee.redhat.com/service/managed-services-api

cd $GOPATH/src/gitlab.cee.redhat.com/service/managed-services-api
git remote add upstream git@gitlab.cee.redhat.com:service/managed-services-api.git
```

Resulting workspace:

```plain
$GOPATH
  /bin
  /pkg
  /src
    /gitlab.cee.rh.com/service/
       /uhc-cluster-service
         /cmd
         /pkg
       /managed-services-api  -- our git root
         /cmd
           /managed-services-api  -- Main CLI entrypoint
         /pkg
           /api      -- type definitions and models
           /config   -- configuration handling
					 /db  		 -- database schema and migrations
           /handlers -- web handlers/controllers
           /services -- interfaces for CRUD and business logic
					 /workers  -- background workers for async reconciliation logic

```

## Generate OpenAPI code

For more information: https://github.com/openapitools/openapi-generator

### On OSX

```sh
# swagger-codegen 3.0 does not support Go w/ OpenAPI spec 3.0.
# This tool supports Go:
brew install openapi-generator
# on other platforms, see https://github.com/openapitools/openapi-generator

# The version of openapi-generator we used last is recorded here:
cat pkg/api/openapi/.openapi-generator/VERSION

# regenerate the API implementation from our swagger file
# (rerun this on on any spec change e.g, you add a new field)
make generate
```

### On Other Linux Platforms

```sh
cat pkg/api/openapi/.openapi-generator/VERSION

# install openapi-generator with the same version ^^, e.g. 3.3.4
npm install @openapitools/openapi-generator-cli@cli-3.3.4 -g

make generate
```

## Managing dependencies

[Go Modules](https://github.com/golang/go/wiki/Modules) are used to manage dependencies.

### Add a dependency

To add a dependency, simply add a new import to any go file in the project and either build or install the application:
```
import (
  "github.com/something/new"
)

...

$ make install
```

The `go.mod` file we automatically be updated with the new required project, the `go.sum` file will be generated.

## Modifying the API definition

All OpenAPI spec modifications must be done through [Apicurio Studo](https://studio.apicur.io/apis/35337) first and manually copied into the repo.

## Testing

### Mocking

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

### Unit Tests

We follow the Go [table driven test](https://github.com/golang/go/wiki/TableDrivenTests) approach.

This means that test cases for a function are defined as a set of structs and executed in a for
loop one by one. For an example of writing a table driven test, see:

- [The table driven test documentation](https://github.com/golang/go/wiki/TableDrivenTests) which has examples
- [The Test_kafkaService_Get test in this repository](pkg/services/kafka_test.go)


### Integration Tests

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