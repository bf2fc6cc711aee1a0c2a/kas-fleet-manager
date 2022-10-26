package observatorium

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"strings"

	"github.com/golang/glog"

	"github.com/pkg/errors"
)

const privateTopicFilter string = "topic!~'__redhat_.*|__consumer_offsets|__transaction_state'"

type APIObservatoriumService interface {
	GetKafkaState(name string, namespaceName string) (KafkaState, error)
	GetMetrics(csMetrics *KafkaMetrics, resourceNamespace string, rq *MetricsReqParams) error
}
type fetcher struct {
	metric string
	labels string
}

// metricQueryResult holds a metric result for each metric query
type metricQueryResult struct {
	metricResult Metric
	metricQuery  string
}

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

// buildQueries takes a list of requested metrics and a list of filters and computes the minimum number of queries
// to run. We need to run one query per label selector, but multiple metrics can share the same label selector.
func (obs *ServiceObservatorium) buildQueries(fetchers []fetcher, rq *MetricsReqParams) []string {
	groupedQueries := map[string][]string{}
	var queries []string

	isMetricRequested := func(metric string) bool {
		// if no filters are provided assume that all metrics were requested
		if rq.Filters == nil {
			return true
		}
		return arrays.Contains(rq.Filters, metric)
	}

	// figure out unique label selectors
	for _, f := range fetchers {
		if isMetricRequested(f.metric) {
			groupedQueries[f.labels] = append(groupedQueries[f.labels], f.metric)
		}
	}

	// build one query per unique label selector
	for labels, metrics := range groupedQueries {
		queries = append(queries, fmt.Sprintf("{__name__=~'%v', %v}", strings.Join(metrics, "|"), labels))
	}

	return queries
}

func (obs *ServiceObservatorium) GetMetrics(metrics *KafkaMetrics, namespace string, rq *MetricsReqParams) error {
	fetchers := []fetcher{
		//Check metrics for available disk space per broker
		{
			`kubelet_volume_stats_available_bytes`,
			fmt.Sprintf(`persistentvolumeclaim=~"data-.*-kafka-[0-9]*$", namespace=~'%s'`, namespace),
		},
		//Check metrics for used disk space per broker
		{
			`kubelet_volume_stats_used_bytes`,
			fmt.Sprintf(`persistentvolumeclaim=~"data-.*-kafka-[0-9]*$", namespace=~'%s'`, namespace),
		},
		//Check metrics for soft limit quota for cluster
		{
			`kafka_broker_quota_softlimitbytes`,
			fmt.Sprintf(`strimzi_io_kind=~'Kafka', namespace=~'%s'`, namespace),
		},
		//Check metrics for hard limit quota for cluster
		{
			`kafka_broker_quota_hardlimitbytes`,
			fmt.Sprintf(`strimzi_io_kind=~'Kafka', namespace=~'%s'`, namespace),
		},
		//Check metrics for used space across the cluster
		{
			`kafka_broker_quota_totalstorageusedbytes`,
			fmt.Sprintf(`strimzi_io_kind=~'Kafka', namespace=~'%s'`, namespace),
		},
		//Check metrics for broker client quota limit
		{
			`kafka_broker_client_quota_limit`,
			fmt.Sprintf(`strimzi_io_kind=~'Kafka', namespace=~'%s'`, namespace),
		},
		//Check metrics for messages in per topic
		{
			`kafka_server_brokertopicmetrics_messages_in_total`,
			fmt.Sprintf(`strimzi_io_kind=~'Kafka', %s, namespace=~'%s'`, privateTopicFilter, namespace),
		},
		//Check metrics for bytes in per topic
		{
			`kafka_server_brokertopicmetrics_bytes_in_total`,
			fmt.Sprintf(`strimzi_io_kind=~'Kafka', %s, namespace=~'%s'`, privateTopicFilter, namespace),
		},
		//Check metrics for bytes out per topic
		{
			`kafka_server_brokertopicmetrics_bytes_out_total`,
			fmt.Sprintf(`strimzi_io_kind=~'Kafka', %s, namespace=~'%s'`, privateTopicFilter, namespace),
		},
		//Check metrics for partition states
		{
			`kafka_controller_kafkacontroller_offline_partitions_count`,
			fmt.Sprintf(`strimzi_io_kind=~'Kafka',namespace=~'%s'`, namespace),
		},
		{
			`kafka_controller_kafkacontroller_global_partition_count`,
			fmt.Sprintf(`strimzi_io_kind=~'Kafka',namespace=~'%s'`, namespace),
		},
		//Check metrics for log size
		{
			`kafka_topic:kafka_log_log_size:sum`,
			fmt.Sprintf(`%s, namespace=~'%s'`, privateTopicFilter, namespace),
		},
		//Check metrics for all traffic in/out
		{
			`kafka_namespace:haproxy_server_bytes_in_total:rate5m`,
			fmt.Sprintf(`exported_namespace=~'%s'`, namespace),
		},
		{
			`kafka_namespace:haproxy_server_bytes_out_total:rate5m`,
			fmt.Sprintf(`exported_namespace=~'%s'`, namespace),
		},
		{
			`kafka_topic:kafka_topic_partitions:sum`,
			fmt.Sprintf(`namespace=~'%s'`, namespace),
		},
		{
			`kafka_topic:kafka_topic_partitions:count`,
			fmt.Sprintf(`namespace=~'%s'`, namespace),
		},
		{
			`consumergroup:kafka_consumergroup_members:count`,
			fmt.Sprintf(`namespace=~'%s'`, namespace),
		},
		{
			`kafka_namespace:kafka_server_socket_server_metrics_connection_count:sum`,
			fmt.Sprintf(`namespace=~'%s'`, namespace),
		},
		{
			`kafka_namespace:kafka_server_socket_server_metrics_connection_creation_rate:sum`,
			fmt.Sprintf(`namespace=~'%s'`, namespace),
		},
		{
			`kafka_topic:kafka_server_brokertopicmetrics_messages_in_total:rate5m`,
			fmt.Sprintf(`namespace=~'%s'`, namespace),
		},
		{
			`kafka_topic:kafka_server_brokertopicmetrics_bytes_in_total:rate5m`,
			fmt.Sprintf(`namespace=~'%s'`, namespace),
		},
		{
			`kafka_topic:kafka_server_brokertopicmetrics_bytes_out_total:rate5m`,
			fmt.Sprintf(`namespace=~'%s'`, namespace),
		},
		{
			`kafka_instance_spec_brokers_desired_count`,
			fmt.Sprintf(`namespace=~'%s'`, namespace),
		},
		{
			`kafka_instance_max_message_size_limit`,
			fmt.Sprintf(`namespace=~'%s'`, namespace),
		},
		{
			`kafka_instance_partition_limit`,
			fmt.Sprintf(`namespace=~'%s'`, namespace),
		},
		{
			`kafka_instance_connection_limit`,
			fmt.Sprintf(`namespace=~'%s'`, namespace),
		},
		{
			`kafka_instance_connection_creation_rate_limit`,
			fmt.Sprintf(`namespace=~'%s'`, namespace),
		},
	}

	// build query per label to reduce query count
	queries := obs.buildQueries(fetchers, rq)

	// distribute metrics queries to observatorium
	resultChan := make(chan metricQueryResult)
	for _, query := range queries {
		go func(query string) {
			metricResult := obs.fetchMetricsResult(query, rq)
			resultChan <- metricQueryResult{
				metricResult: metricResult,
				metricQuery:  query,
			}
		}(query)
	}

	// process the metrics result
	var failedMetrics []string
	for i := 1; i <= len(queries); i++ {
		metricQueryResult := <-resultChan
		if metricQueryResult.metricResult.Err != nil {
			message := fmt.Sprintf("%q:%v", metricQueryResult.metricQuery, metricQueryResult.metricResult.Err)
			failedMetrics = append(failedMetrics, message)
			glog.Errorf("error running query %q: %v", metricQueryResult.metricQuery, metricQueryResult.metricResult.Err)
			continue
		}

		*metrics = append(*metrics, metricQueryResult.metricResult)
	}

	if len(failedMetrics) > 0 {
		return errors.New(fmt.Sprintf("Failed to fetch metrics data [%s]", strings.Join(failedMetrics, ",")))
	}

	return nil
}

func (obs *ServiceObservatorium) fetchMetricsResult(query string, rq *MetricsReqParams) Metric {
	c := obs.client
	var result Metric
	switch rq.ResultType {
	case RangeQuery:
		result = c.QueryRawRange(query, rq.Range)
	case Query:
		result = c.QueryRaw(query)
	default:
		result = Metric{Err: errors.Errorf("Unsupported Result Type %q", rq.ResultType)}
	}
	return result
}
