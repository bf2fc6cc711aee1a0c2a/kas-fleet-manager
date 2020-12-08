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

**IMPORTANT**: The [devtools-bot](https://gitlab.cee.redhat.com/devtools-bot) user needs to be added as a `Maintainer` to your fork in order for your merge requests (MRs) to pass CI checks.

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
           /api      -- type definitions and models (Note. openapi folder is generated - see below)
           /config   -- configuration handling
					 /db  		 -- database schema and migrations
           /handlers -- web handlers/controllers
           /services -- interfaces for CRUD and business logic
					 /workers  -- background workers for async reconciliation logic

```

## Set up Git Hooks
Run the following command to set up git hooks for the project. 

```
make setup/git/hooks
```

The following git hooks are currently available:
- **pre-commit**:
  - This runs checks to ensure that the staged `.go` files passes formatting and standard checks using gofmt and go vet.

## Debugging
### VS Code
Set the following configuration in your **Launch.json** file.
```
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Managed Service API",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/managed-services-api/main.go",
            "env": {
                "OCM_ENV": "development"
            },
            "args": ["serve"]
        }
    ]
}
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
VERSION=$(cat pkg/api/openapi/.openapi-generator/VERSION)

# install openapi-generator with the same version
npm install @openapitools/openapi-generator-cli@cli-$VERSION -g

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

All OpenAPI spec modifications must be done through [Apicurio Studio](https://studio.apicur.io/apis/35337) first and manually copied into the repo.

Once you've made your changes, the second step is to validate it:

```sh
make openapi/validate
```

Once the schema is valid, the remaining step is to generate the openapi modules by using the command:

```sh
make openapi/generate
```

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

For mocking the OCM client, [this wrapper interface](pkg/ocm/client.go). If using the OCM SDK in
any component of the system, please ensure the OCM SDK logic is placed inside this mock-able
wrapper.

### Unit Tests

We follow the Go [table driven test](https://github.com/golang/go/wiki/TableDrivenTests) approach.

This means that test cases for a function are defined as a set of structs and executed in a for
loop one by one. For an example of writing a table driven test, see:

- [The table driven test documentation](https://github.com/golang/go/wiki/TableDrivenTests) which has examples
- [The Test_kafkaService_Get test in this repository](pkg/services/kafka_test.go)


### Integration Tests

Before every integration test, `RegisterIntegration()` must be invoked. This will ensure that the
API server and background workers are running and that the database has been reset to a clean
state. `RegisterIntegration()` also returns a teardown function that should be invoked at the end
of the test. This will stop the API server and background workers. For example:

```
helper, httpClient, teardown := test.RegisterIntegration(t, mockOCMServer)
defer teardown()
```

See [TestKafkaPost integration test](test/integration/kafkas_test.go) as an example of this.

Integration tests in this service can take advantage of running in an "emulated OCM API". This
essentially means a configurable mock OCM API server can be used in place of a "real" OCM API for
testing how the system responds to different failure scenarios in OCM.

The emulated OCM API will be used if the `OCM_ENV` environment variable is set to `integration`.

The emulated OCM API can also be set manually in non-`integration` environments by setting the
`ocm-mock-mode` flag to `emulate-server`.

When handling OCM error scenarios in integration tests, decide which ServiceError you'd like an
endpoint to return, for a full list of ServiceError types, see [this file](pkg/errors/errors.go).
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
- [TestKafkaPost integration test](test/integration/kafkas_test.go)
- [Mock OCM API server](test/mocks/api_server.go)

When adding new features to the service that interact with OCM endpoints that have not currently
been emulated, you must add them. If you do not add them you begin seeing errors in integration
tests around `text/plain` responses being provided by OCM.

Adding a new endpoint involves modifying [the mock OCM API server](test/mocks/api_server.go) and
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

## Logging Standards & Best Practices
  * Log only actionable information, which will be read by a human or a machine for auditing or debugging purposes
    * Logs shall have context and meaning - a single log statement should be useful on its own
    * Logs shall be easily aggregatable
    * Logs shall never contain sensitive information
  * All logs should be logged through our logging interface, `UHCLogger` in `/pkg/logger/logger.go`
    * *Logging interface shall be updated to gracefully handle logs outside of a user context*
  * If a similar log message will be used in more than one place, consider adding a new standardized interface to `UHCLogger`
    * *Logging interface shall be updated to define a new `Log` struct to support standardization of more domain specific log messages*

### Levels
#### Info
Log to this level any non-error based information that might be useful to someone browsing logs for a specific reason. This may or may not include request / response logging, debug information, script output, etc.

#### Warn
Log to this level any error based information that might want to be brought to someone's attention to take action on, but does not seriously impede or affect use of the application (ie. it is recoverable). This may or may not include deprecation notices, retry operations, etc.

#### Error
Log to this level any error that is fatal to the given transaction and affects expected user operation. This may or may not include failed connections, missing expected data, or other unrecoverable outcomes.

#### Fatal
Log to this level any error that is fatal to the service and requires the service to be immediately shutdown in order to prevent data loss or other unrecoverable states. This should be limited to scripts and fail-fast scenarios in service startup *only* and *never* because of a user operation in an otherwise healthy servce.

### Verbosity
Verbosity effects the way in which `Info` logs are written. The best way to see how verbosity works is here: https://play.golang.org/p/iXJiX289VzO

On a scale from 1 -> 10, logging items at `V(10)` would be considered something akin to `TRACE` level logging, whereas `V(1)` would be information you might want to log all of the time.

As a rule of thumb, we use verbosity settings in the following ways. Consider we have:

```
glog.V(1).Info("foo")
glog.V(5).Info("bar")
glog.V(10).Info("biz")
```
* `--v=1`
  * This is production level logging. No unecessary spam and no sensitive information.
  * This means that given the verbosity setting and the above code, we would see `foo` logged.
* `--v=5`
  * This is stage / test level logging. Useful debugging information, but not spammy. No sensitive information.
  * This means that given the verbosity setting and the above code, we would see `foo` and `bar` logged.
* `--v=10`
  * This is local / debug level logging. Useful information for tracing through transactions on a local machine during development.
  * This means that given the verbosity setting and the above code, we would see `foo`, `bar`, and `biz` logged.

### Sentry Logging
Sentry monitors errors/exceptions in a real-time environment. It provides detailed information about captured errors. See [sentry](https://sentry.io/welcome/) for more details.
 
Logging can be enabled by importing the sentry-go package: "github.com/getsentry/sentry-go

Following are possible ways of logging events via Sentry:

```
sentry.CaptureMessage(message) // for logging message
sentry.CaptureEvent(event) // capture the events 
sentry.CaptureException(error) // capture the exception
``` 
Example : 
```
func check(err error, msg string) {
	if err != nil && err != http.ErrServerClosed {
		glog.Errorf("%s: %s", msg, err)
		sentry.CaptureException(err)
	}
}
```
## Definition of Done
* All acceptance criteria specified in JIRA are met
  * Acceptance criteria to include:
    * Required feature functionality
    * Required tests - unit, integration, manual testcases (if relevant)
    * Required documentation
* CI and all relevant tests are passing
* Changes have been verified by one additional reviewer against: 
  * each required environment 
  * each supported upgrade path
* MR has been merged

## Running linter

`golangci-lint` is used to run a static code analysis. This is enabled on per MR check and it will fail, if any new changes don't conform to the rules specified by the linter.

To manually run the check, execute this command from the root of this repository

```
make lint
```
