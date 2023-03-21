# Deploying KAS Fleet Manager to OpenShift

The minimum OpenShift version supported is OpenShift 4.11.

- [Deploying KAS Fleet Manager to OpenShift](#deploying-kas-fleet-manager-to-openshift)
  - [Create a Namespace](#create-a-namespace)
  - [Build and Push the KAS Fleet Manager Image to a Registry](#build-and-push-the-kas-fleet-manager-image-to-a-registry)
    - [Build and Push to the OpenShift Internal Registry](#build-and-push-to-the-openshift-internal-registry)
    - [Build and Push to your own Repository](#build-and-push-to-your-own-repository)
  - [Deploy the Database](#deploy-the-database)
  - [Create the secrets](#create-the-secrets)
  - [(Optional) Deploy the Observatorium Token Refresher](#optional-deploy-the-observatorium-token-refresher)
  - [Deploy KAS Fleet Manager](#deploy-kas-fleet-manager)
    - [Using an Image from a Private External Registry](#using-an-image-from-a-private-external-registry)
  - [Access the service](#access-the-service)
  - [Removing KAS Fleet Manager from OpenShift](#removing-kas-fleet-manager-from-openshift)

## Create a Namespace
Create a namespace where KAS Fleet Manager will be deployed to
```
make deploy/project <OPTIONAL_PARAMETERS>
```

**Optional parameters**:
- `NAMESPACE`: The namespace where the image will be pushed to. Defaults to 'kas-fleet-manager-$USER.'

## Build and Push the KAS Fleet Manager Image to a Registry
### Build and Push to the OpenShift Internal Registry
Login to the OpenShift cluster

>**NOTE**: Ensure that the user used has the correct permissions to push to the OpenShift image registry. For more information, see the [accessing the registry](https://docs.openshift.com/container-platform/4.5/registry/accessing-the-registry.html#prerequisites) guide.
```
oc login <api-url> -u <username> -p <password>
```

Build and push the image to the logged in OpenShift cluster's image registry
```
# Build and push the image to the OpenShift image registry.
GOARCH=amd64 GOOS=linux CGO_ENABLED=0 make image/build/push/internal <OPTIONAL_PARAMETERS>
```

**Optional parameters**:
- `NAMESPACE`: The namespace where the image will be pushed to. Defaults to 'kas-fleet-manager-$USER.'
- `IMAGE_TAG`: Tag for the image. Defaults to a timestamp captured when the command is run (i.e. 1603447837).

### Build and Push to your own Repository
Login to Docker or Quay

* A make target is available for logging into Quay.io
  ```
  make docker/login QUAY_USER="<username>" QUAY_TOKEN="<password>" <OPTIONAL_PARAMETERS>
  ```

  **Optional parameters**:
  - `DOCKER_CONFIG`: The path to your docker config. Defaults to {current_directory}/.docker

Build and push the KAS Fleet Manager image to your own repository
```
make image/push external_image_registry="<your-image-registry>" image_repository="<your-image-repository>" <OPTIONAL_PARAMETERS>
```

**Optional parameters**:
- `image_tag`: Tag for the image. Defaults to a timestamp captured when the command is run (i.e. 1603447837).

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
- `kas-fleet-manager-aws-secret-manager`

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
- `OBSERVATORIUM_SERVICE_TOKEN`: Offline token used to interact with Observatorium. Defaults to value read from _./secrets/observatorium.token_
- `DEX_SECRET`: Dex secret used to authenticate to an Observatorium instance using Dex as authentication. Defaults to value read from _./secrets/dex.secret_
- `DEX_PASSWORD`: Dex password used to authenticate to an Observatorium instance using Dex as authentication. Defaults to value read from _./secrets/dex.secret_
- `MAS_SSO_CLIENT_ID`: The client id for a MAS SSO service account. Defaults to value read from _./secrets/keycloak-service.clientId_
- `MAS_SSO_CLIENT_SECRET`: The client secret for a MAS SSO service account. Defaults to value read from _./secrets/keycloak-service.clientSecret_
- `MAS_SSO_CRT`: The TLS certificate of the MAS SSO instance. Defaults to value read from _./secrets/keycloak-service.crt_
- `MAS_SSO_INSECURE`: Skip TLS insecure verification for the connection to a MAS SSO instance. Defaults to value false.
- `OSD_IDP_MAS_SSO_CLIENT_ID`: The client id for a MAS SSO service account used to configure OpenShift identity provider. Defaults to value read from _./secrets/osd-idp-keycloak-service.clientId_
- `REDHAT_SSO_CLIENT_ID`: The client id for a REDHAT SSO service account. Defaults to value read from _./secrets/redhatsso-service.clientId_
- `REDHAT_SSO_CLIENT_SECRET`: The client secret for a REDHAT SSO service account. Defaults to value read from _./secrets/redhatsso-service.clientSecret_
- `OSD_IDP_MAS_SSO_CLIENT_SECRET`: The client secret for a MAS SSO service account used to configure OpenShift identity provider. Defaults to value read from _./secrets/osd-idp-keycloak-service.clientSecret_
- `IMAGE_PULL_DOCKER_CONFIG`: Base64 encoded Docker config content for pulling private images. Defaults to value read from _./secrets/image-pull.dockerconfigjson_
- `KUBE_CONFIG`: Base64 encoded Kubeconfig content for standalone dataplane clusters communication. Defaults to `''`
- `OBSERVABILITY_RHSSO_LOGS_CLIENT_ID`: The client id for a RHSSO service account that has read logs permission. Defaults to vaue read from _./secrets/rhsso-logs.clientId_
- `OBSERVABILITY_RHSSO_LOGS_SECRET`: The client secret for a RHSSO service account that has read logs permission. Defaults to vaue read from _./secrets/rhsso-logs.clientSecret_
- `OBSERVABILITY_RHSSO_METRICS_CLIENT_ID`: The client id for a RHSSO service account that has remote-write metrics permission. Defaults to vaue read from _./secrets/rhsso-metrics.clientId_
- `OBSERVABILITY_RHSSO_METRICS_SECRET`: The client secret for a RHSSO service account that has remote-write metrics permission. Defaults to vaue read from _./secrets/rhsso-metrics.clientSecret_
- `OBSERVABILITY_RHSSO_METRICS_CLIENT_ID`: The client id for a RHSSO service account that has read metrics permission. Defaults to `''`
- `OBSERVABILITY_RHSSO_METRICS_SECRET`: The client secret for a RHSSO service account that has read metrics permission. Defaults to `''`
- `JWKS_VERIFY_INSECURE`: Skip TLS insecure verification for the connection for fetching jwks certificate. Defaults to value false.
- `ACME_ISSUER_ACCOUNT_KEY`: The ACME Issuer account key used for the automatic management of certificate. This is required when certificate management mode is `automatic`. Defaults to `''`
- `AWS_SECRET_MANAGER_SECRET_ACCESS_KEY`: AWS secret manager secret access key: Defaults to `''`. This is required when certificate management mode is `automatic`.
- `AWS_SECRET_MANAGER_ACCESS_KEY`: AWS secret manager access key: Defaults to `''`. This is required when certificate management mode is `automatic`.
## (Optional) Deploy the Observatorium Token Refresher
>**NOTE**: This is only needed if your Observatorium instance is using RHSSO as authentication.

```
make deploy/token-refresher <OPTIONAL_PARAMETERS>
```

**Optional parameters**
- `OBSERVATORIUM_URL`: URL of the Observatorium instance to connect to. Defaults to `https://observatorium-mst.api.stage.openshift.com/api/metrics/v1/managedkafka`
- `ISSUER_URL`: The issuer URL of your authentication service. Defaults to `https://sso.redhat.com/auth/realms/redhat-external`
- `OBSERVATORIUM_TOKEN_REFRESHER_IMAGE`: The image repository used for the Observatorium token refresher deployment. Defaults to `quay.io/rhoas/mk-token-refresher`.
- `OBSERVATORIUM_TOKEN_REFRESHER_IMAGE_TAG`: The image tag used for the Observatorium token refresher deployment. Defaults to `latest`
- `OBSERVATORIUM_TOKEN_REFRESHER_REPLICAS`: The number of replicas of the Observatorium token refresher deployment. Defaults to `1`.

## Deploy KAS Fleet Manager
```
make deploy/service IMAGE_TAG=<your-image-tag-here> <OPTIONAL_PARAMETERS>
```

**Required parameters**:
- `IMAGE_TAG`: KAS Fleet Manager image tag.

**Optional parameters**:
- `NAMESPACE`: The namespace where the service will be deployed to. Defaults to managed-services-$USER.
- `FLEET_MANAGER_ENV`: Environment used for the KAS Fleet Manager deployment. Options: `development`, `integration`, `testing`, `stage` and `production`, Default: `development`.
- `IMAGE_REGISTRY`: Registry used by the image. Defaults to the OpenShift internal registry.
- `IMAGE_REPOSITORY`: Image repository. Defaults to '\<namespace\>/kas-fleet-manager'.
- `REPLICAS`: Number of replicas of the KAS Fleet Manager deployment. Defaults to `1`.
- `ENABLE_KAFKA_EXTERNAL_CERTIFICATE`: Enable Kafka TLS Certificate. Defaults to `false`.
- `ENABLE_KAFKA_CNAME_REGISTRATION`: Enable Kafka DNS CNAME Registration. Defaults to `false`.
- `ENABLE_KAFKA_LIFE_SPAN`: Enables Kafka expiration. Defaults to `false`.
- `ENABLE_OCM_MOCK`: Enables use of a mocked ocm client. Defaults to `false`.
- `OCM_MOCK_MODE`: The type of mock to use when ocm mock is enabled.Options: `emulate-server` and `stub-server`. Defaults to `emulate-server`.
- `OCM_URL`: OCM API base URL. Defaults to `https://api.stage.openshift.com`.
- `AMS_URL`: AMS API base URL. Defaults to `https://api.stage.openshift.com`.
- `JWKS_URL`: JWK Token Certificate URL. Defaults to `''`.
- `MAS_SSO_ENABLE_AUTH`: Enables MAS SSO authentication for the Data Plane. Defaults to `true`.
- `MAS_SSO_BASE_URL`: MAS SSO base url. Defaults to `https://identity.api.stage.openshift.com`.
- `MAS_SSO_REALM`: MAS SSO realm url. Defaults to `rhoas`.
- `SSO_SPECIAL_MANAGEMENT_ORG_ID`: Special Management Organization ID used for creating internal Service accounts. Defaults to `13640203` which is the special management organisation id  organisation id for Stage environment.
- `MAX_ALLOWED_SERVICE_ACCOUNTS`: The default value of maximum number of service accounts that can be created by users. Defaults to `2`.
- `SERVICE_ACCOUNT_LIMIT_CHECK_SKIP_ORG_ID_LIST`: A list of Org Ids for which service account limit checks dont apply. Defaults to empty list.
- `MAX_LIMIT_FOR_SSO_GET_CLIENTS`: The default value of maximum number of clients fetch from mas-sso. Defaults to `100`.
- `OSD_IDP_MAS_SSO_REALM`: MAS SSO realm for configuring OpenShift Cluster Identity Provider Clients. Defaults to `rhoas-kafka-sre`.
- `TOKEN_ISSUER_URL`: A token issuer url used to validate if JWT token used are coming from the given issuer. Defaults to `https://sso.redhat.com/auth/realms/redhat-external`.
- `OBSERVATORIUM_AUTH_TYPE`: Authentication type for the Observability stack. Options: `dex` and `redhat`, Default: `dex`.
- `DEX_USERNAME`: Username that will be used to authenticate with an Observatorium using Dex as authentication. Defaults to `admin@example.com`.
- `ENABLE_DENY_LIST`: Enable the deny list access control feature. Defaults to `false`.
- `ENABLE_ACCESS_LIST`: Enable the Access list access control feature. Defaults to `false`.
- `DENIED_USERS`: A list of denied users that are not allowed to access the service. A user is identified by its username. Defaults to `[]`.
- `ACCEPTED_ORGANISATIONS`: A list of accepted organisations that are allowed to access the service. An organisation is identified by its orgId. Defaults to `[]`.
- `DEX_URL`: Dex URL. Defaults to `http://dex-dex.apps.pbraun-observatorium.observability.rhmw.io`.
- `OBSERVATORIUM_GATEWAY`: URL of an Observatorium using Dex as authentication. Defaults to `https://observatorium-observatorium.apps.pbraun-observatorium.observability.rhmw.io`.
- `OBSERVATORIUM_TENANT`: Tenant of an Observatorium using Dex as authentication. Defaults to `test`.
- `OBSERVATORIUM_RHSSO_TENANT`: Tenant of an Observatorium using RHSSO as authentication. Defaults to `''`.
- `OBSERVATORIUM_RHSSO_AUTH_SERVER_URL`: RHSSO auth server URL used for Observatorium authentication. Defaults to `''`.
- `OBSERVATORIUM_RHSSO_REALM`: Realm of RHSSO used for Observatorium authentication. Defaults to `''`.
- `OBSERVABILITY_CONFIG_REPO`: URL of the configuration repository used by the Observability stack. Defaults to `quay.io/rhoas/observability-resources-mk`.
- `DATAPLANE_OBSERVABILITY_CONFIG_ENABLE`: Enable sending metrics to the remote write receiver which is configured in the file referenced from `--dataplane-observability-config-file-path`.
- `ENABLE_TERMS_ACCEPTANCE`: Enables terms acceptance through AMS. Defaults to `false`.
- `ALLOW_DEVELOPER_INSTANCE`: Enables creation of developer Kafka instances. Defaults to `true`.
- `QUOTA_TYPE`: Quota management service to be used. Options: `quota-management-list` and `ams`, Default: `quota-management-list`.
- `KAS_FLEETSHARD_OLM_INDEX_IMAGE`: KAS Fleetshard operator OLM index image. Defaults to `quay.io/osd-addons/kas-fleetshard-operator:production-82b42db`.
- `STRIMZI_OLM_INDEX_IMAGE`: Strimzi operator OLM index image. Defaults to `quay.io/osd-addons/managed-kafka:production-82b42db`.
- `OBSERVABILITY_OPERATOR_INDEX_IMAGE`: Observability Operator index image. Defaults to `quay.io/rhoas/observability-operator-index:v4.2.0`.
- `DATAPLANE_CLUSTER_SCALING_TYPE`: Dataplane cluster scaling type. Options: `manual`, `auto` and `none`, Defaults: `manual`.
- `CLUSTER_LOGGING_OPERATOR_ADDON_ID`: The id of the cluster logging operator addon. Defaults to `''`.
- `STRIMZI_OPERATOR_ADDON_ID`: The id of the Strimzi operator addon. Defaults to `managed-kafka-qe`.
- `KAS_FLEETSHARD_ADDON_ID`: The id of the kas-fleetshard operator addon. Defaults to `kas-fleetshard-operator-qe`.
- `CLUSTER_LIST`: The list of data plane cluster configuration to be used. This is to be used when scaling type is `manual`. Defaults to empty list.
- `SUPPORTED_CLOUD_PROVIDERS`: A list of supported cloud providers in a yaml format. Defaults to `[{name: aws, default: true, regions: [{name: us-east-1, default: true, supported_instance_type: {standard: {}, developer: {}}}]}]`.
- `STRIMZI_OLM_PACKAGE_NAME`: Strimzi operator OLM package name. This is optional and to be defined when interacting with standalone data plane clusters. Defaults to `managed-kafka`.
- `KAS_FLEETSHARD_OLM_PACKAGE_NAME`: kas-fleetshard operator OLM package name. This is optional and to be defined when interacting with standalone data plane clusters. Defaults to `kas-fleetshard-operator`.
- `STRIMZI_OPERATOR_STARTING_CSV`: Strimzi operator starting csv. This is only applied for standalone clusters. Defaults to empty string 
- `KAS_FLEETSHARD_OPERATOR_STARTING_CSV`: Kas-fleetshard operator starting csv. This is only applied for standalone clusters. Defaults to empty string
- `OBSERVABILITY_OPERATOR_STARTING_CSV`: Observability Operator starting CSV. Defaults to `observability-operator.v4.2.0`.
- `KAS_FLEETSHARD_OPERATOR_SUBSCRIPTION_CONFIG`: Kas-fleetshard operator subscription config. This is applied for standalone clusters only. The configuration must be of type [SubscriptionConfig](https://pkg.go.dev/github.com/operator-framework/api@v0.3.25/pkg/operators/v1alpha1?utm_source=gopls#SubscriptionConfig). Defaults to an empty object i.e `{}`. See the [config/kas-fleetshard-operator-subscription-spec-config.yaml](../config/kas-fleetshard-operator-subscription-spec-config.yaml) file for example values.
- `STRIMZI_OPERATOR_SUBSCRIPTION_CONFIG`: Strimzi operator subscription config. This is applied for standalone clusters only. The configuration must be of type [SubscriptionConfig](https://pkg.go.dev/github.com/operator-framework/api@v0.3.25/pkg/operators/v1alpha1?utm_source=gopls#SubscriptionConfig). Defaults to an empty object i.e `{}`. See the [config/strimzi-operator-subscription-spec-config.yaml](../config/strimzi-operator-subscription-spec-config.yaml) file for example values. 
- `SSO_PROVIDER_TYPE`: Option to choose between sso providers i.e, mas_sso or redhat_sso, mas_sso by default.
- `REGISTERED_USERS_PER_ORGANISATION`: The list of allowed organisations that are able to create _STANDARD_ kafka instances. This will only be applicable if `QUOTA_TYPE` is set to **quota-management-list**. Defaults to `"[{id: 13640203, any_user: true, max_allowed_instances: 5, registered_users: [], granted_quota: [{instance_type_id: standard, kafka_billing_models: [{id: standard, max_allowed_instances: 5}, {id: marketplace, max_allowed_instances: 5}, {id: enterprise, max_allowed_instances: 5}]}]}, {id: 12147054, any_user: true, max_allowed_instances: 1, registered_users: [], granted_quota: [{instance_type_id: standard, kafka_billing_models: [{id: standard, max_allowed_instances: 1}, {id: enterprise, max_allowed_instances: 1}]}]}, {id: 13639843, any_user: true, max_allowed_instances: 1, registered_users: [], granted_quota: [{instance_type_id: standard, kafka_billing_models: [{id: standard, max_allowed_instances: 1}, {id: enterprise, max_allowed_instances: 1}]}]}, {id: 13785172, any_user: true, max_allowed_instances: 1, registered_users: [], granted_quota: [{instance_type_id: standard, kafka_billing_models: [{id: standard, max_allowed_instances: 1}, {id: enterprise, max_allowed_instances: 1}]}]}, {id: 13645369, any_user: true, max_allowed_instances: 3, registered_users: [], granted_quota: [{instance_type_id: standard, kafka_billing_models: [{id: standard, max_allowed_instances: 3}, {id: enterprise, max_allowed_instances: 3}]}]}]
"`
- `DYNAMIC_SCALING_CONFIG`: The configuration file that contains information about each Kafka instance types, dynamic scaling configuration. Defaults to `"{new_data_plane_openshift_version: '', enable_dynamic_data_plane_scale_up: false, enable_dynamic_data_plane_scale_down: false, compute_machine_per_cloud_provider: {aws: {cluster_wide_workload: {compute_machine_type: m5.2xlarge, compute_node_autoscaling: {min_compute_nodes: 3, max_compute_nodes: 18}}, kafka_workload_per_instance_type: {standard: {compute_machine_type: r5.xlarge, compute_node_autoscaling: {min_compute_nodes: 3, max_compute_nodes: 18}}, developer: {compute_machine_type: m5.2xlarge, compute_node_autoscaling: {min_compute_nodes: 1, max_compute_nodes: 3}}}}, gcp: {cluster_wide_workload: {compute_machine_type: custom-8-32768, compute_node_autoscaling: {min_compute_nodes: 3, max_compute_nodes: 18}}, kafka_workload_per_instance_type: {standard: {compute_machine_type: custom-8-32768, compute_node_autoscaling: {min_compute_nodes: 3, max_compute_nodes: 18}}, developer: {compute_machine_type: custom-8-32768, compute_node_autoscaling: {min_compute_nodes: 1, max_compute_nodes: 3}}}}}}"`
- `NODE_PREWARMING_CONFIG`: The configuration file that contains information about each Kafka instance types, node prewarming configuration. Defaults to `"{}"`
- `ADMIN_AUTHZ_CONFIG`: Configuration file containing endpoints and roles mappings used to grant access to admin API endpoints, Defaults to`"[{method: GET, roles: [kas-fleet-manager-admin-full, kas-fleet-manager-admin-read, kas-fleet-manager-admin-write]}, {method: PATCH, roles: [kas-fleet-manager-admin-full, kas-fleet-manager-admin-write]}, {method: DELETE, roles: [kas-fleet-manager-admin-full]}]
"`
- `ADMIN_API_SSO_BASE_URL`: Base URL of admin API endpints SSO. Defaults to `"https://auth.redhat.com"`
- `ADMIN_API_SSO_ENDPOINT_URI`: admin API SSO endpoint URI. defaults to `"/auth/realms/EmployeeIDP"`
- `ADMIN_API_SSO_REALM`: admin API SSO realm. Defaults to `"EmployeeIDP"`
- `KAFKA_TLS_CERTIFICATE_MANAGEMENT_MUST_STAPLE`: The tls certificate management must staple. Adds the must staple TLS extension to the certificate signing request. The default value is `false`
- `KAFKA_TLS_CERTIFICATE_MANAGEMENT_STRATEGY`: The tls certificate management strategy. Possible options are manual and automatic. The default value is `manual`. In the `manual` mode, the user is expected to manually manage a wildcard certificate that will be applied to all the Kafkas. In `automatic` mode, kas-fleet-manager automatically handles the management of Kafka tls certificate.
- `KAFKA_TLS_CERTIFICATE_MANAGEMENT_STORAGE_TYPE`: The tls certificate management storage type. Available options are in-memory, file and vault. The default value is `vault`.
- `KAFKA_TLS_CERTIFICATE_MANAGEMENT_EMAIL`: The tls certificate management email. This is required when strategy is automatic
- `KAFKA_TLS_CERTIFICATE_MANAGEMENT_RENEWAL_WINDOW_RATIO`: The tls certificate management renewal window ratio i.e how much of a certificate's lifetime becomes the renewal window. The default value is `0.3333333333` - renew certificates a month before their expiry.
- `KAFKA_TLS_CERTIFICATE_MANAGEMENT_SECURE_STORAGE_CACHE_TTL` - the duration of the certificate in the in the secure storage cache. Past this duration, the certificate will be fetched from the remote secure storage. The dafault value is `10m`

### Using an Image from a Private External Registry
If you are using a private external registry, a docker pull secret must be created in the namespace where KAS Fleet Manager is deployed and linked to the service account that KAS Fleet Manager uses.

Create a docker pull secret with credentials that has access to pull the KAS Fleet Manager image from the private external registry.
```
oc create secret generic kas-fleet-manager-pull-secret \
  --from-file=.dockerconfigjson=<path-to-docker-config-json> \
  --type=kubernetes.io/dockerconfigjson
```

Link the pull secret to the KAS Fleet Manager service account
```
oc secrets link kas-fleet-manager kas-fleet-manager-pull-secret --for=pull
```

Delete the KAS Fleet Manager pod(s) to restart the deployment
```
oc get pods -n <namespace>
oc delete pod <kas-fleet-manager-pod>
```

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
