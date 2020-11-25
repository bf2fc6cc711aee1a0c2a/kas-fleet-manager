package constants

// ClusterOperation type
type ClusterOperation string

const (
	// ClusterOperationCreate - OpenShift/k8s cluster create operation
	ClusterOperationCreate ClusterOperation = "create"
)

func (c ClusterOperation) String() string {
	return string(c)
}
