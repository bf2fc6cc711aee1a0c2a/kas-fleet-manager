# Fleet Manager Metrics and Dashboards
This README will outline the adaptations and modifications that need to be made to utilise the fleet manager template metrics and dashboards.

## Template Metrics
The file [metrics.go](pkg/metrics/metrics.go) creates Prometheus metrics of differing types. These are the metrics which are then reported and visualised in each Grafana dashboard.
See [here](https://prometheus.io/docs/concepts/metric_types/) for more info about Prometheus metric types

These metrics are grouped by metric subject: data plane clusters, service ('pineapple' for this template) and reconcilers. These metrics need to be updated with service name (ie 'pineapple' replaced and service name included).

There is also a section of metrics regarding Observatorium API. These metrics are not reported to a Grafana dashboard by this template. See [kas-fleet-manager metrics configmap](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/resources/observability/grafana/grafana-dashboard-kas-fleet-manager-metrics.configmap.yaml#L5460-7247) for an example of a Grafana dashboard reporting these metrics.

The file [metrics_middleware.go](pkg/handlers/metrics_middleware.go) creates metrics concerned with incoming API requests. This file contains useful and important information about how these metrics are written and reported.

## Grafana Dashboards
See the [Getting Started](https://grafana.com/docs/grafana/latest/getting-started/?pg=docs) section of the Grafana website for general information about Grafana and its uses.

See the [JSON Model](https://grafana.com/docs/grafana/latest/dashboards/json-model/?pg=docs) section for more info about Grafana Dashboard JSON models

### Dashboard fields to be updated
In general `pineapple` has been used as placeholder throughout this template and should be replaced with the service being used.

There are three `config.yaml` files located in `observability` folder responsible for generating Grafana dashboards:
* `grafana-dashboard-fleet-manager-metrics.configmap.yaml`
* `grafana-dashboard-fleet-manager-stage-slos.configmap.yaml`
* `grafana-dashboard-fleet-manager-prod-slos.configmap.yaml`

Each file contains the JSON data required for the metrics dashboards. Each dashboard consists of panels. For each panel, within `"targets"` arrays, the `"expr"` fields need to be modified to represent the service being used with the fleet manager. Attention should be paid to labels such as `job`, `namespace`, `exported_namespace` within the `"expr"` field.
The `"expr"` fields contain Prometheus queries. The [Querying section](https://prometheus.io/docs/prometheus/latest/querying/basics/) of the Prometheus website has more information about queries.

The `"title"`, `"transformations"`, `"legendFormat"` fields may also need to be updated to include the service name. Be aware that each panel can have multiple sub-panels each needing to be adapted.

The dashboard `"uid"` needs to be updated also. See [here](https://grafana.com/docs/grafana/latest/http_api/dashboard/) for more info about dashboard uid.

At the end of each file, outside of the JSON data, the `name` and `grafana-folder` fields require an update.

## Further information
See [SLOs README](docs/slos/README.md) for more informtion about metrics and their use in measuring SLIs.

See [here](https://gitlab.cee.redhat.com/service/app-interface#add-a-grafana-dashboard) for information about adding Grafana dashboards in App-Sre