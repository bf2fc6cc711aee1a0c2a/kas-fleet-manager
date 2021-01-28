package observatorium

import (
	prom_v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	pModel "github.com/prometheus/common/model"
	"time"
)

type State string

const (
	ClusterStateUnknown State = "unknown"
	ClusterStateReady   State = "ready"
)

type KafkaState struct {
	State State `json:",omitempty"`
}

type KafkaMetrics []Metric

// Metric holds the Prometheus Matrix model, which contains one or more time series (depending on grouping)
type Metric struct {
	Matrix pModel.Matrix `json:"matrix"`
	Err    error         `json:"-"`
}

// RangeQuery holds common parameters for all kinds of range queries
type RangeQuery struct {
	Filters []string
	prom_v1.Range
	RateInterval string
}

// FillDefaults fills the struct with default parameters
func (q *RangeQuery) FillDefaults() {
	q.End = time.Now()
	q.Start = q.End.Add(-5 * time.Minute)
	q.Step = 30 * time.Second

}
