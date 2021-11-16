# Deploying KAS Fleet Manager to OpenShift

1. [Create a Namespace](#create-a-namespace)
2. [Build and Push KAS Fleet Manager Image](#build-and-push-kas-fleet-manager-image-to-the-openshift-internal-registry)
3. [Deploy the Database](#deploy-the-database)
4. [Create the Secrets](#create-the-secrets)
5. [(Optional) Deploy the Observatorium Token Refresher](#optional-deploy-the-observatorium-token-refresher)
6. [Deploy KAS Fleet Manager](#deploy-kas-fleet-manager)
7. [Access the Service](#access-the-service)
8. [Removing KAS Fleet Manager from OpenShift](#removing-kas-fleet-manager-from-openshift)

## Create a Namespace
Create a namespace where KAS Fleet Manager will be deployed to
```
make deploy/project <OPTIONAL_PARAMETERS>
```

**Optional parameters**:
- `NAMESPACE`: The namespace where the image will be pushed to. Defaults to 'kas-fleet-manager-$USER.'

## Build and Push KAS Fleet Manager Image to the OpenShift Internal Registry
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

## Deploy the Database
```
make deploy/db <OPTIONAL_PARAMETERS>
```

**Optional parameters**:
- `NAMESPACE`: The namespace to be created. Defaults to 'kas-fleet-manager-$USER.'

## Create the secrets
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
- `MAS_SSO_CRT`: The TLS certificate of the MAS SSO instance. Defaults to value read from _./secrets/keycloak-service.crt_
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

## (Optional) Deploy the Observatorium Token Refresher
>**NOTE**: This is only needed if your Observatorium instance is using RHSSO as authentication.

```
make deploy/token-refresher OBSERVATORIUM_URL=<observatorium-url> <OPTIONAL_PARAMETERS>
```

**Optional parameters**
- `ISSUER_URL`: The issuer URL of your authentication service. Defaults to `https://sso.redhat.com/auth/realms/redhat-external`
- `OBSERVATORIUM_TOKEN_REFRESHER_IMAGE`: The image repository used for the Observatorium token refresher deployment. Defaults to `quay.io/rhoas/mk-token-refresher`.
- `OBSERVATORIUM_TOKEN_REFRESHER_IMAGE_TAG`: The image tag used for the Observatorium token refresher deployment. Defaults to `latest`

## Deploy KAS Fleet Manager
```
make deploy/service IMAGE_TAG=<your-image-tag-here> <OPTIONAL_PARAMETERS>
```

**Required parameters**:
- `IMAGE_TAG`: KAS Fleet Manager image tag.

**Optional parameters**:
- `NAMESPACE`: The namespace where the service will be deployed to. Defaults to managed-services-$USER.
- `ENV`: Environment used for the KAS Fleet Manager deployment. Options: `development`, `integration`, `testing`, `stage` and `production`, Default: `development`.
- `IMAGE_REGISTRY`: Registry used by the image. Defaults to the OpenShift internal registry.
- `IMAGE_REPOSITORY`: Image repository. Defaults to '\<namespace\>/kas-fleet-manager'.
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
- `DEFAULT_KAFKA_VERSION`: Version of Kafka when creating Kafka instances. Defaults to `2.7.0`
- `DATAPLANE_CLUSTER_SCALING_TYPE`: Dataplane cluster scaling type. Options: `manual`, `auto` and `none`, Defaults: `manual`.
- `CLUSTER_LOGGING_OPERATOR_ADDON_ID`: The id of the cluster logging operator addon. Defaults to `''`.
- `STRIMZI_OPERATOR_ADDON_ID`: The id of the Strimzi operator addon. Defaults to `managed-kafka-qe`.
- `KAS_FLEETSHARD_ADDON_ID`: The id of the kas-fleetshard operator addon. Defaults to `kas-fleetshard-operator-qe`.

## Access the service
The service can be accessed by via the host of the route created by the service deployment.
```
oc get route kas-fleet-manager
```

## Removing KAS Fleet Manager from OpenShift
```
# Removes all resources created on service deployment
make undeploy
```

**Optional parameters**:
- `NAMESPACE`: The namespace where the KAS Fleet Manager resources will be removed from. Defaults to managed-services-$USER.