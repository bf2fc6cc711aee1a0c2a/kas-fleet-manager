package workers

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"

	. "github.com/onsi/gomega"
	authv1 "github.com/openshift/api/authorization/v1"
	userv1 "github.com/openshift/api/user/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/api/pkg/operators/v1alpha2"
	errors "github.com/zgalor/weberr"

	k8sCoreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	dpMock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/data_plane"
)

var (
	testRegion                    = "us-east-1"
	testProvider                  = "aws"
	strimziAddonID                = "managed-kafka-test"
	clusterLoggingOperatorAddonID = "cluster-logging-operator-test"
	supportedInstanceType         = "eval"
	deprovisionCluster            = api.Cluster{
		Status: api.ClusterDeprovisioning,
	}
	acceptedCluster = api.Cluster{
		Status: api.ClusterAccepted,
	}
	readyCluster = api.Cluster{
		Status: api.ClusterReady,
	}
	clusterWaitingForKasFleetShardOperator = api.Cluster{
		Status: api.ClusterWaitingForKasFleetShardOperator,
	}
	supportedProviders = config.ProviderConfig{
		ProvidersConfig: config.ProviderConfiguration{
			SupportedProviders: config.ProviderList{
				config.Provider{
					Name: "aws",
					Regions: config.RegionList{
						config.Region{
							Name: "us-east-1",
							SupportedInstanceTypes: map[string]config.InstanceTypeConfig{
								"standard": {Limit: nil},
								"eval":     {Limit: nil},
							},
						},
					},
				},
			},
		},
	}
	keycloakRealmConfig = keycloak.KeycloakRealmConfig{
		ValidIssuerURI: "https://foo.bar",
	}
	autoScalingDataPlaneConfig = &config.DataplaneClusterConfig{
		DataPlaneClusterScalingType: config.AutoScaling,
	}
)

func TestClusterManager_GetID(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "should return cluster manager id",
			want: "",
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{}

			Expect(c.GetID()).To(Equal(tt.want))
		})
	}
}

func TestClusterManager_GetWorkerType(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "should return cluster worker type",
			want: "cluster",
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClusterManager(ClusterManagerOptions{})

			Expect(c.GetWorkerType()).To(Equal(tt.want))
		})
	}
}

func TestClusterManager_IsRunning(t *testing.T) {
	type args struct {
		isRunning bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should return false if cluster manager is not running",
			args: args{isRunning: false},
			want: false,
		},
		{
			name: "should return true if cluster manager is running",
			args: args{isRunning: true},
			want: true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClusterManager(ClusterManagerOptions{})
			c.SetIsRunning(tt.args.isRunning)
			Expect(c.IsRunning()).To(Equal(tt.want))
		})
	}
}

func TestClusterManager_reconcile(t *testing.T) {
	type fields struct {
		clusterService         services.ClusterService
		dataplaneClusterConfig *config.DataplaneClusterConfig
		supportedProviders     *config.ProviderConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			// single test case here with default/ empty values is sufficient, as the rest of the functionality called by the
			// reconciler will be tested in the unit tests that "reconcile" calls
			name: "should successfully complete reconciliation with empty return values",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CountByStatusFunc: func([]api.ClusterStatus) ([]services.ClusterStatusCount, *apiErrors.ServiceError) {
						return []services.ClusterStatusCount{}, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, *apiErrors.ServiceError) {
						return []services.ResKafkaInstanceCount{}, nil
					},
					ListByStatusFunc: func(state api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{}, nil
					},
					ListGroupByProviderAndRegionFunc: func(providers []string, regions []string, status []string) ([]*services.ResGroupCPRegion, *apiErrors.ServiceError) {
						return []*services.ResGroupCPRegion{}, nil
					},
				},
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					ClusterConfig:               &config.ClusterConfig{},
				},
				supportedProviders: &config.ProviderConfig{},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:         tt.fields.clusterService,
					DataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
					SupportedProviders:     tt.fields.supportedProviders,
				},
			}

			Expect(len(c.Reconcile()) > 0).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_processMetrics(t *testing.T) {
	type fields struct {
		clusterService         services.ClusterService
		dataplaneClusterConfig *config.DataplaneClusterConfig
		supportedProviders     *config.ProviderConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if CountByStatus called by setClusterStatusCountMetrics fails in ClusterService",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CountByStatusFunc: func([]api.ClusterStatus) ([]services.ClusterStatusCount, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to count by status")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if FindKafkaInstanceCount called by setKafkaPerClusterCountMetrics fails in ClusterService",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CountByStatusFunc: func([]api.ClusterStatus) ([]services.ClusterStatusCount, *apiErrors.ServiceError) {
						return []services.ClusterStatusCount{}, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to find kafka instances count")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if GetInstanceLimit called by setClusterStatusMaxCapacityMetrics fails in SupportedProviders",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CountByStatusFunc: func([]api.ClusterStatus) ([]services.ClusterStatusCount, *apiErrors.ServiceError) {
						return []services.ClusterStatusCount{}, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, *apiErrors.ServiceError) {
						return []services.ResKafkaInstanceCount{}, nil
					},
				},
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					ClusterConfig: config.NewClusterConfig(
						config.ClusterList{
							dpMock.BuildManualCluster(supportedInstanceType),
						}),
				},
				supportedProviders: &config.ProviderConfig{},
			},
			wantErr: true,
		},
		{
			name: "should succeed if no errors occur during the execution",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CountByStatusFunc: func([]api.ClusterStatus) ([]services.ClusterStatusCount, *apiErrors.ServiceError) {
						return []services.ClusterStatusCount{}, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, *apiErrors.ServiceError) {
						return []services.ResKafkaInstanceCount{}, nil
					},
				},
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					ClusterConfig: config.NewClusterConfig(
						config.ClusterList{
							dpMock.BuildManualCluster(supportedInstanceType),
						}),
				},
				supportedProviders: &supportedProviders,
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:         tt.fields.clusterService,
					DataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
					SupportedProviders:     tt.fields.supportedProviders,
				},
			}
			Expect(len(c.processMetrics()) > 0).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_processDeprovisioningClusters(t *testing.T) {
	type fields struct {
		clusterService         services.ClusterService
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if ListByStatus fails in ClusterService",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to list by status")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if reconcileDeprovisioningCluster fails during processing deprovisioned clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							deprovisionCluster,
						}, nil
					},
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, *apiErrors.ServiceError) {
						return &deprovisionCluster, nil
					},
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, apiErrors.GeneralError("failed to delete cluster")
					},
				},
				dataplaneClusterConfig: autoScalingDataPlaneConfig,
			},
			wantErr: true,
		},
		{
			name: "should succeed if no errors are encountered",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							deprovisionCluster,
						}, nil
					},
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, *apiErrors.ServiceError) {
						return &deprovisionCluster, nil
					},
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
				dataplaneClusterConfig: autoScalingDataPlaneConfig,
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:         tt.fields.clusterService,
					DataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
				},
			}
			Expect(len(c.processDeprovisioningClusters()) > 0).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_processCleanupClusters(t *testing.T) {
	type fields struct {
		clusterService             services.ClusterService
		osdIDPKeycloakService      sso.KeycloakService
		kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if ListByStatus fails in ClusterService",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to list by status")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if reconcileCleanupCluster fails during processing cleaned up clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							deprovisionCluster,
						}, nil
					},
				},
				osdIDPKeycloakService: &sso.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaNamespace string) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to deregister client in sso")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should succeed if no errors are encountered",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							deprovisionCluster,
						}, nil
					},
					DeleteByClusterIDFunc: func(clusterID string) *apiErrors.ServiceError {
						return nil
					},
				},
				osdIDPKeycloakService: &sso.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaNamespace string) *apiErrors.ServiceError {
						return nil
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					RemoveServiceAccountFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					OsdIdpKeycloakService:      tt.fields.osdIDPKeycloakService,
					KasFleetshardOperatorAddon: tt.fields.kasFleetshardOperatorAddon,
				},
			}
			Expect(len(c.processCleanupClusters()) > 0).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_processAcceptedClusters(t *testing.T) {
	type fields struct {
		clusterService services.ClusterService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if ListByStatus fails in ClusterService",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to list by status")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if reconcileAcceptedCluster fails during processing accepted clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							acceptedCluster,
						}, nil
					},
					CreateFunc: func(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to create cluster")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should succeed if no errors are encountered",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							acceptedCluster,
						}, nil
					},
					CreateFunc: func(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError) {
						return &acceptedCluster, nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
				},
			}
			Expect(len(c.processAcceptedClusters()) > 0).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_processProvisioningClusters(t *testing.T) {
	type fields struct {
		clusterService services.ClusterService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if ListByStatus fails in ClusterService",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to list by status")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if reconcileClusterStatus fails during processing provisioning clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							acceptedCluster,
						}, nil
					},
					CheckClusterStatusFunc: func(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to check cluster status")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should succeed if no errors are encountered",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							acceptedCluster,
						}, nil
					},
					CheckClusterStatusFunc: func(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError) {
						return &acceptedCluster, nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
				},
			}
			Expect(len(c.processProvisioningClusters()) > 0).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_processProvisionedClusters(t *testing.T) {
	type fields struct {
		clusterService             services.ClusterService
		osdIdpKeycloakService      sso.KeycloakService
		dataplaneClusterConfig     *config.DataplaneClusterConfig
		supportedProviders         *config.ProviderConfig
		observabilityConfiguration *observatorium.ObservabilityConfiguration
		agentOperator              services.KasFleetshardOperatorAddon
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if ListByStatus fails in ClusterService",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to list by status")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if reconcileProvisionedCluster fails during processing provisioned clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							acceptedCluster,
						}, nil
					},
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "", apiErrors.GeneralError("failed to get cluster dns")
					},
				},
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: true,
				},
			},
			wantErr: true,
		},
		{
			name: "should succeed if no errors are encountered",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							acceptedCluster,
						}, nil
					},
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test", nil
					},
					ConfigureAndSaveIdentityProviderFunc: func(cluster *api.Cluster, identityProviderInfo types.IdentityProviderInfo) (*api.Cluster, *apiErrors.ServiceError) {
						return &acceptedCluster, nil
					},
					CountByStatusFunc: func([]api.ClusterStatus) ([]services.ClusterStatusCount, *apiErrors.ServiceError) {
						return []services.ClusterStatusCount{}, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, *apiErrors.ServiceError) {
						return []services.ResKafkaInstanceCount{}, nil
					},
					ListGroupByProviderAndRegionFunc: func(providers []string, regions []string, status []string) ([]*services.ResGroupCPRegion, *apiErrors.ServiceError) {
						return []*services.ResGroupCPRegion{}, nil
					},
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return nil
					},
					InstallStrimziFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
					UpdateStatusAndClientFunc: func(cluster api.Cluster, status api.ClusterStatus, serviceClientId string, serviceClientSecret string) error {
						return nil
					},
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "secret", nil
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloakRealmConfig
					},
				},
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					ClusterConfig:               &config.ClusterConfig{},
				},
				supportedProviders:         &config.ProviderConfig{},
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
				agentOperator: &services.KasFleetshardOperatorAddonMock{
					ProvisionFunc: func(cluster api.Cluster) (bool, services.ParameterList, *apiErrors.ServiceError) {
						return true, services.ParameterList{}, nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					OsdIdpKeycloakService:      tt.fields.osdIdpKeycloakService,
					DataplaneClusterConfig:     tt.fields.dataplaneClusterConfig,
					SupportedProviders:         tt.fields.supportedProviders,
					ObservabilityConfiguration: tt.fields.observabilityConfiguration,
					OCMConfig:                  &ocm.OCMConfig{StrimziOperatorAddonID: strimziAddonID},
					KasFleetshardOperatorAddon: tt.fields.agentOperator,
				},
			}
			Expect(len(c.processProvisionedClusters()) > 0).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_processReadyClusters(t *testing.T) {
	type fields struct {
		clusterService         services.ClusterService
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if ListByStatus fails in ClusterService",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to list by status")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if reconcileEmptyCluster fails during processing ready clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							readyCluster,
						}, nil
					},
					FindNonEmptyClusterByIdFunc: func(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to find non empty cluster by id")
					},
				},
				dataplaneClusterConfig: autoScalingDataPlaneConfig,
			},
			wantErr: true,
		},
		{
			name: "should not return an error if reconcileReadyCluster doesn't throw an error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							readyCluster,
						}, nil
					},
					FindNonEmptyClusterByIdFunc: func(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
						return &readyCluster, apiErrors.GeneralError("failed to find non empty cluster by id")
					},
				},
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType:           "manual",
					EnableReadyDataPlaneClustersReconcile: false,
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:         tt.fields.clusterService,
					DataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
				},
			}
			Expect(len(c.processReadyClusters()) > 0).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileReadyCluster(t *testing.T) {
	type fields struct {
		clusterService             services.ClusterService
		dataplaneClusterConfig     *config.DataplaneClusterConfig
		osdIdpKeycloakService      sso.KeycloakService
		kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon
		observabilityConfiguration *observatorium.ObservabilityConfiguration
	}

	type args struct {
		cluster api.Cluster
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should return no error if EnableReadyDataPlaneClustersReconcile is set to false",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: false,
				},
			},
			wantErr: false,
		},
		{
			name: "should fail if reconcileClusterInstanceType fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: true,
					DataPlaneClusterScalingType:           config.ManualScaling,
					ClusterConfig:                         &config.ClusterConfig{},
				},
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to update cluster")
					},
				},
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
			},
			args: args{
				cluster: readyCluster,
			},
			wantErr: true,
		},
		{
			name: "should fail if reconcileClusterResources fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: true,
					DataPlaneClusterScalingType:           config.ManualScaling,
					ClusterConfig:                         &config.ClusterConfig{},
				},
				clusterService: &services.ClusterServiceMock{
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to apply resources")
					},
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test", nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
			},
			args: args{
				cluster: readyCluster,
			},
			wantErr: true,
		},
		{
			name: "should fail if reconcileClusterIdentityProvider fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: true,
					DataPlaneClusterScalingType:           config.ManualScaling,
					ClusterConfig:                         &config.ClusterConfig{},
				},
				clusterService: &services.ClusterServiceMock{
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return nil
					},
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "", apiErrors.GeneralError("failed to get cluster dns")
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "", apiErrors.GeneralError("failed to register osd cluster client in sso")
					},
				},
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
			},
			args: args{
				cluster: readyCluster,
			},
			wantErr: true,
		},
		{
			name: "should fail if reconcileKasFleetshardOperator fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: true,
					DataPlaneClusterScalingType:           config.ManualScaling,
					ClusterConfig:                         &config.ClusterConfig{},
				},
				clusterService: &services.ClusterServiceMock{
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return nil
					},
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test", nil
					},
					ConfigureAndSaveIdentityProviderFunc: func(cluster *api.Cluster, identityProviderInfo types.IdentityProviderInfo) (*api.Cluster, *apiErrors.ServiceError) {
						return &clusterWaitingForKasFleetShardOperator, nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "secret", nil
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloakRealmConfig
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					ReconcileParametersFunc: func(cluster api.Cluster) (services.ParameterList, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to reconcile params")
					},
				},
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
			},
			args: args{
				cluster: readyCluster,
			},
			wantErr: true,
		},
		{
			name: "should succeed if no errors are thrown during execution",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: true,
					DataPlaneClusterScalingType:           config.ManualScaling,
					ClusterConfig:                         &config.ClusterConfig{},
				},
				clusterService: &services.ClusterServiceMock{
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return nil
					},
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test", nil
					},
					ConfigureAndSaveIdentityProviderFunc: func(cluster *api.Cluster, identityProviderInfo types.IdentityProviderInfo) (*api.Cluster, *apiErrors.ServiceError) {
						return &clusterWaitingForKasFleetShardOperator, nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "secret", nil
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloakRealmConfig
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					ReconcileParametersFunc: func(cluster api.Cluster) (services.ParameterList, *apiErrors.ServiceError) {
						return services.ParameterList{}, nil
					},
				},
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
			},
			args: args{
				cluster: readyCluster,
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					DataplaneClusterConfig:     tt.fields.dataplaneClusterConfig,
					OCMConfig:                  &ocm.OCMConfig{StrimziOperatorAddonID: strimziAddonID},
					OsdIdpKeycloakService:      tt.fields.osdIdpKeycloakService,
					KasFleetshardOperatorAddon: tt.fields.kasFleetshardOperatorAddon,
					ObservabilityConfiguration: tt.fields.observabilityConfiguration,
				},
			}
			Expect(c.reconcileReadyCluster(tt.args.cluster) != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileWaitingForKasFleetshardOperatorCluster(t *testing.T) {
	type fields struct {
		clusterService             services.ClusterService
		dataplaneClusterConfig     *config.DataplaneClusterConfig
		osdIdpKeycloakService      sso.KeycloakService
		kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon
		observabilityConfiguration *observatorium.ObservabilityConfiguration
	}

	type args struct {
		cluster api.Cluster
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should fail if reconcileClusterResources fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: true,
					DataPlaneClusterScalingType:           config.ManualScaling,
					ClusterConfig:                         &config.ClusterConfig{},
				},
				clusterService: &services.ClusterServiceMock{
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to apply resources")
					},
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test", nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
			},
			args: args{
				cluster: readyCluster,
			},
			wantErr: true,
		},
		{
			name: "should fail if reconcileClusterIdentityProvider fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: true,
					EnableReadyDataPlaneClustersReconcile:       true,
					DataPlaneClusterScalingType:                 config.ManualScaling,
					ClusterConfig:                               &config.ClusterConfig{},
				},
				clusterService: &services.ClusterServiceMock{
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return nil
					},
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "", apiErrors.GeneralError("failed to get cluster dns")
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "", apiErrors.GeneralError("failed to register osd cluster client in sso")
					},
				},
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
			},
			args: args{
				cluster: readyCluster,
			},
			wantErr: true,
		},
		{
			name: "should fail if reconcileKasFleetshardOperator fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: true,
					DataPlaneClusterScalingType:           config.ManualScaling,
					ClusterConfig:                         &config.ClusterConfig{},
				},
				clusterService: &services.ClusterServiceMock{
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return nil
					},
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test", nil
					},
					ConfigureAndSaveIdentityProviderFunc: func(cluster *api.Cluster, identityProviderInfo types.IdentityProviderInfo) (*api.Cluster, *apiErrors.ServiceError) {
						return &clusterWaitingForKasFleetShardOperator, nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "secret", nil
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloakRealmConfig
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					ReconcileParametersFunc: func(cluster api.Cluster) (services.ParameterList, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to reconcile params")
					},
				},
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
			},
			args: args{
				cluster: readyCluster,
			},
			wantErr: true,
		},
		{
			name: "should succeed if no errors are thrown during execution",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: true,
					DataPlaneClusterScalingType:           config.ManualScaling,
					ClusterConfig:                         &config.ClusterConfig{},
				},
				clusterService: &services.ClusterServiceMock{
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return nil
					},
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test", nil
					},
					ConfigureAndSaveIdentityProviderFunc: func(cluster *api.Cluster, identityProviderInfo types.IdentityProviderInfo) (*api.Cluster, *apiErrors.ServiceError) {
						return &clusterWaitingForKasFleetShardOperator, nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "secret", nil
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloakRealmConfig
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					ReconcileParametersFunc: func(cluster api.Cluster) (services.ParameterList, *apiErrors.ServiceError) {
						return services.ParameterList{}, nil
					},
				},
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
			},
			args: args{
				cluster: readyCluster,
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					DataplaneClusterConfig:     tt.fields.dataplaneClusterConfig,
					OCMConfig:                  &ocm.OCMConfig{StrimziOperatorAddonID: strimziAddonID},
					OsdIdpKeycloakService:      tt.fields.osdIdpKeycloakService,
					KasFleetshardOperatorAddon: tt.fields.kasFleetshardOperatorAddon,
					ObservabilityConfiguration: tt.fields.observabilityConfiguration,
				},
			}
			Expect(c.reconcileWaitingForKasFleetshardOperatorCluster(tt.args.cluster) != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileProvisionedCluster(t *testing.T) {
	type fields struct {
		clusterService             services.ClusterService
		dataplaneClusterConfig     *config.DataplaneClusterConfig
		osdIdpKeycloakService      sso.KeycloakService
		observabilityConfiguration *observatorium.ObservabilityConfiguration
		agentOperator              services.KasFleetshardOperatorAddon
	}

	type args struct {
		cluster api.Cluster
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should fail if reconcileClusterIdentityProvider fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "", apiErrors.GeneralError("failed to get cluster dns")
					},
				},
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: true,
				},
			},
			args: args{
				cluster: readyCluster,
			},
			wantErr: true,
		},
		{
			name: "should fail if reconcileClusterResources fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: true,
					DataPlaneClusterScalingType:           config.ManualScaling,
					ClusterConfig:                         &config.ClusterConfig{},
				},
				clusterService: &services.ClusterServiceMock{
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to apply resources")
					},
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test", nil
					},
					ConfigureAndSaveIdentityProviderFunc: func(cluster *api.Cluster, identityProviderInfo types.IdentityProviderInfo) (*api.Cluster, *apiErrors.ServiceError) {
						return &clusterWaitingForKasFleetShardOperator, nil
					},
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "secret", nil
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloakRealmConfig
					},
				},
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
			},
			args: args{
				cluster: readyCluster,
			},
			wantErr: true,
		},
		{
			name: "should fail if reconcileAddonOperator fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: true,
					DataPlaneClusterScalingType:           config.ManualScaling,
					ClusterConfig:                         &config.ClusterConfig{},
				},
				clusterService: &services.ClusterServiceMock{
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return nil
					},
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test", nil
					},
					ConfigureAndSaveIdentityProviderFunc: func(cluster *api.Cluster, identityProviderInfo types.IdentityProviderInfo) (*api.Cluster, *apiErrors.ServiceError) {
						return &clusterWaitingForKasFleetShardOperator, nil
					},
					InstallStrimziFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
					UpdateStatusAndClientFunc: func(cluster api.Cluster, status api.ClusterStatus, serviceClientId string, serviceClientSecret string) error {
						return apiErrors.GeneralError("failed to update status and client")
					},
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "secret", nil
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloakRealmConfig
					},
				},
				agentOperator: &services.KasFleetshardOperatorAddonMock{
					ProvisionFunc: func(cluster api.Cluster) (bool, services.ParameterList, *apiErrors.ServiceError) {
						return true, services.ParameterList{}, nil
					},
				},
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
			},
			args: args{
				cluster: readyCluster,
			},
			wantErr: true,
		},
		{
			name: "should succeed if no errors occur during execution",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: true,
					DataPlaneClusterScalingType:           config.ManualScaling,
					ClusterConfig:                         &config.ClusterConfig{},
				},
				clusterService: &services.ClusterServiceMock{
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return nil
					},
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test", nil
					},
					ConfigureAndSaveIdentityProviderFunc: func(cluster *api.Cluster, identityProviderInfo types.IdentityProviderInfo) (*api.Cluster, *apiErrors.ServiceError) {
						return &clusterWaitingForKasFleetShardOperator, nil
					},
					InstallStrimziFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
					UpdateStatusAndClientFunc: func(cluster api.Cluster, status api.ClusterStatus, serviceClientId string, serviceClientSecret string) error {
						return nil
					},
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "secret", nil
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloakRealmConfig
					},
				},
				agentOperator: &services.KasFleetshardOperatorAddonMock{
					ProvisionFunc: func(cluster api.Cluster) (bool, services.ParameterList, *apiErrors.ServiceError) {
						return true, services.ParameterList{}, nil
					},
				},
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
			},
			args: args{
				cluster: readyCluster,
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					DataplaneClusterConfig:     tt.fields.dataplaneClusterConfig,
					OCMConfig:                  &ocm.OCMConfig{StrimziOperatorAddonID: strimziAddonID},
					OsdIdpKeycloakService:      tt.fields.osdIdpKeycloakService,
					KasFleetshardOperatorAddon: tt.fields.agentOperator,
					ObservabilityConfiguration: tt.fields.observabilityConfiguration,
				},
			}
			Expect(c.reconcileProvisionedCluster(tt.args.cluster) != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_processWaitingForKasFleetshardOperatorClusters(t *testing.T) {
	type fields struct {
		clusterService             services.ClusterService
		dataplaneClusterConfig     *config.DataplaneClusterConfig
		observabilityConfiguration *observatorium.ObservabilityConfiguration
		osdIdpKeycloakService      sso.KeycloakService
		kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error if ListByStatus fails in ClusterService",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to list by status")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if reconcileEmptyCluster fails during processing ready clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							clusterWaitingForKasFleetShardOperator,
						}, nil
					},
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to apply resources")
					},
				},
				dataplaneClusterConfig:     config.NewDataplaneClusterConfig(),
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
			},
			wantErr: true,
		},
		{
			name: "should not return an error if no errors occur during execution",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							clusterWaitingForKasFleetShardOperator,
						}, nil
					},
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return nil
					},
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test", nil
					},
					ConfigureAndSaveIdentityProviderFunc: func(cluster *api.Cluster, identityProviderInfo types.IdentityProviderInfo) (*api.Cluster, *apiErrors.ServiceError) {
						return &clusterWaitingForKasFleetShardOperator, nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "secret", nil
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloakRealmConfig
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					ReconcileParametersFunc: func(cluster api.Cluster) (services.ParameterList, *apiErrors.ServiceError) {
						return nil, nil
					},
				},
				dataplaneClusterConfig:     config.NewDataplaneClusterConfig(),
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					DataplaneClusterConfig:     tt.fields.dataplaneClusterConfig,
					ObservabilityConfiguration: tt.fields.observabilityConfiguration,
					OCMConfig:                  &ocm.OCMConfig{StrimziOperatorAddonID: strimziAddonID},
					OsdIdpKeycloakService:      tt.fields.osdIdpKeycloakService,
					KasFleetshardOperatorAddon: tt.fields.kasFleetshardOperatorAddon,
				},
			}
			Expect(len(c.processWaitingForKasFleetshardOperatorClusters()) > 0).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileKasFleetshardOperator(t *testing.T) {
	type fields struct {
		clusterService             services.ClusterService
		kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon
	}
	tests := []struct {
		name    string
		fields  fields
		arg     api.Cluster
		wantErr bool
	}{
		{
			name: "error when ReconcileParametersFunc returns error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					ReconcileParametersFunc: func(cluster api.Cluster) (services.ParameterList, *apiErrors.ServiceError) {
						return nil, &apiErrors.ServiceError{}
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no error when cluster ClientID is set",
			fields: fields{
				clusterService: &services.ClusterServiceMock{},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					ReconcileParametersFunc: func(cluster api.Cluster) (services.ParameterList, *apiErrors.ServiceError) {
						return nil, nil
					},
				},
			},
			arg:     api.Cluster{ClientID: "Client ID", ClientSecret: "secret"},
			wantErr: false,
		},
		{
			name: "error when UpdateFunc returns error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return &apiErrors.ServiceError{}
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					ReconcileParametersFunc: func(cluster api.Cluster) (services.ParameterList, *apiErrors.ServiceError) {
						return nil, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should not receive error when UpdateFunc does not return error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					ReconcileParametersFunc: func(cluster api.Cluster) (services.ParameterList, *apiErrors.ServiceError) {
						return nil, nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					KasFleetshardOperatorAddon: tt.fields.kasFleetshardOperatorAddon,
				},
			}

			Expect(c.reconcileKasFleetshardOperator(tt.arg) != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileClusterStatus(t *testing.T) {
	type fields struct {
		clusterService services.ClusterService
	}
	type args struct {
		cluster *api.Cluster
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "error when getting check cluster status",
			fields: fields{
				clusterService: &services.ClusterServiceMock{CheckClusterStatusFunc: func(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError) {
					return nil, apiErrors.GeneralError("failed")
				}},
			},
			args: args{
				cluster: &api.Cluster{
					ClusterID: "test",
				},
			},
			wantErr: true,
		},
		{
			name: "successful reconcile of failed cluster",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CheckClusterStatusFunc: func(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError) {
						return cluster, nil
					},
				},
			},
			args: args{
				cluster: &api.Cluster{
					ClusterID: "test",
					Status:    api.ClusterFailed,
				},
			},
			wantErr: false,
		},
		{
			name: "successful reconcile",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CheckClusterStatusFunc: func(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError) {
						return cluster, nil
					},
				},
			},
			args: args{
				cluster: &api.Cluster{
					ClusterID: "test",
					Status:    api.ClusterProvisioning,
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
				},
			}
			_, err := c.reconcileClusterStatus(tt.args.cluster)
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileStrimziOperator(t *testing.T) {
	type fields struct {
		clusterService services.ClusterService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "error when installing strimzi",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					InstallStrimziFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, apiErrors.GeneralError("failed to install strimzi")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "strimzi installed successfully",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					InstallStrimziFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					SupportedProviders:         &config.ProviderConfig{},
					ObservabilityConfiguration: &observatorium.ObservabilityConfiguration{},
					DataplaneClusterConfig:     &config.DataplaneClusterConfig{},
					OCMConfig:                  &ocm.OCMConfig{StrimziOperatorAddonID: strimziAddonID},
				},
			}
			_, err := c.reconcileStrimziOperator(api.Cluster{
				ClusterID: "clusterId",
			})
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileClusterLoggingOperator(t *testing.T) {
	type fields struct {
		clusterService services.ClusterService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "error when installing cluster logging operator",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					InstallClusterLoggingFunc: func(cluster *api.Cluster, params []types.Parameter) (bool, *apiErrors.ServiceError) {
						return false, apiErrors.GeneralError("failed to install cluster logging operator")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "cluster logging operator installed successfully",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					InstallClusterLoggingFunc: func(cluster *api.Cluster, params []types.Parameter) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					SupportedProviders:         &config.ProviderConfig{},
					ObservabilityConfiguration: &observatorium.ObservabilityConfiguration{},
					DataplaneClusterConfig:     &config.DataplaneClusterConfig{},
					OCMConfig:                  &ocm.OCMConfig{ClusterLoggingOperatorAddonID: clusterLoggingOperatorAddonID},
				},
			}
			_, err := c.reconcileClusterLoggingOperator(api.Cluster{
				ClusterID: "clusterId",
			})
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileAcceptedCluster(t *testing.T) {
	type fields struct {
		clusterService services.ClusterService
	}

	tests := []struct {
		name    string
		wantErr bool
		fields  fields
	}{
		{
			name: "reconcile cluster with cluster creation requests",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CreateFunc: func(cluster *api.Cluster) (cls *api.Cluster, e *apiErrors.ServiceError) {
						return cluster, nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "failed accepted cluster reconciliation should result in an error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CreateFunc: func(cluster *api.Cluster) (cls *api.Cluster, e *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to create cluster")
					},
				},
			},
			wantErr: true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					ObservabilityConfiguration: &observatorium.ObservabilityConfiguration{},
					DataplaneClusterConfig:     config.NewDataplaneClusterConfig(),
				},
			}

			Expect(c.reconcileAcceptedCluster(&acceptedCluster) != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileClustersForRegions(t *testing.T) {
	type fields struct {
		providerLst            []string
		clusterService         services.ClusterService
		providersConfig        config.ProviderConfig
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name    string
		wantErr bool
		fields  fields
	}{
		{
			name: "creates a missing OSD cluster request automatically when autoscaling is enabled",
			fields: fields{
				dataplaneClusterConfig: autoScalingDataPlaneConfig,
				providerLst:            []string{testRegion},
				clusterService: &services.ClusterServiceMock{
					ListGroupByProviderAndRegionFunc: func(providers []string, regions []string, status []string) (m []*services.ResGroupCPRegion, e *apiErrors.ServiceError) {
						res := []*services.ResGroupCPRegion{
							{
								Provider: testProvider,
								Region:   testRegion,
								Count:    1,
							},
						}
						return res, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				providersConfig: supportedProviders,
			},
			wantErr: false,
		},
		{
			name: "skips reconciliation if autoscaling is disabled",
			fields: fields{
				dataplaneClusterConfig: config.NewDataplaneClusterConfig(),
			},
			wantErr: false,
		},
		{
			name: "should return an error if ListGroupByProviderAndRegion fails",
			fields: fields{
				dataplaneClusterConfig: autoScalingDataPlaneConfig,
				providerLst:            []string{"us-east-1"},
				clusterService: &services.ClusterServiceMock{
					ListGroupByProviderAndRegionFunc: func(providers []string, regions []string, status []string) (m []*services.ResGroupCPRegion, e *apiErrors.ServiceError) {
						res := []*services.ResGroupCPRegion{}
						return res, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to register cluster job")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "failed to create OSD request with empty services.ResGroupCPRegion",
			fields: fields{
				dataplaneClusterConfig: autoScalingDataPlaneConfig,
				providerLst:            []string{"us-east-1"},
				clusterService: &services.ClusterServiceMock{
					ListGroupByProviderAndRegionFunc: func(providers []string, regions []string, status []string) (m []*services.ResGroupCPRegion, e *apiErrors.ServiceError) {
						var res []*services.ResGroupCPRegion
						return res, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				providersConfig: supportedProviders,
			},
			wantErr: false,
		},
		{
			name: "should create OSD request ",
			fields: fields{
				dataplaneClusterConfig: autoScalingDataPlaneConfig,
				providerLst:            []string{"us-east-1"},
				clusterService: &services.ClusterServiceMock{
					ListGroupByProviderAndRegionFunc: func(providers []string, regions []string, status []string) (m []*services.ResGroupCPRegion, e *apiErrors.ServiceError) {
						var res []*services.ResGroupCPRegion
						return res, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to create cluster request")
					},
				},
				providersConfig: supportedProviders,
			},
			wantErr: true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					SupportedProviders:         &tt.fields.providersConfig,
					ObservabilityConfiguration: &observatorium.ObservabilityConfiguration{},
					DataplaneClusterConfig:     tt.fields.dataplaneClusterConfig,
				},
			}
			Expect(c.reconcileClustersForRegions() != nil && !tt.wantErr).To(BeFalse())
		})
	}
}

func TestClusterManager_reconcileAddonOperator(t *testing.T) {
	type fields struct {
		agentOperator  services.KasFleetshardOperatorAddon
		clusterService services.ClusterService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "successful strimzi cluster, logging operator and kas fleetshard operator installation",
			fields: fields{
				agentOperator: &services.KasFleetshardOperatorAddonMock{
					ProvisionFunc: func(cluster api.Cluster) (bool, services.ParameterList, *apiErrors.ServiceError) {
						return true, services.ParameterList{}, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					InstallStrimziFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
					InstallClusterLoggingFunc: func(cluster *api.Cluster, params []types.Parameter) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						if status != api.ClusterWaitingForKasFleetShardOperator {
							t.Errorf("expect status to be %s but got %s", api.ClusterWaitingForKasFleetShardOperator.String(), status)
						}
						return nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "return an error if strimzi installation fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					InstallStrimziFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, apiErrors.GeneralError("failed to install strimzi")
					},
					InstallClusterLoggingFunc: func(cluster *api.Cluster, params []types.Parameter) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "return an error if cluster logging operator installation fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					InstallStrimziFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, apiErrors.GeneralError("failed to install strimzi")
					},
					InstallClusterLoggingFunc: func(cluster *api.Cluster, params []types.Parameter) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "return an error if kas fleetshard operator installation fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					InstallStrimziFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
					InstallClusterLoggingFunc: func(cluster *api.Cluster, params []types.Parameter) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						if status != api.ClusterWaitingForKasFleetShardOperator {
							t.Errorf("expect status to be %s but got %s", api.ClusterWaitingForKasFleetShardOperator.String(), status)
						}
						return nil
					},
				},
				agentOperator: &services.KasFleetshardOperatorAddonMock{
					ProvisionFunc: func(cluster api.Cluster) (bool, services.ParameterList, *apiErrors.ServiceError) {
						return false, services.ParameterList{}, apiErrors.GeneralError("failed to provision kas fleetshard operator")
					},
				},
			},
			wantErr: true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					OCMConfig:                  &ocm.OCMConfig{StrimziOperatorAddonID: strimziAddonID, ClusterLoggingOperatorAddonID: clusterLoggingOperatorAddonID},
					ClusterService:             tt.fields.clusterService,
					KasFleetshardOperatorAddon: tt.fields.agentOperator,
				},
			}
			Expect(c.reconcileAddonOperator(api.Cluster{
				ClusterID: "test-cluster-id",
			}) != nil).To(Equal(tt.wantErr))
		})
	}
}

// buildObservabilityConfig builds a observability coreConfig used for testing
func buildObservabilityConfig() observatorium.ObservabilityConfiguration {
	observabilityConfig := observatorium.ObservabilityConfiguration{
		DexUrl:                         "dex-url",
		DexPassword:                    "dex-password",
		DexUsername:                    "dex-username",
		DexSecret:                      "dex-secret",
		ObservatoriumTenant:            "tenant",
		ObservatoriumGateway:           "gateway",
		ObservabilityConfigRepo:        "obs-config-repo",
		ObservabilityConfigChannel:     "obs-config-channel",
		ObservabilityConfigAccessToken: "obs-config-token",
		ObservabilityConfigTag:         "obs-config-tag",
		RedHatSsoAuthServerUrl:         "red-hat-sso-auth-server-url",
		RedHatSsoRealm:                 "red-hat-sso-realm",
		MetricsClientId:                "metrics-client",
		MetricsSecret:                  "metrics-secret",
		LogsClientId:                   "logs-client",
		LogsSecret:                     "logs-secret",
		AuthType:                       "dex",
	}
	return observabilityConfig
}

func buildResourceSet(observabilityConfig observatorium.ObservabilityConfiguration, clusterConfig config.DataplaneClusterConfig, ingressDNS string, cluster *api.Cluster) (types.ResourceSet, error) {
	strimziNamespace := strimziAddonNamespace
	kasFleetshardNamespace := kasFleetshardAddonNamespace

	resources := []interface{}{
		&userv1.Group{
			TypeMeta: metav1.TypeMeta{
				APIVersion: userv1.SchemeGroupVersion.String(),
				Kind:       "Group",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: mkReadOnlyGroupName,
			},
		},
		&authv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: mkReadOnlyRoleBindingName,
			},
			Subjects: []k8sCoreV1.ObjectReference{
				{
					Kind:       "Group",
					APIVersion: "rbac.authorization.k8s.io",
					Name:       mkReadOnlyGroupName,
				},
			},
			RoleRef: k8sCoreV1.ObjectReference{
				Kind:       "ClusterRole",
				Name:       dedicatedReadersRoleBindingName,
				APIVersion: "rbac.authorization.k8s.io",
			},
		},
		&userv1.Group{
			TypeMeta: metav1.TypeMeta{
				APIVersion: userv1.SchemeGroupVersion.String(),
				Kind:       "Group",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: mkSREGroupName,
			},
		},
		&authv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: mkSRERoleBindingName,
			},
			Subjects: []k8sCoreV1.ObjectReference{
				{
					Kind:       "Group",
					APIVersion: "rbac.authorization.k8s.io",
					Name:       mkSREGroupName,
				},
			},
			RoleRef: k8sCoreV1.ObjectReference{
				Kind:       "ClusterRole",
				Name:       clusterAdminRoleName,
				APIVersion: "rbac.authorization.k8s.io",
			},
		},
		&k8sCoreV1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: observabilityNamespace,
			},
		},
		&k8sCoreV1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: metav1.SchemeGroupVersion.Version,
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      observatoriumDexSecretName,
				Namespace: observabilityNamespace,
			},
			Type: k8sCoreV1.SecretTypeOpaque,
			StringData: map[string]string{
				"authType":    observatorium.AuthTypeDex,
				"dexPassword": observabilityConfig.DexPassword,
				"dexSecret":   observabilityConfig.DexSecret,
				"dexUsername": observabilityConfig.DexUsername,
				"gateway":     observabilityConfig.ObservatoriumGateway,
				"dexUrl":      observabilityConfig.DexUrl,
				"tenant":      observabilityConfig.ObservatoriumTenant,
			},
		},
		&k8sCoreV1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: metav1.SchemeGroupVersion.Version,
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      observatoriumSSOSecretName,
				Namespace: observabilityNamespace,
			},
			Type: k8sCoreV1.SecretTypeOpaque,
			StringData: map[string]string{
				"authType":               observatorium.AuthTypeSso,
				"gateway":                observabilityConfig.RedHatSsoGatewayUrl,
				"tenant":                 observabilityConfig.RedHatSsoTenant,
				"redHatSsoAuthServerUrl": observabilityConfig.RedHatSsoAuthServerUrl,
				"redHatSsoRealm":         observabilityConfig.RedHatSsoRealm,
				"metricsClientId":        observabilityConfig.MetricsClientId,
				"metricsSecret":          observabilityConfig.MetricsSecret,
				"logsClientId":           observabilityConfig.LogsClientId,
				"logsSecret":             observabilityConfig.LogsSecret,
			},
		},
		&v1alpha1.CatalogSource{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "operators.coreos.com/v1alpha1",
				Kind:       "CatalogSource",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      observabilityCatalogSourceName,
				Namespace: observabilityNamespace,
			},
			Spec: v1alpha1.CatalogSourceSpec{
				SourceType: v1alpha1.SourceTypeGrpc,
				Image:      observabilityCatalogSourceImage,
			},
		},
		&v1alpha2.OperatorGroup{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "operators.coreos.com/v1alpha2",
				Kind:       "OperatorGroup",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      observabilityOperatorGroupName,
				Namespace: observabilityNamespace,
			},
			Spec: v1alpha2.OperatorGroupSpec{
				TargetNamespaces: []string{observabilityNamespace},
			},
		},
		&v1alpha1.Subscription{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "operators.coreos.com/v1alpha1",
				Kind:       "Subscription",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      observabilitySubscriptionName,
				Namespace: observabilityNamespace,
			},
			Spec: &v1alpha1.SubscriptionSpec{
				CatalogSource:          observabilityCatalogSourceName,
				Channel:                "alpha",
				CatalogSourceNamespace: observabilityNamespace,
				StartingCSV:            "observability-operator.v3.0.9",
				InstallPlanApproval:    v1alpha1.ApprovalAutomatic,
				Package:                observabilitySubscriptionName,
			},
		},
	}
	if cluster.ProviderType == api.ClusterProviderStandalone {
		strimziNamespace = clusterConfig.StrimziOperatorOLMConfig.Namespace
		kasFleetshardNamespace = clusterConfig.KasFleetshardOperatorOLMConfig.Namespace
		resources = append(resources, &k8sCoreV1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: k8sCoreV1.SchemeGroupVersion.String(),
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: strimziNamespace,
			},
		}, &k8sCoreV1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: k8sCoreV1.SchemeGroupVersion.String(),
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: kasFleetshardNamespace,
			},
		})
	}

	if clusterConfig.ImagePullDockerConfigContent != "" {
		resources = append(resources, &k8sCoreV1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: metav1.SchemeGroupVersion.Version,
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      constants.ImagePullSecretName,
				Namespace: strimziNamespace,
			},
			Type: k8sCoreV1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				k8sCoreV1.DockerConfigJsonKey: []byte(clusterConfig.ImagePullDockerConfigContent),
			},
		},
			&k8sCoreV1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: metav1.SchemeGroupVersion.Version,
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      constants.ImagePullSecretName,
					Namespace: kasFleetshardNamespace,
				},
				Type: k8sCoreV1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					k8sCoreV1.DockerConfigJsonKey: []byte(clusterConfig.ImagePullDockerConfigContent),
				},
			})
	}

	return types.ResourceSet{
		Name:      syncsetName,
		Resources: resources,
	}, nil
}

func TestClusterManager_reconcileClusterResourceSet(t *testing.T) {
	const ingressDNS = "foo.bar.example.com"
	observabilityConfig := buildObservabilityConfig()
	clusterConfig := config.DataplaneClusterConfig{
		ImagePullDockerConfigContent: "image-pull-secret-test",
		StrimziOperatorOLMConfig: config.OperatorInstallationConfig{
			Namespace: "strimzi-namespace",
		},
		KasFleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
			Namespace: "kas-fleet-shard-namespace",
		},
	}
	type fields struct {
		clusterService services.ClusterService
	}
	tests := []struct {
		name    string
		fields  fields
		arg     api.Cluster
		wantErr bool
	}{
		{
			name: "test should pass and resourceset should be created for ocm clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						want, _ := buildResourceSet(observabilityConfig, clusterConfig, ingressDNS, cluster)
						Expect(resources).To(Equal(want))
						return nil
					},
				},
			},
			arg: api.Cluster{ClusterID: "test-cluster-id", ProviderType: "ocm"},
		},
		{
			name: "test should pass and resourceset should be created for standalone clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						want, _ := buildResourceSet(observabilityConfig, clusterConfig, ingressDNS, cluster)
						Expect(resources).To(Equal(want))
						return nil
					},
				},
			},
			arg: api.Cluster{ClusterID: "test-cluster-id", ProviderType: "standalone"},
		},
		{
			name: "should receive error when ApplyResources returns error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to apply resources")
					},
				},
			},
			wantErr: true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					SupportedProviders:         &config.ProviderConfig{},
					ObservabilityConfiguration: &observabilityConfig,
					DataplaneClusterConfig:     &clusterConfig,
					OCMConfig:                  &ocm.OCMConfig{},
				},
			}

			Expect(c.reconcileClusterResources(tt.arg) != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileClusterIdentityProvider(t *testing.T) {
	type fields struct {
		clusterService         services.ClusterService
		osdIdpKeycloakService  sso.KeycloakService
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}
	tests := []struct {
		name    string
		fields  fields
		arg     api.Cluster
		wantErr bool
	}{
		{
			name: "should skip the creation of the identity provider when cluster identity provider has already been set",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					GetClusterDNSFunc: nil, // setting it to nill so that it is not called
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: nil, // setting it to nill so that it is not called
					GetRealmConfigFunc:                nil, // setting it to nill so that it is not called
				},
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: true,
				},
			},
			arg: api.Cluster{
				Meta: api.Meta{
					ID: "cluster-id",
				},
				IdentityProviderID: "some-identity-provider-id", // identity provider already set
			},
			wantErr: false,
		},
		{
			name: "should skip the creation of the identity provider when configuration of it is disabled",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					GetClusterDNSFunc: nil, // setting it to nill so that it is not called
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: nil, // setting it to nill so that it is not called
					GetRealmConfigFunc:                nil, // setting it to nill so that it is not called
				},
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: false, // disabling identity provider configuration
				},
			},
			arg: api.Cluster{
				Meta: api.Meta{
					ID: "cluster-id",
				},
			},
			wantErr: false,
		},
		{
			name: "should receive error when GetClusterDNSFunc returns error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "", apiErrors.GeneralError("failed")
					},
				},
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: true,
				},
			},
			wantErr: true,
		},
		{
			name: "should receive an error when creating the the OSD cluster IDP in keycloak fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test.com", nil
					},
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "", apiErrors.FailedToCreateSSOClient("failure")
					},
					GetRealmConfigFunc: nil, // setting it to nil so that it is not called
				},
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: true,
				},
			},
			arg: api.Cluster{
				Meta: api.Meta{
					ID: "cluster-id",
				},
			},
			wantErr: true,
		},
		{
			name: "should receive error when creating the identity provider throws an error during creation",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: true,
				},
				clusterService: &services.ClusterServiceMock{
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test.com", nil
					},
					ConfigureAndSaveIdentityProviderFunc: func(cluster *api.Cluster, identityProviderInfo types.IdentityProviderInfo) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to configure IDP")
					},
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "secret", nil
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloakRealmConfig
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should create an identity provider when cluster identity provider has not been set",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: true,
				},
				clusterService: &services.ClusterServiceMock{
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test.com", nil
					},
					ConfigureAndSaveIdentityProviderFunc: func(cluster *api.Cluster, identityProviderInfo types.IdentityProviderInfo) (*api.Cluster, *apiErrors.ServiceError) {
						return cluster, nil
					},
				},
				osdIdpKeycloakService: &sso.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "secret", nil
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloakRealmConfig
					},
				},
			},
			arg: api.Cluster{
				Meta: api.Meta{
					ID: "cluster-id",
				},
			},
		},
		{
			name: "should return no error for cluster with IdentityProviderID set",
			arg: api.Cluster{
				IdentityProviderID: "test_id",
			},
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableKafkaSreIdentityProviderConfiguration: true,
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:         tt.fields.clusterService,
					OsdIdpKeycloakService:  tt.fields.osdIdpKeycloakService,
					DataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
				},
			}

			Expect(c.reconcileClusterIdentityProvider(tt.arg) != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileClusterDNS(t *testing.T) {
	type fields struct {
		clusterService services.ClusterService
	}
	tests := []struct {
		name    string
		fields  fields
		arg     api.Cluster
		wantErr bool
	}{
		{
			name: "should return when clusterDNS is already set",
			arg: api.Cluster{
				ClusterDNS: "my-cluster-dns",
			},
			wantErr: false,
		},
		{
			name: "should receive error when GetClusterDNSFunc returns error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "", apiErrors.GeneralError("failed")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should successfully reconcile",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "", nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
				},
			}

			Expect(c.reconcileClusterDNS(tt.arg) != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileDeprovisioningCluster(t *testing.T) {
	type fields struct {
		clusterService         services.ClusterService
		DataplaneClusterConfig *config.DataplaneClusterConfig
	}
	tests := []struct {
		name    string
		fields  fields
		arg     api.Cluster
		wantErr bool
	}{
		{
			name: "should receive error when FindCluster to retrieve sibling cluster returns error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, &apiErrors.ServiceError{}
					},
					UpdateStatusFunc: nil, // set to nil as it should not be called
				},
				DataplaneClusterConfig: autoScalingDataPlaneConfig,
			},
			wantErr: true,
		},
		{
			name: "should update the status back to ready when no sibling cluster found",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
				DataplaneClusterConfig: autoScalingDataPlaneConfig,
			},
			wantErr: false,
		},
		{
			name: "receives an error when delete OCM cluster fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, *apiErrors.ServiceError) {
						return &api.Cluster{ClusterID: "dummy cluster"}, nil
					},
					UpdateStatusFunc: nil,
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, apiErrors.GeneralError("failed to remove cluster")
					},
				},
				DataplaneClusterConfig: autoScalingDataPlaneConfig,
			},
			wantErr: true,
		},
		{
			name: "successful deletion of an OSD cluster when auto configuration is enabled",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, *apiErrors.ServiceError) {
						return &api.Cluster{ClusterID: "dummy cluster"}, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
				},
				DataplaneClusterConfig: autoScalingDataPlaneConfig,
			},
			wantErr: false,
		},
		{
			name: "successful deletion of an OSD cluster when manual configuration is enabled",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: nil, // should not be called
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
				},
				DataplaneClusterConfig: config.NewDataplaneClusterConfig(),
			},
			wantErr: false,
		},
		{
			name: "receives an error when the update status fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return fmt.Errorf("Some errors")
					},
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
				},
				DataplaneClusterConfig: config.NewDataplaneClusterConfig(),
			},
			wantErr: true,
		},
		{
			name: "does not update cluster status when cluster has not been fully deleted from ClusterService",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					DeleteFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return errors.Errorf("this should not be called")
					},
				},
				DataplaneClusterConfig: config.NewDataplaneClusterConfig(),
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:         tt.fields.clusterService,
					DataplaneClusterConfig: tt.fields.DataplaneClusterConfig,
				},
			}

			Expect(c.reconcileDeprovisioningCluster(&tt.arg) != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileCleanupCluster(t *testing.T) {
	type fields struct {
		clusterService             services.ClusterService
		osdIDPKeycloakService      sso.KeycloakService
		kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon
	}
	tests := []struct {
		name    string
		fields  fields
		arg     api.Cluster
		wantErr bool
	}{

		{
			name: "receives an error when deregistering the OSD IDP from keycloak fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					DeleteByClusterIDFunc: func(clusterID string) *apiErrors.ServiceError {
						return nil
					},
				},
				osdIDPKeycloakService: &sso.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaNamespace string) *apiErrors.ServiceError {
						return &apiErrors.ServiceError{}
					},
				},
			},
			wantErr: true,
		},
		{
			name: "receives an error when remove kas-fleetshard-operator service account fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
				osdIDPKeycloakService: &sso.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaNamespace string) *apiErrors.ServiceError {
						return nil
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					RemoveServiceAccountFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return &apiErrors.ServiceError{}
					},
				},
			},
			wantErr: true,
		},
		{
			name: "receives an error when soft delete cluster from database fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					DeleteByClusterIDFunc: func(clusterID string) *apiErrors.ServiceError {
						return &apiErrors.ServiceError{}
					},
				},
				osdIDPKeycloakService: &sso.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaNamespace string) *apiErrors.ServiceError {
						return nil
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					RemoveServiceAccountFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "successfull deletion of an OSD cluster",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					DeleteByClusterIDFunc: func(clusterID string) *apiErrors.ServiceError {
						return nil
					},
				},
				osdIDPKeycloakService: &sso.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaNamespace string) *apiErrors.ServiceError {
						return nil
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					RemoveServiceAccountFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					OsdIdpKeycloakService:      tt.fields.osdIDPKeycloakService,
					KasFleetshardOperatorAddon: tt.fields.kasFleetshardOperatorAddon,
				},
			}
			Expect(c.reconcileCleanupCluster(tt.arg) != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileEmptyCluster(t *testing.T) {
	type fields struct {
		clusterService services.ClusterService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    bool
	}{
		{
			name: "should receive error when FindNonEmptyClusterById returns error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindNonEmptyClusterByIdFunc: func(clusterId string) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, &apiErrors.ServiceError{}
					},
					UpdateStatusFunc:                 nil, // set to nil as it should not be called
					ListGroupByProviderAndRegionFunc: nil, // set to nil as it should not be called
				},
			},
			wantErr: true,
			want:    false,
		},
		{
			name: "should return false when cluster is not empty",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindNonEmptyClusterByIdFunc: func(clusterId string) (*api.Cluster, *apiErrors.ServiceError) {
						return &api.Cluster{ClusterID: clusterId}, nil
					},
					UpdateStatusFunc:                 nil, // set to nil as it should not be called
					ListGroupByProviderAndRegionFunc: nil, // set to nil as it should not be called
				},
			},
			wantErr: false,
			want:    false,
		},
		{
			name: "receives an error when ListGroupByProviderAndRegion returns an error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindNonEmptyClusterByIdFunc: func(clusterId string) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, nil
					},
					UpdateStatusFunc: nil, // set to nil as it should not be called
					ListGroupByProviderAndRegionFunc: func(providers, regions, status []string) ([]*services.ResGroupCPRegion, *apiErrors.ServiceError) {
						return nil, &apiErrors.ServiceError{}
					},
				},
			},
			wantErr: true,
			want:    false,
		},
		{
			name: "should not update the cluster status to deprovisioning when no sibling found",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindNonEmptyClusterByIdFunc: func(clusterId string) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, nil
					},
					UpdateStatusFunc: nil, // set to nil as it not be called
					ListGroupByProviderAndRegionFunc: func(providers, regions, status []string) ([]*services.ResGroupCPRegion, *apiErrors.ServiceError) {
						return []*services.ResGroupCPRegion{{Count: 1}}, nil
					},
				},
			},
			wantErr: false,
			want:    false,
		},
		{
			name: "should return false when updating cluster status fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindNonEmptyClusterByIdFunc: func(clusterId string) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return &apiErrors.ServiceError{}
					},
					ListGroupByProviderAndRegionFunc: func(providers, regions, status []string) ([]*services.ResGroupCPRegion, *apiErrors.ServiceError) {
						return []*services.ResGroupCPRegion{{Count: 2}}, nil
					},
				},
			},
			wantErr: true,
			want:    false,
		},
		{
			name: "should return true and update the cluster status",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindNonEmptyClusterByIdFunc: func(clusterId string) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
					ListGroupByProviderAndRegionFunc: func(providers, regions, status []string) ([]*services.ResGroupCPRegion, *apiErrors.ServiceError) {
						return []*services.ResGroupCPRegion{{Count: 2}}, nil
					},
				},
			},
			wantErr: false,
			want:    true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
				},
			}

			emptyClusterReconciled, err := c.reconcileEmptyCluster(api.Cluster{
				Meta: api.Meta{
					ID: "cluster-id",
				},
			})
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(emptyClusterReconciled).To(Equal(tt.want))
		})
	}
}

func TestClusterManager_reconcileClusterWithManualConfig(t *testing.T) {
	type fields struct {
		clusterService         services.ClusterService
		DataplaneClusterConfig *config.DataplaneClusterConfig
	}
	testOsdConfig := config.NewDataplaneClusterConfig()
	testOsdConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{config.ManualCluster{Schedulable: true, KafkaInstanceLimit: 2}})
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Successfully applies manually configured Cluster without deprovisioning clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListAllClusterIdsFunc: func() ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{{ClusterID: "test02"}}, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, *apiErrors.ServiceError) {
						return []services.ResKafkaInstanceCount{
							{
								Clusterid: "test02",
								Count:     1,
							},
						}, nil
					},
				},
				DataplaneClusterConfig: testOsdConfig,
			},
			wantErr: false,
		},
		{
			name: "Successfully applies manually configured Cluster with deprovisioning clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListAllClusterIdsFunc: func() ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{{ClusterID: "test02"}}, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					UpdateMultiClusterStatusFunc: func(clusterIds []string, status api.ClusterStatus) *apiErrors.ServiceError {
						return nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, *apiErrors.ServiceError) {
						return []services.ResKafkaInstanceCount{
							{
								Clusterid: "test02",
								Count:     0,
							},
						}, nil
					},
				},
				DataplaneClusterConfig: testOsdConfig,
			},
			wantErr: false,
		},
		{
			name: "Should fail if UpdateMultiClusterStatus fails on clusters to deprovision",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListAllClusterIdsFunc: func() ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{{ClusterID: "test02"}}, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					UpdateMultiClusterStatusFunc: func(clusterIds []string, status api.ClusterStatus) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to update multi cluster status")
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, *apiErrors.ServiceError) {
						return []services.ResKafkaInstanceCount{
							{
								Clusterid: "test02",
								Count:     0,
							},
						}, nil
					},
				},
				DataplaneClusterConfig: testOsdConfig,
			},
			wantErr: true,
		},
		{
			name: "Should fail if RegisterClusterJob fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListAllClusterIdsFunc: func() ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{{ClusterID: "test02"}}, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to register cluster job")
					},
				},
				DataplaneClusterConfig: testOsdConfig,
			},
			wantErr: true,
		},
		{
			name: "Should fail if FindKafkaInstanceCount fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListAllClusterIdsFunc: func() ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{{ClusterID: "test02"}}, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to find kafka instance count")
					},
				},
				DataplaneClusterConfig: testOsdConfig,
			},
			wantErr: true,
		},
		{
			name: "Failed to apply manually configured Cluster",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListAllClusterIdsFunc: func() ([]api.Cluster, *apiErrors.ServiceError) {
						return nil, &apiErrors.ServiceError{}
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					UpdateMultiClusterStatusFunc: func(clusterIds []string, status api.ClusterStatus) *apiErrors.ServiceError {
						return nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, *apiErrors.ServiceError) {
						return []services.ResKafkaInstanceCount{}, nil
					},
				},
				DataplaneClusterConfig: testOsdConfig,
			},
			wantErr: true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					DataplaneClusterConfig: tt.fields.DataplaneClusterConfig,
					ClusterService:         tt.fields.clusterService,
				},
			}
			Expect(len(c.reconcileClusterWithManualConfig()) > 0).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileClusterInstanceType(t *testing.T) {
	type fields struct {
		clusterService         services.ClusterService
		dataplaneClusterConfig *config.DataplaneClusterConfig
		cluster                api.Cluster
	}
	testOsdConfig := config.NewDataplaneClusterConfig()
	testOsdConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{dpMock.BuildManualCluster(supportedInstanceType)})
	noScalingDataplaneClusterConfig := config.DataplaneClusterConfig{
		DataPlaneClusterScalingType: config.NoScaling,
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Throw an error when update in database fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return &apiErrors.ServiceError{}
					},
				},
				dataplaneClusterConfig: &noScalingDataplaneClusterConfig,
			},
			wantErr: true,
		},
		{
			name: "Update the cluster instance type in the database to standard,developer when cluster scaling type is not manual",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						if cluster.SupportedInstanceType != api.AllInstanceTypeSupport.String() {
							return &apiErrors.ServiceError{}
						} // the cluster should support both instance types
						return nil
					},
				},
				dataplaneClusterConfig: &noScalingDataplaneClusterConfig,
			},
			wantErr: false,
		},
		{
			name: "Do not update cluster instance type when already set and scaling type is not manual",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: nil,
				},
				dataplaneClusterConfig: &noScalingDataplaneClusterConfig,
				cluster: api.Cluster{
					SupportedInstanceType: api.DeveloperTypeSupport.String(),
				},
			},
			wantErr: false,
		},
		{
			name: "Update the cluster instance type in the database to the one set in manual cluster configuration",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						if cluster.SupportedInstanceType != supportedInstanceType {
							return &apiErrors.ServiceError{}
						} // the cluster should support both instance types
						return nil
					},
				},
				dataplaneClusterConfig: testOsdConfig,
				cluster: api.Cluster{
					SupportedInstanceType: api.StandardTypeSupport.String(),
				},
			},
			wantErr: false,
		},
		{
			name: "Do no update the cluster in the database if not found in manual configuration and already set",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: nil, // should not be called
				},
				dataplaneClusterConfig: testOsdConfig,
				cluster: api.Cluster{
					SupportedInstanceType: api.StandardTypeSupport.String(),
				},
			},
			wantErr: false,
		},
		{
			name: "Update the cluster in the database to support both instance types if not found in manual configuration and not already set",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						if cluster.SupportedInstanceType != api.AllInstanceTypeSupport.String() {
							return &apiErrors.ServiceError{}
						} // the cluster should support both instance types
						return nil
					},
				},
				dataplaneClusterConfig: testOsdConfig,
				cluster: api.Cluster{
					SupportedInstanceType: "",
				},
			},
			wantErr: false,
		},
		{
			name: "Do not update in the database if instance type has not changed",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: nil, // should not be called
				},
				dataplaneClusterConfig: testOsdConfig,
				cluster: api.Cluster{
					SupportedInstanceType: supportedInstanceType,
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					DataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
					ClusterService:         tt.fields.clusterService,
				},
			}
			Expect(c.reconcileClusterInstanceType(tt.fields.cluster) != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_setClusterStatusMaxCapacityMetrics(t *testing.T) {
	type fields struct {
		dataplaneClusterConfig *config.DataplaneClusterConfig
		providersConfig        config.ProviderConfig
	}
	testOsdConfig := config.NewDataplaneClusterConfig()
	testOsdConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{
		dpMock.BuildManualCluster(supportedInstanceType),
		config.ManualCluster{
			Schedulable: false,
		},
	})
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Should return no error and set metrics for supported instance type for given cluster config",
			fields: fields{
				dataplaneClusterConfig: testOsdConfig,
				providersConfig:        supportedProviders,
			},
			wantErr: false,
		},
		{
			name: "Should return error when providersConfig doesn't support instance type from dataplaneClusterConfig",
			fields: fields{
				dataplaneClusterConfig: testOsdConfig,
				providersConfig:        config.ProviderConfig{},
			},
			wantErr: true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					DataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
					SupportedProviders:     &tt.fields.providersConfig,
				},
			}
			Expect(c.setClusterStatusMaxCapacityMetrics() != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_setClusterStatusCountMetrics(t *testing.T) {
	type fields struct {
		clusterService services.ClusterService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should return an error when error is returned from CountByStatus",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CountByStatusFunc: func([]api.ClusterStatus) ([]services.ClusterStatusCount, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to count by status")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should successfully set cluster status count metrics",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CountByStatusFunc: func([]api.ClusterStatus) ([]services.ClusterStatusCount, *apiErrors.ServiceError) {
						return []services.ClusterStatusCount{
							{
								Status: api.ClusterReady,
								Count:  1,
							},
						}, nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
				},
			}
			Expect(c.setClusterStatusCountMetrics() != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_setKafkaPerClusterCountMetrics(t *testing.T) {
	type fields struct {
		clusterService services.ClusterService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should not return an error with nil counters and no error returned from FindKafkaInstanceCount",
			fields: fields{
				clusterService: &services.ClusterServiceMock{FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, *apiErrors.ServiceError) {
					return nil, nil
				}},
			},
			wantErr: false,
		},
		{
			name: "should return an error when error is returned from FindKafkaInstanceCount",
			fields: fields{
				clusterService: &services.ClusterServiceMock{FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, *apiErrors.ServiceError) {
					return nil, apiErrors.GeneralError("failed to find kafka instance count")
				}},
			},
			wantErr: true,
		},
		{
			name: "should return an error when error is returned from GetExternalID",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, *apiErrors.ServiceError) {
						return []services.ResKafkaInstanceCount{
							{
								Clusterid: "test02",
								Count:     1,
							},
						}, nil
					},
					GetExternalIDFunc: func(clusterId string) (string, *apiErrors.ServiceError) {
						return "", apiErrors.GeneralError("failed to get external ID")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should successfully set kafka per cluster count metrics",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, *apiErrors.ServiceError) {
						return []services.ResKafkaInstanceCount{
							{
								Clusterid: "test02",
								Count:     1,
							},
						}, nil
					},
					GetExternalIDFunc: func(clusterId string) (string, *apiErrors.ServiceError) {
						return "cluster-id", nil
					},
				},
			},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
				},
			}
			Expect(c.setKafkaPerClusterCountMetrics() != nil).To(Equal(tt.wantErr))
		})
	}
}
