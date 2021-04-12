package ocm

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// ClusterNamePrefix a prefix used for new OCM cluster names
const (
	ClusterNamePrefix  = "mk-"
	ComputeMachineType = "m5.4xlarge"
)

// NOTE: the current mock generation exports to a _test file, if in the future this should be made public, consider
// moving the type into a ocmtest package.
//go:generate moq -out clusterbuilder_moq.go . ClusterBuilder
// ClusterBuilder wrapper for the OCM-specific builder struct, to allow for mocking.
type ClusterBuilder interface {
	// NewOCMClusterFromCluster create an OCM cluster definition that can be used to create a new cluster with the OCM
	// Cluster Service.
	NewOCMClusterFromCluster(cluster *api.Cluster) (*clustersmgmtv1.Cluster, error)
}

var _ ClusterBuilder = &clusterBuilder{}

// clusterBuilder internal ClusterBuilder implementation.
type clusterBuilder struct {
	// idGenerator generates cluster IDs.
	idGenerator IDGenerator

	// awsConfig contains aws credentials for use with the OCM cluster service.
	awsConfig *config.AWSConfig

	// osdClusterConfig contains cluster creation configuration.
	osdClusterConfig *config.OSDClusterConfig
}

// NewClusterBuilder create a new default implementation of ClusterBuilder.
func NewClusterBuilder(awsConfig *config.AWSConfig, osdClusterConfig *config.OSDClusterConfig) ClusterBuilder {
	return &clusterBuilder{
		idGenerator:      NewIDGenerator(ClusterNamePrefix),
		awsConfig:        awsConfig,
		osdClusterConfig: osdClusterConfig,
	}
}

func (r clusterBuilder) NewOCMClusterFromCluster(cluster *api.Cluster) (*clustersmgmtv1.Cluster, error) {
	// pre-req nil checks
	if err := r.validate(); err != nil {
		return nil, err
	}
	if cluster == nil {
		return nil, errors.New(errors.ErrorValidation, "cluster is not defined")
	}

	clusterBuilder := clustersmgmtv1.NewCluster()
	// the name of the cluster must start with a letter, use a standardised prefix to guarentee this.
	clusterBuilder.Name(r.idGenerator.Generate())
	clusterBuilder.CloudProvider(clustersmgmtv1.NewCloudProvider().ID(cluster.CloudProvider))
	clusterBuilder.Region(clustersmgmtv1.NewCloudRegion().ID(cluster.Region))
	// currently only enabled for MultiAZ.
	clusterBuilder.MultiAZ(true)
	if r.osdClusterConfig.OpenshiftVersion != "" {
		clusterBuilder.Version(clustersmgmtv1.NewVersion().ID(r.osdClusterConfig.OpenshiftVersion))
	}
	// setting CCS to always be true for now as this is the only available cluster type within our quota.
	clusterBuilder.CCS(clustersmgmtv1.NewCCS().Enabled(true))

	clusterBuilder.Managed(true)

	// AWS config read from the secrets/aws.* files
	awsBuilder := clustersmgmtv1.NewAWS().AccountID(r.awsConfig.AccountID).AccessKeyID(r.awsConfig.AccessKey).SecretAccessKey(r.awsConfig.SecretAccessKey)
	clusterBuilder.AWS(awsBuilder)

	// Set compute node size
	clusterBuilder.Nodes(clustersmgmtv1.NewClusterNodes().ComputeMachineType(clustersmgmtv1.NewMachineType().ID(r.osdClusterConfig.ComputeMachineType)))

	return clusterBuilder.Build()
}

// validate validate the state of the clusterBuilder struct.
func (r clusterBuilder) validate() *errors.ServiceError {
	if r.idGenerator == nil {
		return errors.New(errors.ErrorValidation, "idGenerator is not defined")
	}

	if r.awsConfig == nil {
		return errors.New(errors.ErrorValidation, "awsConfig is not defined")
	}

	if r.osdClusterConfig == nil {
		return errors.New(errors.ErrorValidation, "osdClusterConfig is not defined")
	}

	return nil
}
