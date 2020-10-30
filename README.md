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

## User Account setup

- Setup your account in stage OCM: 
[Example MR](https://gitlab.cee.redhat.com/service/ocm-resources/-/blob/master/data/uhc-stage/users/akeating_kafka_service.yaml)
- Once the MR is merged, retrieve your ocm-offline-token from https://qaprodauth.cloud.redhat.com/openshift/token

## Compile from master branch
```
# Change current directory to your source code folder (ie: cd <any_your_source_code_folder>)
$ git clone https://gitlab.cee.redhat.com/service/managed-services-api.git
$ cd managed-services-api
$ git checkout master
$ make binary
$ ./managed-services-api -h
```
## Running the Service locally

1. Set one of the OCM Env (See `Environments` section for list of environments)
```
OCM_ENV=integration
```
2. Clean up and Creating a database 

```
# If you have db already created execute
$ make db/teardown
# Create database tables
$ make db/setup
$ ./managed-services-api migrate
# Verify tables and records are created 
# Login to the database
$ make db/login
# List all the tables
serviceapitests# \dt
                   List of relations
 Schema |      Name      | Type  |        Owner         
--------+----------------+-------+----------------------
 public | clusters       | table | managed_services_api
 public | kafka_requests | table | managed_services_api
 public | migrations     | table | managed_services_api
```

3.  (Only for development) Setup AWS credentials 
Optional: Only required if you wish to create OSD clusters via the managed service api)
```
$ make aws/setup AWS_ACCOUNT_ID=<account_id> AWS_ACCESS_KEY=<aws-access-key> AWS_SECRET_ACCESS_KEY=<aws-secret-key>
```

4. (Only for development) Generate a temporary ocm token 
Generate a temporary ocm token and set it in the secrets/ocm-service.token file
> Note: This will need to be re-generated as this temporary token will expire within a few minutes.
```
$ make ocm/setup OCM_OFFLINE_TOKEN=<ocm-offline-token> OCM_ENV=development
```
5. Running the service locally
```
$ ./managed-services-api serve  (default: http://localhost:8000)
```

## Running the Service on an OpenShift cluster
### Build and Push the Image to the OpenShift Image Registry
Login to the OpenShift internal image registry

**NOTE**: Ensure that the user used has the correct permissions to push to the OpenShift image registry. For more information, see the [accessing the registry](https://docs.openshift.com/container-platform/4.5/registry/accessing-the-registry.html#prerequisites) guide.
```
# Login to the OpenShift cluster
$ oc login <api-url> -u <username> -p <password>

# Login to the OpenShift image registry
$ make docker/login/internal
```

Build and push the image
```
# Create a namespace where the image will be pushed to.
$ make deploy/project

# Build and push the image to the OpenShift image registry. 
$ GOARCH=amd64 GOOS=linux CGO_ENABLED=0 make image/build/push/internal
```

**Optional parameters**:
- `NAMESPACE`: The namespace where the image will be pushed to. Defaults to 'managed-services-$USER.'
- `IMAGE_TAG`: Tag for the image. Defaults to a timestamp captured when the command is run (i.e. 1603447837).

### Deploy the Service using Templates
This will deploy a postgres database and the managed-services-api to a namespace in an OpenShift cluster.

```
# Deploy the service
make deploy OCM_SERVICE_TOKEN=<offline-token> IMAGE_TAG=<image-tag>
```
**Optional parameters**:
- `NAMESPACE`: The namespace where the service will be deployed to. Defaults to managed-services-$USER.
- `IMAGE_REGISTRY`: Registry used by the image. Defaults to the OpenShift internal registry.
- `IMAGE_REPOSITORY`: Image repository. Defaults to '\<namespace\>/managed-services-api'.
- `IMAGE_TAG`: Tag for the image. Defaults to a timestamp captured when the command is run (i.e. 1603447837).
- `OCM_SERVICE_CLIENT_ID`: Client id used to interact with other UHC services.
- `OCM_SERVICE_CLIENT_SECRET`: Client secret used to interact with other UHC services.
- `OCM_SERVICE_TOKEN`: Offline token used to interact with other UHC services. If client id and secret is not defined, this parameter must be specified. See [user account setup](#user-account-setup) section on how to get this offline token.
- `AWS_ACCESS_KEY`: AWS access key. This is only required if you wish to create CCS clusters using the service.
- `AWS_ACCOUNT_ID`: AWS account ID. This is only required if you wish to create CCS clusters using the service.
- `AWS_SECRET_ACCESS_KEY`: AWS secret access key. This is only required if you wish to create CCS clusters using the service.
- `ENABLE_OCM_MOCK`: Enables mock ocm client. Defaults to false.
- `OCM_MOCK_MODE`: The type of mock to use when ocm mock is enabled. Defaults to 'emulate-server'.
- `JWKS_URL`: JWK Token Certificate URL. Defaults to https://api.openshift.com/.well-known/jwks.json.

The service can be accessed by via the host of the route created by the service deployment.
```
oc get route managed-services-api
```

### Removing the Service Deployment from the OpenShift
```
# Removes all resources created on service deployment
$ make undeploy
```

**Optional parameters**:
- `NAMESPACE`: The namespace where the service deployment will be removed from. Defaults to managed-services-$USER.

## Using the Service
#### Creating an OSD Cluster
```
# Create a new cluster (OSD)
$ ./managed-services-api cluster create

# Verify cluster record is created 
# Login to the database
$ make db/login
# Ensure the cluster exists in clusters table and monitor the status. It should change to 'ready' after provisioned.
serviceapitests# select * from clusters;

# Alternatively, verify from ocm-cli
$ ocm login --url=https://api.stage.openshift.com/ --token=<OCM_OFFLINE_TOKEN>
# verify the cluster is in OCM
$ ocm cluster list

# Retrieve the OSD cluster login credentials
$ ocm get /api/clusters_mgmt/v1/clusters/<cluster_id>/credentials | jq '.admin'

# Login to the OSD cluster with the credentials you retrieved above
# Verify the OSD cluster was created successfully and have strimzi-operator installed in namespace 'redhat-managed-kafka-operator'
```
#### Creating a Kafka Cluster
```
# Submit a new Kafka cluster creation request
$ curl -v -XPOST -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/managed-services-api/v1/kafkas?async=true -d '{ "region": "eu-west-1", "cloud_provider": "aws",  "name": "serviceapi"}'

# Login to the database
$ make db/login
# Ensure the bootstrap_url column exists and is populated
serviceapitests# select * from kafka_requests;

# Login to the OSD cluster with the credentials you retrieved earlier
# Verify the Kafka cluster was created successfully in the generated namespace with 4 routes (bootstrap + 3 broker routes) which is the same as the bootstrap server host

# List a kafka request
$ curl -v -XGET -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/managed-services-api/v1/kafkas/<kafka_request_id> | jq

# List all kafka request
$ curl -v -XGET -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/managed-services-api/v1/kafkas | jq

# Delete a kafka request
$ curl -v -X DELETE -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/managed-services-api/v1/kafkas/<kafka_request_id>
```

#### Clean up
```
# Delete the OSD Cluster from the OCM Console manually
# Purge the database
$ make db/teardown
```
## Running Swagger UI
```
# Start Swagger UI container
$ make run/docs

# Launch Swagger UI and Verify from a browser: http://localhost

# Remove Swagger UI conainer
$ make run/docs/teardown
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
make ocm/setup OCM_OFFLINE_TOKEN=<ocm-offline-token> OCM_ENV=development
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
