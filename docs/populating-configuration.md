> NOTE: some step in this document might refer to Red Hat internal components
  that you do not have access to

# Populating Configuration
The document describes how to prepare Fleet Manager to be able to start by
populating its configurations.

For convenience, the following Makefile target is provided:

```
make secrets/setup/empty
```

This target initializes empty files that are necessary for the service to boot.
It can be used to quickly get the service running locally, although it does not
provide a functional kas-fleet-manager.

Follow the subsections to find information about how to configure these files properly.

## Interacting with the Fleet Manager API

See the [Interacting with Fleet Manager](interacting-with-fleet-manager.md)
document.

## Setting up OCM authentication

The Fleet Manager itself requires authentication against OCM so it can interact
with it to perform management of Data Plane clusters.

In order for the Fleet Manager to be able to start, create the following files:
```
touch secrets/ocm-service.clientId
touch secrets/ocm-service.clientSecret
touch secrets/ocm-service.token
```

Fleet Manager supports authentication against OCM in two different ways:
* Using an OCM Service Account
  * To configure Fleet Manager with this authentication mechanism run:
    ```
    echo -n "<your-service-account-client-id>" > secrets/ocm-service.clientId
    echo -n "<your-service-account-client-secret>" secrets/ocm-service.clientSecret
    ```
* Using an OCM offline token
  * To configure Fleet Manager with this authentication mechanism run:
    ```
    make ocm/setup OCM_OFFLINE_TOKEN=<your-retrieved-ocm-offline-token>
    ```

> NOTE: If both OCM Service Account and OCM offline token credentials
  are provided the former takes precedence over the latter

> NOTE: Information about how to retrieve an OCM offline token can be found
  in the [Interacting with Fleet Manager API](interacting-with-fleet-manager.md)
  document

## Allowing creation of *Standard* Kafka instances

Fleet Manager is able to create two types of Kafka instances:
* Developer instances
  * Instances of this type are automatically deleted after 48 hours by default
    > NOTE: This can be controlled by setting the `lifespanSeconds` attribute
      in the `config/kafka-instance-types-configuration.yaml` configuration file
      for developer instances
  * All authenticated users that don't have AMS quotas defined to create other
    kafka instance types can request the creation of a Kafka developer instance
  * There is a limit of one instance per user by default
    > NOTE: This can be controlled by setting the `--max-allowed-developer-instances`
      Fleet Manager binary CLI flag
* Standard instances
  * Instances of this type are not automatically deleted
  * In order to be able to create an instance of this type, the user that
    is creating it must have enough quota

If you are not interested in making use of Standard Kafka instances you
can skip this section. Otherwise, keep reading below.

As commented above, in order to be able to create an instance of this type, the
user must have enough quota to do it. There are currently two ways to define
quotas for users in Fleet Manager:
* Through a Quota Management List configuration file. This is the default
  method used. Follow the the [Quota Management List Configurations](quota-management-list-configuration.md)
  guide for more detail on how to configure it
* By leveraging Red Hat's Account Management Service (AMS). For more information
  about this method, look at the [Quota Management with Account Management Service (AMS) SKU](getting-credentials-and-accounts.md#quota-management-with-account-management-service-ams-sku)

To select the type of quota to be used by Fleet Manager set the `--quota-type`
which accepts either `ams` or `quota-management-list`Fleet Manager binary CLI
flag.

## Setup Fleet Manager AWS configuration
Fleet Manager interacts with AWS to provide the following functionalities:
* To be able to create and manage Data Plane clusters in a specific AWS account
  by passing the needed credentials to OpenShift Cluster Management
* To create [AWS's Route53](https://aws.amazon.com/route53/) DNS records in a
  specific AWS account. This records are DNS records that point to some
  routes related to Kafka instances that are created.
  > NOTE: The domain name used for this records can be configured by setting
    the domain name to be used for Kafka instances. This cane be done
    through the `--kafka-domain-name` Fleet Manager binary CLI flag
For both functionalities, the same underlying AWS account is used.

In order for the Fleet Manager to be able to start, create the following files:
```
touch secrets/aws.accountid
touch secrets/aws.accesskey
touch secrets/aws.secretaccesskey
touch secrets/aws.route53accesskey
touch secrets/aws.route53secretaccesskey
```

If you need any of those functionalities keep reading. Otherwise, this section
can be skipped.

To accomplish the previously mentioned functionalities Fleet Manager needs to
be configured to interact with the AWS account. To do so, provide existing AWS
IAM user credentials to the control plane by running:
```
AWS_ACCOUNT_ID=<aws-account-id> \
AWS_ACCESS_KEY=<aws-iam-user-access-key> \
AWS_SECRET_ACCESS_KEY=<aws-iam-user-secret-access-key> \
ROUTE53_ACCESS_KEY=<aws-iam-user-for-route-53-access-key> \
ROUTE53_SECRET_ACCESS_KEY=<aws-iam-user-for-route-53-secret-access-key> \
make aws/setup
```
> NOTE: If you are in Red Hat, the following [documentation](./getting-credentials-and-accounts.md#aws)
  might be useful to get the IAM user/s credentials

## Setup Fleet Manager GCP configuration
Fleet Manager interacts with GCP to provide the following functionalities:
* To be able to create and manage Data Plane clusters in a specific GCP account
  by passing the needed credentials to OpenShift Cluster Management

If you need any of those functionalities keep reading. Otherwise, this section
can be skipped.

If you intend to configure/provision Data Planes in GCP then
[GCP Service Account](https://cloud.google.com/iam/docs/service-accounts)
JSON credentials need to be provided to Fleet Manager so it can deploy and
provision Data Plane Clusters there.
To create a GCP Service Account and its corresponding JSON credentials see:
  * [Creating GCP Service Accounts](https://cloud.google.com/iam/docs/creating-managing-service-accounts#creating)
  * [Creating GCP Service Account Keys](https://cloud.google.com/iam/docs/creating-managing-service-account-keys#creating)

  Additionally, the GCP Service Account has to meet the following
  requirements:

  * In case the Data Plane Clusters are to be provisioned through OCM then
    the GCP Service Account has to be named `osd-ccs-admin`
  * In case the Data Plane Clusters are to be provisioned through OCM then
    the following GCP IAM Roles have to be granted to the GCP Service Account:
    [Required GCP IAM roles](https://docs.openshift.com/container-platform/latest/installing/installing_gcp/installing-gcp-account.html#installation-gcp-permissions_installing-gcp-account).
    See [Manage Service Account access](https://cloud.google.com/iam/docs/manage-access-service-accounts#single-role)
    for details on how to do it

In order to configure GCP Service Account JSON credentials for Fleet Manager
retrieve them from GCP and configure them. To do so the following
alternatives are available:
* Copy the contents of the JSON credentials into the
  `secrets/gcp.api-credentials` file
* Run the  `gcp/setup/credentials` Makefile target providing the JSON
  credentials content as a base64-encoded string in the `GCP_API_CREDENTIALS`
  environment variable. To do so run:
  ```bash
  GCP_API_CREDENTIALS="<base64-encoded-gcp-serviceaccount-credentials>" make gcp/setup/credentials
  ```

Finally, make sure that `gcp` is listed as a supported Cloud Provider with at
least one configured [GCP region](https://cloud.google.com/compute/docs/regions-zones)
in Fleet Manager in the `config/provider-configuration.yaml` configuration
file. See the documentation in that file for detail on the configuration
schema.

## Setup additional SSO configuration
A SSO server is needed in Fleet Manager to enable some functionalities:
* To enable communication between the Fleetshard operator (in the data plane) and the Fleet Manager
* To configure the OpenShift cluster identity provider in Data Plane clusters
  to give SRE access to them
* To create a SSO Service Account for each Kafka cluster provisioned by the
  Fleet Manager. This SSO Service Account is then used by the Canary deployment
  of the provisioned Kafka cluster in the Data Plane

This additional SSO server must be based on Keycloak.

In order for the Fleet Manager to be able to start, create the following files:
```
touch secrets/keycloak-service.clientId
touch secrets/keycloak-service.clientSecret
touch secrets/osd-idp-keycloak-service.clientId
touch secrets/osd-idp-keycloak-service.clientSecret
```

If you need any of those functionalities keep reading. Otherwise, this section
can be skipped.

Three alternatives are offered here to use as the SSO Keycloak server:
* Use Red Hat SSO
* Use MAS-SSO
  >NOTE: MAS-SSO is soon going to be deprecated. Reach out to the MAS-Security team for info regarding this.
* Run your own Keycloak server instance by following their [getting-started guide](https://www.keycloak.org/getting-started/getting-started-docker). Please note, how to setup Keycloak is out of scope of this guide

To accomplish the previously mentioned functionalities Fleet Manager needs to
be configured to interact with this additional SSO server.

The SSO type to be used can be configured by setting the `--sso-provider-type` CLI flag when running
Fleet Manager. The accepted values for it are `mas_sso` (keycloak is included here) or `redhat`.

Create/Have available two client-id/client-secret pair of credentials (called Keycloak Clients)
in the SSO Keycloak server to be used, one for each previously mentioned
functionality, and then set Fleet Manager to use them by running
the following command:
```
 SSO_CLIENT_ID="<sso-client-id>" \
 SSO_CLIENT_SECRET="<sso-client-secret>" \
 OSD_IDP_SSO_CLIENT_ID="<osd-idp-sso-client-id>" \
 OSD_IDP_SSO_CLIENT_SECRET="<osd-idp-sso-client-secret>" \
 make keycloak/setup
```

Additionally, if `redhat` is the configured SSO provider type then run
the following to configure the credentials:
```
SSO_CLIENT_ID=<sso-client-id> \
SSO_CLIENT_SECRET=<sso-client-secret> \
make redhatsso/setup
```

Additionally, make sure you start the Fleet Manager server with the appropriate
Keycloak SSO Realms and URL for this. See the
[Fleet Manager CLI feature flags](./feature-flags.md#keycloak) for details on it.

## Setup the data plane image pull secret
In the Data Plane cluster, the Strimzi Operator and the FleetShard Deployments
might reference container images that are located in authenticated container
image registries.

Fleet Manager can be configured to send this authenticated
container image registry information as a K8s Secret in [`kubernetes.io/.dockerconfigjson` format](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/#registry-secret-existing-credentials).

In order for the Fleet Manager to be able to start, create the following file:
```
touch secrets/image-pull.dockerconfigjson
```

If you don't need to make use of this functionality you can skip this section.
Otherwise, keep reading below.

To configure the Fleet Manager with this authenticated registry information so
the previously mentioned Data Plane elements can pull container images from it:
* Base-64 encode your [Docker configuration file](https://docs.docker.com/engine/reference/commandline/cli/#docker-cli-configuration-file-configjson-properties).
* Copy the contents generated from the previous point into the `secrets/image-pull.dockerconfigjson` file

## Setup the Observability stack secrets
Fleet Manager provides the ability to provide and configure monitoring metrics
related to the Data Planes and its Kafka instances. To do so it makes
use of what's called the [Observability Stack](./observability/README.md#observability-stack)

To configure and make use of the Observability stack to offer monitoring metrics
related to the Data Planes and its Kafka instances Fleet Manager needs to:
* Access the Observability Stack's Observatorium service to retrieve and then
  offer the metrics as one of the Fleet Manager API endpoints
* To send the credentials/information needed to setup the Observability Stack
  when it is installed in the Data Plane clusters

In order for the Fleet Manager to be able to start, create the following files:
```
	touch secrets/rhsso-logs.clientId
	touch secrets/rhsso-logs.clientSecret
	touch secrets/rhsso-metrics.clientId
	touch secrets/rhsso-metrics.clientSecret
	touch secrets/observability-config-access.token
```

If you are not interested in making use of this functionality you can skip
this section. Otherwise, keep reading below.

The following command is used to setup the various secrets needed by
the Observability stack:
```
OBSERVATORIUM_CONFIG_ACCESS_TOKEN="<observatorium-config-access-token> \
RHSSO_LOGS_CLIENT_ID=<rhsso-logs-client-id> \
RHSSO_LOGS_CLIENT_SECRET=<rhsso-logs-client-secret> \
RHSSO_METRICS_CLIENT_ID=<rhsso-metrics-client-id> \
RHSSO_METRICS_CLIENT_SECRET=<rhsso-metrics-client-secret> \
make observatorium/setup
```

The description of the previously shown parameters are:
* OBSERVATORIUM_CONFIG_ACCESS_TOKEN: A GitHub token required to fetch the configuration from a private git repository. More details can be found [here](https://github.com/redhat-developer/observability-operator#adding-requisite-resources) in the `External config repo secret` bullet point.
* RHSSO_LOGS_CLIENT_ID, RHSSO_LOGS_CLIENT_SECRET: An OAuth's client-id/client-secret credentials pair to push logs to Observatorium. Used to authenticate via `sso.redhat.com` when logging via `config.promtail.enabled` is set in the observability configuration. Provided by the Observatorium team.
* RHSSO_METRICS_CLIENT_ID, RHSSO_METRICS_CLIENT_SECRET: An OAuth's client-id/client-secret credentials pair to push metrics to Observatorium. Used to authenticate via `sso.redhat.com` when `config.prometheus.observatorium` is set in the observability configuration. Provided by the Observatorium team.

An Observatorium token refresher is needed to refresh tokens when
authenticating against Red Hat SSO. To configure the token Refresher
in your local environment run the following command:
```
make observatorium/token-refresher/setup CLIENT_ID=<client-id> CLIENT_SECRET=<client-secret>
```

The description of the previously shown parameters are:
* CLIENT_ID: The client id of a service account that has, at least, permissions
  to read metrics
* ClIENT_SECRET: The client secret of a service account that has, at least,
  permissions to read metrics

Additionally, the following parameters are also supported on the previous command:
* PORT: Port for running the token refresher on.
  Defaults to `8085`
* IMAGE_TAG: Image tag of the [token-refresher image](https://quay.io/repository/rhoas/mk-token-refresher?tab=tags).
  Defaults to `latest`
* ISSUER_URL: URL of your auth issuer.
  Defaults to `https://sso.redhat.com/auth/realms/redhat-external`
* OBSERVATORIUM_URL: URL of your Observatorium instance.
  Defaults to `https://observatorium-mst.api.stage.openshift.com/api/metrics/v1/managedkafka`

Finally, if log delivery from Observability in the Data Plane to AWS CloudWatch
logs is desired setting the Observability CloudWatch Logs configuration is needed.
This can be done by running the following command:
```
OBSERVABILITY_CLOUDWATCHLOGS_CONFIG="<config>" make observability/cloudwatchlogs/setup
```
Information about the Observability Cloudwatch Logging configuration file can be found in the `secrets/observability-cloudwatchlogs-config.yaml.sample` file.
To enable Observability AWS CloudWatch logs delivery run KAS FLeet Manager
with the `--observability-enable-cloudwatchlogging` flag.

See [Observability](./observability/README.md) to learn more about
Observatorium and the Observability Stack.

## Setup a custom TLS certificate for Kafka Host URLs

When Fleet Manager creates Kafka instances, it can be configured to
send a custom TLS certificate to associate to each one of the Kafka instances
host URLs. That custom TLS certificate is sent to the data plane clusters where
those instances are located.

In order for the Fleet Manager to be able to start, create the following files:
```
touch secrets/kafka-tls.crt
touch secrets/kafka-tls.key
```

If you need to setup a custom TLS certificate for the Kafka instances' host
URLs keep reading. Otherwise, this section can be skipped.

To configure Fleet Manager so it sends the custom TLS certificate, provide the
certificate and its corresponding key to the Fleet Manager by running the
following command:
```
KAFKA_TLS_CERT=<kafka-tls-cert> \
KAFKA_TLS_KEY=<kafka-tls-key> \
make kafkacert/setup
```
> NOTE: The certificate domain/s should match the URL endpoint domain if you
  want the certificate to be valid when accessing the endpoint
> NOTE: The expected Certificate and Key values are in PEM format, preserving
  the newlines

Additionally, make sure that the functionality is enabled by setting the
`--enable-kafka-external-certificate` Fleet Manager binary CLI flag

## Configure Sentry logging
Fleet Manager can be configured to send its logs to the
[Sentry](https://sentry.io/) logging service.

In order for the Fleet Manager to be able to start, create the following files:
```
touch secrets/sentry.key
```

If you want to use Sentry set the Sentry Secret key in the `secrets/sentry.key`
previously created.

Additionally, make sure to set the Sentry URL endpoint and Sentry project when
starting the Fleet Manager server. See [Sentry-related CLI flags in Fleet Manager](./feature-flags.md#sentry)
