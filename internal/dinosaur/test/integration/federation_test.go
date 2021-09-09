package integration

import (
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/dinosaurs/types"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test/mocks/kasfleetshardsync"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test/mocks"
	"github.com/dgrijalva/jwt-go"
	. "github.com/onsi/gomega"
)

func TestFederation_GetFederatedMetrics(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewDinosaurHelper(t, ocmServer)
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

	// create dinosaur
	db := h.DBFactory().New()
	account := h.NewRandAccount()
	org, ok := account.GetOrganization()
	if !ok {
		t.Errorf("failed to get organization id for dinosaur owner")
	}
	dinosaurId := h.NewID()
	dinosaur := &dbapi.DinosaurRequest{
		Meta: api.Meta{
			ID: dinosaurId,
		},
		MultiAZ:                 true,
		Owner:                   account.Username(),
		Region:                  "us-east-1",
		CloudProvider:           "aws",
		Name:                    "example-dinosaur",
		OrganisationId:          org.ExternalID(),
		Status:                  constants.DinosaurRequestStatusReady.String(),
		ClusterID:               clusterID,
		ActualDinosaurVersion:   "2.6.0",
		DesiredDinosaurVersion:  "2.6.0",
		ActualStrimziVersion:    "v.23.0",
		DesiredStrimziVersion:   "v0.23.0",
		InstanceType:            types.STANDARD.String(),
		ReauthenticationEnabled: true,
	}

	if err := db.Create(dinosaur).Error; err != nil {
		t.Errorf("failed to create Dinosaur db record due to error: %v", err)
	}

	ctx := h.NewAuthenticatedContext(account, nil)

	federatedMetrics, resp, err := client.DefaultApi.FederateMetrics(ctx, dinosaurId)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to call federation endpoint:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(federatedMetrics).NotTo(BeEmpty())
}

// NOTE: MAS SSO auth support for the /federate endpoint is only a temporary solution.
// This should be removed once we migrate to sso.redhat.com (TODO: to be removed as part of MGDSTRM-6159)
func TestFederation_GetFederatedMetricsUsingMasSsoToken(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewDinosaurHelper(t, ocmServer)
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

	// create dinosaur
	db := h.DBFactory().New()
	owner := h.NewRandAccount()
	org, ok := owner.GetOrganization()
	if !ok {
		t.Errorf("failed to get organization id for dinosaur owner")
	}
	dinosaurId := h.NewID()
	dinosaur := &dbapi.DinosaurRequest{
		Meta: api.Meta{
			ID: dinosaurId,
		},
		MultiAZ:                 true,
		Owner:                   owner.Username(),
		Region:                  "us-east-1",
		CloudProvider:           "aws",
		Name:                    "example-dinosaur",
		OrganisationId:          org.ExternalID(),
		Status:                  constants.DinosaurRequestStatusReady.String(),
		ClusterID:               clusterID,
		ActualDinosaurVersion:   "2.6.0",
		DesiredDinosaurVersion:  "2.6.0",
		ActualStrimziVersion:    "v.23.0",
		DesiredStrimziVersion:   "v0.23.0",
		InstanceType:            types.STANDARD.String(),
		ReauthenticationEnabled: true,
	}

	if err := db.Create(dinosaur).Error; err != nil {
		t.Errorf("failed to create Dinosaur db record due to error: %v", err)
	}

	// Call Dinosaur federate metrics using mas sso token
	masSsoSA := h.NewAccount("service-account-srvc-acct-1", "", "", org.ExternalID())
	var keycloakConfig *keycloak.KeycloakConfig
	h.Env.MustResolveAll(&keycloakConfig)
	claims := jwt.MapClaims{
		"iss":                keycloakConfig.DinosaurRealm.ValidIssuerURI,
		"rh-org-id":          org.ExternalID(),
		"rh-user-id":         masSsoSA.ID(),
		"preferred_username": masSsoSA.Username(),
	}
	masSsoCtx := h.NewAuthenticatedContext(masSsoSA, claims)

	federatedMetrics, resp, err := client.DefaultApi.FederateMetrics(masSsoCtx, dinosaurId)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to call federation endpoint:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(federatedMetrics).NotTo(BeEmpty())
}
