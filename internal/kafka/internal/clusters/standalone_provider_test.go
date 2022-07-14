package clusters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	mock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/data_plane"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/onsi/gomega"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1alpha2 "github.com/operator-framework/api/pkg/operators/v1alpha2"
	"github.com/pkg/errors"
	mocket "github.com/selvatico/go-mocket"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

func TestStandaloneProvider_GetCloudProviders(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}

	tests := []struct {
		name    string
		fields  fields
		want    *types.CloudProviderInfoList
		wantErr bool
		setupFn func()
	}{
		{
			name:    "receives an error when database query fails",
			wantErr: true,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithError(errors.New("some-error"))
			},
		},
		{
			name:    "returns an empty list when no standalone clusters exists",
			wantErr: false,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithReply([]map[string]interface{}{})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			want: &types.CloudProviderInfoList{
				Items: []types.CloudProviderInfo{},
			},
		},
		{
			name:    "returns the list of cloud providers",
			wantErr: false,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithReply([]map[string]interface{}{{"cloud_provider": "aws"}, {"cloud_provider": "azure"}})
			},
			want: &types.CloudProviderInfoList{
				Items: []types.CloudProviderInfo{
					{
						ID:          "aws",
						Name:        "aws",
						DisplayName: "aws",
					},
					{
						ID:          "azure",
						Name:        "azure",
						DisplayName: "azure",
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			test.setupFn()
			provider := newStandaloneProvider(test.fields.connectionFactory, config.NewDataplaneClusterConfig())
			resp, err := provider.GetCloudProviders()
			g.Expect(test.wantErr).To(gomega.Equal(err != nil))
			if !test.wantErr {
				g.Expect(resp.Items).To(gomega.Equal(test.want.Items))
			}
		})
	}
}

func TestStandaloneProvider_GetCloudProviderRegions(t *testing.T) {
	type fields struct {
		connectionFactory *db.ConnectionFactory
	}

	tests := []struct {
		name    string
		fields  fields
		want    *types.CloudProviderRegionInfoList
		wantErr bool
		setupFn func()
	}{
		{
			name:    "receives an error when database query fails",
			wantErr: true,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithError(errors.New("some-error"))
			},
		},
		{
			name:    "returns an empty list when no standalone clusters in a given cloud provider exists",
			wantErr: false,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithReply([]map[string]interface{}{})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
			want: &types.CloudProviderRegionInfoList{
				Items: []types.CloudProviderRegionInfo{},
			},
		},
		{
			name:    "returns the list of cloud providers regions",
			wantErr: false,
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mocket.Catcher.Reset()
				mocket.Catcher.NewMock().WithQuery("SELECT DISTINCT").WithReply([]map[string]interface{}{{"region": "af-east-1", "multi_az": false}, {"region": "eu-central-0", "multi_az": true}})
			},
			want: &types.CloudProviderRegionInfoList{
				Items: []types.CloudProviderRegionInfo{
					{
						ID:              "af-east-1",
						Name:            "af-east-1",
						DisplayName:     "af-east-1",
						CloudProviderID: "aws",
						SupportsMultiAZ: false,
					},
					{
						ID:              "eu-central-0",
						Name:            "eu-central-0",
						DisplayName:     "eu-central-0",
						CloudProviderID: "aws",
						SupportsMultiAZ: true,
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			test.setupFn()
			provider := newStandaloneProvider(test.fields.connectionFactory, config.NewDataplaneClusterConfig())
			resp, err := provider.GetCloudProviderRegions(types.CloudProviderInfo{ID: "aws"})
			g.Expect(test.wantErr).To(gomega.Equal(err != nil))
			if !test.wantErr {
				g.Expect(resp.Items).To(gomega.Equal(test.want.Items))
			}
		})
	}
}

func TestStandaloneProvider_buildOpenIDPClientSecret(t *testing.T) {
	type args struct {
		idpProviderInfo types.IdentityProviderInfo
	}

	tests := []struct {
		name string
		args args
		want *v1.Secret
	}{
		{
			name: "buids a k8s secret with a given client secret",
			args: args{
				idpProviderInfo: types.IdentityProviderInfo{
					OpenID: &types.OpenIDIdentityProviderInfo{
						ClientSecret: "some-client-secret",
					},
				},
			},
			want: &v1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: metav1.SchemeGroupVersion.Version,
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      kafkaSREOpenIDPSecretName,
					Namespace: "openshift-config",
				},
				Type: v1.SecretTypeOpaque,
				StringData: map[string]string{
					"clientSecret": "some-client-secret",
				},
			},
		},
		{
			name: "buids a k8s secret with another given client secret",
			args: args{
				idpProviderInfo: types.IdentityProviderInfo{
					OpenID: &types.OpenIDIdentityProviderInfo{
						ClientSecret: "some-other-client-secret",
					},
				},
			},
			want: &v1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: metav1.SchemeGroupVersion.Version,
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      kafkaSREOpenIDPSecretName,
					Namespace: "openshift-config",
				},
				Type: v1.SecretTypeOpaque,
				StringData: map[string]string{
					"clientSecret": "some-other-client-secret",
				},
			},
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			provider := newStandaloneProvider(db.NewMockConnectionFactory(nil), config.NewDataplaneClusterConfig())
			secret := provider.buildOpenIDPClientSecret(test.args.idpProviderInfo)
			g.Expect(secret).To(gomega.Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildIdentityProviderResource(t *testing.T) {
	type args struct {
		idpProviderInfo types.IdentityProviderInfo
	}

	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{
			name: "buids a k8s secret with a given client secret",
			args: args{
				idpProviderInfo: types.IdentityProviderInfo{
					OpenID: &types.OpenIDIdentityProviderInfo{
						ClientSecret: "some-client-secret",
						ID:           "some-id",
						Name:         "some-name",
						ClientID:     "some-client-id",
						Issuer:       "some-issuer",
					},
				},
			},
			want: map[string]interface{}{
				"apiVersion": "config.openshift.io/v1",
				"kind":       "OAuth",
				"metadata": map[string]string{
					"name": "cluster",
				},
				"spec": map[string]interface{}{
					"identityProviders": []map[string]interface{}{
						{
							"name":          "some-name",
							"mappingMethod": "claim",
							"type":          "OpenID",
							"openID": map[string]interface{}{
								"clientID": "some-client-id",
								"issuer":   "some-issuer",
								"clientSecret": map[string]string{
									"name": kafkaSREOpenIDPSecretName,
								},
								"claims": map[string][]string{
									"email":             {"email"},
									"preferredUsername": {"preferred_username"},
									"last_name":         {"preferred_username"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "buids a k8s secret with another given client secret",
			args: args{
				idpProviderInfo: types.IdentityProviderInfo{
					OpenID: &types.OpenIDIdentityProviderInfo{
						ClientSecret: "some-other-client-secret",
						ID:           "some-id-1",
						Name:         "some-name-1",
						ClientID:     "some-client-id-1",
						Issuer:       "some-issuer-1",
					},
				},
			},
			want: map[string]interface{}{
				"apiVersion": "config.openshift.io/v1",
				"kind":       "OAuth",
				"metadata": map[string]string{
					"name": "cluster",
				},
				"spec": map[string]interface{}{
					"identityProviders": []map[string]interface{}{
						{
							"name":          "some-name-1",
							"mappingMethod": "claim",
							"type":          "OpenID",
							"openID": map[string]interface{}{
								"clientID": "some-client-id-1",
								"issuer":   "some-issuer-1",
								"clientSecret": map[string]string{
									"name": kafkaSREOpenIDPSecretName,
								},
								"claims": map[string][]string{
									"email":             {"email"},
									"preferredUsername": {"preferred_username"},
									"last_name":         {"preferred_username"},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			provider := newStandaloneProvider(db.NewMockConnectionFactory(nil), config.NewDataplaneClusterConfig())
			secret := provider.buildIdentityProviderResource(test.args.idpProviderInfo)
			g.Expect(secret).To(gomega.Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildStrimziOperatorNamespace(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *v1.Namespace
	}{
		{
			name: "buids a namespace with a given name",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					StrimziOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace: "namespace-name",
					},
				},
			},
			want: &v1.Namespace{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace-name",
				},
			},
		},
		{
			name: "buids a namespace with another given name",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					StrimziOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace: "another-namespace-name",
					},
				},
			},
			want: &v1.Namespace{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "another-namespace-name",
				},
			},
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			namespace := provider.buildStrimziOperatorNamespace()
			g.Expect(namespace).To(gomega.Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildStrimziCatalogSource(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *operatorsv1alpha1.CatalogSource
	}{
		{
			name: "buids a catalog source with a given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					StrimziOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:  "namespace-name",
						IndexImage: "index-image-1",
					},
				},
			},
			want: &operatorsv1alpha1.CatalogSource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha1.CatalogSourceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      strimziOperatorCatalogSourceName,
					Namespace: "namespace-name",
				},
				Spec: operatorsv1alpha1.CatalogSourceSpec{
					SourceType: operatorsv1alpha1.SourceTypeGrpc,
					Image:      "index-image-1",
				},
			},
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			catalogSource := provider.buildStrimziOperatorCatalogSource()
			g.Expect(catalogSource).To(gomega.Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildStrimziOperatorGroup(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *operatorsv1alpha2.OperatorGroup
	}{
		{
			name: "buids a operator group with a given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					StrimziOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:  "namespace-name",
						IndexImage: "index-image-1",
					},
				},
			},
			want: &operatorsv1alpha2.OperatorGroup{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha2.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha2.OperatorGroupKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      strimziOperatorOperatorGroupName,
					Namespace: "namespace-name",
				},
				Spec: operatorsv1alpha2.OperatorGroupSpec{},
			},
		},
		{
			name: "buids an operator group with another given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					StrimziOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:  "another-namespace-name",
						IndexImage: "index-image-2",
					},
				},
			},
			want: &operatorsv1alpha2.OperatorGroup{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha2.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha2.OperatorGroupKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      strimziOperatorOperatorGroupName,
					Namespace: "another-namespace-name",
				},
				Spec: operatorsv1alpha2.OperatorGroupSpec{},
			},
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			operatorGroup := provider.buildStrimziOperatorOperatorGroup()
			g.Expect(operatorGroup).To(gomega.Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildStrimziOperatorSubscription(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *operatorsv1alpha1.Subscription
	}{
		{
			name: "buids a operator subscription with a given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					StrimziOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:           "namespace-name",
						IndexImage:          "index-image-1",
						SubscriptionChannel: "alpha",
						Package:             "package-1",
					},
				},
			},
			want: &operatorsv1alpha1.Subscription{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha1.SubscriptionKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      strimziOperatorSubscriptionName,
					Namespace: "namespace-name",
				},
				Spec: &operatorsv1alpha1.SubscriptionSpec{
					CatalogSource:          strimziOperatorCatalogSourceName,
					Channel:                "alpha",
					CatalogSourceNamespace: "namespace-name",
					InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
					Package:                "package-1",
				},
			},
		},
		{
			name: "buids an operator subscription with another given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					StrimziOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:           "another-namespace-name",
						IndexImage:          "index-image-2",
						SubscriptionChannel: "beta",
						Package:             "package-2",
					},
				},
			},
			want: &operatorsv1alpha1.Subscription{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha1.SubscriptionKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      strimziOperatorSubscriptionName,
					Namespace: "another-namespace-name",
				},
				Spec: &operatorsv1alpha1.SubscriptionSpec{
					CatalogSource:          strimziOperatorCatalogSourceName,
					Channel:                "beta",
					CatalogSourceNamespace: "another-namespace-name",
					InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
					Package:                "package-2",
				},
			},
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			subscription := provider.buildStrimziOperatorSubscription()
			g.Expect(subscription).To(gomega.Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildKasFleetshardOperatorNamespace(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *v1.Namespace
	}{
		{
			name: "buids a namespace with a given name",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					KasFleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace: "namespace-name",
					},
				},
			},
			want: &v1.Namespace{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace-name",
				},
			},
		},
		{
			name: "buids a namespace with another given name",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					KasFleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace: "another-namespace-name",
					},
				},
			},
			want: &v1.Namespace{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1.SchemeGroupVersion.String(),
					Kind:       "Namespace",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "another-namespace-name",
				},
			},
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			namespace := provider.buildKASFleetShardOperatorNamespace()
			g.Expect(namespace).To(gomega.Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildKasFleetshardSyncSecret(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	type args struct {
		params []types.Parameter
	}

	tests := []struct {
		name   string
		args   args
		fields fields
		want   *v1.Secret
	}{
		{
			name: "buids a secrets with a given parameters",
			args: args{
				params: []types.Parameter{
					{
						Id:    "id-1",
						Value: "value-1",
					},
				},
			},
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					KasFleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace: "namespace-name",
					},
				},
			},
			want: &v1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      kasFleetShardOperatorParametersSecretName,
					Namespace: "namespace-name",
				},
				StringData: map[string]string{
					"id-1": "value-1",
				},
			},
		},
		{
			name: "buids a secret with another given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					KasFleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace: "another-namespace-name",
					},
				},
			},
			args: args{
				params: []types.Parameter{
					{
						Id:    "id",
						Value: "value",
					},
					{
						Id:    "id-2",
						Value: "value-2",
					},
				},
			},
			want: &v1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      kasFleetShardOperatorParametersSecretName,
					Namespace: "another-namespace-name",
				},
				StringData: map[string]string{
					"id":   "value",
					"id-2": "value-2",
				},
			},
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			secret := provider.buildKASFleetShardSyncSecret(test.args.params)
			g.Expect(secret).To(gomega.Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildKasFleetshardCatalogSource(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *operatorsv1alpha1.CatalogSource
	}{
		{
			name: "buids a catalog source with a given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					KasFleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:  "namespace-name",
						IndexImage: "index-image-1",
					},
				},
			},
			want: &operatorsv1alpha1.CatalogSource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha1.CatalogSourceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      kasFleetShardOperatorCatalogSourceName,
					Namespace: "namespace-name",
				},
				Spec: operatorsv1alpha1.CatalogSourceSpec{
					SourceType: operatorsv1alpha1.SourceTypeGrpc,
					Image:      "index-image-1",
				},
			},
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			catalogSource := provider.buildKASFleetShardOperatorCatalogSource()
			g.Expect(catalogSource).To(gomega.Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildKasFleetshardOperatorGroup(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *operatorsv1alpha2.OperatorGroup
	}{
		{
			name: "buids a operator group with a given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					KasFleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:  "namespace-name",
						IndexImage: "index-image-1",
					},
				},
			},
			want: &operatorsv1alpha2.OperatorGroup{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha2.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha2.OperatorGroupKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      kasFleetShardOperatorOperatorGroupName,
					Namespace: "namespace-name",
				},
				Spec: operatorsv1alpha2.OperatorGroupSpec{},
			},
		},
		{
			name: "buids an operator group with another given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					KasFleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:  "another-namespace-name",
						IndexImage: "index-image-2",
					},
				},
			},
			want: &operatorsv1alpha2.OperatorGroup{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha2.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha2.OperatorGroupKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      kasFleetShardOperatorOperatorGroupName,
					Namespace: "another-namespace-name",
				},
				Spec: operatorsv1alpha2.OperatorGroupSpec{},
			},
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			operatorGroup := provider.buildKASFleetShardOperatorOperatorGroup()
			g.Expect(operatorGroup).To(gomega.Equal(test.want))
		})
	}
}

func TestStandaloneProvider_buildKasFleetshardOperatorSubscription(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}

	tests := []struct {
		name   string
		fields fields
		want   *operatorsv1alpha1.Subscription
	}{
		{
			name: "buids a operator subscription with a given parameters",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: mock.BuildValidDataPlaneClusterConfigKasFleetshardOperatorOLMConfig(),
			},
			want: &operatorsv1alpha1.Subscription{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha1.SubscriptionKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      kasFleetShardOperatorSubscriptionName,
					Namespace: "namespace-name",
				},
				Spec: &operatorsv1alpha1.SubscriptionSpec{
					CatalogSource:          kasFleetShardOperatorCatalogSourceName,
					Channel:                "alpha",
					CatalogSourceNamespace: "namespace-name",
					InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
					Package:                "package-1",
				},
			},
		},
		{
			name: "buids an operator subscription with another given parameters",
			fields: fields{
				connectionFactory: db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: &config.DataplaneClusterConfig{
					KasFleetshardOperatorOLMConfig: config.OperatorInstallationConfig{
						Namespace:           "another-namespace-name",
						IndexImage:          "index-image-2",
						SubscriptionChannel: "beta",
						Package:             "package-2",
					},
				},
			},
			want: &operatorsv1alpha1.Subscription{
				TypeMeta: metav1.TypeMeta{
					APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
					Kind:       operatorsv1alpha1.SubscriptionKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      kasFleetShardOperatorSubscriptionName,
					Namespace: "another-namespace-name",
				},
				Spec: &operatorsv1alpha1.SubscriptionSpec{
					CatalogSource:          kasFleetShardOperatorCatalogSourceName,
					Channel:                "beta",
					CatalogSourceNamespace: "another-namespace-name",
					InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
					Package:                "package-2",
				},
			},
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			subscription := provider.buildKASFleetShardOperatorSubscription()
			g.Expect(subscription).To(gomega.Equal(test.want))
		})
	}
}
func TestStandaloneProvider_InstallStrimzi(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}
	type args struct {
		clusterSpec *types.ClusterSpec
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "should InstallStrimzi without error",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: mock.BuildValidDataPlaneClusterConfigKasFleetshardOperatorOLMConfig(),
			},
			args: args{
				clusterSpec: &types.ClusterSpec{},
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			ok, err := provider.InstallStrimzi(test.args.clusterSpec)
			g.Expect(err != nil).To(gomega.Equal(test.wantErr))
			g.Expect(ok).To(gomega.Equal(test.want))
		})
	}
}

func TestStandaloneProvider_InstallKasFleetshard(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}
	type args struct {
		clusterSpec *types.ClusterSpec
		params      []types.Parameter
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "should Install Kas Fleet Shard without error",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: mock.BuildValidDataPlaneClusterConfigKasFleetshardOperatorOLMConfig(),
			},
			args: args{
				clusterSpec: &types.ClusterSpec{},
				params: []types.Parameter{
					{
						Id:    "param1",
						Value: "param-value-1",
					},
				},
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			ok, err := provider.InstallKasFleetshard(test.args.clusterSpec, test.args.params)
			g.Expect(err != nil).To(gomega.Equal(test.wantErr))
			g.Expect(ok).To(gomega.Equal(test.want))
		})
	}
}

func TestStandaloneProvider_AddIdentityProvider(t *testing.T) {
	type fields struct {
		connectionFactory      *db.ConnectionFactory
		dataplaneClusterConfig *config.DataplaneClusterConfig
	}
	type args struct {
		clusterSpec      *types.ClusterSpec
		identityProvider types.IdentityProviderInfo
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *types.IdentityProviderInfo
		wantErr bool
	}{
		{
			name: "should Add Identity Provider without issue",
			fields: fields{
				connectionFactory:      db.NewMockConnectionFactory(nil),
				dataplaneClusterConfig: mock.BuildValidDataPlaneClusterConfigKasFleetshardOperatorOLMConfig(),
			},
			args: args{
				clusterSpec: &types.ClusterSpec{},
				identityProvider: types.IdentityProviderInfo{
					OpenID: &types.OpenIDIdentityProviderInfo{
						Name:         "test-name",
						ClientID:     "test-client-id",
						ClientSecret: "test-client-secret",
						Issuer:       "test-issuer",
					},
				},
			},
			want: &types.IdentityProviderInfo{
				OpenID: &types.OpenIDIdentityProviderInfo{
					Name:         "test-name",
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
					Issuer:       "test-issuer",
				},
			},
		},
	}

	for _, testcase := range tests {
		test := testcase
		t.Run(test.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			provider := newStandaloneProvider(test.fields.connectionFactory, test.fields.dataplaneClusterConfig)
			ok, err := provider.AddIdentityProvider(test.args.clusterSpec, test.args.identityProvider)
			g.Expect(err != nil).To(gomega.Equal(test.wantErr))
			g.Expect(ok).To(gomega.Equal(test.want))
		})
	}
}

func Test_shouldApplyChanges(t *testing.T) {
	type args struct {
		dynamicClient    dynamic.ResourceInterface
		existingObj      *unstructured.Unstructured
		newConfiguration string
	}
	var obj unstructured.Unstructured
	newAnnotations := obj.GetAnnotations()
	obj.SetAnnotations(newAnnotations)

	var newObj unstructured.Unstructured
	newAnnotations = map[string]string{
		lastAppliedConfigurationAnnotation: "lastAppliedConfigurationAnnotation",
	}
	newObj.SetAnnotations(newAnnotations)

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Should return true if exsistingObj is nil",
			args: args{
				existingObj: nil,
			},
			want: true,
		},
		{
			name: "Should return true if existingObj.GetAnnotations() == nil",
			args: args{
				dynamicClient:    nil,
				existingObj:      &unstructured.Unstructured{},
				newConfiguration: "",
			},
			want: true,
		},
		{
			name: "Should return true if changes are applied",
			args: args{
				dynamicClient:    nil,
				existingObj:      &obj,
				newConfiguration: "",
			},
			want: true,
		},
		{
			name: "Should return false if changes are not applied",
			args: args{
				dynamicClient:    nil,
				existingObj:      &newObj,
				newConfiguration: "lastAppliedConfigurationAnnotation",
			},
			want: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got := shouldApplyChanges(tt.args.dynamicClient, tt.args.existingObj, tt.args.newConfiguration)
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func TestStandaloneProvider_GetMachinePool(t *testing.T) {
	sampleMachinePoolID := "test-machinepool-id"
	type args struct {
		clusterID     string
		machinePoolID string
	}
	tests := []struct {
		name    string
		args    args
		want    *types.MachinePoolInfo
		wantErr bool
	}{
		{
			name: "Always returns a MachinePool with the provided MachinePool ID",
			args: args{
				clusterID:     "test-cluster-id",
				machinePoolID: sampleMachinePoolID,
			},
			want: &types.MachinePoolInfo{
				ID: sampleMachinePoolID,
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			standaloneProvider := newStandaloneProvider(nil, nil)
			got, err := standaloneProvider.GetMachinePool(tt.args.clusterID, tt.args.machinePoolID)
			gotErr := err != nil
			g.Expect(gotErr).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func TestStandaloneProvider_CreateMachinePool(t *testing.T) {
	type args struct {
		machinePoolRequest types.MachinePoolRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *types.MachinePoolRequest
		wantErr bool
	}{
		{
			name: "Always returns nil and no error",
			args: args{
				machinePoolRequest: types.MachinePoolRequest{},
			},
			want:    nil,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			standaloneProvider := newStandaloneProvider(nil, nil)

			got, err := standaloneProvider.CreateMachinePool(&tt.args.machinePoolRequest)
			gotErr := err != nil
			g.Expect(gotErr).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}
