package constants

// ClusterOperation type
type ClusterOperation string

const (
	// ClusterOperationCreate - OpenShift/k8s cluster create operation
	ClusterOperationCreate ClusterOperation = "create"

	// ClusterOperationDelete - OpenShift/k8s cluster delete operation
	ClusterOperationDelete ClusterOperation = "delete"

	// ClusterOperationHardDelete - OpenShift/k8s cluster hard delete operation
	ClusterOperationHardDelete ClusterOperation = "hard_delete"

	// The DNS prefixes used for traffic ingress
	ManagedKafkaIngressDnsNamePrefix = "kas"
	DefaultIngressDnsNamePrefix      = "apps"

	//ImagePullSecretName is the name of the secret used to pull images
	ImagePullSecretName = "rhoas-image-pull-secret"
)

func (c ClusterOperation) String() string {
	return string(c)
}
