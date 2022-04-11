package presenters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	mocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/service_accounts"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	. "github.com/onsi/gomega"
)

func TestConvertServiceAccountRequest(t *testing.T) {
	type args struct {
		from public.ServiceAccountRequest
	}

	tests := []struct {
		name string
		args args
		want *api.ServiceAccountRequest
	}{
		{
			name: "should successfully convert non-empty ServiceAccountRequest",
			args: args{
				from: mocks.BuildServiceAccountRequest(nil),
			},
			want: mocks.BuildApiServiceAccountRequest(nil),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(ConvertServiceAccountRequest(tt.args.from)).To(Equal(tt.want))
		})
	}
}

func TestPresentServiceAccount(t *testing.T) {
	type args struct {
		from *api.ServiceAccount
	}

	tests := []struct {
		name string
		args args
		want *public.ServiceAccount
	}{
		{
			name: "should return ServiceAccount as presented to the end user",
			args: args{
				from: mocks.BuildApiServiceAccount(nil),
			},
			want: mocks.BuildServiceAccount(nil),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(PresentServiceAccount(tt.args.from)).To(Equal(tt.want))
		})
	}
}

func TestPresentServiceAccountListItem(t *testing.T) {
	type args struct {
		from *api.ServiceAccount
	}

	tests := []struct {
		name string
		args args
		want public.ServiceAccountListItem
	}{
		{
			name: "should present ServiceAccountListItem",
			args: args{
				from: mocks.BuildApiServiceAccount(nil),
			},
			want: mocks.BuildServiceAccountListItem(nil),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(PresentServiceAccountListItem(tt.args.from)).To(Equal(tt.want))
		})
	}
}

func TestPresentSsoProvider(t *testing.T) {
	type args struct {
		from *api.SsoProvider
	}

	tests := []struct {
		name string
		args args
		want public.SsoProvider
	}{
		{
			name: "should present PresentSsoProvider",
			args: args{
				from: mocks.BuildApiSsoProvier(nil),
			},
			want: mocks.BuildSsoProvider(nil),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(PresentSsoProvider(tt.args.from)).To(Equal(tt.want))
		})
	}
}
