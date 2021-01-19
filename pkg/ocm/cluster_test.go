package ocm

import (
	"reflect"
	"testing"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/clusterservicetest"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
)

const openshiftVersion = "openshift-v4.6.1"

func Test_clusterBuilder_NewOCMClusterFromCluster(t *testing.T) {
	awsConfig := &config.AWSConfig{}

	clusterCreationConfig := &config.ClusterCreationConfig{
		OpenshiftVersion:   openshiftVersion,
		ComputeMachineType: ComputeMachineType,
	}

	clusterAWS := clustersmgmtv1.
		NewAWS().
		AccountID(awsConfig.AccountID).
		AccessKeyID(awsConfig.AccessKey).
		SecretAccessKey(awsConfig.SecretAccessKey)

	type fields struct {
		idGenerator           IDGenerator
		awsConfig             *config.AWSConfig
		clusterCreationConfig *config.ClusterCreationConfig
	}

	type args struct {
		cluster *api.Cluster
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantFn  func() *clustersmgmtv1.Cluster
		wantErr bool
	}{
		{
			name: "nil aws config results in error",
			fields: fields{
				idGenerator:           NewIDGenerator(""),
				awsConfig:             nil,
				clusterCreationConfig: clusterCreationConfig,
			},
			args: args{
				cluster: &api.Cluster{},
			},
			wantErr: true,
		},
		{
			name: "nil cluster creation config results in error",
			fields: fields{
				idGenerator:           NewIDGenerator(""),
				awsConfig:             awsConfig,
				clusterCreationConfig: nil,
			},
			args: args{
				cluster: &api.Cluster{},
			},
			wantErr: true,
		},
		{
			name: "nil cluster results in error",
			fields: fields{
				idGenerator:           NewIDGenerator(""),
				awsConfig:             awsConfig,
				clusterCreationConfig: clusterCreationConfig,
			},
			args: args{
				cluster: nil,
			},
			wantErr: true,
		},
		{
			name: "nil id generator results in error",
			fields: fields{
				idGenerator:           nil,
				awsConfig:             awsConfig,
				clusterCreationConfig: clusterCreationConfig,
			},
			args: args{
				cluster: &api.Cluster{},
			},
			wantErr: true,
		},
		{
			name: "successful conversion of all supported provided values",
			fields: fields{
				idGenerator: &IDGeneratorMock{
					GenerateFunc: func() string {
						return ""
					},
				},
				awsConfig:             awsConfig,
				clusterCreationConfig: clusterCreationConfig,
			},
			args: args{
				cluster: &api.Cluster{
					CloudProvider: clusterservicetest.MockClusterCloudProvider,
					ClusterID:     clusterservicetest.MockClusterID,
					ExternalID:    clusterservicetest.MockClusterExternalID,
					Region:        clusterservicetest.MockClusterRegion,
					BYOC:          clusterservicetest.MockClusterBYOC,
					Managed:       clusterservicetest.MockClusterManaged,
					MultiAZ:       clusterservicetest.MockClusterMultiAZ,
				},
			},
			wantFn: func() *clustersmgmtv1.Cluster {
				cluster, err := clusterservicetest.NewMockCluster(func(builder *clustersmgmtv1.ClusterBuilder) {
					// these values will be ignored by the conversion as they're unsupported. so expect different
					// values than we provide.
					builder.BYOC(true)
					builder.Managed(true)
					builder.Name("")
					builder.AWS(clusterAWS)
					builder.MultiAZ(true)
					builder.Version(clustersmgmtv1.NewVersion().ID(openshiftVersion))
					builder.Nodes(clustersmgmtv1.NewClusterNodes().ComputeMachineType(clustersmgmtv1.NewMachineType().ID(ComputeMachineType)))
				})
				if err != nil {
					panic(err)
				}
				return cluster
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		if tt.wantFn == nil {
			tt.wantFn = func() *clustersmgmtv1.Cluster {
				return nil
			}
		}
		t.Run(tt.name, func(t *testing.T) {
			r := clusterBuilder{
				idGenerator:           tt.fields.idGenerator,
				awsConfig:             tt.fields.awsConfig,
				clusterCreationConfig: tt.fields.clusterCreationConfig,
			}
			got, err := r.NewOCMClusterFromCluster(tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOCMClusterFromCluster() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.wantFn()) {
				t.Errorf("NewOCMClusterFromCluster() got = %+v, want %+v", got, tt.wantFn())
			}
		})
	}
}
