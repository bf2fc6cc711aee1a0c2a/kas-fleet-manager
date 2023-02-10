package integration

import (
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	mockkafka "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"

	"github.com/onsi/gomega"
)

func TestAdminKafka_KafkaTLSCertificateRevocation(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, nil)
	defer teardown()

	db := test.TestServices.DBFactory.New()
	kafka := mockkafka.BuildKafkaRequest(
		mockkafka.WithPredefinedTestValues(),
	)

	kafka.Status = constants.KafkaRequestStatusReady.String()
	if err := db.Create(kafka).Error; err != nil {
		t.Errorf("failed to create Kafka db record due to error: %v", err)
	}

	adminClient := test.NewAdminPrivateAPIClient(h)

	// setup private admin client with permission
	adminClientContextWithPermission := NewAuthenticatedContextForAdminEndpoints(h, []string{testFullRole})

	// successfully revokes a certificate for the kafka
	successfulResponse, err := adminClient.DefaultApi.RevokeKafkaTLSCertificateBKafkaID(adminClientContextWithPermission, kafka.ID, private.KafkacertificateRevocationRequest{
		RevocationReason: 1,
	})
	if successfulResponse != nil {
		defer successfulResponse.Body.Close()
	}
	g.Expect(err).ToNot(gomega.HaveOccurred(), "should successfully revoke Kafka cert")
	g.Expect(successfulResponse.StatusCode).To(gomega.Equal(http.StatusNoContent))

	// return an error when bad request error when certificate reason is invalid
	badRequestResponse, err := adminClient.DefaultApi.RevokeKafkaTLSCertificateBKafkaID(adminClientContextWithPermission, kafka.ID, private.KafkacertificateRevocationRequest{
		RevocationReason: -1,
	})
	if badRequestResponse != nil {
		defer badRequestResponse.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred(), "should fail to revoke certificate due to invalid reason")
	g.Expect(badRequestResponse.StatusCode).To(gomega.Equal(http.StatusBadRequest), "should fail with bad request error")

	// return an error when kafka cannot be found
	kafkaNotFoundResponse, err := adminClient.DefaultApi.RevokeKafkaTLSCertificateBKafkaID(adminClientContextWithPermission, api.NewID(), private.KafkacertificateRevocationRequest{
		RevocationReason: -1,
	})
	if kafkaNotFoundResponse != nil {
		defer kafkaNotFoundResponse.Body.Close()
	}
	g.Expect(err).To(gomega.HaveOccurred(), "should fail to revoke certificate due to to kafka not being found")
	g.Expect(kafkaNotFoundResponse.StatusCode).To(gomega.Equal(http.StatusNotFound), "should fail with not found error")

	// setup private admin client without permission
	adminClientContextWithoutPermission := NewAuthenticatedContextForAdminEndpoints(h, []string{testReadRole, testWriteRole})

	// successfully revokes a certificate for the kafka
	notFoundResponseDueToMissingPermission, err := adminClient.DefaultApi.RevokeKafkaTLSCertificateBKafkaID(adminClientContextWithoutPermission, kafka.ID, private.KafkacertificateRevocationRequest{
		RevocationReason: 1,
	})
	if notFoundResponseDueToMissingPermission != nil {
		defer notFoundResponseDueToMissingPermission.Body.Close()
	}

	g.Expect(err).To(gomega.HaveOccurred(), "should return forbidden")
	g.Expect(notFoundResponseDueToMissingPermission.StatusCode).To(gomega.Equal(http.StatusNotFound), "should fail with routes not found error due to missing permission")
}
