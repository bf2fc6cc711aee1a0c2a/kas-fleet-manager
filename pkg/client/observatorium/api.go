package observatorium

import (
	"fmt"
	"github.com/golang/glog"
	"strings"
)

type APIObservatoriumService interface {
	GetKafkaState(name string, namespaceName string) (KafkaState, error)
	GetMetrics(csMetrics *KafkaMetrics, resourceNamespace string, rq *MetricsReqParams) error
}
type fetcher struct {
	metric   string
	labels   string
	callback callback
}
type callback func(m Metric)

type ServiceObservatorium struct {
	client *Client
}

func (obs *ServiceObservatorium) GetKafkaState(name string, resourceNamespace string) (KafkaState, error) {
	KafkaState := KafkaState{}
	c := obs.client
	metric := `strimzi_resource_state{%s}`
	labels := fmt.Sprintf(`kind=~'Kafka', name=~'%s',resource_namespace=~'%s'`, name, resourceNamespace)
	result := c.Query(metric, labels)
	if result.Err != nil {
		return KafkaState, result.Err
	}

	for _, s := range result.Vector {

		if s.Value == 1 {
			KafkaState.State = ClusterStateReady
		} else {
			KafkaState.State = ClusterStateUnknown
		}
	}
	return KafkaState, nil
}

func (obs *ServiceObservatorium) GetMetrics(metrics *KafkaMetrics, namespace string, rq *MetricsReqParams) error {
	failedMetrics := []string{}
	fetchers := map[string]fetcher{
		//Check metrics for available disk space per broker
		"kubelet_volume_stats_available_bytes": {
			`kubelet_volume_stats_available_bytes{%s}`,
			fmt.Sprintf(`persistentvolumeclaim=~"data-.*-kafka-[0-9]*$", namespace=~'%s'`, namespace),
			func(m Metric) {
				*metrics = append(*metrics, m)
			},
		},
		//Check metrics for messages in per topic
		"kafka_server_brokertopicmetrics_messages_in_total": {
			`kafka_server_brokertopicmetrics_messages_in_total{%s}`,
			fmt.Sprintf(`strimzi_io_kind=~'Kafka', namespace=~'%s'`, namespace),
			func(m Metric) {
				*metrics = append(*metrics, m)
			},
		},
		//Check metrics for bytes in per topic
		"kafka_server_brokertopicmetrics_bytes_in_total": {
			`kafka_server_brokertopicmetrics_bytes_in_total{%s}`,
			fmt.Sprintf(`strimzi_io_kind=~'Kafka', namespace=~'%s'`, namespace),
			func(m Metric) {
				*metrics = append(*metrics, m)
			},
		},
		//Check metrics for bytes out per topic
		"kafka_server_brokertopicmetrics_bytes_out_total": {
			`kafka_server_brokertopicmetrics_bytes_out_total{%s}`,
			fmt.Sprintf(`strimzi_io_kind=~'Kafka', namespace=~'%s'`, namespace),
			func(m Metric) {
				*metrics = append(*metrics, m)
			},
		},
		//Check metrics for partition states
		"kafka_controller_kafkacontroller_offline_partitions_count": {
			`kafka_controller_kafkacontroller_offline_partitions_count{%s}`,
			fmt.Sprintf(`strimzi_io_kind=~'Kafka',namespace=~'%s'`, namespace),
			func(m Metric) {
				*metrics = append(*metrics, m)
			},
		},
		"kafka_controller_kafkacontroller_global_partition_count": {
			`kafka_controller_kafkacontroller_global_partition_count{%s}`,
			fmt.Sprintf(`strimzi_io_kind=~'Kafka',namespace=~'%s'`, namespace),
			func(m Metric) {
				*metrics = append(*metrics, m)
			},
		},
		//Check metrics for log size
		"kafka_log_log_size": {
			`sum by (namespace, topic)(kafka_log_log_size{%s})`,
			fmt.Sprintf(`strimzi_io_kind=~'Kafka',namespace=~'%s'`, namespace),
			func(m Metric) {
				*metrics = append(*metrics, m)
			},
		},
	}

	for msg, f := range fetchers {
		fetchAll := len(rq.Filters) == 0
		if fetchAll {
			result := obs.fetchMetricsResult(rq, &f)
			if result.Err != nil {
				glog.Error("error from metric ", result.Err)
				failedMetrics = append(failedMetrics, msg)
			}
			f.callback(result)
		}
		if !fetchAll {
			for _, filter := range rq.Filters {
				if filter == msg {
					result := obs.fetchMetricsResult(rq, &f)
					if result.Err != nil {
						glog.Error("error from metric ", result.Err)
						failedMetrics = append(failedMetrics, msg)
					}
					f.callback(result)
				}
			}

		}

	}
	if len(failedMetrics) > 0 {
		glog.Infof("Failed to fetch metrics data [%s]", strings.Join(failedMetrics, ","))
	}
	return nil
}

func (obs *ServiceObservatorium) fetchMetricsResult(rq *MetricsReqParams, f *fetcher) Metric {
	c := obs.client
	var result  Metric
	switch rq.ResultType {
	case RangeQuery:
		result = c.QueryRange(f.metric, f.labels, rq.Range)
	case Query:
		result = c.Query(f.metric, f.labels)
	default:
		result = Metric{Err: fmt.Errorf("Unsupported Result Type %q", rq.ResultType)}
	}
	return result
}
