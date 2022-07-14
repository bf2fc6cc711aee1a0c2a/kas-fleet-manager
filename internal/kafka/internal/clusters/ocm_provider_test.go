package clusters

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha2"
	"github.com/pkg/errors"
	k8sCorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestOCMProvider_Create(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterReq types.ClusterRequest
	}
	awsConfig := &config.AWSConfig{
		AccountID:       "",
		AccessKey:       "",
		SecretAccessKey: "",
	}
	osdCreateConfig := &config.DataplaneClusterConfig{
		OpenshiftVersion: "4.7",
	}
	cb := NewClusterBuilder(awsConfig, osdCreateConfig)

	internalId := "test-internal-id"
	externalId := "test-external-id"

	cr := types.ClusterRequest{
		CloudProvider:  "aws",
		Region:         "east-1",
		MultiAZ:        true,
		AdditionalSpec: nil,
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.ClusterSpec
		wantErr bool
	}{
		{
			name: "should return created cluster",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateClusterFunc: func(cluster *clustersmgmtv1.Cluster) (*clustersmgmtv1.Cluster, error) {
						return clustersmgmtv1.NewCluster().ID(internalId).ExternalID(externalId).Build()
					},
				},
			},
			args: args{
				clusterReq: cr,
			},
			want: &types.ClusterSpec{
				InternalID:     internalId,
				ExternalID:     externalId,
				Status:         api.ClusterProvisioning,
				AdditionalInfo: nil,
			},
			wantErr: false,
		},
		{
			name: "should return error when create cluster failed from OCM",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateClusterFunc: func(cluster *clustersmgmtv1.Cluster) (*clustersmgmtv1.Cluster, error) {
						return nil, errors.Errorf("failed to create cluster")
					},
				},
			},
			args:    args{clusterReq: cr},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := newOCMProvider(test.fields.ocmClient, cb, &ocm.OCMConfig{})
			resp, err := p.Create(&test.args.clusterReq)
			g.Expect(resp).To(gomega.Equal(test.want))
			if test.wantErr {
				g.Expect(err).NotTo(gomega.BeNil())
			}
		})
	}
}

func TestOCMProvider_CheckClusterStatus(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec *types.ClusterSpec
	}

	internalId := "test-internal-id"
	externalId := "test-external-id"

	clusterFailedProvisioningErrorText := "cluster provisioning failed test message"

	spec := &types.ClusterSpec{
		InternalID:     internalId,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.ClusterSpec
		wantErr bool
	}{
		{
			name: "should return cluster status ready",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterFunc: func(clusterID string) (*clustersmgmtv1.Cluster, error) {
						sb := clustersmgmtv1.NewClusterStatus().State(clustersmgmtv1.ClusterStateReady)
						return clustersmgmtv1.NewCluster().Status(sb).ExternalID(externalId).Build()
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			want: &types.ClusterSpec{
				InternalID:     internalId,
				ExternalID:     externalId,
				Status:         api.ClusterProvisioned,
				AdditionalInfo: nil,
			},
			wantErr: false,
		},
		{
			name: "should return cluster status failed",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterFunc: func(clusterID string) (*clustersmgmtv1.Cluster, error) {
						sb := clustersmgmtv1.NewClusterStatus().State(clustersmgmtv1.ClusterStateError).ProvisionErrorMessage(clusterFailedProvisioningErrorText)
						return clustersmgmtv1.NewCluster().Status(sb).ExternalID(externalId).Build()
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			want: &types.ClusterSpec{
				InternalID:     internalId,
				ExternalID:     externalId,
				Status:         api.ClusterFailed,
				StatusDetails:  clusterFailedProvisioningErrorText,
				AdditionalInfo: nil,
			},
			wantErr: false,
		},
		{
			name: "should return error when failed to get cluster from OCM",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterFunc: func(clusterID string) (*clustersmgmtv1.Cluster, error) {
						return nil, errors.Errorf("failed to get cluster")
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			wantErr: true,
			want:    nil,
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.CheckClusterStatus(test.args.clusterSpec)
			g.Expect(resp).To(gomega.Equal(test.want))
			if test.wantErr {
				g.Expect(err).NotTo(gomega.BeNil())
			}
		})
	}
}

func TestOCMProvider_Delete(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec *types.ClusterSpec
	}

	internalId := "test-internal-id"

	spec := &types.ClusterSpec{
		InternalID:     internalId,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "should return true if cluster is not found from OCM",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					DeleteClusterFunc: func(clusterID string) (int, error) {
						return http.StatusNotFound, nil
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "should return false if the cluster still exists in OCM",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					DeleteClusterFunc: func(clusterID string) (int, error) {
						return http.StatusConflict, nil
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "should return error",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					DeleteClusterFunc: func(clusterID string) (int, error) {
						return 0, errors.Errorf("failed to delete cluster from OCM")
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			want:    false,
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.Delete(test.args.clusterSpec)
			g.Expect(resp).To(gomega.Equal(test.want))
			if test.wantErr {
				g.Expect(err).NotTo(gomega.BeNil())
			}
		})
	}
}

func TestOCMProvider_GetClusterDNS(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec *types.ClusterSpec
	}

	internalId := "test-internal-id"

	spec := &types.ClusterSpec{
		InternalID:     internalId,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	dns := "test.foo.bar.com"

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "should return dns value from OCM",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterDNSFunc: func(clusterID string) (string, error) {
						return dns, nil
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			want:    dns,
			wantErr: false,
		},
		{
			name: "should return error",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetClusterDNSFunc: func(clusterID string) (string, error) {
						return "", errors.Errorf("failed to get dns value from OCM")
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.GetClusterDNS(test.args.clusterSpec)
			g.Expect(resp).To(gomega.Equal(test.want))
			if test.wantErr {
				g.Expect(err).NotTo(gomega.BeNil())
			}
		})
	}
}

func TestOCMProvider_AddIdentityProvider(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec          *types.ClusterSpec
		identityProviderInfo types.IdentityProviderInfo
	}

	internalId := "test-internal-id"

	spec := &types.ClusterSpec{
		InternalID:     internalId,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	testIdpId := "test-idp-ocm-id"
	testIdpName := "test-idp-name"
	testIdpClientId := "test-client-id"
	testIdpClientSecret := "test-client-secret"
	testIdpIssuer := "test-issuer"
	idpInfo := types.IdentityProviderInfo{OpenID: &types.OpenIDIdentityProviderInfo{
		Name:         testIdpName,
		ClientID:     testIdpClientId,
		ClientSecret: testIdpClientSecret,
		Issuer:       testIdpIssuer,
	}}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.IdentityProviderInfo
		wantErr bool
	}{
		{
			name: "should create IDP",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateIdentityProviderFunc: func(clusterID string, identityProvider *clustersmgmtv1.IdentityProvider) (*clustersmgmtv1.IdentityProvider, error) {
						return clustersmgmtv1.NewIdentityProvider().ID(testIdpId).Build()
					},
					GetIdentityProviderListFunc: func(clusterID string) (*clustersmgmtv1.IdentityProviderList, error) {
						return nil, errors.Errorf("this should not be called")
					},
				},
			},
			args: args{
				clusterSpec:          spec,
				identityProviderInfo: idpInfo,
			},
			want: &types.IdentityProviderInfo{
				OpenID: &types.OpenIDIdentityProviderInfo{
					ID:           testIdpId,
					Name:         testIdpName,
					ClientID:     testIdpClientId,
					ClientSecret: testIdpClientSecret,
					Issuer:       testIdpIssuer,
				},
			},
			wantErr: false,
		},
		{
			name: "should not return error if IDP already exists",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateIdentityProviderFunc: func(clusterID string, identityProvider *clustersmgmtv1.IdentityProvider) (*clustersmgmtv1.IdentityProvider, error) {
						return nil, errors.Errorf("idp already exists")
					},
					GetIdentityProviderListFunc: func(clusterID string) (*clustersmgmtv1.IdentityProviderList, error) {
						idp := clustersmgmtv1.NewIdentityProvider().ID(testIdpId).Name(testIdpName)
						return clustersmgmtv1.NewIdentityProviderList().Items(idp).Build()
					},
				},
			},
			args: args{
				clusterSpec:          spec,
				identityProviderInfo: idpInfo,
			},
			want: &types.IdentityProviderInfo{
				OpenID: &types.OpenIDIdentityProviderInfo{
					ID:           testIdpId,
					Name:         testIdpName,
					ClientID:     testIdpClientId,
					ClientSecret: testIdpClientSecret,
					Issuer:       testIdpIssuer,
				},
			},
			wantErr: false,
		},
		{
			name: "should return error",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateIdentityProviderFunc: func(clusterID string, identityProvider *clustersmgmtv1.IdentityProvider) (*clustersmgmtv1.IdentityProvider, error) {
						return nil, errors.Errorf("unexepcted error")
					},
					GetIdentityProviderListFunc: func(clusterID string) (*clustersmgmtv1.IdentityProviderList, error) {
						return nil, errors.Errorf("this should not be called")
					},
				},
			},
			args: args{
				clusterSpec:          spec,
				identityProviderInfo: idpInfo,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.AddIdentityProvider(test.args.clusterSpec, test.args.identityProviderInfo)
			g.Expect(resp).To(gomega.Equal(test.want))
			if test.wantErr {
				g.Expect(err).NotTo(gomega.BeNil())
			}
		})
	}
}

func TestOCMProvider_ApplyResources(t *testing.T) {
	g := gomega.NewWithT(t)
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec *types.ClusterSpec
		resources   types.ResourceSet
	}

	internalId := "test-internal-id"

	spec := &types.ClusterSpec{
		InternalID:     internalId,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	name := "test-resource-set"
	resources := types.ResourceSet{
		Name:      name,
		Resources: []interface{}{sampleProjectCR(), sampleOperatorGroup()},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.ResourceSet
		wantErr bool
	}{
		{
			name: "should create resource set",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetSyncSetFunc: func(clusterID string, syncSetID string) (*clustersmgmtv1.Syncset, error) {
						return nil, apiErrors.NotFound("not found error")
					},
					CreateSyncSetFunc: func(clusterID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {
						g.Expect(syncset.ID()).To(gomega.Equal(resources.Name))
						g.Expect(syncset.Resources()).To(gomega.Equal(resources.Resources))
						return nil, nil
					},
					UpdateSyncSetFunc: func(clusterID string, syncSetID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {
						return nil, errors.Errorf("UpdateSyncSet should not be called")
					},
				},
			},
			args: args{
				clusterSpec: spec,
				resources:   resources,
			},
			want: &types.ResourceSet{
				Name:      name,
				Resources: []interface{}{sampleProjectCR(), sampleOperatorGroup()},
			},
			wantErr: false,
		},
		{
			name: "should update resource set if ResourceSet is changed",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetSyncSetFunc: func(clusterID string, syncSetID string) (*clustersmgmtv1.Syncset, error) {
						p, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(sampleProjectCR())
						return clustersmgmtv1.NewSyncset().ID(name).Resources(p).Build()
					},
					CreateSyncSetFunc: func(clusterID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {
						return nil, errors.New("CreateSyncSet should not be called")
					},
					UpdateSyncSetFunc: func(clusterID string, syncSetID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {
						g.Expect(syncset.Resources()).To(gomega.Equal(resources.Resources))
						return nil, nil
					},
				},
			},
			args: args{
				clusterSpec: spec,
				resources:   resources,
			},
			want: &types.ResourceSet{
				Name:      name,
				Resources: []interface{}{sampleProjectCR(), sampleOperatorGroup()},
			},
			wantErr: false,
		},
		{
			name: "should not update resource set if ResourceSet is not changed",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetSyncSetFunc: func(clusterID string, syncSetID string) (*clustersmgmtv1.Syncset, error) {
						p, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(sampleProjectCR())
						g, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(sampleOperatorGroup())
						return clustersmgmtv1.NewSyncset().ID(name).Resources(p, g).Build()
					},
					CreateSyncSetFunc: func(clusterID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {
						return nil, errors.New("CreateSyncSet should not be called")
					},
					UpdateSyncSetFunc: func(clusterID string, syncSetID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {
						return nil, errors.New("UpdateSyncSetFunc should not be called")
					},
				},
			},
			args: args{
				clusterSpec: spec,
				resources:   resources,
			},
			want: &types.ResourceSet{
				Name:      name,
				Resources: []interface{}{sampleProjectCR(), sampleOperatorGroup()},
			},
			wantErr: false,
		},
		{
			name: "should return error when get resources failed",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetSyncSetFunc: func(clusterID string, syncSetID string) (*clustersmgmtv1.Syncset, error) {
						return nil, errors.Errorf("error")
					},
					CreateSyncSetFunc: func(clusterID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {
						return nil, errors.Errorf("CreateSyncSet should not be called")
					},
					UpdateSyncSetFunc: func(clusterID string, syncSetID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {
						return nil, errors.Errorf("UpdateSyncSet should not be called")
					},
				},
			},
			args: args{
				clusterSpec: spec,
				resources:   resources,
			},
			want:    nil,
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.ApplyResources(test.args.clusterSpec, test.args.resources)
			g.Expect(resp).To(gomega.Equal(test.want))
			if test.wantErr {
				g.Expect(err).NotTo(gomega.BeNil())
			}
		})
	}
}

func TestOCMProvider_InstallAddon(t *testing.T) {
	g := gomega.NewWithT(t)
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec *types.ClusterSpec
		addonID     string
	}

	internalId := "test-internal-id"

	spec := &types.ClusterSpec{
		InternalID:     internalId,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	testAddonId := "test-addon-id"

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "should create addon but not ready",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return clustersmgmtv1.NewAddOnInstallation().Build()
					},
					CreateAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						g.Expect(addonId).To(gomega.Equal(testAddonId))
						return clustersmgmtv1.NewAddOnInstallation().State(clustersmgmtv1.AddOnInstallationStateInstalling).Build()
					},
				},
			},
			args: args{
				clusterSpec: spec,
				addonID:     testAddonId,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "should not create addon and it's ready",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						g.Expect(addonId).To(gomega.Equal(testAddonId))
						return clustersmgmtv1.NewAddOnInstallation().ID("test-addon-id").State(clustersmgmtv1.AddOnInstallationStateReady).Build()
					},
					CreateAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return nil, errors.Errorf("CreateAddon should not be called")
					},
				},
			},
			args: args{
				clusterSpec: spec,
				addonID:     testAddonId,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "should return error when failed to get addon",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return nil, errors.Errorf("failed to get addon")
					},
				},
			},
			args: args{
				clusterSpec: spec,
				addonID:     testAddonId,
			},
			wantErr: true,
			want:    false,
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.installAddon(test.args.clusterSpec, test.args.addonID)
			g.Expect(resp).To(gomega.Equal(test.want))
			if test.wantErr {
				g.Expect(err).NotTo(gomega.BeNil())
			}
		})
	}
}

func TestOCMProvider_InstallAddonWithParams(t *testing.T) {
	g := gomega.NewWithT(t)
	type fields struct {
		ocmClient ocm.Client
	}
	type args struct {
		clusterSpec *types.ClusterSpec
		addonID     string
		params      []types.Parameter
	}

	internalId := "test-internal-id"

	spec := &types.ClusterSpec{
		InternalID:     internalId,
		ExternalID:     "",
		Status:         "",
		AdditionalInfo: nil,
	}

	testAddonId := "test-addon-id"
	testParams := []types.Parameter{
		{
			Id:    "param1",
			Value: "param-value-1",
		},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "should create addon with params if addon is not installed",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return clustersmgmtv1.NewAddOnInstallation().Build()
					},
					CreateAddonWithParamsFunc: func(clusterId string, addonId string, params []types.Parameter) (*clustersmgmtv1.AddOnInstallation, error) {
						g.Expect(addonId).To(gomega.Equal(testAddonId))
						g.Expect(params).To(gomega.Equal(testParams))
						return clustersmgmtv1.NewAddOnInstallation().State(clustersmgmtv1.AddOnInstallationStateInstalling).Build()
					},
				},
			},
			args: args{
				clusterSpec: spec,
				addonID:     testAddonId,
				params:      testParams,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "should not update addon if it's already installed",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return clustersmgmtv1.NewAddOnInstallation().ID("test-addon-id").State(clustersmgmtv1.AddOnInstallationStateReady).Build()
					},
					UpdateAddonParametersFunc: func(clusterId string, addonId string, parameters []types.Parameter) (*clustersmgmtv1.AddOnInstallation, error) {
						g.Expect(addonId).To(gomega.Equal(testAddonId))
						g.Expect(parameters).To(gomega.Equal(testParams))
						return clustersmgmtv1.NewAddOnInstallation().State(clustersmgmtv1.AddOnInstallationStateReady).Build()
					},
				},
			},
			args: args{
				clusterSpec: spec,
				addonID:     testAddonId,
				params:      testParams,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "should return error when failed to get addon",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetAddonFunc: func(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
						return nil, errors.Errorf("failed to get addon")
					},
				},
			},
			args: args{
				clusterSpec: spec,
			},
			wantErr: true,
			want:    false,
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.installAddonWithParams(test.args.clusterSpec, test.args.addonID, test.args.params)
			g.Expect(resp).To(gomega.Equal(test.want))
			if test.wantErr {
				g.Expect(err).NotTo(gomega.BeNil())
			}
		})
	}
}

func TestOCMProvider_GetCloudProviders(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}

	providerId1 := "provider-id-1"
	providerName1 := "provider-name-1"
	providerDisplayName1 := "provider-display-name-1"

	tests := []struct {
		name    string
		fields  fields
		want    *types.CloudProviderInfoList
		wantErr bool
	}{
		{
			name: "should return cloud providers when there are no cloud providers returned from ocm",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetCloudProvidersFunc: func() (*clustersmgmtv1.CloudProviderList, error) {
						return clustersmgmtv1.NewCloudProviderList().Build()
					},
				},
			},
			want:    &types.CloudProviderInfoList{Items: nil},
			wantErr: false,
		},
		{
			name: "should return cloud providers when there are cloud providers returned from ocm",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetCloudProvidersFunc: func() (*clustersmgmtv1.CloudProviderList, error) {
						p := clustersmgmtv1.NewCloudProvider().ID(providerId1).Name(providerName1).DisplayName(providerDisplayName1)
						return clustersmgmtv1.NewCloudProviderList().Items(p).Build()
					},
				},
			},
			want: &types.CloudProviderInfoList{Items: []types.CloudProviderInfo{{
				ID:          providerId1,
				Name:        providerName1,
				DisplayName: providerDisplayName1,
			}}},
			wantErr: false,
		},
		{
			name: "should return error when failed to get cloud providers",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetCloudProvidersFunc: func() (*clustersmgmtv1.CloudProviderList, error) {
						return nil, errors.Errorf("failed to get cloud providers")
					},
				},
			},
			wantErr: true,
			want:    nil,
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.GetCloudProviders()
			g.Expect(resp).To(gomega.Equal(test.want))
			if test.wantErr {
				g.Expect(err).NotTo(gomega.BeNil())
			}
		})
	}
}

func TestOCMProvider_GetCloudProviderRegions(t *testing.T) {
	g := gomega.NewWithT(t)

	type fields struct {
		ocmClient ocm.Client
	}

	type args struct {
		providerInfo types.CloudProviderInfo
	}

	providerId1 := "provider-id-1"
	providerName1 := "provider-name-1"
	providerDisplayName1 := "provider-display-name-1"

	regionId1 := "region-id-1"
	regionName1 := "region-name-1"
	regionDisplayName1 := "region-display-name-1"
	regionSupportsMultiAZ1 := true

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.CloudProviderRegionInfoList
		wantErr bool
	}{
		{
			name: "should return cloud providers when there are no cloud providers returned from ocm",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetRegionsFunc: func(provider *clustersmgmtv1.CloudProvider) (*clustersmgmtv1.CloudRegionList, error) {
						g.Expect(provider.ID()).To(gomega.Equal(providerId1))
						g.Expect(provider.Name()).To(gomega.Equal(providerName1))
						g.Expect(provider.DisplayName()).To(gomega.Equal(providerDisplayName1))
						return clustersmgmtv1.NewCloudRegionList().Build()
					},
				},
			},
			args: args{providerInfo: types.CloudProviderInfo{
				ID:          providerId1,
				Name:        providerName1,
				DisplayName: providerDisplayName1,
			}},
			want:    &types.CloudProviderRegionInfoList{Items: nil},
			wantErr: false,
		},
		{
			name: "should return cloud providers when there are cloud providers returned from ocm",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetRegionsFunc: func(provider *clustersmgmtv1.CloudProvider) (*clustersmgmtv1.CloudRegionList, error) {
						g.Expect(provider.ID()).To(gomega.Equal(providerId1))
						g.Expect(provider.Name()).To(gomega.Equal(providerName1))
						g.Expect(provider.DisplayName()).To(gomega.Equal(providerDisplayName1))
						p := clustersmgmtv1.NewCloudProvider().ID(providerId1)
						r := clustersmgmtv1.NewCloudRegion().ID(regionId1).CloudProvider(p).Name(regionName1).DisplayName(regionDisplayName1).SupportsMultiAZ(regionSupportsMultiAZ1)
						return clustersmgmtv1.NewCloudRegionList().Items(r).Build()
					},
				},
			},
			args: args{providerInfo: types.CloudProviderInfo{
				ID:          providerId1,
				Name:        providerName1,
				DisplayName: providerDisplayName1,
			}},
			want: &types.CloudProviderRegionInfoList{
				Items: []types.CloudProviderRegionInfo{
					{
						ID:              regionId1,
						CloudProviderID: providerId1,
						Name:            regionName1,
						DisplayName:     regionDisplayName1,
						SupportsMultiAZ: regionSupportsMultiAZ1,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should return error when failed to get cloud provider regions",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetRegionsFunc: func(provider *clustersmgmtv1.CloudProvider) (*clustersmgmtv1.CloudRegionList, error) {
						return nil, errors.Errorf("failed get cloud provider regions")
					},
				},
			},
			wantErr: true,
			want:    nil,
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			p := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			resp, err := p.GetCloudProviderRegions(test.args.providerInfo)
			g.Expect(resp).To(gomega.Equal(test.want))
			if test.wantErr {
				g.Expect(err).NotTo(gomega.BeNil())
			}
		})
	}
}

func TestOCMProvider_GetMachinePool(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}

	sampleMachinePoolID := "test-machinepool-id"
	sampleClusterID := "test-cluster-id"

	type args struct {
		clusterID     string
		machinePoolID string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.MachinePoolInfo
		wantErr bool
	}{
		{
			name: "getting an existing machinepool succeeds",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetMachinePoolFunc: func(clusterID, machinePoolID string) (*clustersmgmtv1.MachinePool, error) {
						machinePoolBuilder := clustersmgmtv1.NewMachinePool()
						machinePoolBuilder.ID(machinePoolID)
						machinePool, err := machinePoolBuilder.Build()
						if err != nil {
							return nil, err
						}
						return machinePool, nil
					},
				},
			},
			args: args{
				clusterID:     sampleClusterID,
				machinePoolID: sampleMachinePoolID,
			},
			want: &types.MachinePoolInfo{
				ID: sampleMachinePoolID,
			},
			wantErr: false,
		},
		{
			name: "getting an existing machinepool succeeds second test",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetMachinePoolFunc: func(clusterID, machinePoolID string) (*clustersmgmtv1.MachinePool, error) {
						autoscalingBuilder := clustersmgmtv1.NewMachinePoolAutoscaling()
						autoscalingBuilder.MinReplicas(3)
						autoscalingBuilder.MaxReplicas(6)
						machinePoolBuilder := clustersmgmtv1.NewMachinePool()
						machinePoolBuilder.ID(machinePoolID)
						machinePoolBuilder.Autoscaling(autoscalingBuilder)
						machinePoolBuilder.AvailabilityZones("us-east-1", "eu-west-1")
						machinePool, err := machinePoolBuilder.Build()
						if err != nil {
							return nil, err
						}
						return machinePool, nil
					},
				},
			},
			args: args{
				clusterID:     sampleClusterID,
				machinePoolID: sampleMachinePoolID,
			},
			want: &types.MachinePoolInfo{
				ID:                 sampleMachinePoolID,
				MultiAZ:            true,
				AutoScalingEnabled: true,
				AutoScaling: types.MachinePoolAutoScaling{
					MinNodes: 3,
					MaxNodes: 6,
				},
			},
			wantErr: false,
		},
		{
			name: "getting an unexisting machinepool from OCM returns nil as a result",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetMachinePoolFunc: func(clusterID, machinePoolID string) (*clustersmgmtv1.MachinePool, error) {
						return nil, nil
					},
				},
			},
			args: args{
				clusterID:     sampleClusterID,
				machinePoolID: sampleMachinePoolID,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "getting an error from OCM returns an error as a result",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					GetMachinePoolFunc: func(clusterID, machinePoolID string) (*clustersmgmtv1.MachinePool, error) {
						return nil, fmt.Errorf("this is an example error")
					},
				},
			},
			args: args{
				clusterID:     sampleClusterID,
				machinePoolID: sampleMachinePoolID,
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		test := tc
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			ocmProvider := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			res, err := ocmProvider.GetMachinePool(test.args.clusterID, test.args.machinePoolID)
			gotErr := err != nil
			g.Expect(gotErr).To(gomega.Equal(test.wantErr))
			g.Expect(res).To(gomega.Equal(test.want))
		})
	}
}

func TestOCMProvider_CreateMachinePool(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}

	sampleMachinePoolID := "test-machinepool-id"
	sampleClusterID := "test-cluster-id"

	type args struct {
		machinePoolRequest types.MachinePoolRequest
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.MachinePoolRequest
		wantErr bool
	}{
		{
			name: "creting a machinepool succeeds",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateMachinePoolFunc: func(clusterID string, machinePool *clustersmgmtv1.MachinePool) (*clustersmgmtv1.MachinePool, error) {
						machinePoolBuilder := clustersmgmtv1.NewMachinePool()
						machinePoolBuilder.ID(machinePool.ID())
						machinePool, err := machinePoolBuilder.Build()
						if err != nil {
							return nil, err
						}
						return machinePool, nil
					},
				},
			},
			args: args{
				machinePoolRequest: types.MachinePoolRequest{
					ID:        sampleMachinePoolID,
					ClusterID: sampleClusterID,
				},
			},
			want: &types.MachinePoolRequest{
				ID:        sampleMachinePoolID,
				ClusterID: sampleClusterID,
			},
			wantErr: false,
		},
		{
			name: "an error is returned when machinepool min nodes is greater than max nodes",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateMachinePoolFunc: func(clusterID string, machinePool *clustersmgmtv1.MachinePool) (*clustersmgmtv1.MachinePool, error) {
						return nil, fmt.Errorf("test error")
					},
				},
			},
			args: args{
				machinePoolRequest: types.MachinePoolRequest{
					ID:                 sampleMachinePoolID,
					ClusterID:          sampleClusterID,
					AutoScalingEnabled: true,
					AutoScaling: types.MachinePoolAutoScaling{
						MinNodes: 6,
						MaxNodes: 3,
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "when MultiAZ is enabled the number of nodes to specify in the OCM machine pool is a multiple of OCM MultiAZ requirement",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateMachinePoolFunc: func(clusterID string, machinePool *clustersmgmtv1.MachinePool) (*clustersmgmtv1.MachinePool, error) {
						mpAutoscaling := machinePool.Autoscaling()
						if mpAutoscaling == nil {
							return nil, fmt.Errorf("test fail")
						}
						if mpAutoscaling.MinReplicas() != shared.RoundUp(2, ocmMultiAZClusterNodeScalingMultiple) {
							return nil, fmt.Errorf("test fail: unexpected number of min replicas")
						}
						if mpAutoscaling.MaxReplicas() != shared.RoundUp(4, ocmMultiAZClusterNodeScalingMultiple) {
							return nil, fmt.Errorf("test fail: unexpected number of max replicas")
						}

						machinePoolBuilder := clustersmgmtv1.NewMachinePool()
						machinePoolBuilder.ID(machinePool.ID())
						autoscalingBuilder := clustersmgmtv1.NewMachinePoolAutoscaling()
						autoscalingBuilder.MinReplicas(mpAutoscaling.MinReplicas())
						autoscalingBuilder.MaxReplicas(mpAutoscaling.MaxReplicas())
						machinePool, err := machinePoolBuilder.Build()
						if err != nil {
							return nil, err
						}
						return machinePool, nil
					},
				},
			},
			args: args{
				machinePoolRequest: types.MachinePoolRequest{
					ID:                 sampleMachinePoolID,
					ClusterID:          sampleClusterID,
					MultiAZ:            true,
					AutoScalingEnabled: true,
					AutoScaling: types.MachinePoolAutoScaling{
						MinNodes: 2,
						MaxNodes: 4,
					},
				},
			},
			want: &types.MachinePoolRequest{
				ID:                 sampleMachinePoolID,
				ClusterID:          sampleClusterID,
				AutoScalingEnabled: true,
				MultiAZ:            true,
				AutoScaling: types.MachinePoolAutoScaling{
					MinNodes: 2,
					MaxNodes: 4,
				},
			},
			wantErr: false,
		},
		{
			name: "when MultiAZ is disabled the number of nodes to specify in the OCM machine pool is a multiple of OCM MultiAZ requirement",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateMachinePoolFunc: func(clusterID string, machinePool *clustersmgmtv1.MachinePool) (*clustersmgmtv1.MachinePool, error) {
						mpAutoscaling := machinePool.Autoscaling()
						if mpAutoscaling == nil {
							return nil, fmt.Errorf("test fail")
						}
						if mpAutoscaling.MinReplicas() != 2 {
							return nil, fmt.Errorf("test fail: unexpected number of min replicas")
						}
						if mpAutoscaling.MaxReplicas() != 4 {
							return nil, fmt.Errorf("test fail: unexpected number of max replicas")
						}

						machinePoolBuilder := clustersmgmtv1.NewMachinePool()
						machinePoolBuilder.ID(machinePool.ID())
						autoscalingBuilder := clustersmgmtv1.NewMachinePoolAutoscaling()
						autoscalingBuilder.MinReplicas(mpAutoscaling.MinReplicas())
						autoscalingBuilder.MaxReplicas(mpAutoscaling.MaxReplicas())
						machinePool, err := machinePoolBuilder.Build()
						if err != nil {
							return nil, err
						}
						return machinePool, nil
					},
				},
			},
			args: args{
				machinePoolRequest: types.MachinePoolRequest{
					ID:                 sampleMachinePoolID,
					ClusterID:          sampleClusterID,
					MultiAZ:            false,
					AutoScalingEnabled: true,
					AutoScaling: types.MachinePoolAutoScaling{
						MinNodes: 2,
						MaxNodes: 4,
					},
				},
			},
			want: &types.MachinePoolRequest{
				ID:                 sampleMachinePoolID,
				ClusterID:          sampleClusterID,
				AutoScalingEnabled: true,
				MultiAZ:            false,
				AutoScaling: types.MachinePoolAutoScaling{
					MinNodes: 2,
					MaxNodes: 4,
				},
			},
			wantErr: false,
		},
		{
			name: "an error is returned when OCM fails to create a machinepool",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateMachinePoolFunc: func(clusterID string, machinePool *clustersmgmtv1.MachinePool) (*clustersmgmtv1.MachinePool, error) {
						return nil, fmt.Errorf("test error")
					},
				},
			},
			args: args{
				machinePoolRequest: types.MachinePoolRequest{
					ID:        sampleMachinePoolID,
					ClusterID: sampleClusterID,
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		test := tc
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			ocmProvider := newOCMProvider(test.fields.ocmClient, nil, &ocm.OCMConfig{})
			res, err := ocmProvider.CreateMachinePool(&test.args.machinePoolRequest)
			gotErr := err != nil
			g.Expect(gotErr).To(gomega.Equal(test.wantErr))
			g.Expect(res).To(gomega.Equal(test.want))
		})
	}
}

func sampleProjectCR() *k8sCorev1.Namespace {
	return &k8sCorev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: k8sCorev1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
	}
}

func sampleOperatorGroup() *v1alpha2.OperatorGroup {
	t := metav1.NewTime(time.Unix(0, 0))
	return &v1alpha2.OperatorGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "operators.coreos.com/v1alpha2",
			Kind:       "OperatorGroup",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-operator-group-name",
			Namespace: "test-project",
		},
		Spec: v1alpha2.OperatorGroupSpec{
			TargetNamespaces: []string{"test-project"},
		},
		Status: v1alpha2.OperatorGroupStatus{
			LastUpdated: &t,
		},
	}
}
