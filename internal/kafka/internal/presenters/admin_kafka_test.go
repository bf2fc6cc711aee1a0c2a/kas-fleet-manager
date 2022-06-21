package presenters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	mock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/account"

	. "github.com/onsi/gomega"
)

func TestGetRoutesFromKafkaRequest(t *testing.T) {
	type args struct {
		dbKafkaRequest *dbapi.KafkaRequest
	}

	var emptyKafkaRoutes []private.KafkaAllOfRoutes

	tests := []struct {
		name string
		args args
		want []private.KafkaAllOfRoutes
	}{
		{
			name: "should return empty routes without any routes in the KafkaRequest",
			args: args{
				dbKafkaRequest: &dbapi.KafkaRequest{},
			},
			want: emptyKafkaRoutes,
		},
		{
			name: "should return empty routes if the KafkaRequest routes are malformed",
			args: args{
				dbKafkaRequest: mock.BuildKafkaRequest(
					mock.WithPredefinedTestValues(),
					mock.WithRoutes(mock.GetMalformedRoutes()),
				),
			},
			want: emptyKafkaRoutes,
		},
		{
			name: "should return routes if not empty and in correct format in the KafkaRequest",
			args: args{
				dbKafkaRequest: mock.BuildKafkaRequest(
					mock.WithPredefinedTestValues(),
					mock.WithRoutes(mock.GetRoutes()),
				),
			},
			want: mock.GetSampleKafkaAllRoutes(),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			Expect(GetRoutesFromKafkaRequest(tt.args.dbKafkaRequest)).To(Equal(tt.want))
		})
	}
}

func TestPresentKafkaRequestAdminEndpoint(t *testing.T) {
	type args struct {
		dbKafkaRequest *dbapi.KafkaRequest
		accountService account.AccountService
	}

	storageSize := "1000Gi"
	tests := []struct {
		name    string
		args    args
		want    *private.Kafka
		wantErr bool
	}{
		{
			name: "should return admin kafka request",
			args: args{
				dbKafkaRequest: mock.BuildKafkaRequest(
					mock.WithPredefinedTestValues(),
					mock.With(mock.STORAGE_SIZE, storageSize),
				),
				accountService: account.NewMockAccountService(),
			},
			want: mock.BuildAdminKafkaRequest(func(kafka *private.Kafka) {
				kafka.KafkaStorageSize = storageSize
				kafka.OrganisationId = mock.DefaultOrganisationId

				dataRetentionSizeQuantity := config.Quantity(storageSize)
				dataRetentionSizeBytes, err := dataRetentionSizeQuantity.ToInt64()
				Expect(err).ToNot(HaveOccurred(), "failed to convert kafka data retention size '%s' to bytes", storageSize)

				kafka.MaxDataRetentionSize = private.SupportedKafkaSizeBytesValueItem{
					Bytes: dataRetentionSizeBytes,
				}
			}),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			converted, err := PresentKafkaRequestAdminEndpoint(tt.args.dbKafkaRequest, tt.args.accountService)
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for PresentKafkaRequestAdminEndpoint: %v", err)
				return
			}
			Expect(converted).To(Equal(tt.want))
		})
	}
}
