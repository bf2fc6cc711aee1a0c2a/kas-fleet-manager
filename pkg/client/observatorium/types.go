package observatorium

type State string

const (
	ClusterStateUnknown State = "unknown"
	ClusterStateReady   State = "ready"
)

type KafkaState struct {
	State State `json:",omitempty"`
}
