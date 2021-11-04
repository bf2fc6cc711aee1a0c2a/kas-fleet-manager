# Configuring Kafka /metrics/federate Endpoint as a Prometheus Scrape Target

The **/kafkas/{id}/metrics/federate** endpoint returns Kafka metrics in a Prometheus Text Format. This can be configured as a scrape target which allows you to easily collect your Kafka metrics and integrate them with your own metrics platform.

This document will guide you on how to set up a Prometheus scrape target to collect metrics of your Kafka instances in OpenShift Streams for Apache Kafka.

## Pre-requisites
- Prometheus instance.
- RHOSAK service account (create one [here](https://console.redhat.com/application-services/service-accounts)).

### Configuring via Prometheus CR
> The following steps are based on this [guide](https://github.com/prometheus-operator/prometheus-operator/blob/master/Documentation/additional-scrape-config.md#additional-scrape-configuration) from Prometheus.

1. Create a file called `kafka-federate.yaml` with the following content:
    ```
    - job_name: "kafka-federate"
      static_configs:
      - targets: ["api.openshift.com"]
      scheme: "https"
      metrics_path: "/api/kafkas_mgmt/v1/kafkas/<replace-this-with-your-kafka-id>/metrics/federate"
      oauth2:
        client_id: "<replace-this-with-your-service-account-client-id>"
        client_secret: "<replace-this-with-your-service-account-client-secret>"
        token_url: "https://identity.api.openshift.com/auth/realms/rhoas/protocol/openid-connect/token"
    ```
2. Create a secret which has the configuration specified in step 1.
    ```
    kubectl create secret generic additional-scrape-configs --from-file=kafka-federate.yaml --dry-run -o yaml | kubectl apply -f - -n <namespace>
    ```
3. Reference this secret in your Prometheus CR
    ```
    apiVersion: monitoring.coreos.com/v1
    kind: Prometheus
    metadata:
        ...
    spec:
        ...
        additionalScrapeConfigs:
            name: additional-scrape-configs
            key: kafka-federate.yaml
    ```
4. The scrape target should be available once the configuration has been reloaded.

### Configuring via a Configuration File

1. Add the following to your Prometheus configuration file (see this [documentation](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#configuration-file) for more information about the Prometheus configuration file)
    ```
    ...
    scrape_configs:
    - job_name: "kafka-federate"
      static_configs:
      - targets: ["api.openshift.com"]
      scheme: "https"
      metrics_path: "/api/kafkas_mgmt/v1/kafkas/<replace-this-with-your-kafka-id>/metrics/federate"
      oauth2:
        client_id: "<replace-this-with-your-service-account-client-id>"
        client_secret: "<replace-this-with-your-service-account-client-secret>"
        token_url: "https://identity.api.openshift.com/auth/realms/rhoas/protocol/openid-connect/token"
    ...
    ```
2. The scrape target should be available once the configuration has been reloaded.
