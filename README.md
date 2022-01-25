Kafka Service Fleet Manager
---
![build status badge](https://github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/actions/workflows/ci.yaml/badge.svg)

A service for provisioning and managing fleets of Kafka instances.

For more information on how the service works, see [the implementation documentation](docs/implementation.md).

## Prerequisites
* [Golang 1.16+](https://golang.org/dl/)
* [Docker](https://docs.docker.com/get-docker/) - to create database
* [ocm cli](https://github.com/openshift-online/ocm-cli/releases) - ocm command line tool
* [Node.js v12.20+](https://nodejs.org/en/download/) and [npm](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm)

There are a number of prerequisites required for running kas-fleet-manager due to its interaction with external services. All of the below are required to run kas-fleet-manager locally.
### User Account & Organization Setup
1. Request additional permissions for your user account in OCM stage. [Example MR](https://gitlab.cee.redhat.com/service/ocm-resources/-/merge_requests/812).
    - Ensure your user has the role `ManagedKafkaService`. This allows your user to create Syncsets.
3. Ensure the organization your account or service account belongs to has quota for installing the Managed Kafka Add-on, see this [example](https://gitlab.cee.redhat.com/service/ocm-resources/-/blob/master/data/uhc-stage/orgs/13640203.yaml).
    - Find your organization by its `external_id` beneath [ocm-resources/uhc-stage/orgs](https://gitlab.cee.redhat.com/service/ocm-resources/-/tree/master/data/uhc-stage/orgs).

### Configuring Observability
The Observability stack requires a Personal Access Token to read externalized configuration from within the bf2 organization.
For development cycles, you will need to generate a personal token for your own GitHub user (with bf2 access) and place it within
the `secrets/observability-config-access.token` file.

To generate a new token:
1. Follow the steps [found here](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token),
making sure to check **ONLY** the `repo` box at the top of the scopes/permissions list (which will check each of the subcategory boxes beneath it).
2. Copy the value of your Personal Access Token to a secure private location. Once you leave the page, you cannot access the value
again & you will be forced to reset the token to receive a new value should you lose the original.
3. Paste the token value in the `secrets/observability-config-access.token` file.

### Data Plane OSD cluster setup
Kas-fleet-manager can be started without a dataplane OSD cluster, however, no Kafkas will be placed or provisioned. To setup a data plane OSD cluster, please follow the `Using an existing OSD cluster with manual scaling enabled` option in the [data-plane-osd-cluster-options.md](docs/data-plane-osd-cluster-options.md) guide.

### Populating Configuration
1. Add your organization's `external_id` to the [Quota Management List Configurations](./docs/quota-management-list-configuration.md) 
if you need to create STANDARD kafka instances. Follow the guide in [Quota Management List Configurations](./docs/access-control.md)
2. Follow the guide in [Access Control Configurations](./docs/access-control.md) to configure access control as required.
3. Retrieve your ocm-offline-token from https://qaprodauth.cloud.redhat.com/openshift/token and save it to `secrets/ocm-service.token` 
4. Setup AWS configuration
```
make aws/setup
```
4. Setup MAS SSO configuration
    - keycloak cert
    ```
    echo "" | openssl s_client -servername identity.api.stage.openshift.com -connect identity.api.stage.openshift.com:443 -prexit 2>/dev/null | sed -n -e '/BEGIN\ CERTIFICATE/,/END\ CERTIFICATE/ p' > secrets/keycloak-service.crt
    ```
    - mas sso client id & client secret
    ```
    make keycloak/setup MAS_SSO_CLIENT_ID=<mas_sso_client_id> MAS_SSO_CLIENT_SECRET=<mas_sso_client_secret> OSD_IDP_MAS_SSO_CLIENT_ID=<osd_idp_mas_sso_client_id> OSD_IDP_MAS_SSO_CLIENT_SECRET=<osd_idp_mas_sso_client_secret>
    ```
    > Values can be found in [Vault](https://vault.devshift.net/ui/vault/secrets/managed-application-services/show/MK-Control-Plane-team/integration-tests).
5. Setup Kafka TLS cert
```
make kafkacert/setup
```
6. Setup the image pull secret
    - Image pull secret for RHOAS can be found in [Vault](https://vault.devshift.net/ui/vault/secrets/managed-services/show/quay-org-accounts/rhoas/robots/rhoas-pull), copy the content for the `config.json` key and paste it to `secrets/image-pull.dockerconfigjson` file.
7. Setup the Observability stack secrets
```
make observatorium/setup
```

## Running a Local Observatorium Token Refresher 
> NOTE: This is only required if your Observatorium instance is authenticated using sso.redhat.com.

Run the following make target:
```
make observatorium/token-refresher/setup CLIENT_ID=<client-id> CLIENT_SECRET=<client-secret> [OPTIONAL PARAMETERS]
```

**Required Parameters**:
- CLIENT_ID: The client id of a service account that has, at least, permissions to read metrics.
- ClIENT_SECRET: The client secret of a service account that has, at least, permissions to read metrics.

**Optional Parameters**:
- PORT: Port for running the token refresher on. Defaults to `8085`
- IMAGE_TAG: Image tag of the [token-refresher image](https://quay.io/repository/rhoas/mk-token-refresher?tab=tags). Defaults to `latest`
- ISSUER_URL: URL of your auth issuer. Defaults to `https://sso.redhat.com/auth/realms/redhat-external`
- OBSERVATORIUM_URL: URL of your Observatorium instance. Defaults to `https://observatorium-mst.api.stage.openshift.com/api/metrics/v1/managedkafka`

## Running the Service locally
Please make sure you have followed all of the prerequisites above first.

1. Compile the binary
```
make binary
```
2. Clean up and Creating the database
    - If you have db already created execute
    ```
    make db/teardown
    ```
    - Create database tables
    ```
    make db/setup && make db/migrate
    ```
    - Optional - Verify tables and records are created
    ```
    make db/login
    ```
    ```
    # List all the tables
    serviceapitests# \dt
                       List of relations
    Schema |        Name        | Type  |       Owner
    --------+--------------------+-------+-------------------
    public | clusters           | table | kas_fleet_manager
    public | connector_clusters | table | kas_fleet_manager
    public | connectors         | table | kas_fleet_manager
    public | kafka_requests     | table | kas_fleet_manager
    public | leader_leases      | table | kas_fleet_manager
    public | migrations         | table | kas_fleet_manager
    ```
3. Generate OCM token secret
    ```
    make ocm/setup OCM_OFFLINE_TOKEN=<ocm-offline-token> OCM_ENV=development
    ```
4. Start the service
    ```
    ./kas-fleet-manager serve
    ```
    >**NOTE**: The service has numerous feature flags which can be used to enable/disable certain features of the service. Please see the [feature flag](./docs/feature-flags.md) documentation for more information.
5. Verify the local service is working
    ```
    curl -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/kafkas_mgmt/v1/kafkas
   {"kind":"KafkaRequestList","page":1,"size":0,"total":0,"items":[]}
    ```

## Running the Service on an OpenShift cluster
Follow this [guide](./docs/deploying-kas-fleet-manager-to-openshift.md) on how to deploy the KAS Fleet Manager service to an OpenShift cluster.

## Using the Service
### Kafkas
#### Creating a Kafka Cluster
```
# Submit a new Kafka cluster creation request
curl -v -XPOST -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/kafkas_mgmt/v1/kafkas?async=true -d '{ "region": "us-east-1", "cloud_provider": "aws",  "name": "serviceapi", "multi_az":true}'

# List a kafka request
curl -v -XGET -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/kafkas_mgmt/v1/kafkas/<kafka_request_id> | jq

# List all kafka request
curl -v -XGET -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/kafkas_mgmt/v1/kafkas | jq

# Delete a kafka request
curl -v -X DELETE -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/kafkas_mgmt/v1/kafkas/<kafka_request_id>
```
#### Using RHOAS CLI to manage Kafka instances

The locally installed kas-fleet-manager doesn't deploy TLS enabled kafka admin server but the default URL scheme used by the app-service-cli is HTTPS. So, the url scheme should be changed to http in the CLI [code](https://github.com/redhat-developer/app-services-cli/blob/main/pkg/core/connection/api/defaultapi/default_client.go#L155) and it should be built locally
```
make binary
```
Then, it can be pointed towards the locally running fleet manager
```
./rhoas login --mas-auth-url=stage --api-gateway=http://localhost:8000
``` 
Now, various kafka specific operations can be performed as described [here](http://appservices.tech/commands/rhoas_kafka)

#### Using the Kafka admin-server API
 - The admin-server API is used for managing topics, acls, and consumer groups. The API specification can be found [here](https://github.com/bf2fc6cc711aee1a0c2a/kafka-admin-api/blob/main/kafka-admin/src/main/resources/openapi-specs/kafka-admin-rest.yaml)
- To get the API endpoint, use the following command
```
oc get routes -n kafka-c7ndprea8ueq4mmrm3c0 | grep admin-server

# Here c7ndprea8ueq4mmrm3c0 is the kafka instance ID.
```

### View the API docs
```
# Start Swagger UI container
make run/docs

# Launch Swagger UI and Verify from a browser: http://localhost:8082

# Remove Swagger UI conainer
make run/docs/teardown
```
## Using podman instead of docker
Install the podman-docker utility. This will create a symbolic link for ```/run/docker.sock``` to ```/run/podman/podman.sock```
```
#Fedora and RHEL8
dnf -y install podman-docker

#Ubuntu 21.10 or higher
apt -y install podman-docker
```
Note: As this is running rootless containers, please check the etc/subuid and etc/subgid files and make sure that the configured range includes the UID of current user. Please find more details [here](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md#enable-user-namespaces-on-rhel7-machines)

## Additional CLI commands

In addition to the REST API exposed via `make run`, there are additional commands to interact directly
with the service (i.e. cluster creation/scaling, Kafka creation, Errors list, etc.) without having to use a REST API client.

To use these commands, run `make binary` to create the `./kas-fleet-manager` CLI.

Run `./kas-fleet-manager -h` for information on the additional commands.
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


## Running the Tests
### Running unit tests
```
make test
```

### Running integration tests

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
OCM_ENV=integration make test/integration
```

To run integration tests with a real OCM environment, run:

```
make test/integration
```

To stop and remove the database container when finished, run:
```
make db/teardown
```

### Connector Service

The https://github.com/bf2fc6cc711aee1a0c2a/cos-fleet-manager is used to build the `cos-fleet-manager` 
binary which is a fleet manager for connectors similar to how `kas-fleet-manager` is fleet manager for Kafka 
instances.  The `cos-fleet-manager` just imports most of the code from the `kas-fleet-manager` enabling only
connector APIs that are in this repo's `internal/connector` package.

Connector integration tests require most of the security and access configuration listed in [Prerequisites](#Prerequisites). 
Connector service uses AWS secrets manager as a connector specific _vault service_ for storing connector secret 
properties such as usernames, passwords, etc.

Before running integration tests, the required AWS secrets files in the `secrets` directory MUST be configured using the command:
```
make aws/setup
``` 

## Additional documentation:
* [kas-fleet-manager Implementation](docs/implementation.md)
* [Data Plane Cluster dynamic scaling architecture](docs/architecture/data-plane-osd-cluster-dynamic-scaling.md)
* [Explanation of JWT token claims used across the kas-fleet-manager](docs/jwt-claims.md)
