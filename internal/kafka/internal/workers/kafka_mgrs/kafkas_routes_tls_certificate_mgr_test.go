package kafka_mgrs

import (
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"

	"github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	w "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
)

func TestKafkasRoutesTLSCertificateManager_Reconcile(t *testing.T) {
	type fields struct {
		kafkaService services.KafkaService
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Should throw an error if listing kafkas fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return nil, errors.GeneralError("some error")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "should succeed if no kafkas are returned",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
			wantErr: false,
		},
		{
			name: "should throw an error if reconciles kafkas route tls certificate fails",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{{
							Meta: api.Meta{
								ID: "some-id",
							},
							KafkasRoutesBaseDomainName:      "some-base-domain",
							KafkasRoutesBaseDomainTLSKeyRef: "some-key-ref",
							KafkasRoutesBaseDomainTLSCrtRef: "some-crt-ref",
						}}, nil
					},
					ManagedKafkasRoutesTLSCertificateFunc: func(kafkaRequest *dbapi.KafkaRequest) error {
						return fmt.Errorf("some error")
					},
				},
			},
			wantErr: true,
		},
		{
			name: "successfully reconciles kafkas route tls certificate",
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					ListAllFunc: func() (dbapi.KafkaList, *errors.ServiceError) {
						return []*dbapi.KafkaRequest{{
							Meta: api.Meta{
								ID: "some-id",
							},
							KafkasRoutesBaseDomainName:      "some-base-domain",
							KafkasRoutesBaseDomainTLSKeyRef: "some-key-ref",
							KafkasRoutesBaseDomainTLSCrtRef: "some-crt-ref",
						}}, nil
					},
					ManagedKafkasRoutesTLSCertificateFunc: func(kafkaRequest *dbapi.KafkaRequest) error {
						return nil
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
			t.Parallel()
			certificateManager := NewKafkasRoutesTLSCertificateManager(tt.fields.kafkaService, w.Reconciler{})
			g.Expect(len(certificateManager.Reconcile()) > 0).To(gomega.Equal(tt.wantErr))
		})
	}
}
