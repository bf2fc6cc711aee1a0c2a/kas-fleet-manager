Managed Service API for Kafka
---

A service for provisioning and managing fleets of Kafka instances.

For more information on how the service works, see [the implementation documentation](docs/implementation.md).

## Prerequisites
* [OpenAPI Generator (4.3.1)](https://openapi-generator.tech/docs/installation/)
* [Golang](https://golang.org/dl/)
* [Docker](https://docs.docker.com/get-docker/) - to create database
* [go-bindata (3.1.2+)](https://github.com/go-bindata/go-bindata) - for code generation
* [ocm cli](https://github.com/openshift-online/ocm-cli/releases) - ocm command line tool
* [moq](https://github.com/matryer/moq) - for mock generation

## User Account & Organization Setup

- Setup your account in stage OCM: 
[Example MR](https://gitlab.cee.redhat.com/service/ocm-resources/-/merge_requests/812) (Skip this step if you have a service account)
    - Ensure your user has the role `ManagedKafkaService`. This allows your user to create Syncsets.
    - Once the MR is merged, retrieve your ocm-offline-token from https://qaprodauth.cloud.redhat.com/openshift/token

- Ensure that the organization's `external_id` appears in the [Allow List Configurations](config/allow-list-configuration.yaml). Follow the guide in [Allow List Configurations](#allow-list-configurations). 

- Ensure the organization your personal OCM account or service account belongs to has quota for installing the Managed Kafka Add-on, see this [example](https://gitlab.cee.redhat.com/service/ocm-resources/-/blob/master/data/uhc-stage/orgs/13640203.yaml).
    - Find your organization by its `external_id` beneath [ocm-resources/uhc-stage/orgs](https://gitlab.cee.redhat.com/service/ocm-resources/-/tree/master/data/uhc-stage/orgs).

## Allow List Configurations

The service is [limited to certain organisation and users (given by their username)](config/allow-list-configuration.yaml). To configure this list, you'll need to have username and or the organisation id. 

The username is the account in question. 

To get the org id: 
- Login to `cloud.redhat.com/openshift/token` with the account in question. 
- Use the supplied command to login to `ocm`, 
- Then run `ocm whoami` and get the organisations id from `external_id` field. 


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
    OCM_ENV=development
    ```
2. Clean up and Creating a database 

    ```
    # If you have db already created execute
    $ make db/teardown
    # Create database tables
    $ make db/setup
    $ make db/migrate
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

3.  Setup AWS credentials 
    
    #### Option A)
    Needed when ENV != (testing|integration)
    ```
    $ make aws/setup AWS_ACCOUNT_ID=<account_id> AWS_ACCESS_KEY=<aws-access-key> AWS_SECRET_ACCESS_KEY=<aws-secret-key>
    ```
    
    #### Option B)
    Works when ENV == (testing|integration)
    ```
    $ make aws/setup
    ```
    #### mas sso setup

    ##### keycloak cert
    ```
    echo "" | openssl s_client -connect keycloak-edge-redhat-rhoam-user-sso.apps.mas-sso-stage.1gzl.s1.devshift.org:443 -prexit 2>/dev/null | sed -n -e '/BEGIN\ CERTIFICATE/,/END\ CERTIFICATE/ p' > secrets/keycloak-service.crt
    ```
    ##### mas sso client id & client secret from keepassdb
    ```
    $ make keycloak/setup MAS_SSO_CLIENT_ID=<mas_sso_client_id> MAS_SSO_CLIENT_SECRET=<mas_sso_client_secret>
    ```

4. Setup external certificate for kafka brokers
    #### Option A)
    Needed when ENV != (stage|production)
    ```
    $ make kafkacert/setup
    ```
    
    #### Option B)
    Works when ENV == (stage|production)
    The certificate and private key can be retrieved from Vault
    ```
    $ make kafkacert/setup KAFKA_TLS_CERT=<kafka_tls_cert> KAFKA_TLS_KEY=<kafka_tls_key>
    ```

5. Generate a temporary ocm token
    Generate a temporary ocm token and set it in the secrets/ocm-service.token file
    > **Note**: This will need to be re-generated as this temporary token will expire within a few minutes.
    ```
    $ make ocm/setup OCM_OFFLINE_TOKEN="$(ocm token)" OCM_ENV=development
    ```
   
6. Running the service locally
    ```
    $ ./managed-services-api serve  (default: http://localhost:8000)
    ```

## Running the Service on an OpenShift cluster
### Build and Push the Image to the OpenShift Image Registry
Login to the OpenShift internal image registry

>**NOTE**: Ensure that the user used has the correct permissions to push to the OpenShift image registry. For more information, see the [accessing the registry](https://docs.openshift.com/container-platform/4.5/registry/accessing-the-registry.html#prerequisites) guide.
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
- `ROUTE53_ACCESS_KEY`: AWS route 53 access key for creating CNAME records
- `ROUTE53_SECRET_ACCESS_KEY`: AWS route 53 secret access key for creating CNAME records
- `KAFKA_TLS_CERT`: Kafka TLS external certificate.
- `KAFKA_TLS_KEY`: Kakfa TLS external certificate private key.
- `OBSERVATORIUM_SERVICE_TOKEN`: Token for observatorium service.

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
# Create a new cluster (OSD). 
# The following command will register a cluster request which will be reconciled by the cluster worker
$ ./managed-services-api cluster create

# Verify cluster record is created 
# Login to the database
$ make db/login
# Ensure the cluster exists in clusters table and monitor the status. It should change to 'ready' after provisioned.
serviceapitests# select * from clusters;

# Alternatively, verify from ocm-cli
$ ocm login --url=https://api.stage.openshift.com/ --token=<OCM_OFFLINE_TOKEN>
# verify the cluster is in OCM
$ ocm list clusters

# Retrieve the OSD cluster login credentials
$ ocm get /api/clusters_mgmt/v1/clusters/<cluster_id>/credentials | jq '.admin'

# Login to the OSD cluster with the credentials you retrieved above
# Verify the OSD cluster was created successfully and have strimzi-operator installed in namespace 'redhat-managed-kafka-operator'
```
#### Using an existing OSD Cluster
Any OSD cluster can be used by the service, it does not have to be created with the service itself. If you already have an existing OSD
cluster, you will need to register it in the database so that it can be used by the service for incoming Kafka requests.  The cluster must have been created with multizone availability.

1. Get the ID of your cluster (e.g. `1h95qckof3s31h3622d35d5eoqh5vtuq`). There are two ways of getting this:
   - From the cluster overview URL. 
        - Go to the `OpenShift Cluster Management Dashboard` > `Clusters`. 
        - Select your cluster from the cluster list to go to the overview page.
        - The ID should be located in the URL:
          `https://cloud.redhat.com/openshift/details/<cluster-id>#overview`
   - From the CLI
        - Run `ocm list clusters`
        - The ID should be displayed under the ID column

2. Register the cluster to the service
    - Run the following command to generate an **INSERT** command:
      ```
      make db/generate/insert/cluster CLUSTER_ID=<your-cluster-id> 
      ```
    - Run the command generated above in your local database.
        - Login to the local database using `make db/login`
        - Ensure that the **clusters** table is available.
            - Create the binary by running `make binary`
            - Run `./managed-services-api migrate`
        - Once the table is available, the generated **INSERT** command can now be run.

3. Ensure the cluster is ready to be used for incoming Kafka requests.
    - Take note of the status of the cluster, `cluster_provisioned`, when you registered it to the database in step 2. This means that the cluster has been successfully provisioned but still have remaining resources to set up (i.e. Strimzi operator installation).
    - Run the service using `make run` and let it reconcile resources required in order to make the cluster ready to be used by Kafka requests.
    - Once done, the cluster status in your database should have changed to `ready`. This means that the service can now assign this cluster to any incoming Kafka requests so that the service can process them.

#### Creating a Kafka Cluster
```
# Submit a new Kafka cluster creation request
$ curl -v -XPOST -H "Authorization: Bearer $(ocm token)" http://localhost:8000/api/managed-services-api/v1/kafkas?async=true -d '{ "region": "us-east-1", "cloud_provider": "aws",  "name": "serviceapi", "multi_az":true}'

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
with the service (i.e. cluster creation/scaling, Kafka creation, Errors list, etc.) without having to use a REST API client.

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
>**NOTE**: This uses golangci-lint which needs to be installed in your `GOPATH/bin`
