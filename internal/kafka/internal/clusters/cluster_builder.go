package clusters

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"
)

// ClusterNamePrefix a prefix used for new OCM cluster names
const (
	ClusterNamePrefix = "mk-"
)

// NOTE: the current mock generation exports to a _test file, if in the future this should be made public, consider
// moving the type into a ocmtest package.
// ClusterBuilder wrapper for the OCM-specific builder struct, to allow for mocking.
//
//go:generate moq -out clusterbuilder_moq.go . ClusterBuilder
type ClusterBuilder interface {
	// NewOCMClusterFromCluster create an OCM cluster definition that can be used to create a new cluster with the OCM
	// Cluster Service.
	NewOCMClusterFromCluster(clusterRequest *types.ClusterRequest) (*clustersmgmtv1.Cluster, error)
}

var _ ClusterBuilder = &clusterBuilder{}

// clusterBuilder internal ClusterBuilder implementation.
type clusterBuilder struct {
	// idGenerator generates cluster IDs.
	idGenerator ocm.IDGenerator

	// awsConfig contains aws credentials for use with the OCM cluster service.
	awsConfig *config.AWSConfig

	// gcpConfig contains GCP credentials for use with the OCM cluster service
	gcpConfig *config.GCPConfig

	// dataplaneClusterConfig contains cluster creation configuration.
	dataplaneClusterConfig *config.DataplaneClusterConfig
}

// NewClusterBuilder create a new default implementation of ClusterBuilder.
func NewClusterBuilder(awsConfig *config.AWSConfig, gcpConfig *config.GCPConfig, dataplaneClusterConfig *config.DataplaneClusterConfig) ClusterBuilder {
	return &clusterBuilder{
		idGenerator:            ocm.NewIDGenerator(ClusterNamePrefix),
		awsConfig:              awsConfig,
		gcpConfig:              gcpConfig,
		dataplaneClusterConfig: dataplaneClusterConfig,
	}
}

func (r clusterBuilder) NewOCMClusterFromCluster(clusterRequest *types.ClusterRequest) (*clustersmgmtv1.Cluster, error) {
	if clusterRequest == nil {
		return nil, errors.Errorf("cluster request is nil")
	}
	// pre-req nil checks
	if err := r.validate(clusterRequest); err != nil {
		return nil, err
	}

	clusterBuilder := clustersmgmtv1.NewCluster()
	// the name of the cluster must start with a letter, use a standardised prefix to guarentee this.
	clusterBuilder.Name(r.idGenerator.Generate())
	clusterBuilder.CloudProvider(clustersmgmtv1.NewCloudProvider().ID(clusterRequest.CloudProvider))
	clusterBuilder.Region(clustersmgmtv1.NewCloudRegion().ID(clusterRequest.Region))
	clusterBuilder.MultiAZ(clusterRequest.MultiAZ)
	if r.dataplaneClusterConfig.OpenshiftVersion != "" {
		clusterBuilder.Version(clustersmgmtv1.NewVersion().ID(r.dataplaneClusterConfig.OpenshiftVersion))
	}
	// setting CCS to always be true for now as this is the only available cluster type within our quota.
	clusterBuilder.CCS(clustersmgmtv1.NewCCS().Enabled(true))

	clusterBuilder.Managed(true)

	cloudProviderID := cloudproviders.ParseCloudProviderID(clusterRequest.CloudProvider)
	r.setCloudProviderBuilder(cloudProviderID, clusterBuilder)

	computeMachineType := r.dataplaneClusterConfig.DefaultComputeMachineType(cloudProviderID)
	if computeMachineType == "" {
		return nil, fmt.Errorf("Cloud provider %q is not a recognized cloud provider", clusterRequest.CloudProvider)
	}

	clusterBuilder.Nodes(clustersmgmtv1.NewClusterNodes().
		ComputeMachineType(clustersmgmtv1.NewMachineType().ID(computeMachineType)).
		AutoscaleCompute(clustersmgmtv1.NewMachinePoolAutoscaling().MinReplicas(constants.MinNodesForDefaultMachinePool).MaxReplicas(constants.MaxNodesForDefaultMachinePool)))

	return clusterBuilder.Build()
}

func (r clusterBuilder) setCloudProviderBuilder(cloudProviderID cloudproviders.CloudProviderID, ocmClusterBuilder *clustersmgmtv1.ClusterBuilder) {
	switch cloudProviderID {
	case cloudproviders.AWS:
		awsBuilder := clustersmgmtv1.NewAWS().AccountID(r.awsConfig.AccountID).AccessKeyID(r.awsConfig.AccessKey).SecretAccessKey(r.awsConfig.SecretAccessKey)
		ocmClusterBuilder.AWS(awsBuilder)
	case cloudproviders.GCP:
		gcpBuilder := clustersmgmtv1.NewGCP()
		gcpBuilder.AuthProviderX509CertURL(r.gcpConfig.GCPCredentials.AuthProviderX509CertURL)
		gcpBuilder.AuthURI(r.gcpConfig.GCPCredentials.AuthURI)
		gcpBuilder.ClientEmail(r.gcpConfig.GCPCredentials.ClientEmail)
		gcpBuilder.ClientID(r.gcpConfig.GCPCredentials.ClientID)
		gcpBuilder.ClientX509CertURL(r.gcpConfig.GCPCredentials.ClientX509CertURL)
		gcpBuilder.PrivateKey(r.gcpConfig.GCPCredentials.PrivateKey)
		gcpBuilder.PrivateKeyID(r.gcpConfig.GCPCredentials.PrivateKeyID)
		gcpBuilder.ProjectID(r.gcpConfig.GCPCredentials.ProjectID)
		gcpBuilder.TokenURI(r.gcpConfig.GCPCredentials.TokenURI)
		gcpBuilder.Type(r.gcpConfig.GCPCredentials.Type)
		ocmClusterBuilder.GCP(gcpBuilder)
	}
}

// validate validate the state of the clusterBuilder struct.
func (r clusterBuilder) validate(clusterRequest *types.ClusterRequest) error {
	if r.idGenerator == nil {
		return errors.Errorf("idGenerator is not defined")
	}

	if clusterRequest.CloudProvider == cloudproviders.AWS.String() && r.awsConfig == nil {
		return errors.Errorf("awsConfig is not defined")
	}

	if clusterRequest.CloudProvider == cloudproviders.GCP.String() && r.gcpConfig == nil {
		return errors.Errorf("gcpConfig is not defined")
	}

	if r.dataplaneClusterConfig == nil {
		return errors.Errorf("dataplaneClusterConfig is not defined")
	}

	return nil
}
