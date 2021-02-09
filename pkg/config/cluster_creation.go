package config

import (
	"github.com/spf13/pflag"
)

type ClusterCreationConfig struct {
	AutoOSDCreation             bool   `json:"auto_osd_creation"`
	OpenshiftVersion            string `json:"cluster_openshift_version"`
	ComputeMachineType          string `json:"cluster_compute_machine_type"`
	EnableKasFleetshardOperator bool   `json:"enable_kas_fleetshard_operator"`
	StrimziOperatorVersion      string `json:"strimzi_operator_version"`
}

func NewClusterCreationConfig() *ClusterCreationConfig {
	return &ClusterCreationConfig{
		AutoOSDCreation:             false,
		OpenshiftVersion:            "",
		ComputeMachineType:          "m5.4xlarge",
		EnableKasFleetshardOperator: false,
		StrimziOperatorVersion:      "v0.21.3",
	}
}

func (s *ClusterCreationConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&s.AutoOSDCreation, "auto-osd-creation", s.AutoOSDCreation, "Enable Auto Creation of supported OSD cluster")
	fs.StringVar(&s.OpenshiftVersion, "cluster-openshift-version", s.OpenshiftVersion, "The version of openshift installed on the cluster. An empty string indicates that the latest stable version should be used")
	fs.StringVar(&s.ComputeMachineType, "cluster-compute-machine-type", s.ComputeMachineType, "The compute machine type")
	fs.BoolVar(&s.EnableKasFleetshardOperator, "enable-kas-fleetshard-operator", s.EnableKasFleetshardOperator, "Enable kas-fleetshard-operator when creating an OSD cluster")
	fs.StringVar(&s.StrimziOperatorVersion, "strimzi-operator-version", s.StrimziOperatorVersion, "The version of the Strimzi operator to install")
}
