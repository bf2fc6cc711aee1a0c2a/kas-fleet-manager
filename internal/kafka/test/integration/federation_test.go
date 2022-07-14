package integration

import (
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/golang-jwt/jwt/v4"
	"github.com/onsi/gomega"
)

func TestFederation_GetFederatedMetrics(t *testing.T) {
	g := gomega.NewWithT(t)
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	// create kafka
	db := h.DBFactory().New()
	account := h.NewRandAccount()
	org, ok := account.GetOrganization()
	if !ok {
		t.Errorf("failed to get organization id for kafka owner")
	}
	kafkaId := h.NewID()
	kafka := &dbapi.KafkaRequest{
		Meta: api.Meta{
			ID: kafkaId,
		},
		MultiAZ:                 true,
		Owner:                   account.Username(),
		Region:                  "us-east-1",
		CloudProvider:           "aws",
		Name:                    "example-kafka",
		OrganisationId:          org.ExternalID(),
		Status:                  constants.KafkaRequestStatusReady.String(),
		ClusterID:               clusterID,
		ActualKafkaVersion:      "2.6.0",
		DesiredKafkaVersion:     "2.6.0",
		ActualStrimziVersion:    "v.23.0",
		DesiredStrimziVersion:   "v0.23.0",
		InstanceType:            types.STANDARD.String(),
		ReauthenticationEnabled: true,
	}

	if err := db.Create(kafka).Error; err != nil {
		t.Errorf("failed to create Kafka db record due to error: %v", err)
	}

	ctx := h.NewAuthenticatedContext(account, nil)

	federatedMetrics, resp, err := client.DefaultApi.FederateMetrics(ctx, kafkaId)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to call federation endpoint:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(federatedMetrics).NotTo(gomega.BeEmpty())
}

// NOTE: MAS SSO auth support for the /federate endpoint is only a temporary solution.
// This should be removed once we migrate to sso.redhat.com (TODO: to be removed as part of MGDSTRM-6159)
func TestFederation_GetFederatedMetricsUsingMasSsoToken(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()
	defer mockKasfFleetshardSync.Stop()

	clusterID, getClusterErr := common.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	// create kafka
	db := h.DBFactory().New()
	owner := h.NewRandAccount()
	org, ok := owner.GetOrganization()
	if !ok {
		t.Errorf("failed to get organization id for kafka owner")
	}
	kafkaId := h.NewID()
	kafka := &dbapi.KafkaRequest{
		Meta: api.Meta{
			ID: kafkaId,
		},
		MultiAZ:                 true,
		Owner:                   owner.Username(),
		Region:                  "us-east-1",
		CloudProvider:           "aws",
		Name:                    "example-kafka",
		OrganisationId:          org.ExternalID(),
		Status:                  constants.KafkaRequestStatusReady.String(),
		ClusterID:               clusterID,
		ActualKafkaVersion:      "2.6.0",
		DesiredKafkaVersion:     "2.6.0",
		ActualStrimziVersion:    "v.23.0",
		DesiredStrimziVersion:   "v0.23.0",
		InstanceType:            types.STANDARD.String(),
		ReauthenticationEnabled: true,
	}

	if err := db.Create(kafka).Error; err != nil {
		t.Errorf("failed to create Kafka db record due to error: %v", err)
	}

	// Call Kafka federate metrics using mas sso token
	masSsoSA := h.NewAccount("service-account-srvc-acct-1", "", "", org.ExternalID())
	var keycloakConfig *keycloak.KeycloakConfig
	h.Env.MustResolveAll(&keycloakConfig)
	claims := jwt.MapClaims{
		"iss":                keycloakConfig.SSOProviderRealm().ValidIssuerURI,
		"rh-org-id":          org.ExternalID(),
		"rh-user-id":         masSsoSA.ID(),
		"preferred_username": masSsoSA.Username(),
	}
	masSsoCtx := h.NewAuthenticatedContext(masSsoSA, claims)

	federatedMetrics, resp, err := client.DefaultApi.FederateMetrics(masSsoCtx, kafkaId)
	if resp != nil {
		resp.Body.Close()
	}
	g.Expect(err).NotTo(gomega.HaveOccurred(), "Error occurred when attempting to call federation endpoint:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(federatedMetrics).NotTo(gomega.BeEmpty())
}
