package presenters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	mock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	mockSa "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/service_accounts"
	v1 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/managedkafkas.managedkafka.bf2.org/v1"

	. "github.com/onsi/gomega"
)

func TestPresentManagedKafka(t *testing.T) {
	type args struct {
		from *v1.ManagedKafka
	}

	tests := []struct {
		name string
		args args
		want private.ManagedKafka
	}{
		{
			name: "should return ManagedKafka with non-empty 'from' spec",
			args: args{
				from: mock.BuildManagedKafka(nil),
			},
			want: *mock.BuildPrivateKafka(func(kafka *private.ManagedKafka) {
				kafka.Spec.ServiceAccounts = getServiceAccounts([]v1.ServiceAccount{})
			}),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(PresentManagedKafka(tt.args.from)).To(Equal(tt.want))
		})
	}
}

func TestGetOpenAPIManagedKafkaEndpointTLS(t *testing.T) {
	type args struct {
		from *v1.TlsSpec
	}

	tests := []struct {
		name string
		args args
		want *private.ManagedKafkaAllOfSpecEndpointTls
	}{
		{
			name: "should return ManagedKafkaAllOfSpecEndpointTls with non-empty 'from' spec",
			args: args{
				from: mock.BuildTlsSpec(nil),
			},
			want: mock.BuildManagedKafkaAllOfSpecEndpointTls(nil),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(getOpenAPIManagedKafkaEndpointTLS(tt.args.from)).To(Equal(tt.want))
		})
	}
}

func TestGetOpenAPIManagedKafkaOAuthTLSTrustedCertificate(t *testing.T) {
	type args struct {
		from *v1.OAuthSpec
	}

	tests := []struct {
		name string
		args args
		want *string
	}{
		{
			name: "should return OpenAPIManagedKafkaOAuthTLSTrustedCertificate with non-empty 'from' spec",
			args: args{
				from: mock.BuildOAuthSpec(nil),
			},
			want: mock.GetTrustedCertValue(),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(getOpenAPIManagedKafkaOAuthTLSTrustedCertificate(tt.args.from)).To(Equal(tt.want))
		})
	}
}

func TestGetServiceAccounts(t *testing.T) {
	type args struct {
		from []v1.ServiceAccount
	}

	tests := []struct {
		name string
		args args
		want []private.ManagedKafkaAllOfSpecServiceAccounts
	}{
		{
			name: "should return ServiceAccounts with non-empty 'from' service accounts",
			args: args{
				from: mockSa.GetServiceAccounts(),
			},
			want: mockSa.GetManagedKafkaAllOfSpecServiceAccounts(),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(getServiceAccounts(tt.args.from)).To(Equal(tt.want))
		})
	}
}
