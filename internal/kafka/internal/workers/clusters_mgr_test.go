package workers

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm"
	"testing"

	ingressoperatorv1 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/ingressoperator/v1"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/types"
	authv1 "github.com/openshift/api/authorization/v1"
	projectv1 "github.com/openshift/api/project/v1"
	userv1 "github.com/openshift/api/user/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/api/pkg/operators/v1alpha2"
	errors "github.com/zgalor/weberr"
	k8sCoreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	ocmErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
)

var (
	testRegion                    = "us-west-1"
	testProvider                  = "aws"
	strimziAddonID                = "managed-kafka-test"
	clusterLoggingOperatorAddonID = "cluster-logging-operator-test"
)

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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
				},
			}
			_, err := c.reconcileClusterStatus(tt.args.cluster)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileClusterStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
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
			name: "error when getting managed kafka addon from ocm fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					InstallAddonFunc: func(cluster *api.Cluster, addonID string) (bool, *apiErrors.ServiceError) {
						return false, apiErrors.GeneralError("failed to install addon")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "strimzi addon installed successfully",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					InstallAddonFunc: func(cluster *api.Cluster, addonID string) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
					ConfigService: coreServices.NewConfigService(
						&config.ApplicationConfig{
							SupportedProviders:         &config.ProviderConfig{},
							AccessControlList:          &config.AccessControlListConfig{},
							ObservabilityConfiguration: &config.ObservabilityConfiguration{},
							OSDClusterConfig:           &config.OSDClusterConfig{},
						},
					),
					OCMConfig: &config.OCMConfig{StrimziOperatorAddonID: strimziAddonID},
				},
			}
			_, err := c.reconcileStrimziOperator(api.Cluster{
				ClusterID: "clusterId",
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileStrimziOperator() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
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
			name: "error when getting cluster logging operator addon from ocm fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					InstallAddonWithParamsFunc: func(cluster *api.Cluster, addonID string, addonParams []ocm.AddonParameter) (bool, *ocmErrors.ServiceError) {
						return false, apiErrors.GeneralError("failed to install cluster logging operator addon")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "cluster logging operator addon installed successfully",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					InstallAddonWithParamsFunc: func(cluster *api.Cluster, addonID string, addonParams []ocm.AddonParameter) (bool, *ocmErrors.ServiceError) {
						return true, nil
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
					ConfigService: coreServices.NewConfigService(
						&config.ApplicationConfig{
							SupportedProviders:         &config.ProviderConfig{},
							AccessControlList:          &config.AccessControlListConfig{},
							ObservabilityConfiguration: &config.ObservabilityConfiguration{},
							OSDClusterConfig:           &config.OSDClusterConfig{},
						},
					),
					OCMConfig: &config.OCMConfig{ClusterLoggingOperatorAddonID: clusterLoggingOperatorAddonID},
				},
			}
			_, err := c.reconcileClusterLoggingOperator(api.Cluster{
				ClusterID: "clusterId",
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcileClusterLoggingOperator() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
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
					CreateFunc: func(cluster *api.Cluster) (cls *api.Cluster, e *ocmErrors.ServiceError) {
						return cluster, nil
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
					ConfigService: coreServices.NewConfigService(
						&config.ApplicationConfig{
							AccessControlList:          &config.AccessControlListConfig{},
							ObservabilityConfiguration: &config.ObservabilityConfiguration{},
							OSDClusterConfig:           config.NewOSDClusterConfig(),
						}),
				},
			}

			clusterRequest := &api.Cluster{
				Region:        testRegion,
				CloudProvider: testProvider,
				Status:        "cluster_accepted",
			}

			err := c.reconcileAcceptedCluster(clusterRequest)
			if err != nil && !tt.wantErr {
				t.Errorf("reconcileAcceptedCluster() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClusterManager_reconcileClustersForRegions(t *testing.T) {
	type fields struct {
		providerLst     []string
		clusterService  services.ClusterService
		providersConfig config.ProviderConfig
	}

	tests := []struct {
		name    string
		wantErr bool
		fields  fields
	}{
		{
			name: "creates a missing OSD cluster request automatically",
			fields: fields{
				providerLst: []string{"us-east-1"},
				clusterService: &services.ClusterServiceMock{
					ListGroupByProviderAndRegionFunc: func(providers []string, regions []string, status []string) (m []*services.ResGroupCPRegion, e *ocmErrors.ServiceError) {
						var res []*services.ResGroupCPRegion
						return res, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return nil
					},
				},
				providersConfig: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name: "aws",
								Regions: config.RegionList{
									config.Region{
										Name: "us-east-1",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "failed to create OSD request",
			fields: fields{
				providerLst: []string{"us-east-1"},
				clusterService: &services.ClusterServiceMock{
					ListGroupByProviderAndRegionFunc: func(providers []string, regions []string, status []string) (m []*services.ResGroupCPRegion, e *ocmErrors.ServiceError) {
						var res []*services.ResGroupCPRegion
						return res, nil
					},
					RegisterClusterJobFunc: func(clusterReq *api.Cluster) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to create cluster request")
					},
				},
				providersConfig: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name: "aws",
								Regions: config.RegionList{
									config.Region{
										Name: "us-east-1",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "failed to retrieve OSD cluster info from database",
			fields: fields{
				providerLst: []string{"us-east-1"},
				clusterService: &services.ClusterServiceMock{
					ListGroupByProviderAndRegionFunc: func(providers []string, regions []string, status []string) (m []*services.ResGroupCPRegion, e *ocmErrors.ServiceError) {
						return nil, ocmErrors.New(ocmErrors.ErrorGeneral, "Database retrieval failed")
					},
				},
				providersConfig: config.ProviderConfig{
					ProvidersConfig: config.ProviderConfiguration{
						SupportedProviders: config.ProviderList{
							config.Provider{
								Name: "aws",
								Regions: config.RegionList{
									config.Region{
										Name: "us-east-1",
									},
								},
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{

					ClusterService: tt.fields.clusterService,
					ConfigService: coreServices.NewConfigService(&config.ApplicationConfig{
						SupportedProviders:         &tt.fields.providersConfig,
						AccessControlList:          &config.AccessControlListConfig{},
						ObservabilityConfiguration: &config.ObservabilityConfiguration{},
						OSDClusterConfig:           config.NewOSDClusterConfig(),
					}),
				},
			}
			err := c.reconcileClustersForRegions()
			if err != nil && !tt.wantErr {
				t.Errorf("reconcileClustersForRegions() error = %v, wantErr %v", err, tt.wantErr)
			}
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
			name: "successful strimzi cluster logging operator and kas fleetshard operator addon installation",
			fields: fields{
				agentOperator: &services.KasFleetshardOperatorAddonMock{
					ProvisionFunc: func(cluster api.Cluster) (bool, *apiErrors.ServiceError) {
						return true, nil
					},
				},
				clusterService: &services.ClusterServiceMock{
					InstallAddonFunc: func(cluster *api.Cluster, addonID string) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
					InstallAddonWithParamsFunc: func(cluster *api.Cluster, addonID string, addonParams []ocm.AddonParameter) (bool, *ocmErrors.ServiceError) {
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
					InstallAddonFunc: func(cluster *api.Cluster, addonID string) (bool, *apiErrors.ServiceError) {
						return false, apiErrors.GeneralError("failed to install strimzi addon")
					},
					InstallAddonWithParamsFunc: func(cluster *api.Cluster, addonID string, addonParams []ocm.AddonParameter) (bool, *ocmErrors.ServiceError) {
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
					InstallAddonFunc: func(cluster *api.Cluster, addonID string) (bool, *apiErrors.ServiceError) {
						return false, apiErrors.GeneralError("failed to install strimzi addon")
					},
					InstallAddonWithParamsFunc: func(cluster *api.Cluster, addonID string, addonParams []ocm.AddonParameter) (bool, *ocmErrors.ServiceError) {
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
					InstallAddonFunc: func(cluster *api.Cluster, addonID string) (bool, *apiErrors.ServiceError) {
						return false, nil
					},
					InstallAddonWithParamsFunc: func(cluster *api.Cluster, addonID string, addonParams []ocm.AddonParameter) (bool, *ocmErrors.ServiceError) {
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
					ProvisionFunc: func(cluster api.Cluster) (bool, *apiErrors.ServiceError) {
						return false, apiErrors.GeneralError("failed to provision kas fleetshard operator")
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					OCMConfig:                  &config.OCMConfig{StrimziOperatorAddonID: strimziAddonID, ClusterLoggingOperatorAddonID: clusterLoggingOperatorAddonID},
					ClusterService:             tt.fields.clusterService,
					ConfigService:              coreServices.NewConfigService(&config.ApplicationConfig{}),
					KasFleetshardOperatorAddon: tt.fields.agentOperator,
				},
			}
			err := c.reconcileAddonOperator(api.Cluster{
				ClusterID: "test-cluster-id",
			})
			if err != nil && !tt.wantErr {
				t.Errorf("reconcileAddonOperator() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClusterManager_reconcileClusterResourceSet(t *testing.T) {
	const ingressDNS = "foo.bar.example.com"
	observabilityConfig := buildObservabilityConfig()
	clusterCreateConfig := config.OSDClusterConfig{
		ImagePullDockerConfigContent: "image-pull-secret-test",
		IngressControllerReplicas:    12,
	}
	type fields struct {
		clusterService services.ClusterService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "test should pass and resourceset should be created",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return ingressDNS, nil
					},
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						want, _ := buildResourceSet(observabilityConfig, clusterCreateConfig, ingressDNS)
						Expect(resources).To(Equal(want))
						return nil
					},
				},
			},
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
			name: "should receive error when ApplyResources returns error",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test.com", nil
					},
					ApplyResourcesFunc: func(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
						return apiErrors.GeneralError("failed to apply resources")
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{

					ClusterService: tt.fields.clusterService,
					ConfigService: coreServices.NewConfigService(&config.ApplicationConfig{
						SupportedProviders:         &config.ProviderConfig{},
						AccessControlList:          &config.AccessControlListConfig{},
						ObservabilityConfiguration: &observabilityConfig,
						OSDClusterConfig:           &clusterCreateConfig,
						Kafka:                      &config.KafkaConfig{},
					}),
				},
			}

			err := c.reconcileClusterResources(api.Cluster{ClusterID: "test-cluster-id"})
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestClusterManager_reconcileClusterIdentityProvider(t *testing.T) {
	type fields struct {
		clusterService        services.ClusterService
		osdIdpKeycloakService coreServices.KeycloakService
	}
	tests := []struct {
		name    string
		fields  fields
		arg     api.Cluster
		wantErr bool
	}{
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
			name: "should receive an error when creating the the OSD cluster IDP in keycloak fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test.com", nil
					},
				},
				osdIdpKeycloakService: &coreServices.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "", apiErrors.FailedToCreateSSOClient("failure")
					},
					GetRealmConfigFunc: nil, // setting it to nill so that it is not called
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
				clusterService: &services.ClusterServiceMock{
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test.com", nil
					},
					ConfigureAndSaveIdentityProviderFunc: func(cluster *api.Cluster, identityProviderInfo types.IdentityProviderInfo) (*api.Cluster, *apiErrors.ServiceError) {
						return nil, apiErrors.GeneralError("failed to configure IDP")
					},
				},
				osdIdpKeycloakService: &coreServices.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "secret", nil
					},
					GetRealmConfigFunc: func() *config.KeycloakRealmConfig {
						return &config.KeycloakRealmConfig{
							ValidIssuerURI: "https://foo.bar",
						}
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should create an identity provider when cluster identity provider has not been set",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					GetClusterDNSFunc: func(clusterID string) (string, *apiErrors.ServiceError) {
						return "test.com", nil
					},
					ConfigureAndSaveIdentityProviderFunc: func(cluster *api.Cluster, identityProviderInfo types.IdentityProviderInfo) (*api.Cluster, *apiErrors.ServiceError) {
						return cluster, nil
					},
				},
				osdIdpKeycloakService: &coreServices.KeycloakServiceMock{
					RegisterOSDClusterClientInSSOFunc: func(clusterId, clusterOathCallbackURI string) (string, *apiErrors.ServiceError) {
						return "secret", nil
					},
					GetRealmConfigFunc: func() *config.KeycloakRealmConfig {
						return &config.KeycloakRealmConfig{
							ValidIssuerURI: "https://foo.bar",
						}
					},
				},
			},
			arg: api.Cluster{
				Meta: api.Meta{
					ID: "cluster-id",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:        tt.fields.clusterService,
					OsdIdpKeycloakService: tt.fields.osdIdpKeycloakService,
				},
			}

			err := c.reconcileClusterIdentityProvider(tt.arg)
			gomega.Expect(err != nil).To(Equal(tt.wantErr))
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
				},
			}

			err := c.reconcileClusterDNS(tt.arg)
			gomega.Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileDeprovisioningCluster(t *testing.T) {
	type fields struct {
		clusterService services.ClusterService
		configService  coreServices.ConfigService
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
				configService: coreServices.NewConfigService(&config.ApplicationConfig{
					OSDClusterConfig: &config.OSDClusterConfig{
						DataPlaneClusterScalingType: "auto",
					},
				}),
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
				configService: coreServices.NewConfigService(&config.ApplicationConfig{
					OSDClusterConfig: &config.OSDClusterConfig{
						DataPlaneClusterScalingType: "auto",
					},
				}),
			},
			wantErr: false,
		},
		{
			name: "recieves an error when delete OCM cluster fails",
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
				configService: coreServices.NewConfigService(&config.ApplicationConfig{
					OSDClusterConfig: &config.OSDClusterConfig{
						DataPlaneClusterScalingType: "auto",
					},
				}),
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
				configService: coreServices.NewConfigService(&config.ApplicationConfig{
					OSDClusterConfig: &config.OSDClusterConfig{
						DataPlaneClusterScalingType: "auto",
					},
				}),
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
				configService: coreServices.NewConfigService(&config.ApplicationConfig{
					OSDClusterConfig: &config.OSDClusterConfig{
						DataPlaneClusterScalingType: "manual",
					},
				}),
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
				configService: coreServices.NewConfigService(&config.ApplicationConfig{
					OSDClusterConfig: &config.OSDClusterConfig{
						DataPlaneClusterScalingType: "manual",
					},
				}),
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
				configService: coreServices.NewConfigService(&config.ApplicationConfig{
					OSDClusterConfig: &config.OSDClusterConfig{
						DataPlaneClusterScalingType: "manual",
					},
				}),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService: tt.fields.clusterService,
					ConfigService:  tt.fields.configService,
				},
			}

			err := c.reconcileDeprovisioningCluster(&tt.arg)
			gomega.Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func TestClusterManager_reconcileCleanupCluster(t *testing.T) {
	type fields struct {
		clusterService             services.ClusterService
		osdIDPKeycloakService      coreServices.KeycloakService
		kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon
	}
	tests := []struct {
		name    string
		fields  fields
		arg     api.Cluster
		wantErr bool
	}{

		{
			name: "recieves an error when deregistering the OSD IDP from keycloak fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					DeleteByClusterIDFunc: func(clusterID string) *apiErrors.ServiceError {
						return nil
					},
				},
				osdIDPKeycloakService: &coreServices.KeycloakServiceMock{
					DeRegisterClientInSSOFunc: func(kafkaNamespace string) *apiErrors.ServiceError {
						return &apiErrors.ServiceError{}
					},
				},
			},
			wantErr: true,
		},
		{
			name: "recieves an error when remove kas-fleetshard-operator service account fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					UpdateStatusFunc: func(cluster api.Cluster, status api.ClusterStatus) error {
						return nil
					},
				},
				osdIDPKeycloakService: &coreServices.KeycloakServiceMock{
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
			name: "recieves an error when soft delete cluster from database fails",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					DeleteByClusterIDFunc: func(clusterID string) *apiErrors.ServiceError {
						return &apiErrors.ServiceError{}
					},
				},
				osdIDPKeycloakService: &coreServices.KeycloakServiceMock{
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
				osdIDPKeycloakService: &coreServices.KeycloakServiceMock{
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ClusterService:             tt.fields.clusterService,
					OsdIdpKeycloakService:      tt.fields.osdIDPKeycloakService,
					KasFleetshardOperatorAddon: tt.fields.kasFleetshardOperatorAddon,
				},
			}

			err := c.reconcileCleanupCluster(tt.arg)
			gomega.Expect(err != nil).To(Equal(tt.wantErr))
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gomega.RegisterTestingT(t)
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
			gomega.Expect(err != nil).To(Equal(tt.wantErr))
			gomega.Expect(emptyClusterReconciled).To(Equal(tt.want))
		})
	}
}

// buildObservabilityConfig builds a observability config used for testing
func buildObservabilityConfig() config.ObservabilityConfiguration {
	observabilityConfig := config.ObservabilityConfiguration{
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
	}
	return observabilityConfig
}

func buildResourceSet(observabilityConfig config.ObservabilityConfiguration, clusterCreateConfig config.OSDClusterConfig, ingressDNS string) (types.ResourceSet, error) {
	r := int32(clusterCreateConfig.IngressControllerReplicas)
	resources := []interface{}{
		&ingressoperatorv1.IngressController{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "operator.openshift.io/v1",
				Kind:       "IngressController",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sharded-nlb",
				Namespace: openshiftIngressNamespace,
			},
			Spec: ingressoperatorv1.IngressControllerSpec{
				Domain: ingressDNS,
				RouteSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						IngressLabelName: IngressLabelValue,
					},
				},
				EndpointPublishingStrategy: &ingressoperatorv1.EndpointPublishingStrategy{
					LoadBalancer: &ingressoperatorv1.LoadBalancerStrategy{
						ProviderParameters: &ingressoperatorv1.ProviderLoadBalancerParameters{
							AWS: &ingressoperatorv1.AWSLoadBalancerParameters{
								Type: ingressoperatorv1.AWSNetworkLoadBalancer,
							},
							Type: ingressoperatorv1.AWSLoadBalancerProvider,
						},
						Scope: ingressoperatorv1.ExternalLoadBalancer,
					},
					Type: ingressoperatorv1.LoadBalancerServiceStrategyType,
				},
				Replicas: &r,
				NodePlacement: &ingressoperatorv1.NodePlacement{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"node-role.kubernetes.io/worker": "",
						},
					},
				},
			},
		},
		&projectv1.Project{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "project.openshift.io/v1",
				Kind:       "Project",
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
				Name:      observabilityDexCredentials,
				Namespace: observabilityNamespace,
			},
			Type: k8sCoreV1.SecretTypeOpaque,
			StringData: map[string]string{
				"password": observabilityConfig.DexPassword,
				"secret":   observabilityConfig.DexSecret,
				"username": observabilityConfig.DexUsername,
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
				StartingCSV:            "observability-operator.v3.0.2",
				InstallPlanApproval:    v1alpha1.ApprovalAutomatic,
				Package:                observabilitySubscriptionName,
			},
		},
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
	}
	if clusterCreateConfig.ImagePullDockerConfigContent != "" {
		resources = append(resources, &k8sCoreV1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: metav1.SchemeGroupVersion.Version,
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      imagePullSecretName,
				Namespace: strimziAddonNamespace,
			},
			Type: k8sCoreV1.SecretTypeDockercfg,
			Data: map[string][]byte{
				k8sCoreV1.DockerConfigKey: []byte(clusterCreateConfig.ImagePullDockerConfigContent),
			},
		},
			&k8sCoreV1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: metav1.SchemeGroupVersion.Version,
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      imagePullSecretName,
					Namespace: kasFleetshardAddonNamespace,
				},
				Type: k8sCoreV1.SecretTypeDockercfg,
				Data: map[string][]byte{
					k8sCoreV1.DockerConfigKey: []byte(clusterCreateConfig.ImagePullDockerConfigContent),
				},
			})
	}

	return types.ResourceSet{
		Name:      syncsetName,
		Resources: resources,
	}, nil
}

func TestClusterManager_reconcileClusterWithManualConfig(t *testing.T) {
	type fields struct {
		clusterService services.ClusterService
		configService  coreServices.ConfigService
	}
	testOsdConfig := config.NewOSDClusterConfig()
	testOsdConfig.ClusterConfig = config.NewClusterConfig(config.ClusterList{config.ManualCluster{Schedulable: true, KafkaInstanceLimit: 2}})
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Successfully applies manually configurated Cluster",
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					ListAllClusterIdsFunc: func() ([]api.Cluster, *apiErrors.ServiceError) {
						var list []api.Cluster
						list = append(list, api.Cluster{ClusterID: "test02"})
						return list, nil
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
								Count:     1,
							},
						}, nil
					},
				},
				configService: coreServices.NewConfigService(&config.ApplicationConfig{
					OSDClusterConfig: testOsdConfig,
				}),
			},
			wantErr: false,
		},
		{
			name: "Failed to apply manually configurated Cluster",
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
				configService: coreServices.NewConfigService(&config.ApplicationConfig{
					OSDClusterConfig: testOsdConfig,
				}),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &ClusterManager{
				ClusterManagerOptions: ClusterManagerOptions{
					ConfigService:  tt.fields.configService,
					ClusterService: tt.fields.clusterService,
				},
			}
			if err := c.reconcileClusterWithManualConfig(); (len(err) > 0) != tt.wantErr {
				t.Errorf("reconcileClusterWithManualConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
