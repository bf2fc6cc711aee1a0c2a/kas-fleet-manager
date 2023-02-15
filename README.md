Kafka Service Fleet Manager
---

![build status badge](https://github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/actions/workflows/ci.yaml/badge.svg)
![codecov](https://codecov.io/gh/bf2fc6cc711aee1a0c2a/kas-fleet-manager/branch/main/graph/badge.svg)

A service for provisioning and managing fleets of Kafka instances.

For more information on how the service
works, see the [implementation documentation](docs/implementation.md).

## Prerequisites

* [Golang 1.19+](https://golang.org/dl/)
* [Docker](https://docs.docker.com/get-docker/) - to create database
* [ocm cli](https://github.com/openshift-online/ocm-cli/releases) - ocm command line tool
* [Node.js v14.17+](https://nodejs.org/en/download/) and [npm](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm)

There are some additional prerequisites required for running kas-fleet-manager
due to its interaction with external services which are described below.

## Running Fleet Manager in your local environment

1. Follow the [populating configuration guide](docs/populating-configuration.md)
   to prepare Fleet Manager with its needed configurations

1. Compile the Fleet Manager binary
   ```
   make binary
   ```
1. Create and setup the Fleet Manager database
    * Create and setup the database container and the initial database schemas
    ```
    make db/setup && make db/migrate
    ```
    * Optional - Verify tables and records are created
    ```
    # Login to the database to get a SQL prompt
    make db/login
    ```
    ```
    # List all the tables
    serviceapitests# \dt
    ```
    ```
    # Verify that the `migrations` table contains multiple records
    serviceapitests# select * from migrations;
    ```

1. Start the Fleet Manager service in your local environment
    ```
    ./kas-fleet-manager serve
    ```

    This will start the Fleet Manager server and it will expose its API on
    port 8000 by default

    >**NOTE**: The service has numerous feature flags which can be used to enable/disable certain features
    of the service. Please see the [feature flag](./docs/feature-flags.md) documentation for more information.
1. Verify the local service is working
    ```
    curl -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/kafkas_mgmt/v1/kafkas
   {"kind":"KafkaRequestList","page":1,"size":0,"total":0,"items":[]}
    ```

   >NOTE: Make sure you are logged in to OCM through the CLI before running
          this command. Details on that can be found [here](./docs/interacting-with-fleet-manager.md#interacting-directly-with-the-api)

### Configuring a Data Plane OSD cluster for Fleet Manager

Kas-fleet-manager can be started without a dataplane OSD cluster, however,
no Kafkas will be placed or provisioned.

To manually setup a data plane OSD cluster, please follow the
`Using an existing OSD cluster with manual scaling enabled` option in
the [data-plane-osd-cluster-options.md](docs/data-plane-osd-cluster-options.md)
guide.

## Running Fleet Manager on an OpenShift cluster

Follow this [guide](./docs/deploying-kas-fleet-manager-to-openshift.md) on how
to deploy the KAS Fleet Manager service to a OpenShift cluster.

## Using the Fleet Manager service

### Interacting with Fleet Manager

See the [Interacting with Fleet Manager document](docs/interacting-with-fleet-manager.md)

### Viewing the API docs

```
# Start Swagger UI container
make run/docs

# Launch Swagger UI and Verify from a browser: http://localhost:8082

# Remove Swagger UI conainer
make run/docs/teardown
```

### Running additional CLI commands

In addition to starting and running a Fleet Manager server, the Fleet Manager
binary provides additional commands to interact with the service (i.e. running
data migrations)

To use these commands, run `make binary` to create the `./fleet-manager` binary.

Then run `./fleet-manager -h` for information on the additional available
commands.

### Fleet Manager Environments

The service can be run in a number of different environments. Environments are
essentially bespoke sets of configuration that the service uses to make it
function differently. Environments can be set using the `OCM_ENV` environment
variable. Below are the list of known environments and their
details.

* `development` - The `staging` OCM environment is used. Sentry is disabled.
   Debugging utilities are enabled. This should be used in local development.
   This is the default environment used when directly running the Fleet
   Manager binary and the `OCM_ENV` variable has not been set.
* `testing` - The OCM API is mocked/stubbed out, meaning network calls to OCM
   will fail. The auth service is mocked. This should be used for unit testing.
* `integration` - Identical to `testing` but using an emulated OCM API server
   to respond to OCM API calls, instead of a basic mock. This can be used for
   integration testing to mock OCM behaviour.
* `production` - Debugging utilities are disabled, Sentry is enabled.
   environment can be ignored in most development and is only used when
   the service is deployed.

The `OCM_ENV` environment variable should be set before running any Fleet
Manager binary command or Makefile target.

## Running the Tests

See the [testing](docs/testing.md) document.

## Contributing

See the [contributing guide](CONTRIBUTING.md) for general guidelines.

## Connector Service

The https://github.com/bf2fc6cc711aee1a0c2a/cos-fleet-manager is used to
build the `cos-fleet-manager`  binary which is a fleet manager for connectors
similar to how `kas-fleet-manager` is fleet manager for Kafka  instances.
The `cos-fleet-manager` just imports most of the code from
the `kas-fleet-manager` enabling only connector APIs that are in this
repo's `internal/connector` package.

Connector integration tests require most of the security and access
configuration listed in the [populating configuration guide](docs/populating-configuration.md)
document. Connector service uses AWS secrets manager as a
connector specific _vault service_ for storing connector secret properties
such as usernames, passwords, etc.

Before running integration tests, the required AWS secrets files
in the `secrets/vault` directory MUST be configured in the files:
```
secrets/vault/aws_access_key_id
secrets/vault/aws_secret_access_key
```

## Additional documentation

Additional documentation can be found in the [docs](docs) directory.

Some relevant documents are:
* [Running the Service on an OpenShift cluster](./docs/deploying-kas-fleet-manager-to-openshift.md)
* [Adding new endpoint](docs/adding-a-new-endpoint.md)
* [Adding new CLI flag](docs/adding-new-flags.md)
* [Automated testing](docs/automated-testing.md)
* [Requesting credentials and accounts](docs/getting-credentials-and-accounts.md)
* [Data Plane Setup](docs/data-plane-osd-cluster-options.md)
* [Access Control](docs/access-control.md)
* [Quota Management](docs/quota-management-list-configuration.md)
* [Explanation of JWT token claims used across the fleet-manager](docs/jwt-claims.md)
* [kas-fleet-manager implementation information](docs/implementation.md)
* [Data Plane Cluster dynamic scaling architecture](docs/architecture/data-plane-osd-cluster-dynamic-scaling.md)
* [Explanation of JWT token claims used across the kas-fleet-manager](docs/jwt-claims.md)