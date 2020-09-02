OCM Managed Service API
---

This project is based on OCM microservice.

To help you while reading the code the service implements a simple collection
of _dinosaurs_, so you can inmediatelly know when something is infrastructure or
business logic. Anything that talks about dinosaurs is business logic, which you
will want to replace with your our concepts. The rest is infrastructure, and you
will probably want to preserve without change.

For a real service written using the same infrastructure see the
https://gitlab.cee.redhat.com/service/uhc-account-manager

To contact the people that created this infrastructure go to the
https://coreos.slack.com/ #service-development-b channel.

## Prerequisites
[OpenAPI Generator](https://openapi-generator.tech/docs/installation/)
[go-bindata](https://github.com/go-bindata/go-bindata#installation)
[docker](https://docs.docker.com/get-docker/) - to create database
[gotestsum](https://github.com/gotestyourself/gotestsum#install) - to run the tests

## Running the tests locally
### Setup Postgres
An instance of Postgres is required to run this service locally, the following steps will install and setup a postgres locally for you with Docker. 
```
make db/setup
```

### Run the tests
```
make test
```

### Run the integration tests
```
make test-integration
```

To stop and remove the container when finished, run:
```
make db/teardown
```

