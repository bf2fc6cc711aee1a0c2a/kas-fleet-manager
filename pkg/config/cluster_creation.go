package config

import (
	"github.com/spf13/pflag"
)

type ClusterCreationConfig struct {
	AutoOSDCreation    bool   `json:"auto_osd_creation"`
	OpenshiftVersion   string `json:"cluster_openshift_version"`
	ComputeMachineType string `json:"cluster_compute_machine_type"`
}

func NewClusterCreationConfig() *ClusterCreationConfig {
	return &ClusterCreationConfig{
		AutoOSDCreation:    false,
		OpenshiftVersion:   "",
		ComputeMachineType: "m5.4xlarge",
	}
}

func (s *ClusterCreationConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&s.AutoOSDCreation, "auto-osd-creation", s.AutoOSDCreation, "Enable Auto Creation of supported OSD cluster")
	fs.StringVar(&s.OpenshiftVersion, "cluster-openshift-version", s.OpenshiftVersion, "The version of openshift installed on the cluster. An empty string indicates that the latest stable version should be used")
	fs.StringVar(&s.ComputeMachineType, "cluster-compute-machine-type", s.ComputeMachineType, "The compute machine type")
}
