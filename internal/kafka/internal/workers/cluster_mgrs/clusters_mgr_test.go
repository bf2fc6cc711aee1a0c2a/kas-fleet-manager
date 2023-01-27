package cluster_mgrs

import (
	"context"
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services/kafka_tls_certificate_management"
	dpMock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/data_plane"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"
	"github.com/onsi/gomega"
	authv1 "github.com/openshift/api/authorization/v1"
	userv1 "github.com/openshift/api/user/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/api/pkg/operators/v1alpha2"
	errors "github.com/pkg/errors"
	k8sCoreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	strimziAddonID                = "managed-kafka-test"
	clusterLoggingOperatorAddonID = "cluster-logging-operator-test"
	supportedInstanceType         = "developer"
	acceptedCluster               = api.Cluster{
		Status: api.ClusterAccepted,
	}
	readyCluster = api.Cluster{
		Status:      api.ClusterReady,
		ClusterType: api.ManagedDataPlaneClusterType.String(),
	}
	clusterWaitingForKasFleetShardOperator = api.Cluster{
		Status:      api.ClusterWaitingForKasFleetShardOperator,
		ClusterType: api.ManagedDataPlaneClusterType.String(),
	}
	keycloakRealmConfig = keycloak.KeycloakRealmConfig{
		ValidIssuerURI: "https://foo.bar",
	}
	enterpriseAcceptedCluster = api.Cluster{
		Status:      api.ClusterAccepted,
		ClusterType: api.EnterpriseDataPlaneClusterType.String(),
	}
	provisioningCluster = api.Cluster{
		Status:      api.ClusterProvisioning,
		ClusterType: api.ManagedDataPlaneClusterType.String(),
	}
	enterpriseProvisioningCluster = api.Cluster{
		Status:      api.ClusterProvisioning,
		ClusterType: api.EnterpriseDataPlaneClusterType.String(),
		ClusterDNS:  "some-cluster-dns",
	}
	provisionedCluster = api.Cluster{
		Status:      api.ClusterProvisioned,
		ClusterType: api.ManagedDataPlaneClusterType.String(),
	}
	enterpriseProvisionedCluster = api.Cluster{
		Status:      api.ClusterProvisioned,
		ClusterType: api.EnterpriseDataPlaneClusterType.String(),
		ClusterDNS:  "some-cluster-dns",
	}
	observabilityConfig = &observatorium.ObservabilityConfiguration{
		DexUrl:                             "http://dummy",
		DexUsername:                        "dummy",
		DexPassword:                        "dummy",
		DexSecret:                          "dummy",
		DexSecretFile:                      "dummy",
		DexPasswordFile:                    "dummy",
		RedHatSsoGatewayUrl:                "http://dummy",
		RedHatSsoAuthServerUrl:             "http://dummy",
		RedHatSsoRealm:                     "dummy",
		RedHatSsoTenant:                    "dummy",
		RedHatSsoTokenRefresherUrl:         "http://dummy",
		MetricsClientId:                    "dummy",
		MetricsClientIdFile:                "dummy",
		MetricsSecret:                      "dummy",
		MetricsSecretFile:                  "dummy",
		LogsClientId:                       "dummy",
		LogsClientIdFile:                   "dummy",
		LogsSecret:                         "dummy",
		LogsSecretFile:                     "dummy",
		ObservatoriumGateway:               "http://dummy",
		ObservatoriumTenant:                "dummy",
		AuthType:                           "redhat",
		AuthToken:                          "dummy",
		AuthTokenFile:                      "dummy",
		ObservabilityConfigTag:             "main",
		ObservabilityConfigRepo:            "dummy",
		ObservabilityConfigChannel:         "resources",
		ObservabilityConfigAccessToken:     "dummy",
		ObservabilityConfigAccessTokenFile: "dummy",
	}
	dataplaneClusterConfig = &config.DataplaneClusterConfig{
		ImagePullDockerConfigContent: "dummy",
		StrimziOperatorOLMConfig: config.OperatorInstallationConfig{
			Namespace:               "dummy",
			IndexImage:              "dummy",
			Package:                 "dummy",
			SubscriptionChannel:     "dummy",
			SubscriptionConfigFile:  "dummy",
			SubscriptionStartingCSV: "dummy",
		},
		KasFleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
			Namespace:               "dummy",
			IndexImage:              "dummy",
			Package:                 "dummy",
			SubscriptionChannel:     "dummy",
			SubscriptionConfigFile:  "dummy",
			SubscriptionStartingCSV: "dummy",
		},
		ObservabilityOperatorOLMConfig: config.OperatorInstallationConfig{
			Namespace:               "dummy",
			IndexImage:              "dummy",
			Package:                 "dummy",
			SubscriptionChannel:     "dummy",
			SubscriptionConfigFile:  "dummy",
			SubscriptionStartingCSV: "dummy",
		},
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{}

			g.Expect(c.GetID()).To(gomega.Equal(tt.want))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := NewClusterManager(ClusterManagerOptions{})

			g.Expect(c.GetWorkerType()).To(gomega.Equal(tt.want))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := NewClusterManager(ClusterManagerOptions{})
			c.SetIsRunning(tt.args.isRunning)
			g.Expect(c.IsRunning()).To(gomega.Equal(tt.want))
		})
	}
}

func TestClusterManager_reconcile(t *testing.T) {
	type fields struct {
		clusterService         services.ClusterService
		dataplaneClusterConfig *config.DataplaneClusterConfig
		supportedProviders     *config.ProviderConfig
		OCMConfig              *ocm.OCMConfig
		ProviderFactory        clusters.ProviderFactory
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
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
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
				OCMConfig: &ocm.OCMConfig{
					SelfToken: "selfToken",
				},
				ProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{GetClusterResourceQuotaCostsFunc: func() ([]types.QuotaCost, error) {
							return []types.QuotaCost{}, nil
						}}, nil
					},
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:         tt.fields.clusterService,
					DataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
					SupportedProviders:     tt.fields.supportedProviders,
					OCMConfig:              tt.fields.OCMConfig,
					ProviderFactory:        tt.fields.ProviderFactory,
				},
			}

			g.Expect(len(c.Reconcile()) > 0).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_processMetrics(t *testing.T) {
	type fields struct {
		clusterService         services.ClusterService
		dataplaneClusterConfig *config.DataplaneClusterConfig
		supportedProviders     *config.ProviderConfig
		ocmConfig              *ocm.OCMConfig
		providerFactory        clusters.ProviderFactory
	}
	tests := []struct {
		name     string
		fields   fields
		wantErrs []error
	}{
		{
			name: "should return one error if setClusterStatusCountMetrics fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CountByStatusFunc: func([]api.ClusterStatus) ([]services.ClusterStatusCount, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to count by status")
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
						return nil, nil
					},
				},
				ocmConfig: &ocm.OCMConfig{},
			},
			wantErrs: []error{errors.New("failed to set cluster status count metrics: KAFKAS-MGMT-9: failed to count by status")},
		},
		{
			name: "should return one error if setKafkaPerClusterCountMetrics fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CountByStatusFunc: func([]api.ClusterStatus) ([]services.ClusterStatusCount, *apiErrors.ServiceError) {
						return nil, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
						return nil, errors.New("failed to find kafka instance count")
					},
				},
				ocmConfig: &ocm.OCMConfig{},
			},
			wantErrs: []error{errors.New("failed to set kafka per cluster count metrics: failed to find kafka instance count")},
		},
		{
			name: "should return one error if setClusterProviderResourceQuotaMetrics fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CountByStatusFunc: func([]api.ClusterStatus) ([]services.ClusterStatusCount, *apiErrors.ServiceError) {
						return nil, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
						return nil, nil
					},
				},
				ocmConfig: &ocm.OCMConfig{
					SelfToken: "test-token",
				},
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							GetClusterResourceQuotaCostsFunc: func() ([]types.QuotaCost, error) {
								return nil, errors.New("failed to get quota costs")
							},
						}, nil
					},
				},
			},
			wantErrs: []error{errors.New("failed to set cluster provider resource quota metrics: failed to get quota costs")},
		},
		{
			name: "should return multiple errors if all cluster metrics fails to process",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CountByStatusFunc: func([]api.ClusterStatus) ([]services.ClusterStatusCount, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to count by status")
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
						return nil, errors.New("failed to find kafka instance count")
					},
				},
				ocmConfig: &ocm.OCMConfig{
					SelfToken: "test-token",
				},
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							GetClusterResourceQuotaCostsFunc: func() ([]types.QuotaCost, error) {
								return nil, errors.New("failed to get quota costs")
							},
						}, nil
					},
				},
			},
			wantErrs: []error{
				errors.New("failed to set cluster status count metrics: KAFKAS-MGMT-9: failed to count by status"),
				errors.New("failed to set kafka per cluster count metrics: failed to find kafka instance count"),
				errors.New("failed to set cluster provider resource quota metrics: failed to get quota costs"),
			},
		},
		{
			name: "should return no errors if all cluster metrics process successfully",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					CountByStatusFunc: func([]api.ClusterStatus) ([]services.ClusterStatusCount, *apiErrors.ServiceError) {
						return nil, nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
						return nil, nil
					},
				},
				ocmConfig: &ocm.OCMConfig{
					SelfToken: "test-token",
				},
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							GetClusterResourceQuotaCostsFunc: func() ([]types.QuotaCost, error) {
								return nil, nil
							},
						}, nil
					},
				},
			},
			wantErrs: []error{},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:         tt.fields.clusterService,
					DataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
					SupportedProviders:     tt.fields.supportedProviders,
					OCMConfig:              tt.fields.ocmConfig,
					ProviderFactory:        tt.fields.providerFactory,
				},
			}
			// processMetrics accumulates all the errors encountered during metrics processing in an array.
			gotErrors := c.processMetrics()
			g.Expect(len(gotErrors)).To(gomega.Equal(len(tt.wantErrs)), "Errors received: %v", gotErrors)
			if len(gotErrors) > 0 {
				for i, err := range gotErrors {
					g.Expect(err.Error()).To(gomega.Equal(tt.wantErrs[i].Error()))
				}
			}
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
		{
			name: "should succeed if no errors are encountered when dealing with enterprise cluster",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							enterpriseAcceptedCluster,
						}, nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should fail if an error is encountered when dealing with enterprise cluster",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							enterpriseAcceptedCluster,
						}, nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to update cluster")
					},
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
				},
			}
			g.Expect(len(c.processAcceptedClusters()) > 0).To(gomega.Equal(tt.wantErr))
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
							provisioningCluster,
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
							provisioningCluster,
						}, nil
					},
					CheckClusterStatusFunc: func(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError) {
						return &provisioningCluster, nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should succeed if no errors are encountered when dealing with enterprise cluster",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							enterpriseProvisioningCluster,
						}, nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should fail if an error is encountered when dealing with enterprise cluster",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							enterpriseProvisioningCluster,
						}, nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to update cluster")
					},
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
				},
			}
			g.Expect(len(c.processProvisioningClusters()) > 0).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_processProvisionedClusters(t *testing.T) {
	type fields struct {
		clusterService                       services.ClusterService
		osdIdpKeycloakService                sso.OSDKeycloakService
		dataplaneClusterConfig               *config.DataplaneClusterConfig
		supportedProviders                   *config.ProviderConfig
		observabilityConfiguration           *observatorium.ObservabilityConfiguration
		agentOperator                        services.KasFleetshardOperatorAddon
		providerFactory                      clusters.ProviderFactory
		kafkaTLSCertificateManagementService kafka_tls_certificate_management.KafkaTLSCertificateManagementService
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
				providerFactory: &clusters.ProviderFactoryMock{},
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
				providerFactory: &clusters.ProviderFactoryMock{},
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
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
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
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						providerMock := &clusters.ProviderMock{
							GetMachinePoolFunc: func(clusterID, id string) (*types.MachinePoolInfo, error) {
								return &types.MachinePoolInfo{
									ID:        id,
									ClusterID: clusterID,
								}, nil
							},
						}
						return providerMock, nil
					},
				},
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{TLSCertRef: "some-crt-ref", TLSKeyRef: "some-key-ref"}, nil
					},
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should succeed if no errors are encountered when dealing with enterprise cluster",
			fields: fields{
				observabilityConfiguration: observabilityConfig,
				dataplaneClusterConfig:     dataplaneClusterConfig,
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{TLSCertRef: "some-crt-ref", TLSKeyRef: "some-key-ref"}, nil
					},
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
				},
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							enterpriseProvisionedCluster,
						}, nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should apply correct resource set when dealing with an enterprise cluster",
			fields: fields{
				observabilityConfiguration: observabilityConfig,
				dataplaneClusterConfig:     dataplaneClusterConfig,
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{TLSCertRef: "some-crt-ref", TLSKeyRef: "some-key-ref"}, nil
					},
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
				},
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							enterpriseProvisionedCluster,
						}, nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						// the resource set must contain 8 items: the observability resources plus two image pull secrets
						if len(resources.Resources) != 8 {
							return apiErrors.GeneralError(fmt.Sprintf("expected 8 items in the resource set but got %v", len(resources.Resources)))
						}
						return nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should fail if an error is encountered when dealing with enterprise cluster",
			fields: fields{
				observabilityConfiguration: observabilityConfig,
				dataplaneClusterConfig:     dataplaneClusterConfig,
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							enterpriseProvisionedCluster,
						}, nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to update cluster")
					},
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return nil
					},
				},
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
				},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:                       tt.fields.clusterService,
					OsdIdpKeycloakService:                tt.fields.osdIdpKeycloakService,
					DataplaneClusterConfig:               tt.fields.dataplaneClusterConfig,
					SupportedProviders:                   tt.fields.supportedProviders,
					ObservabilityConfiguration:           tt.fields.observabilityConfiguration,
					OCMConfig:                            &ocm.OCMConfig{StrimziOperatorAddonID: strimziAddonID},
					KasFleetshardOperatorAddon:           tt.fields.agentOperator,
					ProviderFactory:                      tt.fields.providerFactory,
					KafkaTLSCertificateManagementService: tt.fields.kafkaTLSCertificateManagementService,
					KafkaConfig: &config.KafkaConfig{
						KafkaDomainName: "some-domain-name",
					},
				},
			}
			g.Expect(len(c.processProvisionedClusters()) > 0).To(gomega.Equal(tt.wantErr))
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
			name: "should not return an error if reconcileReadyCluster doesn't throw an error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							readyCluster,
						}, nil
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:         tt.fields.clusterService,
					DataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
				},
			}
			g.Expect(len(c.processReadyClusters()) > 0).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileReadyCluster(t *testing.T) {
	type fields struct {
		clusterService                       services.ClusterService
		dataplaneClusterConfig               *config.DataplaneClusterConfig
		osdIdpKeycloakService                sso.OSDKeycloakService
		kasFleetshardOperatorAddon           services.KasFleetshardOperatorAddon
		observabilityConfiguration           *observatorium.ObservabilityConfiguration
		kafkaTLSCertificateManagementService kafka_tls_certificate_management.KafkaTLSCertificateManagementService
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{}, nil
					},
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
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
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{}, nil
					},
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
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
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "", apiErrors.GeneralError("failed to register osd cluster client in sso")
					},
				},
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{}, nil
					},
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
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
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{}, nil
					},
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
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
			name: "should fail if certificate management fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: true,
					DataPlaneClusterScalingType:           config.ManualScaling,
					ClusterConfig:                         &config.ClusterConfig{},
				},
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{}, errors.New("some errors")
					},
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
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
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{
							TLSCertRef: "some-crt-ref",
							TLSKeyRef:  "some-key-ref",
						}, nil
					},
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
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
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:                       tt.fields.clusterService,
					DataplaneClusterConfig:               tt.fields.dataplaneClusterConfig,
					OCMConfig:                            &ocm.OCMConfig{StrimziOperatorAddonID: strimziAddonID},
					OsdIdpKeycloakService:                tt.fields.osdIdpKeycloakService,
					KasFleetshardOperatorAddon:           tt.fields.kasFleetshardOperatorAddon,
					ObservabilityConfiguration:           tt.fields.observabilityConfiguration,
					KafkaTLSCertificateManagementService: tt.fields.kafkaTLSCertificateManagementService,
					KafkaConfig: &config.KafkaConfig{
						KafkaDomainName: "some-domaine",
					},
				},
			}
			g.Expect(c.reconcileReadyCluster(tt.args.cluster) != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileWaitingForKasFleetshardOperatorCluster(t *testing.T) {
	type fields struct {
		clusterService                       services.ClusterService
		dataplaneClusterConfig               *config.DataplaneClusterConfig
		osdIdpKeycloakService                sso.OSDKeycloakService
		kasFleetshardOperatorAddon           services.KasFleetshardOperatorAddon
		observabilityConfiguration           *observatorium.ObservabilityConfiguration
		kafkaTLSCertificateManagementService kafka_tls_certificate_management.KafkaTLSCertificateManagementService
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
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
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "", apiErrors.GeneralError("failed to register osd cluster client in sso")
					},
				},
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
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
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
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
			name: "should fail if certificate management fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: true,
					DataPlaneClusterScalingType:           config.ManualScaling,
					ClusterConfig:                         &config.ClusterConfig{},
				},
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{}, errors.New("some errors")
					},
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
					},
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
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{
							TLSCertRef: "some-crt-ref",
							TLSKeyRef:  "some-key-ref",
						}, nil
					},
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return false
					},
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
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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

	for _, testcase := range tests {
		tt := testcase
		g := gomega.NewWithT(t)
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:                       tt.fields.clusterService,
					DataplaneClusterConfig:               tt.fields.dataplaneClusterConfig,
					OCMConfig:                            &ocm.OCMConfig{StrimziOperatorAddonID: strimziAddonID},
					OsdIdpKeycloakService:                tt.fields.osdIdpKeycloakService,
					KasFleetshardOperatorAddon:           tt.fields.kasFleetshardOperatorAddon,
					ObservabilityConfiguration:           tt.fields.observabilityConfiguration,
					KafkaTLSCertificateManagementService: tt.fields.kafkaTLSCertificateManagementService,
					KafkaConfig: &config.KafkaConfig{
						KafkaDomainName: "some-domaine",
					},
				},
			}
			g.Expect(c.reconcileWaitingForKasFleetshardOperatorCluster(tt.args.cluster) != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileProvisionedCluster(t *testing.T) {
	type fields struct {
		clusterService                       services.ClusterService
		dataplaneClusterConfig               *config.DataplaneClusterConfig
		osdIdpKeycloakService                sso.OSDKeycloakService
		observabilityConfiguration           *observatorium.ObservabilityConfiguration
		agentOperator                        services.KasFleetshardOperatorAddon
		providerFactory                      clusters.ProviderFactory
		kafkaTLSCertificateManagementService kafka_tls_certificate_management.KafkaTLSCertificateManagementService
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
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "secret", nil
					},
					GetRealmConfigFunc: func() *keycloak.KeycloakRealmConfig {
						return &keycloakRealmConfig
					},
				},
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
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
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to update status and client")
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return false
					},
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{}, nil
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
			name: "should fail if reconcileClusterMachinePools fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: true,
					DataPlaneClusterScalingType:           config.AutoScaling,
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
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to update status and client")
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return false
					},
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{}, nil
					},
				},
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{}, fmt.Errorf("test error")
					},
				},
			},
			args: args{
				cluster: readyCluster,
			},
			wantErr: true,
		},
		{
			name: "should fail if certificate management fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					EnableReadyDataPlaneClustersReconcile: true,
					DataPlaneClusterScalingType:           config.AutoScaling,
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
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{}, errors.New("some error")
					},
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return false
					},
				},
				observabilityConfiguration: &observatorium.ObservabilityConfiguration{},
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{}, nil
					},
				},
			},
			args: args{
				cluster: readyCluster,
			},
			wantErr: true,
		},
		{
			name: "should fail when updating status in database fails after all elements are reconciled successfully",
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
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return apiErrors.GeneralError("some errors")
					},
				},
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{}, nil
					},
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return true
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
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{
							TLSCertRef: "some-crt-ref",
							TLSKeyRef:  "some-key-ref",
						}, nil
					},
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return false
					},
				},
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:                       tt.fields.clusterService,
					DataplaneClusterConfig:               tt.fields.dataplaneClusterConfig,
					OCMConfig:                            &ocm.OCMConfig{StrimziOperatorAddonID: strimziAddonID},
					OsdIdpKeycloakService:                tt.fields.osdIdpKeycloakService,
					KasFleetshardOperatorAddon:           tt.fields.agentOperator,
					ObservabilityConfiguration:           tt.fields.observabilityConfiguration,
					ProviderFactory:                      tt.fields.providerFactory,
					KafkaTLSCertificateManagementService: tt.fields.kafkaTLSCertificateManagementService,
					KafkaConfig: &config.KafkaConfig{
						KafkaDomainName: "some-domaine",
					},
				},
			}
			g.Expect(c.reconcileProvisionedCluster(tt.args.cluster) != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_processWaitingForKasFleetshardOperatorClusters(t *testing.T) {
	type fields struct {
		clusterService                       services.ClusterService
		dataplaneClusterConfig               *config.DataplaneClusterConfig
		observabilityConfiguration           *observatorium.ObservabilityConfiguration
		osdIdpKeycloakService                sso.OSDKeycloakService
		kasFleetshardOperatorAddon           services.KasFleetshardOperatorAddon
		kafkaTLSCertificateManagementService kafka_tls_certificate_management.KafkaTLSCertificateManagementService
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
			name: "should return an error if it occurs during processing waiting for kas-fleetshard clusters",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListByStatusFunc: func(api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{
							clusterWaitingForKasFleetShardOperator,
						}, nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return apiErrors.GeneralError("some errors")
					},
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to apply resources")
					},
				},
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return false
					},
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{}, nil
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
				kafkaTLSCertificateManagementService: &kafka_tls_certificate_management.KafkaTLSCertificateManagementServiceMock{
					ManageCertificateFunc: func(ctx context.Context, domain string) (kafka_tls_certificate_management.CertificateManagementOutput, error) {
						return kafka_tls_certificate_management.CertificateManagementOutput{TLSCertRef: "some-crt-ref", TLSKeyRef: "some-key-ref"}, nil
					},
					IsAutomaticCertificateManagementEnabledFunc: func() bool {
						return false
					},
				},
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:                       tt.fields.clusterService,
					DataplaneClusterConfig:               tt.fields.dataplaneClusterConfig,
					ObservabilityConfiguration:           tt.fields.observabilityConfiguration,
					OCMConfig:                            &ocm.OCMConfig{StrimziOperatorAddonID: strimziAddonID},
					OsdIdpKeycloakService:                tt.fields.osdIdpKeycloakService,
					KasFleetshardOperatorAddon:           tt.fields.kasFleetshardOperatorAddon,
					KafkaTLSCertificateManagementService: tt.fields.kafkaTLSCertificateManagementService,
					KafkaConfig: &config.KafkaConfig{
						KafkaDomainName: "some-domain-name",
					},
				},
			}
			g.Expect(len(c.processWaitingForKasFleetshardOperatorClusters()) > 0).To(gomega.Equal(tt.wantErr))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					KasFleetshardOperatorAddon: tt.fields.kasFleetshardOperatorAddon,
				},
			}

			g.Expect(c.reconcileKasFleetshardOperator(tt.arg) != nil).To(gomega.Equal(tt.wantErr))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
				},
			}
			_, err := c.reconcileClusterStatus(tt.args.cluster)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
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
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
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
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					ObservabilityConfiguration: &observatorium.ObservabilityConfiguration{},
					DataplaneClusterConfig:     config.NewDataplaneClusterConfig(),
				},
			}

			g.Expect(c.reconcileAcceptedCluster(&acceptedCluster) != nil).To(gomega.Equal(tt.wantErr))
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
		arg     api.Cluster
		want    bool
		wantErr bool
	}{
		{
			name: "should not return an error when operators installation have been accepted but not ready yet",
			fields: fields{
				agentOperator: &services.KasFleetshardOperatorAddonMock{
					ProvisionFunc: func(cluster api.Cluster) (bool, services.ParameterList, *apiErrors.ServiceError) {
						return false, services.ParameterList{}, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					InstallStrimziFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
					InstallClusterLoggingFunc: func(cluster *api.Cluster, params []types.Parameter) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					UpdateStatusFunc: nil, // set to nil as it should not be called as operators installation status is false
				},
			},
			arg: api.Cluster{
				ClusterID: "test-cluster-id",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "should return an error operators installation have been accepted but not ready yet but updating of client id and client secrets fails in the database",
			fields: fields{
				agentOperator: &services.KasFleetshardOperatorAddonMock{
					ProvisionFunc: func(cluster api.Cluster) (bool, services.ParameterList, *apiErrors.ServiceError) {
						return false, services.ParameterList{}, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					InstallStrimziFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
					InstallClusterLoggingFunc: func(cluster *api.Cluster, params []types.Parameter) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return apiErrors.GeneralError("some errors")
					},
					UpdateStatusFunc: nil, // set to nil as it should not be called as operators installation status is false
				},
			},
			arg: api.Cluster{
				ClusterID: "test-cluster-id",
			},
			wantErr: true,
		},
		{
			name: "should not store the client_id and client_secret if they are already set",
			fields: fields{
				agentOperator: &services.KasFleetshardOperatorAddonMock{
					ProvisionFunc: func(cluster api.Cluster) (bool, services.ParameterList, *apiErrors.ServiceError) {
						return false, services.ParameterList{}, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					InstallStrimziFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
					InstallClusterLoggingFunc: func(cluster *api.Cluster, params []types.Parameter) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
					UpdateFunc:       nil, // set to nil as this should not be called since client_id and client_secret are already set
					UpdateStatusFunc: nil, // set to nil as it should not be called as operators installation status is false
				},
			},
			arg: api.Cluster{
				ClusterID:    "test-cluster-id",
				ClientID:     "some-client-id",
				ClientSecret: "some-client-secret",
			},
			want:    false,
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
			arg: api.Cluster{
				ClusterID: "test-cluster-id",
			},
			want:    false,
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
			arg: api.Cluster{
				ClusterID: "test-cluster-id",
			},
			want:    false,
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
			arg: api.Cluster{
				ClusterID: "test-cluster-id",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "successful reconciles operators installation are ready",
			fields: fields{
				agentOperator: &services.KasFleetshardOperatorAddonMock{
					ProvisionFunc: func(cluster api.Cluster) (bool, services.ParameterList, *apiErrors.ServiceError) {
						return true, services.ParameterList{}, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					InstallStrimziFunc: func(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
					InstallClusterLoggingFunc: func(cluster *api.Cluster, params []types.Parameter) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
			},
			arg: api.Cluster{
				ClusterID: "test-cluster-id",
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					OCMConfig:                  &ocm.OCMConfig{StrimziOperatorAddonID: strimziAddonID, ClusterLoggingOperatorAddonID: clusterLoggingOperatorAddonID},
					ClusterService:             tt.fields.clusterService,
					KasFleetshardOperatorAddon: tt.fields.agentOperator,
				},
			}
			reconciled, err := c.reconcileAddonOperator(tt.arg)
			if err != nil && !tt.wantErr {
				t.Errorf("reconcileAddonOperator() error = %v, wantErr %v", err, tt.wantErr)
			}
			if reconciled != tt.want {
				t.Errorf("reconcileAddonOperator() got = %v, want %v", reconciled, tt.want)
			}
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
				Image:      clusterConfig.ObservabilityOperatorOLMConfig.IndexImage,
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
				StartingCSV:            clusterConfig.ObservabilityOperatorOLMConfig.SubscriptionStartingCSV,
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
				Name:   strimziNamespace,
				Labels: clusters.StrimziOperatorCommonLabels(),
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
	g := gomega.NewWithT(t)

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
		ObservabilityOperatorOLMConfig: config.OperatorInstallationConfig{
			IndexImage:              "quay.io/rhoas/observability-operator-index:v4.0.2",
			SubscriptionStartingCSV: "observability-operator.v4.0.2",
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
						g.Expect(resources).To(gomega.Equal(want))
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
						g.Expect(resources).To(gomega.Equal(want))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					SupportedProviders:         &config.ProviderConfig{},
					ObservabilityConfiguration: &observabilityConfig,
					DataplaneClusterConfig:     &clusterConfig,
					OCMConfig:                  &ocm.OCMConfig{},
				},
			}

			g.Expect(c.reconcileClusterResources(tt.arg) != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileClusterIdentityProvider(t *testing.T) {
	type fields struct {
		clusterService         services.ClusterService
		osdIdpKeycloakService  sso.OSDKeycloakService
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
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: nil, // setting it to nill so that it is not called
					GetRealmConfigFunc:      nil, // setting it to nill so that it is not called
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
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: nil, // setting it to nill so that it is not called
					GetRealmConfigFunc:      nil, // setting it to nill so that it is not called
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
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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
				osdIdpKeycloakService: &sso.OSDKeycloakServiceMock{
					RegisterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:         tt.fields.clusterService,
					OsdIdpKeycloakService:  tt.fields.osdIdpKeycloakService,
					DataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
				},
			}

			g.Expect(c.reconcileClusterIdentityProvider(tt.arg) != nil).To(gomega.Equal(tt.wantErr))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
				},
			}

			g.Expect(c.reconcileClusterDNS(tt.arg) != nil).To(gomega.Equal(tt.wantErr))
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
					ListNonEnterpriseClusterIDsFunc: func() ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{{ClusterID: "test02"}}, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
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
					ListNonEnterpriseClusterIDsFunc: func() ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{{ClusterID: "test02"}}, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					UpdateMultiClusterStatusFunc: func(clusterIDs []string, status api.ClusterStatus) *apiErrors.ServiceError {
						return nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
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
					ListNonEnterpriseClusterIDsFunc: func() ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{{ClusterID: "test02"}}, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					UpdateMultiClusterStatusFunc: func(clusterIDs []string, status api.ClusterStatus) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to update multi cluster status")
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
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
					ListNonEnterpriseClusterIDsFunc: func() ([]api.Cluster, *apiErrors.ServiceError) {
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
					ListNonEnterpriseClusterIDsFunc: func() ([]api.Cluster, *apiErrors.ServiceError) {
						return []api.Cluster{{ClusterID: "test02"}}, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
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
					ListNonEnterpriseClusterIDsFunc: func() ([]api.Cluster, *apiErrors.ServiceError) {
						return nil, &apiErrors.ServiceError{}
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return nil
					},
					UpdateMultiClusterStatusFunc: func(clusterIDs []string, status api.ClusterStatus) *apiErrors.ServiceError {
						return nil
					},
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
						return []services.ResKafkaInstanceCount{}, nil
					},
				},
				DataplaneClusterConfig: testOsdConfig,
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					DataplaneClusterConfig: tt.fields.DataplaneClusterConfig,
					ClusterService:         tt.fields.clusterService,
				},
			}
			g.Expect(len(c.reconcileClusterWithManualConfig()) > 0).To(gomega.Equal(tt.wantErr))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					DataplaneClusterConfig: tt.fields.dataplaneClusterConfig,
					ClusterService:         tt.fields.clusterService,
				},
			}
			g.Expect(c.reconcileClusterInstanceType(tt.fields.cluster) != nil).To(gomega.Equal(tt.wantErr))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
				},
			}
			g.Expect(c.setClusterStatusCountMetrics() != nil).To(gomega.Equal(tt.wantErr))
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
				clusterService: &services.ClusterServiceMock{FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
					return nil, nil
				}},
			},
			wantErr: false,
		},
		{
			name: "should return an error when error is returned from FindKafkaInstanceCount",
			fields: fields{
				clusterService: &services.ClusterServiceMock{FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
					return nil, apiErrors.GeneralError("failed to find kafka instance count")
				}},
			},
			wantErr: true,
		},
		{
			name: "should return an error when error is returned from GetExternalID",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
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
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
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
		{
			name: "should not call GetExternalIDFunc when Clusterid is empty",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindKafkaInstanceCountFunc: func(clusterIDs []string) ([]services.ResKafkaInstanceCount, error) {
						return []services.ResKafkaInstanceCount{
							{
								Clusterid: "",
								Count:     1,
							},
						}, nil
					},
					GetExternalIDFunc: nil, // set to nil as it never be called
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
				},
			}
			g.Expect(c.setKafkaPerClusterCountMetrics() != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileClusterMachinePool(t *testing.T) {

	type fields struct {
		dataplaneClusterConfig *config.DataplaneClusterConfig
		providerFactory        clusters.ProviderFactory
		clusterService         services.ClusterService
	}
	tests := []struct {
		name    string
		fields  fields
		arg     api.Cluster
		want    bool
		wantErr bool
	}{
		{
			name: "should return true if all machinepools for the given cluster already exist",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
				},
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							GetMachinePoolFunc: func(clusterID, id string) (*types.MachinePoolInfo, error) {
								return &types.MachinePoolInfo{ID: id, ClusterID: clusterID}, nil
							},
							CreateMachinePoolFunc: func(request *types.MachinePoolRequest) (*types.MachinePoolRequest, error) {
								return request, nil
							},
						}, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
			},
			arg: api.Cluster{
				ClusterID:             "test-cluster-id",
				CloudProvider:         cloudproviders.AWS.String(),
				SupportedInstanceType: "developer,standard",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "should return false and error if all machinepools for the given cluster already exist but update to database fails",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
				},
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							GetMachinePoolFunc: func(clusterID, id string) (*types.MachinePoolInfo, error) {
								return &types.MachinePoolInfo{ID: id, ClusterID: clusterID}, nil
							},
							CreateMachinePoolFunc: func(request *types.MachinePoolRequest) (*types.MachinePoolRequest, error) {
								return request, nil
							},
						}, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return &apiErrors.ServiceError{}
					},
				},
			},
			arg: api.Cluster{
				ClusterID:             "test-cluster-id",
				CloudProvider:         cloudproviders.AWS.String(),
				SupportedInstanceType: "developer,standard",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "should return true if new machinepools are created and the creation succeeds",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					DynamicScalingConfig: config.DynamicScalingConfig{
						ComputeMachinePerCloudProvider: map[cloudproviders.CloudProviderID]config.ComputeMachinesConfig{
							cloudproviders.AWS: {
								KafkaWorkloadPerInstanceType: map[string]config.ComputeMachineConfig{
									api.DeveloperTypeSupport.String(): {
										ComputeMachineType: "testmachinetype",
										ComputeNodesAutoscaling: &config.ComputeNodesAutoscalingConfig{
											MaxComputeNodes: 3,
										},
									},
								},
							},
						},
					},
				},
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							GetMachinePoolFunc: func(clusterID, id string) (*types.MachinePoolInfo, error) {
								// We simulate that a kafka-developer machinepool has still not been created
								// and that the kafka-standard machinepool is already created
								if id == "kafka-developer" {
									return nil, nil
								}
								return &types.MachinePoolInfo{ID: id, ClusterID: clusterID}, nil
							},
							CreateMachinePoolFunc: func(request *types.MachinePoolRequest) (*types.MachinePoolRequest, error) {
								return request, nil
							},
						}, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
			},
			arg: api.Cluster{
				ClusterID:             "test-cluster-id",
				SupportedInstanceType: "developer,standard",
				CloudProvider:         cloudproviders.AWS.String(),
			},
			want:    true,
			wantErr: false,
		},

		{
			name: "should return true if data plane cluster scaling is set to manual",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.ManualScaling,
				},
			},
			arg: api.Cluster{
				ClusterID:             "test-cluster-id",
				SupportedInstanceType: "developer,standard",
				CloudProvider:         cloudproviders.AWS.String(),
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "should return true if data plane cluster scaling is set to none",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.NoScaling,
				},
			},
			arg: api.Cluster{
				ClusterID:             "test-cluster-id",
				SupportedInstanceType: "developer,standard",
				CloudProvider:         cloudproviders.AWS.String(),
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "should return an error if the provider cannot be retrieved",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
				},
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{}, fmt.Errorf("test error returning provider")
					},
				},
			},
			arg: api.Cluster{
				ClusterID:             "test-cluster-id",
				SupportedInstanceType: "developer,standard",
				CloudProvider:         cloudproviders.AWS.String(),
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "should return an error if an error is returned when trying to retrieve a machinepool from the provider",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
				},
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							GetMachinePoolFunc: func(clusterID, id string) (*types.MachinePoolInfo, error) {
								return nil, fmt.Errorf("test error returning machinepool from provider")
							},
						}, nil
					},
				},
			},
			arg: api.Cluster{
				ClusterID:             "test-cluster-id",
				CloudProvider:         cloudproviders.AWS.String(),
				SupportedInstanceType: "developer,standard",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "should return an error if an error is returned when trying to create a machinepool",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
				},
				providerFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							GetMachinePoolFunc: func(clusterID, id string) (*types.MachinePoolInfo, error) {
								return nil, nil
							},
							CreateMachinePoolFunc: func(request *types.MachinePoolRequest) (*types.MachinePoolRequest, error) {
								return nil, fmt.Errorf("test error creating machinepool using the provider")
							},
						}, nil
					},
				},
			},
			arg: api.Cluster{
				ClusterID:             "test-cluster-id",
				CloudProvider:         cloudproviders.AWS.String(),
				SupportedInstanceType: "developer",
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		test := tc
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					DataplaneClusterConfig: test.fields.dataplaneClusterConfig,
					ProviderFactory:        test.fields.providerFactory,
					ClusterService:         test.fields.clusterService,
				},
			}
			reconciled, err := c.reconcileClusterMachinePools(test.arg)
			gotErr := err != nil
			g.Expect(gotErr).To(gomega.Equal(test.wantErr))
			g.Expect(reconciled).To(gomega.Equal(test.want))
		})
	}

}

func TestClusterManager_reconcileDynamicCapacityInfo(t *testing.T) {
	t.Parallel()
	type fields struct {
		dataplaneClusterConfig *config.DataplaneClusterConfig
		clusterService         services.ClusterService
	}
	tests := []struct {
		name    string
		fields  fields
		arg     api.Cluster
		wantErr bool
	}{
		{
			name: "resets capacity to empty object when autoscaling is not enabled",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.ManualScaling,
				},
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						capacityInfoToBePersisted := cluster.RetrieveDynamicCapacityInfo()

						if len(capacityInfoToBePersisted) != 0 {
							t.Errorf("Persisted dynamic capacity size should be 0")
						}
						return nil
					},
				},
			},
			arg: api.Cluster{
				ClusterID:             "test-cluster-id",
				SupportedInstanceType: "developer,standard",
				CloudProvider:         "cp",
				DynamicCapacityInfo:   api.JSON([]byte(`{"key1":{"max_nodes":1,"max_units":1,"remaining_units":1}}`)),
			},
			wantErr: false,
		},
		{
			name: "return no error if capacity info already there when autoscaling is enabled",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
				},
				clusterService: nil, // set to nil as it not be called
			},
			arg: api.Cluster{
				ClusterID:             "test-cluster-id",
				CloudProvider:         "cp",
				SupportedInstanceType: "developer,standard",
				DynamicCapacityInfo:   api.JSON([]byte(`{"key1":{"max_nodes":1,"max_units":1,"remaining_units":1}}`)),
			},
			wantErr: false,
		},
		{
			name: "return error when cluster service update returns an error and when autoscaling is enabled",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
				},
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						return &apiErrors.ServiceError{}
					},
				},
			},
			arg: api.Cluster{
				ClusterID:             "test-cluster-id",
				SupportedInstanceType: "developer,standard",
				CloudProvider:         "cp",
				DynamicCapacityInfo:   api.JSON([]byte(`{}`)),
			},
			wantErr: true,
		},
		{
			name: "return no error when cluster service update succeed both instance types and when autoscaling is enabled",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					DynamicScalingConfig: config.DynamicScalingConfig{
						ComputeMachinePerCloudProvider: map[cloudproviders.CloudProviderID]config.ComputeMachinesConfig{
							cloudproviders.AWS: {
								KafkaWorkloadPerInstanceType: map[string]config.ComputeMachineConfig{
									api.StandardTypeSupport.String(): {
										ComputeNodesAutoscaling: &config.ComputeNodesAutoscalingConfig{
											MaxComputeNodes: 2,
										},
									},
									api.DeveloperTypeSupport.String(): {
										ComputeNodesAutoscaling: &config.ComputeNodesAutoscalingConfig{
											MaxComputeNodes: 3,
										},
									},
								},
							},
						},
					},
				},
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						capacityInfoToBePersisted := cluster.RetrieveDynamicCapacityInfo()

						if len(capacityInfoToBePersisted) != 2 {
							t.Errorf("Persisted dynamic capacity size should be 2")
						}

						// check standard is there
						standardCapacity, standardCapacityFound := capacityInfoToBePersisted[api.StandardTypeSupport.String()]
						if !standardCapacityFound {
							t.Error("standard dynamic capacity info should be there")
						}

						if standardCapacity.MaxNodes != 2 {
							t.Error("standard max nodes should be 2")
						}

						// check developer is there
						developerCapacity, developerCapacityFound := capacityInfoToBePersisted[api.DeveloperTypeSupport.String()]
						if !developerCapacityFound {
							t.Error("developer dynamic capacity info should be there")
						}

						if developerCapacity.MaxNodes != 3 {
							t.Error("standard max nodes should be 3")
						}
						return nil
					},
				},
			},
			arg: api.Cluster{
				ClusterID:             "test-cluster-id",
				SupportedInstanceType: "developer,standard",
				CloudProvider:         cloudproviders.AWS.String(),
				DynamicCapacityInfo:   api.JSON([]byte(`{}`)),
			},
			wantErr: false,
		},
		{
			name: "return error when cluster cloud provider is missing from the dynamic scaling configuration",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					DynamicScalingConfig: config.DynamicScalingConfig{
						ComputeMachinePerCloudProvider: map[cloudproviders.CloudProviderID]config.ComputeMachinesConfig{
							"another-cloud-provider": {
								KafkaWorkloadPerInstanceType: map[string]config.ComputeMachineConfig{
									api.StandardTypeSupport.String(): {
										ComputeNodesAutoscaling: &config.ComputeNodesAutoscalingConfig{
											MaxComputeNodes: 2,
										},
									},
									api.DeveloperTypeSupport.String(): {
										ComputeNodesAutoscaling: &config.ComputeNodesAutoscalingConfig{
											MaxComputeNodes: 3,
										},
									},
								},
							},
						},
					},
				},
				clusterService: nil,
			}, // set to nil as it should never be called
			arg: api.Cluster{
				ClusterID:             "test-cluster-id",
				SupportedInstanceType: "developer,standard",
				CloudProvider:         "cp",
				DynamicCapacityInfo:   api.JSON([]byte(`{}`)),
			},
			wantErr: true,
		},
		{
			name: "return no error when cluster service update succeed when cluster supports only one instance type and when autoscaling is enabled",
			fields: fields{
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					DataPlaneClusterScalingType: config.AutoScaling,
					DynamicScalingConfig: config.DynamicScalingConfig{
						ComputeMachinePerCloudProvider: map[cloudproviders.CloudProviderID]config.ComputeMachinesConfig{
							cloudproviders.AWS: {
								KafkaWorkloadPerInstanceType: map[string]config.ComputeMachineConfig{
									api.StandardTypeSupport.String(): {
										ComputeNodesAutoscaling: &config.ComputeNodesAutoscalingConfig{
											MaxComputeNodes: 20,
										},
									},
									api.DeveloperTypeSupport.String(): {
										ComputeNodesAutoscaling: &config.ComputeNodesAutoscalingConfig{
											MaxComputeNodes: 3,
										},
									},
								},
							},
						},
					},
				},
				clusterService: &services.ClusterServiceMock{
					UpdateFunc: func(cluster api.Cluster) *apiErrors.ServiceError {
						capacityInfoToBePersisted := cluster.RetrieveDynamicCapacityInfo()

						if len(capacityInfoToBePersisted) != 1 {
							t.Errorf("Persisted dynamic capacity size should be 1")
						}

						// check standard is there
						standardCapacity, standardCapacityFound := capacityInfoToBePersisted[api.StandardTypeSupport.String()]
						if !standardCapacityFound {
							t.Error("standard max nodes should be there")
						}

						if standardCapacity.MaxNodes != 20 {
							t.Error("standard max nodes should be 20")
						}

						// check developer is not there
						_, developerCapacityFound := capacityInfoToBePersisted[api.DeveloperTypeSupport.String()]
						if developerCapacityFound {
							t.Error("develper should not be there as it is not supported by the cluster")
						}
						return nil
					},
				},
			},
			arg: api.Cluster{
				ClusterID:             "test-cluster-id",
				SupportedInstanceType: "standard",
				CloudProvider:         cloudproviders.AWS.String(),
				DynamicCapacityInfo:   api.JSON([]byte(`{}`)),
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		test := tc
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					DataplaneClusterConfig: test.fields.dataplaneClusterConfig,
					ClusterService:         test.fields.clusterService,
				},
			}
			err := c.reconcileDynamicCapacityInfo(test.arg)
			gotErr := err != nil
			g.Expect(gotErr).To(gomega.Equal(test.wantErr))
		})
	}
}

func TestClusterManager_setClusterProviderResourceQuotaMetrics(t *testing.T) {
	tests := []struct {
		name               string
		clusterManagerOpts ClusterManagerOptions
		wantErr            bool
	}{
		{
			name: "should skip metric update if ocm credentials are not specified",
			clusterManagerOpts: ClusterManagerOptions{
				OCMConfig: &ocm.OCMConfig{},
			},
			wantErr: false,
		},
		{
			name: "should skip metric update if ocm client id and secret are both not specified",
			clusterManagerOpts: ClusterManagerOptions{
				OCMConfig: &ocm.OCMConfig{
					ClientID: "test-client-id",
				},
			},
			wantErr: false,
		},
		{
			name: "should return an error if GetProvider fails",
			clusterManagerOpts: ClusterManagerOptions{
				OCMConfig: &ocm.OCMConfig{
					SelfToken: "test-token",
				},
				ProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return nil, errors.New("failed to get provider")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return an error if GetQuotaCosts fails",
			clusterManagerOpts: ClusterManagerOptions{
				OCMConfig: &ocm.OCMConfig{
					SelfToken: "test-token",
				},
				ProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							GetClusterResourceQuotaCostsFunc: func() ([]types.QuotaCost, error) {
								return nil, errors.New("failed to get quota list")
							},
						}, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should return no error if ocm client id and secret are specified and metric update is successful",
			clusterManagerOpts: ClusterManagerOptions{
				OCMConfig: &ocm.OCMConfig{
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
				},
				ProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							GetClusterResourceQuotaCostsFunc: func() ([]types.QuotaCost, error) {
								return []types.QuotaCost{
									{
										ID:         "test-quota-id",
										MaxAllowed: 1,
										Consumed:   0,
									},
								}, nil
							},
						}, nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should return no error if ocm token is specified and metric update is successful",
			clusterManagerOpts: ClusterManagerOptions{
				OCMConfig: &ocm.OCMConfig{
					SelfToken: "test-token",
				},
				ProviderFactory: &clusters.ProviderFactoryMock{
					GetProviderFunc: func(providerType api.ClusterProviderType) (clusters.Provider, error) {
						return &clusters.ProviderMock{
							GetClusterResourceQuotaCostsFunc: func() ([]types.QuotaCost, error) {
								return []types.QuotaCost{
									{
										ID:         "test-quota-id",
										MaxAllowed: 1,
										Consumed:   0,
									},
								}, nil
							},
						}, nil
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := &ClusterManager{
				ClusterManagerOptions: tc.clusterManagerOpts,
			}

			err := c.setClusterProviderResourceQuotaMetrics()
			g.Expect(err != nil).To(gomega.Equal(tc.wantErr))
		})
	}
}
