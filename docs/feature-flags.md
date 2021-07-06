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

- **allow_any_registered_users**: Allows any user registered against redhat.com access to the service.
    - `allow-list-config-file` [Required]: The `allow_any_registered_users` flag is declared within the allow list configuration file. This configuration is used to specify the path to this config file (default: `'config/allow-list-configuration.yaml'`, example: [allow-list-configuration.yaml](../config/allow-list-configuration.yaml)).
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
- **enable-deletion-of-expired-kafka**: Enables deletion of Kafka instances when its life span has expired.
    - `kafka-lifespan` [Optional]: The desired lifespan of a Kafka instance in hour(s) (default: `48`).
    - `long-lived-kafkas-config-file` [Optional]: The path to the file containing a list of Kafkas, specified by ID, that should not expire (default: `'config/long-lived-kafkas-configuration.yaml'`, example: [long-lived-kafkas-configuration.yaml](../config/long-lived-kafkas-configuration.yaml)).
- **enable-kafka-external-certificate**: Enables custom Kafka TLS certificate.
    - `kafka-tls-cert-file` [Required]: The path to the file containing the Kafka TLS certificate (default: `'secrets/kafka-tls.crt'`).
    - `kafka-tls-key-file` [Required]: The path to the file containing the Kafka TLS private key (default: `'secrets/kafka-tls.key'`).
- **quota-type**: Sets the quota service to be used for access control when requesting Kafka instances (options: `ams` or `allow-list`, default: `allow-list`).
    > For more information on the quota service implementation, see the [quota service architecture](./architecture/quota-service-implementation) architecture documentation.
    - If this is set to `allow-list`, quotas will be managed via the allow list configuration. 
        > See [access control](./access-control.md) documentation for more information about the allow list.
        - `enable-instance-limit-control` [Required]: Enables enforcement of limits on how much Kafka instances a user can create (default: `false`). 
        
            If enabled, the maximum instances a user can create can be specified in one of the following ways:
            - `allow-list-config-file` [Optional]: Allows setting of Kafka instance limit per organisation via _allowed_users_per_organisation_ or per service account via _allowed_service_accounts_ (default: `'config/allow-list-configuration.yaml'`, example: [allow-list-configuration.yaml](../config/allow-list-configuration.yaml)). 
            - `max-allowed-instances` [Optional]: The default maximum Kafka instance limit a user can create (default: `1`).

            > See the [max allowed instances](./access-control.md#max-allowed-instances) section for more information about setting Kafka instance limits for users.
    - If this is set to `ams`, quotas will be managed via OCM's accounts management service (AMS).
       - `product-type` [Required]: Sets the product type to be used to reserve quota in AMS (RHOSAK or RHOSAKTrial)

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
        - `cluster-compute-machine-type` [Optional]: The compute machine type to be used for provisioning a new dataplane cluster (default: `m5.4xlarge`).
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
- **enable-jwt**: Enables JSON Web Token (JWT) validation.
    - One or more of the following configuration must be specified:
        - `jwks-url`: The URL of the JWT signing certificate (default: `https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/certs`).
        - `jwks-file`: The path to the file containing the JWT signing certificates (default: `'config/jwks-file.json'`).
    - `mas-sso-base-url` [Required]: The base URL of the Keycloak instance to be used for retrieving the JWT signing certificate.
    - `mas-sso-realm` [Required]: The Keycloak realm to be used for retrieving the JWT signing certificate.
- **enable-terms-acceptance**: Enables terms acceptance verification.
