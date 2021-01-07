package constants

// ClusterOperation type
type ClusterOperation string

const (
	// ClusterOperationCreate - OpenShift/k8s cluster create operation
	ClusterOperationCreate ClusterOperation = "create"

	// ClusterNodeScaleIncrement - default increment/ decrement node count when scaling multiAZ clusters
	ClusterNodeScaleIncrement = 3

	// The DNS prefixes used for traffic ingress
	ManagedKafkaIngressDnsNamePrefix = "mk"
	DefaultIngressDnsNamePrefix      = "apps"
)

func (c ClusterOperation) String() string {
	return string(c)
}
