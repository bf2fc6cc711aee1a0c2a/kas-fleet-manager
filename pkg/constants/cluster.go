package constants

// ClusterOperation type
type ClusterOperation string

const (
	// ClusterOperationCreate - OpenShift/k8s cluster create operation
	ClusterOperationCreate ClusterOperation = "create"

	// The DNS prefixes used for traffic ingress
	ManagedKafkaIngressDnsNamePrefix = "mk"
	DefaultIngressDnsNamePrefix      = "apps"
)

func (c ClusterOperation) String() string {
	return string(c)
}
