package clusters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm/clusterservicetest"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/onsi/gomega"
)

const (
	testOpenshiftVersion      = "openshift-v4.6.1"
	testAWSComputeMachineType = "m5.2xlarge"
	testGCPComputeMachineType = "testgcpmachinetype"
)

func Test_clusterBuilder_NewOCMClusterFromCluster(t *testing.T) {
	awsConfig := &config.AWSConfig{}

	awsComputeMachineConfig := config.ComputeMachinesConfig{
		ClusterWideWorkload: &config.ComputeMachineConfig{
			ComputeMachineType: testAWSComputeMachineType,
			ComputeNodesAutoscaling: &config.ComputeNodesAutoscalingConfig{
				MaxComputeNodes: 18,
				MinComputeNodes: 3,
			},
		},
	}
	gcpComputeMachineConfig := config.ComputeMachinesConfig{
		ClusterWideWorkload: &config.ComputeMachineConfig{
			ComputeMachineType: testGCPComputeMachineType,
			ComputeNodesAutoscaling: &config.ComputeNodesAutoscalingConfig{
				MaxComputeNodes: 8,
				MinComputeNodes: 6,
			},
		},
	}
	dataplaneClusterConfig := &config.DataplaneClusterConfig{
		DynamicScalingConfig: config.DynamicScalingConfig{
			NewDataPlaneOpenShiftVersion: testOpenshiftVersion,
			ComputeMachinePerCloudProvider: map[cloudproviders.CloudProviderID]config.ComputeMachinesConfig{
				cloudproviders.AWS: awsComputeMachineConfig,
				cloudproviders.GCP: gcpComputeMachineConfig,
			},
		},
	}

	clusterAWS := clustersmgmtv1.
		NewAWS().
		AccountID(awsConfig.ConfigForOSDClusterCreation.AccountID).
		AccessKeyID(awsConfig.ConfigForOSDClusterCreation.AccessKey).
		SecretAccessKey(awsConfig.ConfigForOSDClusterCreation.SecretAccessKey)

	clusterGCP := clustersmgmtv1.
		NewGCP().
		AuthProviderX509CertURL("testvalue").
		AuthURI("testvalue").
		ClientEmail("testvalue").
		ClientID("testvalue").
		ClientX509CertURL("testvalue").
		PrivateKey("testvalue").
		PrivateKeyID("testvalue").
		ProjectID("testvalue").
		TokenURI("testvalue").
		Type("testvalue")

	type fields struct {
		idGenerator            ocm.IDGenerator
		awsConfig              *config.AWSConfig
		gcpConfig              *config.GCPConfig
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	type args struct {
		clusterRequest *types.ClusterRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantFn  func() *clustersmgmtv1.Cluster
		wantErr bool
	}{
		{
			name: "nil aws config results in error when AWS is the cloud provider to be used",
			fields: fields{
				idGenerator:            ocm.NewIDGenerator(""),
				awsConfig:              nil,
				dataplaneClusterConfig: dataplaneClusterConfig,
			},
			args: args{
				clusterRequest: &types.ClusterRequest{
					CloudProvider: cloudproviders.AWS.String(),
				},
			},
			wantErr: true,
		},
		{
			name: "nil gcp config results in error when GCP is the cloud provider to be used",
			fields: fields{
				idGenerator:            ocm.NewIDGenerator(""),
				awsConfig:              &config.AWSConfig{},
				gcpConfig:              nil,
				dataplaneClusterConfig: dataplaneClusterConfig,
			},
			args: args{
				clusterRequest: &types.ClusterRequest{
					CloudProvider: cloudproviders.GCP.String(),
				},
			},
			wantErr: true,
		},
		{
			name: "nil cluster creation config results in error",
			fields: fields{
				idGenerator:            ocm.NewIDGenerator(""),
				awsConfig:              awsConfig,
				dataplaneClusterConfig: nil,
			},
			args: args{
				clusterRequest: &types.ClusterRequest{},
			},
			wantErr: true,
		},
		{
			name: "nil dataplane cluster config results in error",
			fields: fields{
				idGenerator:            ocm.NewIDGenerator(""),
				awsConfig:              awsConfig,
				dataplaneClusterConfig: dataplaneClusterConfig,
			},
			args: args{
				clusterRequest: nil,
			},
			wantErr: true,
		},
		{
			name: "nil id generator results in error",
			fields: fields{
				idGenerator:            nil,
				awsConfig:              awsConfig,
				dataplaneClusterConfig: dataplaneClusterConfig,
			},
			args: args{
				clusterRequest: &types.ClusterRequest{},
			},
			wantErr: true,
		},
		{
			name: "successful conversion of all supported provided values (AWS provider)",
			fields: fields{
				idGenerator: &ocm.IDGeneratorMock{
					GenerateFunc: func() string {
						return ""
					},
				},
				awsConfig:              awsConfig,
				dataplaneClusterConfig: dataplaneClusterConfig,
			},
			args: args{
				clusterRequest: &types.ClusterRequest{
					CloudProvider: clusterservicetest.MockClusterCloudProvider,
					Region:        clusterservicetest.MockClusterRegion,
					MultiAZ:       clusterservicetest.MockClusterMultiAZ,
				},
			},
			wantFn: func() *clustersmgmtv1.Cluster {
				cluster, err := clusterservicetest.NewMockCluster(func(builder *clustersmgmtv1.ClusterBuilder) {
					// these values will be ignored by the conversion as they're unsupported. so expect different
					// values than we provide.
					builder.CCS(clustersmgmtv1.NewCCS().Enabled(true))
					builder.Managed(true)
					builder.Name("")
					builder.CloudProvider(clustersmgmtv1.NewCloudProvider().ID(cloudproviders.AWS.String()))
					builder.AWS(clusterAWS)
					builder.GCP(nil)
					builder.MultiAZ(true)
					builder.Version(clustersmgmtv1.NewVersion().ID(testOpenshiftVersion))
					builder.Nodes(clustersmgmtv1.NewClusterNodes().
						ComputeMachineType(clustersmgmtv1.NewMachineType().ID(testAWSComputeMachineType)).
						AutoscaleCompute(clustersmgmtv1.NewMachinePoolAutoscaling().MinReplicas(awsComputeMachineConfig.ClusterWideWorkload.ComputeNodesAutoscaling.MinComputeNodes).MaxReplicas(awsComputeMachineConfig.ClusterWideWorkload.ComputeNodesAutoscaling.MaxComputeNodes)))
				})
				if err != nil {
					panic(err)
				}
				return cluster
			},
			wantErr: false,
		},
		{
			name: "successful conversion of all supported provided values (GCP provider)",
			fields: fields{
				idGenerator: &ocm.IDGeneratorMock{
					GenerateFunc: func() string {
						return ""
					},
				},
				gcpConfig: &config.GCPConfig{
					GCPCredentials: config.GCPCredentials{
						AuthProviderX509CertURL: "testvalue",
						AuthURI:                 "testvalue",
						ClientEmail:             "testvalue",
						ClientID:                "testvalue",
						ClientX509CertURL:       "testvalue",
						PrivateKey:              "testvalue",
						PrivateKeyID:            "testvalue",
						ProjectID:               "testvalue",
						TokenURI:                "testvalue",
						Type:                    "testvalue",
					},
				},
				dataplaneClusterConfig: dataplaneClusterConfig,
			},
			args: args{
				clusterRequest: &types.ClusterRequest{
					CloudProvider: cloudproviders.GCP.String(),
					Region:        clusterservicetest.MockClusterRegion,
					MultiAZ:       clusterservicetest.MockClusterMultiAZ,
				},
			},
			wantFn: func() *clustersmgmtv1.Cluster {
				cluster, err := clusterservicetest.NewMockCluster(func(builder *clustersmgmtv1.ClusterBuilder) {
					// these values will be ignored by the conversion as they're unsupported. so expect different
					// values than we provide.
					builder.CCS(clustersmgmtv1.NewCCS().Enabled(true))
					builder.Managed(true)
					builder.Name("")
					builder.CloudProvider(clustersmgmtv1.NewCloudProvider().ID(cloudproviders.GCP.String()))
					builder.AWS(nil)
					builder.GCP(clusterGCP)
					builder.MultiAZ(true)
					builder.Version(clustersmgmtv1.NewVersion().ID(testOpenshiftVersion))
					builder.Nodes(clustersmgmtv1.NewClusterNodes().
						ComputeMachineType(clustersmgmtv1.NewMachineType().ID(testGCPComputeMachineType)).
						AutoscaleCompute(clustersmgmtv1.NewMachinePoolAutoscaling().MinReplicas(gcpComputeMachineConfig.ClusterWideWorkload.ComputeNodesAutoscaling.MinComputeNodes).MaxReplicas(gcpComputeMachineConfig.ClusterWideWorkload.ComputeNodesAutoscaling.MaxComputeNodes)))
				})
				if err != nil {
					panic(err)
				}
				return cluster
			},
			wantErr: false,
		},
		{
			name: "successful conversion of all supported provided values with multi_az set to false",
			fields: fields{
				idGenerator: &ocm.IDGeneratorMock{
					GenerateFunc: func() string {
						return ""
					},
				},
				awsConfig:              awsConfig,
				dataplaneClusterConfig: dataplaneClusterConfig,
			},
			args: args{
				clusterRequest: &types.ClusterRequest{
					CloudProvider: clusterservicetest.MockClusterCloudProvider,
					Region:        clusterservicetest.MockClusterRegion,
					MultiAZ:       false,
				},
			},
			wantFn: func() *clustersmgmtv1.Cluster {
				cluster, err := clusterservicetest.NewMockCluster(func(builder *clustersmgmtv1.ClusterBuilder) {
					// these values will be ignored by the conversion as they're unsupported. so expect different
					// values than we provide.
					builder.CCS(clustersmgmtv1.NewCCS().Enabled(true))
					builder.Managed(true)
					builder.Name("")
					builder.AWS(clusterAWS)
					builder.MultiAZ(false)
					builder.Version(clustersmgmtv1.NewVersion().ID(testOpenshiftVersion))
					builder.Nodes(clustersmgmtv1.NewClusterNodes().
						ComputeMachineType(clustersmgmtv1.NewMachineType().ID(testAWSComputeMachineType)).
						AutoscaleCompute(clustersmgmtv1.NewMachinePoolAutoscaling().MinReplicas(awsComputeMachineConfig.ClusterWideWorkload.ComputeNodesAutoscaling.MinComputeNodes).MaxReplicas(awsComputeMachineConfig.ClusterWideWorkload.ComputeNodesAutoscaling.MaxComputeNodes)))
				})
				if err != nil {
					panic(err)
				}
				return cluster
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		if tt.wantFn == nil {
			tt.wantFn = func() *clustersmgmtv1.Cluster {
				return nil
			}
		}
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			r := clusterBuilder{
				idGenerator:            tt.fields.idGenerator,
				awsConfig:              tt.fields.awsConfig,
				gcpConfig:              tt.fields.gcpConfig,
				dataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
			}
			got, err := r.NewOCMClusterFromCluster(tt.args.clusterRequest)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOCMClusterFromCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.wantFn()))
		})
	}
}
