package presenters

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	mock "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"

	. "github.com/onsi/gomega"
)

func TestConvertKafkaRequest(t *testing.T) {
	type args struct {
		kafkaRequestPayload public.KafkaRequestPayload
		dbKafkaRequests     []*dbapi.KafkaRequest
	}

	reauthEnabled := true
	reauthDisabled := false

	tests := []struct {
		name string
		args args
		want *dbapi.KafkaRequest
	}{
		{
			name: "should return converted KafkaRequest from empty db KafkaRequest",
			args: args{
				kafkaRequestPayload: *mock.BuildKafkaRequestPayload(nil),
				dbKafkaRequests:     []*dbapi.KafkaRequest{},
			},
			want: mock.BuildKafkaRequest(
				mock.With(mock.STATUS, ""),
				mock.With(mock.OWNER, ""),
				mock.With(mock.CLUSTER_ID, ""),
				mock.WithReauthenticationEnabled(reauthEnabled),
				mock.WithDeleted(false),
			),
		},
		{
			name: "Should return empty kafka request with reauthentication disabled",
			args: args{
				kafkaRequestPayload: *mock.BuildKafkaRequestPayload(func(payload *public.KafkaRequestPayload) {
					payload.ReauthenticationEnabled = &reauthDisabled
				}),
				dbKafkaRequests: []*dbapi.KafkaRequest{mock.BuildKafkaRequest(
					mock.WithReauthenticationEnabled(reauthEnabled),
				)},
			},
			want: mock.BuildKafkaRequest(
				mock.WithReauthenticationEnabled(reauthDisabled),
			),
		},
		{
			name: "should convert and return kafka request from non-empty db kafka request slice with reauth enabled",
			args: args{
				kafkaRequestPayload: *mock.BuildKafkaRequestPayload(nil),
				dbKafkaRequests: []*dbapi.KafkaRequest{mock.BuildKafkaRequest(
					mock.WithReauthenticationEnabled(true),
				)},
			},
			want: mock.BuildKafkaRequest(
				mock.WithReauthenticationEnabled(true),
			),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(ConvertKafkaRequest(tt.args.kafkaRequestPayload, tt.args.dbKafkaRequests...)).To(Equal(tt.want))
		})
	}
}

func TestPresentKafkaRequest(t *testing.T) {
	type args struct {
		dbKafkaRequest *dbapi.KafkaRequest
	}

	bootstrapServer := "http://test.com"
	failedReason := ""
	version := "2.8.0"
	instanceType := types.STANDARD.String()
	reauthEnabled := true
	kafkaStorageSize := "1000Gi"

	tests := []struct {
		name string
		args args
		want public.KafkaRequest
	}{
		{
			name: "should return kafka request as presented to an end user",
			args: args{
				dbKafkaRequest: mock.BuildKafkaRequest(
					mock.WithReauthenticationEnabled(reauthEnabled),
					mock.With(mock.BOOTSTRAP_SERVER_HOST, bootstrapServer),
					mock.With(mock.FAILED_REASON, failedReason),
					mock.With(mock.ACTUAL_KAFKA_VERSION, version),
					mock.With(mock.INSTANCE_TYPE, instanceType),
					mock.With(mock.STORAGE_SIZE, kafkaStorageSize),
				),
			},
			want: *mock.BuildPublicKafkaRequest(func(kafkaRequest *public.KafkaRequest) {
				kafkaRequest.ReauthenticationEnabled = reauthEnabled
				kafkaRequest.BootstrapServerHost = setBootstrapServerHost(bootstrapServer)
				kafkaRequest.FailedReason = failedReason
				kafkaRequest.InstanceType = instanceType
				kafkaRequest.KafkaStorageSize = kafkaStorageSize
				kafkaRequest.BrowserUrl = "//dashboard"
			}),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(PresentKafkaRequest(tt.args.dbKafkaRequest, "")).To(Equal(tt.want))
		})
	}
}

func TestSetBootstrapServerHost(t *testing.T) {
	type args struct {
		bootstrapServerHost string
	}
	nonEmptyBootstrapServerHost := "http://some-url.com"

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "non empty bootstrap server host conversion should return its value with colon and port number as a suffix",
			args: args{
				bootstrapServerHost: nonEmptyBootstrapServerHost,
			},
			want: fmt.Sprintf("%s:443", nonEmptyBootstrapServerHost),
		},
		{
			name: "empty bootstrap server host conversion should return empty value",
			args: args{
				bootstrapServerHost: "",
			},
			want: "",
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(setBootstrapServerHost(tt.args.bootstrapServerHost)).To(Equal(tt.want))
		})
	}
}
