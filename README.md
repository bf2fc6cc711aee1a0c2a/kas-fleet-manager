Kafka Service Fleet Manager
---
![build status badge](https://github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/actions/workflows/ci.yaml/badge.svg)

A service for provisioning and managing fleets of Kafka instances.

For more information on how the service works, see [the implementation documentation](docs/implementation.md).

## Prerequisites
* [Golang 1.15+](https://golang.org/dl/)
* [Docker](https://docs.docker.com/get-docker/) - to create database
* [ocm cli](https://github.com/openshift-online/ocm-cli/releases) - ocm command line tool

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
    > Values can be found in [Vault](https://vault.devshift.net/ui/vault/secrets/managed-services-ci/show/managed-service-api/integration-tests).
5. Setup Kafka TLS cert
```
make kafkacert/setup
```
6. Setup the image pull secret
    - Image pull secret for RHOAS can be found in [Vault](https://vault.devshift.net/ui/vault/secrets/managed-services/show/quay-org-accounts/rhoas/robots/rhoas-pull), copy the content for the `config.json` key and paste it to `secrets/image-pull.dockerconfigjson` file.

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

3. Start the service
    ```
    ./kas-fleet-manager serve
    ```
    >**NOTE**: The service has numerous feature flags which can be used to enable/disable certain features of the service. Please see the [feature flag](./docs/feature-flags.md) documentation for more information.
4. Verify the local service is working
    ```
    curl -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/kafkas_mgmt/v1/kafkas
   {"kind":"KafkaRequestList","page":1,"size":0,"total":0,"items":[]}
    ```

## Running the Service on an OpenShift cluster
### Create a Namespace
Create a namespace where KAS Fleet Manager will be deployed to
```
make deploy/project <OPTIONAL_PARAMETERS>
```

**Optional parameters**:
- `NAMESPACE`: The namespace where the image will be pushed to. Defaults to 'kas-fleet-manager-$USER.'

### Build and Push the Image to the OpenShift Image Registry
Login to the OpenShift internal image registry

>**NOTE**: Ensure that the user used has the correct permissions to push to the OpenShift image registry. For more information, see the [accessing the registry](https://docs.openshift.com/container-platform/4.5/registry/accessing-the-registry.html#prerequisites) guide.
```
# Login to the OpenShift cluster
oc login <api-url> -u <username> -p <password>

# Login to the OpenShift image registry
make docker/login/internal
```

Build and push the image
```
# Build and push the image to the OpenShift image registry.
GOARCH=amd64 GOOS=linux CGO_ENABLED=0 make image/build/push/internal <OPTIONAL_PARAMETERS>
```

**Optional parameters**:
- `NAMESPACE`: The namespace where the image will be pushed to. Defaults to 'kas-fleet-manager-$USER.'
- `IMAGE_TAG`: Tag for the image. Defaults to a timestamp captured when the command is run (i.e. 1603447837).

### Deploy the Service
#### Deploy a postgres database
```
make deploy/db <OPTIONAL_PARAMETERS>
```

**Optional parameters**:
- `NAMESPACE`: The namespace to be created. Defaults to 'kas-fleet-manager-$USER.'

#### Create the secrets
This will create the following secrets in the given namespace:
- `kas-fleet-manager`
- `kas-fleet-manager-dataplane-certificate`
- `kas-fleet-manager-observatorium-configuration-red-hat-sso`
- `kas-fleet-manager-rds`

```
make deploy/secrets <OPTIONAL_PARAMETERS>
```

**Optional parameters**:
- `NAMESPACE`: The namespace where the secrets will be created in. Defaults to 'kas-fleet-manager-$USER.'
- `OCM_SERVICE_CLIENT_ID`: The client id for an OCM service account. Defaults to value read from _./secrets/ocm-service.clientId_
- `OCM_SERVICE_CLIENT_SECRET`: The client secret for an OCM service account. Defaults to value read from _./secrets/ocm-service.clientSecret_
- `OCM_SERVICE_TOKEN`: An offline token for an OCM service account. Defaults to value read from _./secrets/ocm-service.token_
- `SENTRY_KEY`: Token used to authenticate with Sentry. Defaults to value read from _./secrets/sentry.key_
- `AWS_ACCESS_KEY`: The access key of an AWS account used to provision OpenShift clusters. Defaults to value read from _./secrets/aws.accesskey_
- `AWS_ACCOUNT_ID`: The account id of an AWS account used to provision OpenShift clusters. Defaults to value read from _./secrets/aws.accountid_
- `AWS_SECRET_ACCESS_KEY`: The secret access key of an AWS account used to provision OpenShift clusters. Defaults to value read from _./secrets/aws.secretaccesskey_
- `ROUTE53_ACCESS_KEY`: The access key of an AWS account that has Route53 permissions. Defaults to value read from _./secrets/aws.route53accesskey_
- `ROUTE53_SECRET_ACCESS_KEY`: The secret access key of an AWS account that has Route53 permissions. Defaults to value read from _./secrets/aws.route53secretaccesskey_
- `VAULT_ACCESS_KEY`: AWS secrets manager access key. Defaults to value read from _./secrets/vault.accesskey_
- `VAULT_SECRET_ACCESS_KEY`: AWS secrets manager secret access key. Defaults to value read from _./secrets/vault.secretaccesskey_
- `OBSERVATORIUM_SERVICE_TOKEN`: Offline token used to interact with Observatorium. Defaults to value read from _./secrets/observatorium.token_
- `DEX_SECRET`: Dex secret used to authenticate to an Observatorium instance using Dex as authentication. Defaults to value read from _./secrets/dex.secret_
- `DEX_PASSWORD`: Dex password used to authenticate to an Observatorium instance using Dex as authentication. Defaults to value read from _./secrets/dex.secret_
- `MAS_SSO_CLIENT_ID`: The client id for a MAS SSO service account. Defaults to value read from _./secrets/keycloak-service.clientId_
- `MAS_SSO_CLIENT_SECRET`: The client secret for a MAS SSO service account. Defaults to value read from _./secrets/keycloak-service.clientSecret_
- `OSD_IDP_MAS_SSO_CLIENT_ID`: The client id for a MAS SSO service account used to configure OpenShift identity provider. Defaults to value read from _./secrets/osd-idp-keycloak-service.clientId_
- `OSD_IDP_MAS_SSO_CLIENT_SECRET`: The client secret for a MAS SSO service account used to configure OpenShift identity provider. Defaults to value read from _./secrets/osd-idp-keycloak-service.clientSecret_
- `IMAGE_PULL_DOCKER_CONFIG`: Docker config content for pulling private images. Defaults to value read from _./secrets/image-pull.dockerconfigjson_
- `KUBE_CONFIG`: Kubeconfig content for standalone dataplane clusters communication. Defaults to `''`
- `OBSERVABILITY_RHSSO_LOGS_CLIENT_ID`: The client id for a RHSSO service account that has read logs permission. Defaults to vaue read from _./secrets/rhsso-logs.clientId_
- `OBSERVABILITY_RHSSO_LOGS_SECRET`: The client secret for a RHSSO service account that has read logs permission. Defaults to vaue read from _./secrets/rhsso-logs.clientSecret_
- `OBSERVABILITY_RHSSO_METRICS_CLIENT_ID`: The client id for a RHSSO service account that has remote-write metrics permission. Defaults to vaue read from _./secrets/rhsso-metrics.clientId_
- `OBSERVABILITY_RHSSO_METRICS_SECRET`: The client secret for a RHSSO service account that has remote-write metrics permission. Defaults to vaue read from _./secrets/rhsso-metrics.clientSecret_
- `OBSERVABILITY_RHSSO_METRICS_CLIENT_ID`: The client id for a RHSSO service account that has read metrics permission. Defaults to `''`
- `OBSERVABILITY_RHSSO_METRICS_SECRET`: The client secret for a RHSSO service account that has read metrics permission. Defaults to `''`

#### (Optional) Deploy the Observatorium Token Refresher
>**NOTE**: This is only needed if your Observatorium instance is using RHSSO as authentication.

```
make deploy/token-refresher OBSERVATORIUM_URL=<observatorium-url> <OPTIONAL_PARAMETERS>
```

**Optional parameters**
- `ISSUER_URL`: The issuer URL of your authentication service. Defaults to `https://sso.redhat.com/auth/realms/redhat-external`
- `OBSERVATORIUM_TOKEN_REFRESHER_IMAGE`: The image repository used for the Observatorium token refresher deployment. Defaults to `quay.io/rhoas/mk-token-refresher`.
- `OBSERVATORIUM_TOKEN_REFRESHER_IMAGE_TAG`: The image tag used for the Observatorium token refresher deployment. Defaults to `latest`

#### Deploy KAS Fleet Manager
```
make deploy/service <OPTIONAL_PARAMETERS>
```

**Optional parameters**:
- `NAMESPACE`: The namespace where the service will be deployed to. Defaults to managed-services-$USER.
- `ENV`: Environment used for the KAS Fleet Manager deployment. Options: `development`, `integration`, `testing`, `stage` and `production`, Default: `development`.
- `IMAGE_REGISTRY`: Registry used by the image. Defaults to the OpenShift internal registry.
- `IMAGE_REPOSITORY`: Image repository. Defaults to '\<namespace\>/kas-fleet-manager'.
- `IMAGE_TAG`: Tag for the image. Defaults to a timestamp captured when the command is run (i.e. 1603447837).
- `REPLICAS`: Number of replicas of the KAS Fleet Manager deployment. Defaults to `1`.
- `ENABLE_KAFKA_EXTERNAL_CERTIFICATE`: Enable Kafka TLS Certificate. Defaults to `false`.
- `ENABLE_KAFKA_LIFE_SPAN`: Enables Kafka expiration. Defaults to `false`.
- `KAFKA_LIFE_SPAN`: Kafka expiration lifetime in hours. Defaults to `48`.
- `ENABLE_OCM_MOCK`: Enables use of a mocked ocm client. Defaults to `false`.
- `OCM_MOCK_MODE`: The type of mock to use when ocm mock is enabled.Options: `emulate-server` and `stub-server`. Defaults to `emulate-server`.
- `OCM_URL`: OCM API base URL. Defaults to `https://api.stage.openshift.com`.
- `AMS_URL`: AMS API base URL. Defaults to `''`.
- `JWKS_URL`: JWK Token Certificate URL. Defaults to `''`.
- `MAS_SSO_ENABLE_AUTH`: Enables MAS SSO authentication for the Data Plane. Defaults to `true`.
- `MAS_SSO_BASE_URL`: MAS SSO base url. Defaults to `https://identity.api.stage.openshift.com`.
- `MAS_SSO_REALM`: MAS SSO realm url. Defaults to `rhoas`.
- `MAX_ALLOWED_SERVICE_ACCOUNTS`: The default value of maximum number of service accounts that can be created by users. Defaults to `2`.
- `MAX_LIMIT_FOR_SSO_GET_CLIENTS`: The default value of maximum number of clients fetch from mas-sso. Defaults to `100`.
- `OSD_IDP_MAS_SSO_REALM`: MAS SSO realm for configuring OpenShift Cluster Identity Provider Clients. Defaults to `rhoas-kafka-sre`.
- `TOKEN_ISSUER_URL`: A token issuer url used to validate if JWT token used are coming from the given issuer. Defaults to `https://sso.redhat.com/auth/realms/redhat-external`.
- `VAULT_KIND`: The type of vault to use to store secrets. Defaults to `tmp`.
- `OBSERVATORIUM_AUTH_TYPE`: Authentication type for the Observability stack. Options: `dex` and `redhat`, Default: `dex`.
- `DEX_USERNAME`: Username that will be used to authenticate with an Observatorium using Dex as authentication. Defaults to `admin@example.com`.
- `DEX_URL`: Dex URL. Defaults to `http://dex-dex.apps.pbraun-observatorium.observability.rhmw.io`.
- `OBSERVATORIUM_GATEWAY`: URL of an Observatorium using Dex as authentication. Defaults to `https://observatorium-observatorium.apps.pbraun-observatorium.observability.rhmw.io`.
- `OBSERVATORIUM_TENANT`: Tenant of an Observatorium using Dex as authentication. Defaults to `test`.
- `OBSERVATORIUM_RHSSO_GATEWAY`: URL of an Observatorium using RHSSO as authentication. Defaults to `''`.
- `OBSERVATORIUM_RHSSO_TENANT`: Tenant of an Observatorium using RHSSO as authentication. Defaults to `''`.
- `OBSERVATORIUM_RHSSO_AUTH_SERVER_URL`: RHSSO auth server URL used for Observatorium authentication. Defaults to `''`.
- `OBSERVATORIUM_RHSSO_REALM`: Realm of RHSSO used for Observatorium authentication. Defaults to `''`.
- `OBSERVABILITY_CONFIG_REPO`: URL of the configuration repository used by the Observability stack. Defaults to `https://api.github.com/repos/bf2fc6cc711aee1a0c2a/observability-resources-mk/contents`.
- `ENABLE_TERMS_ACCEPTANCE`: Enables terms acceptance through AMS. Defaults to `false`.
- `ALLOW_EVALUATOR_INSTANCE`: Enables creation of evaluator Kafka instances. Defaults to `true`.
- `QUOTA_TYPE`: Quota management service to be used. Options: `quota-management-list` and `ams`, Default: `quota-management-list`.
- `KAS_FLEETSHARD_OLM_INDEX_IMAGE`: KAS Fleetshard operator OLM index image. Defaults to `quay.io/osd-addons/kas-fleetshard-operator:production-82b42db`.
- `STRIMZI_OLM_INDEX_IMAGE`: Strimzi operator OLM index image. Defaults to `quay.io/osd-addons/managed-kafka:production-82b42db`.
- `STRIMZI_OPERATOR_VERSION`: Version of Strimzi operator to install. Defaults to `''`.
- `DEFAULT_KAFKA_VERSION`: Version of Kafka when creating Kafka instances. Defaults to `2.7.0`
- `DATAPLANE_CLUSTER_SCALING_TYPE`: Dataplane cluster scaling type. Options: `manual`, `auto` and `none`, Defaults: `manual`.
- `CLUSTER_LOGGING_OPERATOR_ADDON_ID`: The id of the cluster logging operator addon. Defaults to `''`.
- `STRIMZI_OPERATOR_ADDON_ID`: The id of the Strimzi operator addon. Defaults to `managed-kafka-qe`.
- `KAS_FLEETSHARD_ADDON_ID`: The id of the kas-fleetshard operator addon. Defaults to `kas-fleetshard-operator-qe`.

#### Update the public-host-url of the KAS Fleet Manager deployment
Get the KAS Fleet Manager host
```
oc get routes/kas-fleet-manager -o jsonpath='{.spec.host}' -n <namespace>
```

Set the `--public-host-url` parameter of the `serve` cmd in the KAS Fleet Manager deployment to `https://\<kas-fleet-manager-host>
```
oc edit deployment/kas-fleet-manager -n <namespace>

# KAS Fleet Manager Deployment
...
spec:
  ...
  template:
    ...
    spec:
      containers:
      ...
      command:
        - /usr/local/bin/kas-fleet-manager
        - serve
        ...
        - '--public-host-url=https://<kas-fleet-manager-host>'
```

#### Access the service
The service can be accessed by via the host of the route created by the service deployment.
```
oc get route kas-fleet-manager
```

### Removing the Service Deployment from the OpenShift
```
# Removes all resources created on service deployment
make undeploy
```

**Optional parameters**:
- `NAMESPACE`: The namespace where the service deployment will be removed from. Defaults to managed-services-$USER.

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

### Running performance tests
See this [README](./test/performance/README.md) for more info about performance tests

### Connector Service

The https://github.com/bf2fc6cc711aee1a0c2a/cos-fleet-manager is used to build the `cos-fleet-manager` 
binary which is a fleet manager for connectors similar to how `kas-fleet-manager` is fleet manager for Kafka 
instances.  The `cos-fleet-manager` just imports most of the code from the `kas-fleet-manager` enabling only
connector APIs that are in this repo's `internal/connector` package.

## Additional documentation:
* [kas-fleet-manager Implementation](docs/implementation.md)
* [Data Plane Cluster dynamic scaling architecture](docs/architecture/data-plane-osd-cluster-dynamic-scaling.md)
* [Explanation of JWT token claims used across the kas-fleet-manager](docs/jwt-claims.md)
