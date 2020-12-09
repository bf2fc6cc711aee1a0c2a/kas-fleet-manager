package observatorium

import (
	"fmt"
)

type APIObservatoriumService interface {
	GetKafkaState(name string, namespaceName string) (KafkaState, error)
}

type ServiceObservatorium struct {
	client *Client
}

func (obs *ServiceObservatorium) GetKafkaState(name string, resourceNamespace string) (KafkaState, error) {
	KafkaState := KafkaState{}
	c := obs.client
	metric := `strimzi_resource_state{%s}`
	labels := fmt.Sprintf(`kind=~'Kafka', name=~'%s',resource_namespace=~'%s'`, name, resourceNamespace)
	vec, err := c.Query(metric, labels)
	if err != nil {
		return KafkaState, err
	}

	for _, s := range *vec {
		if s.Value == 1 {
			KafkaState.State = ClusterStateReady
		} else {
			KafkaState.State = ClusterStateUnknown
		}
	}
	return KafkaState, nil
}
