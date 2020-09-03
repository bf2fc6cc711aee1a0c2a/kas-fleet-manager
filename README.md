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

### Run locally

```
make run
```

To stop and remove the container when finished, run:
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
