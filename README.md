OCM Managed Service API
---

This project is based on OCM microservice.

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
```
make test-integration
```

To stop and remove the database container when finished, run:
```
make db/teardown
```

## Merge request checks 
Upon opening or updating a merge request, a pr check job will be triggered in jenkins. 

This job runs the `pr_check.sh` script, which starts a docker container with a postgres database and executes various make targets. For now, this includes only the `make verify` target. The tests that are run inside the container are defined in the `pr_check_docker.sh` script. 

The container can be run locally by executing the `pr_check.sh` script, or simply running `make test/run`.

## Staging deployments 
Upon making changes to the master branch, a build job will be triggered in jenkins. 

This job will run the `build_deploy.sh` script:
- Two environment variables are expected to be defined: `QUAY_USER` and `QUAY_TOKEN`. These define how to reach the quay repository where the resulting image should be pushed. These should be defined already in Jenkins.
- The `VERSION` of the build is defined as the first 7 digits of the commit hash. This is used as the `image_tag` supplied to the deployment template in later steps
- The image is built and pushed to the Quay repo using `make version=VERSION image/push`.
