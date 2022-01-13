Dinosaur Service Fleet Manager
---
![build status badge](https://github.com/bf2fc6cc711aee1a0c2a/fleet-manager/actions/workflows/ci.yaml/badge.svg)

A service for provisioning and managing fleets of Dinosaur instances.

For more information on how the service works, see [the implementation documentation](docs/implementation.md).

## Prerequisites
* [Golang 1.15+](https://golang.org/dl/)
* [Docker](https://docs.docker.com/get-docker/) - to create database
* [ocm cli](https://github.com/openshift-online/ocm-cli/releases) - ocm command line tool

There are a number of prerequisites required for running fleet-manager due to its interaction with external services. All of the below are required to run fleet-manager locally.
### User Account & Organization Setup
1. Request additional permissions for your user account in OCM stage. [Example MR](https://gitlab.cee.redhat.com/service/ocm-resources/-/merge_requests/812).
    - Ensure your user has the role `ManagedDinosaurService`. This allows your user to create Syncsets.
3. Ensure the organization your account or service account belongs to has quota for installing the Managed Dinosaur Add-on, see this [example](https://gitlab.cee.redhat.com/service/ocm-resources/-/blob/master/data/uhc-stage/orgs/13640203.yaml).
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
fleet-manager can be started without a dataplane OSD cluster, however, no Dinosaurs will be placed or provisioned. To setup a data plane OSD cluster, please follow the `Using an existing OSD cluster with manual scaling enabled` option in the [data-plane-osd-cluster-options.md](docs/data-plane-osd-cluster-options.md) guide.

### Populating Configuration
1. Add your organization's `external_id` to the [Quota Management List Configurations](./docs/quota-management-list-configuration.md) 
if you need to create STANDARD dinosaur instances. Follow the guide in [Quota Management List Configurations](./docs/access-control.md)
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
    > Values can be found in [Vault](https://vault.devshift.net/ui/vault/secrets/managed-services-ci/show/managed-service-api/integration-tests).
5. Setup Dinosaur TLS cert
```
make dinosaurcert/setup
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
- OBSERVATORIUM_URL: URL of your Observatorium instance. Defaults to `https://observatorium-mst.api.stage.openshift.com/api/metrics/v1/manageddinosaur`

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
    public | clusters           | table | fleet_manager
    public | dinosaur_requests  | table | fleet_manager
    public | leader_leases      | table | fleet_manager
    public | migrations         | table | fleet_manager
    ```

3. Start the service
    ```
    ./fleet-manager serve
    ```
    >**NOTE**: The service has numerous feature flags which can be used to enable/disable certain features of the service. Please see the [feature flag](./docs/feature-flags.md) documentation for more information.
4. Verify the local service is working
    ```
    curl -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/dinosaurs_mgmt/v1/dinosaurs
   {"kind":"DinosaurRequestList","page":1,"size":0,"total":0,"items":[]}
    ```

## Running the Service on an OpenShift cluster
Follow this [guide](./docs/deploying-fleet-manager-to-openshift.md) on how to deploy the Fleet Manager service to an OpenShift cluster.

## Using the Service
### Dinosaurs
#### Creating a Dinosaur Cluster
```
# Submit a new Dinosaur cluster creation request
curl -v -XPOST -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/dinosaurs_mgmt/v1/dinosaurs?async=true -d '{ "region": "us-east-1", "cloud_provider": "aws",  "name": "serviceapi", "multi_az":true}'

# List a dinosaur request
curl -v -XGET -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/dinosaurs_mgmt/v1/dinosaurs/<dinosaur_request_id> | jq

# List all dinosaur request
curl -v -XGET -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/dinosaurs_mgmt/v1/dinosaurs | jq

# Delete a dinosaur request
curl -v -X DELETE -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/dinosaurs_mgmt/v1/dinosaurs/<dinosaur_request_id>
```

### View the API docs
```
# Start Swagger UI container
make run/docs

# Launch Swagger UI and Verify from a browser: http://localhost

# Remove Swagger UI conainer
make run/docs/teardown
```
## Additional CLI commands

In addition to the REST API exposed via `make run`, there are additional commands to interact directly
with the service (i.e. cluster creation/scaling, Dinosaur creation, Errors list, etc.) without having to use a REST API client.

To use these commands, run `make binary` to create the `./fleet-manager` CLI.

Run `./fleet-manager -h` for information on the additional commands.
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

## Additional documentation:
* [fleet-manager Implementation](docs/implementation.md)
* [Data Plane Cluster dynamic scaling architecture](docs/architecture/data-plane-osd-cluster-dynamic-scaling.md)
* [Explanation of JWT token claims used across the fleet-manager](docs/jwt-claims.md)
