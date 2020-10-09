OCM Managed Service API
---

A service for provisioning and managing fleets of Kafka instances.

For more information on how the service works, see [the implementation documentation](docs/implementation.md).

## Prerequisites
* [OpenAPI Generator](https://openapi-generator.tech/docs/installation/)
* [Golang](https://golang.org/dl/)
* [Docker](https://docs.docker.com/get-docker/) - to create database
* [gotestsum](https://github.com/gotestyourself/gotestsum#install) - to run the tests
* [go-bindata (3.1.2+)](https://github.com/go-bindata/go-bindata) - for code generation
* [ocm cli](https://github.com/openshift-online/ocm-cli/releases) - ocm command line tool

## Running the service locally
An instance of Postgres is required to run this service locally, the following steps will install and setup a postgres locally for you with Docker. 
```
make db/setup
```

To log in to the database: 
```
make db/login
```

Set up the AWS credential files (only needed if creating new OSD clusters):
```
make aws/setup AWS_ACCOUNT_ID=<account_id> AWS_ACCESS_KEY=<aws-access-key> AWS_SECRET_ACCESS_KEY=<aws-secret-key>
```

Set up the `ocm-service.token` file in the `secrets` directory to point to your temporary ocm token.
ocm-offline-token can be retrieved from https://qaprodauth.cloud.redhat.com/openshift/token
```
make ocm/setup OCM_OFFLINE_TOKEN=<ocm-offline-token>
```

To run the service: 
```
make run 
```

## Environments

The service can be run in a number of different environments. Environments are essentially bespoke
sets of configuration that the service uses to make it function differently. Environments can be
set using the `OCM_ENV` environment variable. Below are the list of known environments and their
details.

- `development` - The `staging` OCM environment is used. Sentry is disabled. Debugging utilities
   are enabled. This should be used in local development.
- `testing` - The OCM API is mocked/stubbed out, meaning network calls to OCM will fail. The auth
   service is mocked. This should be used for unit testing.
- `integration` - Identical to `testing` but using an emulated OCM API server to respond to OCM API
   calls, instead of a basic mock. This can be used for integration testing to mock OCM behaviour.
- `production` - Debugging utilities are disabled, Sentry is enabled. environment can be ignored in
   most development and is only used when the service is deployed.

## Contributing

See the [contributing guide](CONTRIBUTING.md) for general guidelines.

### Setup Git Hooks
See the [setup git hooks](CONTRIBUTING.md#set-up-git-hooks) section in the contributing guide for more information.

### Additional CLI commands

In addition to the REST API exposed via `make run`, there are additional commands to interact directly
with cluster creation logic etc without having to run the server.

To use these commands, run `make binary` to create the `./managed-services-api` CLI.

Run `./managed-services-api -h` for information on the additional commands.

### Run the tests
```
make test
```

### Run the integration tests

Integration tests can be executed against a real or "emulated" OCM environment. Executing against
an emulated environment can be useful to get fast feedback as OpenShift clusters will not actually
be provisioned, reducing testing time greatly.

Both scenarios require a database and OCM token to be setup before running integration tests, run:

```
make db/setup
make ocm/setup OCM_OFFLINE_TOKEN=<ocm-offline-token>
```

To run integration tests with an "emulated" OCM environment, run:

```
OCM_ENV=integration make test-integration
```

To run integration tests with a real OCM environment, run:

```
make test-integration
```

To stop and remove the database container when finished, run:
```
make db/teardown
```

### Additional Checks

In addition to the unit and integration tests, we should ensure that our code passes standard go checks.

To verify that the code passes standard go checks, run:
```
make verify
```

To verify that the code passes lint checks, run:
```
make lint
```
**NOTE**: This uses golangci-lint which needs to be installed in your GOPATH/bin