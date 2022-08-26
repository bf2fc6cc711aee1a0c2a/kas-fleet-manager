package constants

// ClusterOperation type
type ClusterOperation string

const (
	// ClusterOperationCreate - OpenShift/k8s cluster create operation
	ClusterOperationCreate ClusterOperation = "create"

	// ClusterOperationDelete - OpenShift/k8s cluster delete operation
	ClusterOperationDelete ClusterOperation = "delete"

	// The DNS prefixes used for traffic ingress
	ManagedKafkaIngressDnsNamePrefix = "kas"
	DefaultIngressDnsNamePrefix      = "apps"

	//ImagePullSecretName is the name of the secret used to pull images
	ImagePullSecretName = "rhoas-image-pull-secret"

	//MinNodesForDefaultMachinePool is the minimum number of worker nodes for the default machine pool
	MinNodesForDefaultMachinePool = 3

	//MaxNodesForDefaultMachinePool is the maximum number of worker nodes for the default machine pool
	MaxNodesForDefaultMachinePool = 18
)

func (c ClusterOperation) String() string {
	return string(c)
}
