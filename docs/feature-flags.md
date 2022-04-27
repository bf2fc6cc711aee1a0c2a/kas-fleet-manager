# Feature Flags
This lists the feature flags and their sub-configurations to enable/disable and configure features of the KAS Fleet Manager. This set of features can be seen below.

   - [Feature Flags](#feature-flags)
  - [Access Control](#access-control)
  - [Connectors](#connectors)
  - [Database](#database)
  - [Health Check Server](#health-check-server)
  - [Kafka](#kafka)
  - [Keycloak](#keycloak)
  - [Metrics Server](#metrics-server)
  - [Observability](#observability)
  - [OpenShift Cluster Manager](#openshift-cluster-manager)
  - [Dataplane Cluster Management](#dataplane-cluster-management)
  - [Sentry](#sentry)
  - [Server](#server)

## Access Control
> For more information on access control for KAS Fleet Manager, see this [documentation](./access-control.md).

- **enable-deny-list**: Enables access control for denied users.
    - `deny-list-config-file` [Required]: The path to the file containing the list of users that should be denied access to the service. (default: `'config/deny-list-configuration.yaml'`, example: [deny-list-configuration.yaml](../config/deny-list-configuration.yaml)).

## Connectors
- **enable-connectors**: Enables Kafka Connectors.
    - `mas-sso-base-url` [Required]: The base URL of the Keycloak instance to be used for authentication.
    - `mas-sso-realm` [Required]: The Keycloak realm to be used for authentication.
    - `connector-types` [Optional]: Directory containing connector type service URLs (default: `'config/connector-types'`).

## Database
- **enable-db-debug**: Enables Postgres debug logging.

## Health Check Server
- **enable-health-check-https**: Enable HTTPS for health check server.
    - `https-cert-file` [Required]: The path to the file containing the TLS certificate. 
    - `https-key-file` [Required]: The path to the file containing the TLS private key.

## Kafka
- **enable-deletion-of-expired-kafka**: Enables deletion of developer Kafka instances when its life span has expired.
    - `kafka-lifespan` [Optional]: The desired lifespan of a Kafka instance in hour(s) (default: `48`).
- **enable-kafka-external-certificate**: Enables custom Kafka TLS certificate.
    - `kafka-tls-cert-file` [Required]: The path to the file containing the Kafka TLS certificate (default: `'secrets/kafka-tls.crt'`).
    - `kafka-tls-key-file` [Required]: The path to the file containing the Kafka TLS private key (default: `'secrets/kafka-tls.key'`).
- **enable-developer-instance**: Enable the creation of one kafka developer instances per user    
- **quota-type**: Sets the quota service to be used for access control when requesting Kafka instances (options: `ams` or `quota-management-list`, default: `quota-management-list`).
    > For more information on the quota service implementation, see the [quota service architecture](./architecture/quota-service-implementation) architecture documentation.
    - If this is set to `quota-management-list`, quotas will be managed via the quota management list configuration. 
        > See [quota control](./quota-management-list-configuration.md) documentation for more information about the quota management list.
        - `enable-instance-limit-control` [Required]: Enables enforcement of limits on how much Kafka instances a user can create (default: `false`). 
        
            If enabled, the maximum instances a user can create can be specified in one of the following ways:
            - `quota-management-list-config-file` [Optional]: Allows setting of Kafka instance limit per organisation 
              via _registered_users_per_organisation_ or per service account via _registered_service_accounts_ 
              (default: `'config/quota-management-list-configuration.yaml'`, 
              example: [quota-management-list-configuration.yaml](../config/quota-management-list-configuration.yaml)). 
            - `max-allowed-instances` [Optional]: The default maximum Kafka instance limit a user can create (default: `1`).

            > See the [max allowed instances](./access-control.md#max-allowed-instances) section for more information about setting Kafka instance limits for users.
    - If this is set to `ams`, quotas will be managed via OCM's accounts management service (AMS).

## Keycloak
- **mas-sso-debug**: Enables Keycloak debug logging.
- **mas-sso-enable-auth**: Enables Kafka authentication via Keycloak.
    - `mas-sso-base-url` [Required]: The base URL of the Keycloak instance.
    - `mas-sso-cert-file` [Optional]: File containing tls cert for the mas-sso. Useful when mas-sso uses a self-signed certificate. If the provided file does not exist, is the empty string or the provided file content is empty then no custom MAS SSO certificate is used (default `secrets/keycloak-service.crt`).
    - `mas-sso-client-id-file` [Required]: The path to the file containing a Keycloak account client ID that has access to the Kafka service accounts realm (default: `'secrets/keycloak-service.clientId'`).
    - `mas-sso-client-secret-file` [Required]: The path to the file containing a Keycloak account client secret that has access to the Kafka service accounts realm (default: `'secrets/keycloak-service.clientSecret'`).
    - `mas-sso-realm` [Required]: The Keycloak realm to be used for the Kafka service accounts.
- **mas-sso-insecure**: Disables Keycloak TLS verification.

## Metrics Server
- **enable-metrics-https**: Enables HTTPS for the metrics server.
    - `https-cert-file` [Required]: The path to the file containing the TLS certificate. 
    - `https-key-file` [Required]: The path to the file containing the TLS private key.

## Observability
- **enable-observatorium-mock**: Enables use of a mock Observatorium client.
    - `observatorium-timeout` [Optional]: Timeout to be used for Observatorium requests (default: `240s`).
    - `observatorium-token-file` [Optional]: The path to the file containing a token for authenticating with Observatorium (default: `'secrets/observatorium.token'`).
- **observatorium-debug**: Enables Observatorium debug logging.
- **observatorium-ignore-ssl**: Disables Observatorium TLS verification.
- **observatorium-auth-type**[Optional]: This allows for the choice of either Red Hat SSO (`redhat`) or Dex
(`dex`) as the authentication medium for interaction between kas-fleet-manager and Observatorium (default: `dex`, options: `redhat` or `dex`).

### Dex Authentication
- The '[Required]' in the following denotes that these flags are required to use Dex Authentication with the service.
    - `dex-password-file`[Required]: The path to the file containing the Dex password for use with Dex
    authentication.
    - `dex-secret-file`[Required]: The path to the file containing the Dex secret for use with Dex
    authentication.
    - `dex-username`[Required]: The Dex username for authentication (default: `admin@example.com`)
    - `dex-url`[Required]: The Dex URL for authentication (default: `http://dex-dex.apps.pbraun-observatorium.observability.rhmw.io`).
    - `observatorium-gateway`[Required]: The Observatorium URL for use with dex authentication (default: `https://observatorium-observatorium.apps.pbraun-observatorium.observability.rhmw.io`).
    - `observatorium-tenant`[Required]: The Observatorium tenant name for use with dex authentication (default: `test`).

### Red Hat SSO Authentication
- The '[Required]' in the following denotes that these flags are required to use Red Hat SSO Authentication with the service.
    - `observability-red-hat-sso-auth-server-url`[Required]: Red Hat SSO authentication server URL (default: `https://sso.redhat.com/auth`).
    - `observability-red-hat-sso-realm`[Required]: Red Hat SSO realm (default: `redhat-external`).
    - `observability-red-hat-sso-token-refresher-url`[Required]: Red Hat SSO token refresher URL (default: `www.test.com`).
    - `observability-red-hat-sso-observatorium-gateway`[Required]: Red Hat SSO observatorium gateway (default: `https://observatorium-mst.api.stage.openshift.com`).
    - `observability-red-hat-sso-tenant`[Required]: Red Hat SSO tenant (default: `managedKafka`).
    - `observability-red-hat-sso-logs-client-id-file`[Required]: The path to the file containing the client
    ID for the logs service account for use with Red Hat SSO.
    - `observability-red-hat-sso-logs-secret-file`[Required]: The path to the file containing the client
    secret for the logs service account for use with Red Hat SSO.
    - `observability-red-hat-sso-metrics-client-id-file`[Required]: The path to the file containing the client
    ID for the metrics service account for use with Red Hat SSO.
    - `observability-red-hat-sso-metrics-secret-file`[Required]: The path to the file containing the client
    secret for the metrics service account for use with Red Hat SSO.

## OpenShift Cluster Manager
- **enable-ocm-mock**: Enables use of a mock OCM client.
    - `ocm-mock-mode` [Optional]: Sets the ocm client mock type (default: `stub-server`).
- **ocm-debug**: Enables OpenShift Cluster Manager (OCM) debug logging.

## Dataplane Cluster Management
- **enable-ready-dataplane-clusters-reconcile**: Enables reconciliation of data plane clusters in a `Ready` state.
- **kubeconfig**: A path to kubeconfig file used to communicate with standalone dataplane clusters.
- **dataplane-cluster-scaling-type**: Sets the behaviour of how the service manages and scales OSD clusters (options: `manual`, `auto` or `none`).
    > For more information on the different dataplane cluster scaling types and their behaviour, see the [dataplane osd cluster options](./data-plane-osd-cluster-options.md) documentation.
    
    - If this is set to `manual`, the following configuration must be specified:
        - `dataplane-cluster-config-file` [Required]: The path to the file that contains a list of data plane clusters and their details for the service to manage (default: `'config/dataplane-cluster-configuration.yaml'`, example: [dataplane-cluster-configuration.yaml](../config/dataplane-cluster-configuration.yaml)).
    - If this is set to `auto`, the following configurations can be specified:
        - `providers-config-file` [Required]: The path to the file containing a list of supported cloud providers that the service can provision dataplane clusters to (default: `'config/provider-configuration.yaml'`, example: [provider-configuration.yaml](../config/provider-configuration.yaml)).
        - `cluster-compute-machine-type` [Optional]: The compute machine type to be used for provisioning a new dataplane cluster (default: `m5.2xlarge`).
        - `cluster-openshift-version` [Optional]: The OpenShift version to be installed on the dataplane cluster (default: `""`, empty string indicates that the latest stable version will be used). 
- **cluster-logging-operator-addon-id**: Enables the Cluster Logging Operator addon with Cloud Watch and application level logs enabled. (default: `""`, An empty string indicates that the operator should not be installed).
- **strimzi-operator-cs-namespace**: Strimzi operator catalog source namespace.
- **strimzi-operator-index-image**: Strimzi operator index image name
- **strimzi-operator-namespace**: Strimzi operator namespace
- **strimzi-operator-package**: Strimzi operator package name
- **strimzi-operator-sub-channel**: Strimzi operator subscription channel
- **kas-fleetshard-operator-cs-namespace**: kas-fleetshard operator catalog source namespace
- **kas-fleetshard-operator-index-image**: kas-fleetshard operator index image name
- **kas-fleetshard-operator-namespace**: kas-fleetshard operator namespace
- **kas-fleetshard-operator-package**: kas-fleetshard operator package name
- **kas-fleetshard-operator-sub-channel**: kas-fleetshard operator subscription channel

## Sentry
- **enable-sentry**: Enables Sentry error reporting.
    - `sentry-key-file` [Required]: The path to the file containing the Sentry key (default: `'secrets/sentry.key'`).
    - `sentry-project` [Required]: The Sentry project ID.
    - `sentry-url` [Required]: The base URL of the Sentry instance.
    - `enable-sentry-debug` [Optional]: Enables Sentry debug logging (default: `false`).
    - `sentry-timeout` [Optional]: The timeout duration for requests to Sentry.

## Server
- **enable-https**: Enables HTTPS for the KAS Fleet Manager server.
    - `https-cert-file` [Required]: The path to the file containing the TLS certificate. 
    - `https-key-file` [Required]: The path to the file containing the TLS private key.
- **enable-terms-acceptance**: Enables terms acceptance verification.
