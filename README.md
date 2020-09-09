OCM Managed Service API
---

This project is based on OCM microservice.

## Prerequisites
* [OpenAPI Generator](https://openapi-generator.tech/docs/installation/)
* [Golang](https://golang.org/dl/)
* [Docker](https://docs.docker.com/get-docker/) - to create database
* [gotestsum](https://github.com/gotestyourself/gotestsum#install) - to run the tests
* [ocm cli](https://github.com/openshift-online/ocm-cli/releases) - ocm command line tool

## Running the service locally
An instance of Postgres is required to run this service locally, the following steps will install and setup a postgres locally for you with Docker. 
```
make db/setup
```

To log in to the database: 
```
docker exec -it managed-services-api-db psql -d serviceapitests -U ocm_managed_service_api -W
password: foobar-bizz-buzz
```

Set up the AWS credential files (only needed if creating new OSD clusters):
```
make aws/setup AWS_ACCOUNT_ID=<account_id> AWS_ACCESS_KEY=<aws-access-key> AWS_SECRET_ACCESS_KEY=<aws-secret-key>
```

Modify the `ocm-service.token` file in the `secrets` directory to point to your temporary ocm token. 
```sh
# Log in to ocm
# osd_offline_token can be retrieved from https://qaprodauth.cloud.redhat.com/openshift/token
ocm login --url="http://localhost:8000" --token="<osd_offline_token>"

# Generate a new temporary token
# and update ocm-service.token file
echo -n $(ocm token) > secrets/ocm-service.token

```

To run the service: 
```
make run 
```

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
