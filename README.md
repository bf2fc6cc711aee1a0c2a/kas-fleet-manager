Dinosaur Service Fleet Manager
---
![build status badge](https://github.com/bf2fc6cc711aee1a0c2a/fleet-manager/actions/workflows/ci.yaml/badge.svg)

A service for provisioning and managing fleets of Dinosaur instances.

For more information on how the service works, see [the implementation documentation](docs/implementation.md).

## Prerequisites
* [Golang 1.16+](https://golang.org/dl/)
* [Docker](https://docs.docker.com/get-docker/) - to create database
* [ocm cli](https://github.com/openshift-online/ocm-cli/releases) - ocm command line tool

Apart from the installed software prerequisites, here are other ones required for running fleet-manager due to its interaction with external services. All of the below are required to run fleet-manager locally.

### User Account & Organization Setup
1. Create new organization and users being members of this organization:
* Make sure that you are logged out from any existing `access.redhat.com` sessions. Use the `register new organization` flow to create an organization
* Create new users using user management UI. It is recommended to append or prefix usernames of your team members, e.g. `dffrench_control_plane` for `dffrench` username being a member of `Control Plane` team.

2. To onboard a fleet-manager to [app-interface](https://gitlab.cee.redhat.com/service/app-interface) a new role will be required for resources (e.g. Syncsets, etc) creation (see [example](https://gitlab.cee.redhat.com/service/uhc-account-manager/-/blob/master/pkg/api/roles/managed_kafka_service.go))
3. Once the role is created, users required to have this role need to get it assigned to their OCM account: [example MR](https://gitlab.cee.redhat.com/service/ocm-resources/-/merge_requests/812).
4. Very likely your organization will need to have quota for installing Add-ons specific for your fleet-manager. See an example of Add-on quotas [here](https://gitlab.cee.redhat.com/service/ocm-resources/-/blob/master/data/uhc-stage/orgs/13640203.yaml) (find your organization by its `external_id` beneath [ocm-resources/uhc-stage/orgs](https://gitlab.cee.redhat.com/service/ocm-resources/-/tree/master/data/uhc-stage/orgs)).

### Vault
For automation (fleet-manager deployments and app-interface pipelines) it is required to use [vault](https://gitlab.cee.redhat.com/service/app-interface#manage-secrets-via-app-interface-openshiftnamespace-1yml-using-vault) for secrets management

### AWS
AWS accounts are required for provisioning development OSD clusters and to access AWS Route53 for Domain Name System. To request a AWS accounts, have a chat with your team's manager.  
> NOTE, if you are in the Middleware, make sure to send an email to `mw-billing-leaders@redhat.com` request the account to be created. The email should follow the below format:
```
*Request Type*
New AWS account
*Team*
<Whoever the requesting team is>
 
*Cost*
<Calculations for the estimated monthly charges>
*Why?*
<The reason for requesting the account> 
```
> NOTE
Within the AWS account used to provision OSD clusters, you must create an osdCcsAdmin IAM user with the following requirements:
- This user needs at least Programmatic access enabled.
- This user must have the AdministratorAccess policy attached to it.
See https://docs.openshift.com/dedicated/osd_planning/aws-ccs.html#ccs-aws-customer-procedure_aws-ccs for more information.

### SSO authentication
Depending on whether interacting with public or private endpoints, the authentication should be set up as follows:
* sso.redhat.com for public endpoints
* MAS-SSO for private endpoints (see this [doc](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/docs/mas-sso/service-architecture/service-architecture.md) to understand how kas-fleet-manager is using MAS-SSO for authentication)

See [feature-flags](docs/feature-flags.md#Keycloak) to understand flags used for authentication.

### sso.redhat.com service account
To avail of all required OCM services the fleet-manager depends on, it is required to create an sso.redhat.com service account that will be used for the communication between the fleet-manager and OCM. It is a help desk ticket to get this created which gets routed to Red Hat IT. 
The link to create the request is https://redhat.service-now.com/help?id=sc_cat_item&sys_id=7ab45993131c9380196f7e276144b054

### Quota Management with Account Management Service (AMS) SKU
The [Account Management Service](https://api.openshift.com/?urls.primaryName=Accounts%20management%20service) manages users subscriptions. The leverage the service offered by AMS to manage quota. Quota comes in form of stock keeping unit (SKU) assigned to a given organisation. The process is the same as requesting SKU for addons which is described in the following link https://gitlab.cee.redhat.com/service/managed-tenants/-/blob/main/docs/tenants/requesting_skus.md.

### Deploying the Fleet Manager with AppInterface and Onboarding with AppSRE

To onboard with Service Delivery AppSRE, please follow the following [onboarding document](https://gitlab.cee.redhat.com/app-sre/contract/-/blob/master/content/service/service_onboarding_flow.md) which details the whole process. 

#### Envoy Configuration and Rate Limiting with AppInterface

All traffic goes through the Envoy container, which sends requests to the [3scale Limitador](https://github.com/Kuadrant/limitador) instance for global rate limiting across all API endpoints. If Limitador is unavailable, the response behaviour can be configured as desired. For this setup, the Envoy configuration when Limitador is unavailable is setup to forward all requests to the fleet-manager container. If the Envoy sidecar container itself was unavailable, the API would be unavailable.

1. The Envoy configuration for your deployment will need to be provided in AppInterface. Follow the [manage a config map](https://gitlab.cee.redhat.com/service/app-interface#example-manage-a-configmap-via-app-interface-openshiftnamespace-1yml) guideline. 
The file [kas-fleet-manager Envoy ConfigMap](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/resources/services/managed-services/production/kas-fleet-manager-Envoy.configmap.yaml) can serve as a starting template for your fleet manager. This file is further referenced in [kas-fleet-manager namespace](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/data/services/managed-services/namespaces/managed-services-production.yml#L58) so that the config map gets created. 

2. Rate Limiting configuration is managed in [rate-limiting saas file](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/data/services/rate-limiting/cicd/saas.yaml). Follow the [`kas-fleet-manager` example](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/data/services/rate-limiting/cicd/saas.yaml#L197) to setup your configuration.

For more information on the setup, please see the [Rate Limiting template](https://gitlab.cee.redhat.com/service/rate-limiting-templates) and engage AppSRE for help. 

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
4. Setup SSO configuration
    - keycloak cert
    ```
    echo "" | openssl s_client -servername identity.api.stage.openshift.com -connect identity.api.stage.openshift.com:443 -prexit 2>/dev/null | sed -n -e '/BEGIN\ CERTIFICATE/,/END\ CERTIFICATE/ p' > secrets/keycloak-service.crt
    ```
    - mas sso client id & client secret
    ```
    make keycloak/setup SSO_CLIENT_ID=<SSO_client_id> SSO_CLIENT_SECRET=<SSO_client_secret> OSD_IDP_SSO_CLIENT_ID=<osd_idp_SSO_client_id> OSD_IDP_SSO_CLIENT_SECRET=<osd_idp_SSO_client_secret>
    ```
5. Setup Dinosaur TLS cert
```
make dinosaurcert/setup
```
6. Setup the image pull secret
    - To be able to pull docker images from quay, copy the content of your account's secret (`config.json` key) and paste it to `secrets/image-pull.dockerconfigjson` file.

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
