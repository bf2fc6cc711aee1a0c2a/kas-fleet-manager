package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test/common"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test/mocks/kasfleetshardsync"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/keycloak"

	coreTest "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test/mocks"
	"github.com/dgrijalva/jwt-go"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/pkg/errors"
	"gopkg.in/resty.v1"
)

type TestServer struct {
	OcmServer     *httptest.Server
	TearDown      func()
	ClusterID     string
	Token         string
	Client        *public.APIClient
	PrivateClient *private.APIClient
	Helper        *coreTest.Helper
	Ctx           context.Context
}

type claimsFunc func(account *v1.Account, clusterId string, h *coreTest.Helper) jwt.MapClaims

func setup(t *testing.T, claims claimsFunc, startupHook interface{}) TestServer {

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	h, client, tearDown := test.NewDinosaurHelperWithHooks(t, ocmServer, startupHook)

	clusterId, getClusterErr := common.GetOSDClusterID(h, t, nil)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details: %v", getClusterErr)
	}

	account := h.NewAllowedServiceAccount()
	ctx := h.NewAuthenticatedContext(account, claims(account, clusterId, h))
	token := h.CreateJWTStringWithClaim(account, claims(account, clusterId, h))

	config := private.NewConfiguration()
	config.BasePath = fmt.Sprintf("http://%s", test.TestServices.ServerConfig.BindAddress)
	config.DefaultHeader = map[string]string{
		"Authorization": "Bearer " + token,
	}
	privateClient := private.NewAPIClient(config)

	return TestServer{
		OcmServer:     ocmServer,
		Client:        client,
		PrivateClient: privateClient,
		TearDown: func() {
			ocmServer.Close()
			tearDown()
		},
		ClusterID: clusterId,
		Token:     token,
		Helper:    h,
		Ctx:       ctx,
	}
}

func TestDataPlaneEndpoints_AuthzSuccess(t *testing.T) {
	clusterId := "test-cluster-id"
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		return jwt.MapClaims{
			"iss": test.TestServices.KeycloakConfig.DinosaurRealm.ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"kas-fleetshard-operator-cluster-id": clusterId,
		}
	}, nil)

	defer testServer.TearDown()

	body := map[string]private.DataPlanePineappleStatus{
		testServer.ClusterID: {},
	}
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(body).
		Put(testServer.Helper.RestURL("/agent-clusters/" + clusterId + "/dinosaurs/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest)) //the clusterId is not valid

	clusterStatusUpdateRequest := private.DataPlaneClusterUpdateStatusRequest{}
	restyResp, err = resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(clusterStatusUpdateRequest).
		Put(testServer.Helper.RestURL("/agent-clusters/" + clusterId + "/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest)) //the clusterId is not valid
}

func TestDataPlaneEndpoints_AuthzFailWhenNoRealmRole(t *testing.T) {
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		return jwt.MapClaims{
			"iss":                                test.TestServices.KeycloakConfig.DinosaurRealm.ValidIssuerURI,
			"kas-fleetshard-operator-cluster-id": "test-cluster-id",
		}
	}, nil)

	defer testServer.TearDown()

	body := map[string]private.DataPlanePineappleStatus{
		"test-cluster-id": {},
	}
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(body).
		Put(testServer.Helper.RestURL("/agent-clusters/" + testServer.ClusterID + "/dinosaurs/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusNotFound))

	clusterStatusUpdateRequest := private.DataPlaneClusterUpdateStatusRequest{}
	restyResp, err = resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(clusterStatusUpdateRequest).
		Put(testServer.Helper.RestURL("/agent-clusters/" + testServer.ClusterID + "/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusNotFound))
}

func TestDataPlaneEndpoints_AuthzFailWhenClusterIdNotMatch(t *testing.T) {
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		return jwt.MapClaims{
			"iss": test.TestServices.KeycloakConfig.DinosaurRealm.ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"kas-fleetshard-operator-cluster-id": "different-cluster-id",
		}
	}, nil)
	defer testServer.TearDown()

	body := map[string]private.DataPlanePineappleStatus{
		"test-cluster-id": {},
	}
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(body).
		Put(testServer.Helper.RestURL("/agent-clusters/" + testServer.ClusterID + "/dinosaurs/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusNotFound))

	clusterStatusUpdateRequest := private.DataPlaneClusterUpdateStatusRequest{}
	restyResp, err = resty.R().
		SetHeader("Content-Type", "application/json").
		SetAuthToken(testServer.Token).
		SetBody(clusterStatusUpdateRequest).
		Put(testServer.Helper.RestURL("/agent-clusters/" + testServer.ClusterID + "/status"))

	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusNotFound))
}

func TestDataPlaneEndpoints_GetAndUpdateManagedDinosaurs(t *testing.T) {
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.DinosaurRealm.ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"kas-fleetshard-operator-cluster-id": cid,
		}
	}, nil)
	defer testServer.TearDown()
	bootstrapServerHost := "some-bootstrap⁻host"
	ssoClientID := "some-sso-client-id"
	ssoSecret := "some-sso-secret"

	var testDinosaurs = []*dbapi.DinosaurRequest{
		{
			ClusterID:              testServer.ClusterID,
			MultiAZ:                false,
			Name:                   mockDinosaurName1,
			Namespace:              "mk-1",
			Status:                 constants2.DinosaurRequestStatusDeprovision.String(),
			BootstrapServerHost:    bootstrapServerHost,
			SsoClientID:            ssoClientID,
			SsoClientSecret:        ssoSecret,
			DesiredDinosaurVersion: "2.7.0",
			DesiredStrimziVersion:  "strimzi-cluster-operator.v0.23.0-0",
		},
		{
			ClusterID:              testServer.ClusterID,
			MultiAZ:                false,
			Name:                   mockDinosaurName2,
			Namespace:              "mk-2",
			Status:                 constants2.DinosaurRequestStatusProvisioning.String(),
			BootstrapServerHost:    bootstrapServerHost,
			SsoClientID:            ssoClientID,
			SsoClientSecret:        ssoSecret,
			DesiredDinosaurVersion: "2.6.0",
			DesiredStrimziVersion:  "strimzi-cluster-operator.v0.23.0-0",
		},
		{
			ClusterID:              testServer.ClusterID,
			MultiAZ:                false,
			Name:                   mockDinosaurName3,
			Namespace:              "mk-3",
			Status:                 constants2.DinosaurRequestStatusPreparing.String(),
			BootstrapServerHost:    bootstrapServerHost,
			SsoClientID:            ssoClientID,
			SsoClientSecret:        ssoSecret,
			DesiredDinosaurVersion: "2.7.1",
			DesiredStrimziVersion:  "strimzi-cluster-operator.v0.23.0-0",
		},
		{
			ClusterID:              testServer.ClusterID,
			MultiAZ:                false,
			Name:                   mockDinosaurName4,
			Namespace:              "mk-4",
			Status:                 constants2.DinosaurRequestStatusReady.String(),
			BootstrapServerHost:    bootstrapServerHost,
			SsoClientID:            ssoClientID,
			SsoClientSecret:        ssoSecret,
			DesiredDinosaurVersion: "2.7.2",
			DesiredStrimziVersion:  "strimzi-cluster-operator.v0.23.0-0",
		},
		{
			ClusterID:              testServer.ClusterID,
			MultiAZ:                false,
			Namespace:              "mk-5",
			Name:                   mockDinosaurName4,
			Status:                 constants2.DinosaurRequestStatusFailed.String(),
			BootstrapServerHost:    bootstrapServerHost,
			SsoClientID:            ssoClientID,
			SsoClientSecret:        ssoSecret,
			DesiredDinosaurVersion: "2.7.2",
			DesiredStrimziVersion:  "strimzi-cluster-operator.v0.23.0-0",
		},
	}

	db := test.TestServices.DBFactory.New()

	// create dummy dinosaurs
	if err := db.Create(&testDinosaurs).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	// create an additional dinosaur in failed state without "ssoSecret", "ssoClientID" and bootstrapServerHost. This indicates that the
	// dinosaur failed in preparing state and should not be returned in the list
	additionalDinosaur := &dbapi.DinosaurRequest{
		ClusterID:              testServer.ClusterID,
		MultiAZ:                false,
		Name:                   mockDinosaurName4,
		Namespace:              "mk",
		Status:                 constants2.DinosaurRequestStatusFailed.String(),
		DesiredDinosaurVersion: "2.7.2",
	}

	if err := db.Save(additionalDinosaur).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetPineapples(testServer.Ctx, testServer.ClusterID)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(list.Items)).To(Equal(4)) // only count valid Managed Dinosaur CR

	find := func(slice []private.ManagedPineapple, match func(dinosaur private.ManagedPineapple) bool) *private.ManagedPineapple {
		for _, item := range slice {
			if match(item) {
				return &item
			}
		}
		return nil
	}

	for _, k := range testDinosaurs {
		if k.Status != constants2.DinosaurRequestStatusPreparing.String() {
			if mk := find(list.Items, func(item private.ManagedPineapple) bool { return item.Metadata.Annotations.MasId == k.ID }); mk != nil {
				Expect(mk.Metadata.Name).To(Equal(k.Name))
				Expect(mk.Metadata.Annotations.MasPlacementId).To(Equal(k.PlacementId))
				Expect(mk.Metadata.Annotations.MasId).To(Equal(k.ID))
				Expect(mk.Metadata.Namespace).NotTo(BeEmpty())
				Expect(mk.Spec.Deleted).To(Equal(k.Status == constants2.DinosaurRequestStatusDeprovision.String()))
				Expect(mk.Spec.Versions.Pineapple).To(Equal(k.DesiredDinosaurVersion))
				Expect(mk.Spec.Endpoint.Tls).To(BeNil())
			} else {
				t.Error("failed matching manageddinosaur id with dinosaurrequest id")
				break
			}
		}
	}

	var readyClusters, deletedClusters []string
	updates := map[string]private.DataPlanePineappleStatus{}
	for _, item := range list.Items {
		if !item.Spec.Deleted {
			updates[item.Metadata.Annotations.MasId] = private.DataPlanePineappleStatus{
				Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{{
					Type:   "Ready",
					Status: "True",
					Reason: "StrimziUpdating",
				}},
				Versions: private.DataPlanePineappleStatusVersions{
					Pineapple:         fmt.Sprintf("dinosaur-new-version-%s", item.Metadata.Annotations.MasId),
					PineappleOperator: fmt.Sprintf("strimzi-new-version-%s", item.Metadata.Annotations.MasId),
				},
			}
			readyClusters = append(readyClusters, item.Metadata.Annotations.MasId)
		} else {
			updates[item.Metadata.Annotations.MasId] = private.DataPlanePineappleStatus{
				Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{{
					Type:   "Ready",
					Status: "False",
					Reason: "Deleted",
				}},
			}
			deletedClusters = append(deletedClusters, item.Metadata.Annotations.MasId)
		}
	}

	// routes will be stored the first time status are updated
	_, err = testServer.PrivateClient.AgentClustersApi.UpdatePineappleClusterStatus(testServer.Ctx, testServer.ClusterID, updates)
	Expect(err).NotTo(HaveOccurred())

	// wait for the CNAMEs for routes to be created
	waitErr := common.NewPollerBuilder(test.TestServices.DBFactory).
		IntervalAndTimeout(1*time.Second, 1*time.Minute).
		RetryLogMessage("waiting for Dinosaur routes to be created").
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			c := &dbapi.DinosaurRequest{}
			if err := db.First(c, "routes IS NOT NULL").Error; err != nil {
				return false, err
			}
			// if one route is created, it is safe to assume all routes are created
			return c.RoutesCreated, nil
		}).Build().Poll()
	Expect(waitErr).To(BeNil())

	// Send the requests again, this time the instances should be ready because routes are created
	_, err = testServer.PrivateClient.AgentClustersApi.UpdatePineappleClusterStatus(testServer.Ctx, testServer.ClusterID, updates)
	Expect(err).NotTo(HaveOccurred())

	for _, cid := range readyClusters {
		c := &dbapi.DinosaurRequest{}
		if err := db.First(c, "id = ?", cid).Error; err != nil {
			t.Errorf("failed to find dinosaur cluster with id %s due to error: %v", cid, err)
		}

		sentUpdate, ok := updates[cid]
		if !ok {
			t.Errorf("failed to find sent dinosaur status update related to cluster with id %s", cid)
		}

		// Test version related reported fields
		Expect(c.Status).To(Equal(constants2.DinosaurRequestStatusReady.String()))
		Expect(c.ActualDinosaurVersion).To(Equal(sentUpdate.Versions.Pineapple))
		Expect(c.ActualStrimziVersion).To(Equal(sentUpdate.Versions.PineappleOperator))
		Expect(c.StrimziUpgrading).To(BeTrue()) // should always be true since Condition.Reason is set to StrimziUpgrading

		// TODO test when dinosaur is being upgraded when kas fleet shard operator side
		// appropriately reports it
	}

	for _, cid := range deletedClusters {
		c := &dbapi.DinosaurRequest{}
		// need to use Unscoped here as there is a chance the entry is soft deleted already
		if err := db.Unscoped().Where("id = ?", cid).First(c).Error; err != nil {
			t.Errorf("failed to find dinosaur cluster with id %s due to error: %v", cid, err)
		}
		Expect(c.Status).To(Equal(constants2.DinosaurRequestStatusDeleting.String()))
	}

	for _, cid := range readyClusters {
		// update the status to ready again and remove reason field to simulate the end of upgrade process as reported by kas-fleet-shard
		_, err = testServer.PrivateClient.AgentClustersApi.UpdatePineappleClusterStatus(testServer.Ctx, testServer.ClusterID, map[string]private.DataPlanePineappleStatus{
			cid: {
				Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{{
					Type:   "Ready",
					Status: "True",
				}},
				Versions: private.DataPlanePineappleStatusVersions{
					Pineapple:         fmt.Sprintf("dinosaur-new-version-%s", cid),
					PineappleOperator: fmt.Sprintf("strimzi-new-version-%s", cid),
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())

		c := &dbapi.DinosaurRequest{}
		if err := db.First(c, "id = ?", cid).Error; err != nil {
			t.Errorf("failed to find dinosaur cluster with id %s due to error: %v", cid, err)
		}

		// Make sure that the dinosaur stays in ready state and status of strimzi upgrade is false.
		Expect(c.Status).To(Equal(constants2.DinosaurRequestStatusReady.String()))
		Expect(c.StrimziUpgrading).To(BeFalse())
	}
}

func TestDataPlaneEndpoints_GetAndUpdateManagedDinosaursWithTlsCerts(t *testing.T) {
	cert := "some-fake-cert"
	key := "some-fake-key"
	startHook := func(c *config.DinosaurConfig) {
		c.EnableDinosaurExternalCertificate = true
		c.DinosaurTLSCert = cert
		c.DinosaurTLSKey = key
	}
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.DinosaurRealm.ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"kas-fleetshard-operator-cluster-id": cid,
		}
	}, startHook)
	defer testServer.TearDown()
	bootstrapServerHost := "some-bootstrap⁻host"
	ssoClientID := "some-sso-client-id"
	ssoSecret := "some-sso-secret"
	canaryServiceAccountClientId := "canary-servie-account-client-id"
	canaryServiceAccountClientSecret := "canary-service-account-client-secret"

	testDinosaur := &dbapi.DinosaurRequest{
		ClusterID:                        testServer.ClusterID,
		MultiAZ:                          false,
		Name:                             mockDinosaurName1,
		Status:                           constants2.DinosaurRequestStatusReady.String(),
		BootstrapServerHost:              bootstrapServerHost,
		SsoClientID:                      ssoClientID,
		SsoClientSecret:                  ssoSecret,
		CanaryServiceAccountClientID:     canaryServiceAccountClientId,
		CanaryServiceAccountClientSecret: canaryServiceAccountClientSecret,
		PlacementId:                      "some-placement-id",
		DesiredDinosaurVersion:           "2.7.0",
	}

	db := test.TestServices.DBFactory.New()

	// create dummy dinosaur
	if err := db.Save(testDinosaur).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetPineapples(testServer.Ctx, testServer.ClusterID)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(list.Items)).To(Equal(1)) // we should have one managed dinosaur cr

	find := func(slice []private.ManagedPineapple, match func(dinosaur private.ManagedPineapple) bool) *private.ManagedPineapple {
		for _, item := range slice {
			if match(item) {
				return &item
			}
		}
		return nil
	}
	if mk := find(list.Items, func(item private.ManagedPineapple) bool { return item.Metadata.Annotations.MasId == testDinosaur.ID }); mk != nil {
		Expect(mk.Spec.Endpoint.Tls.Cert).To(Equal(cert))
		Expect(mk.Spec.Endpoint.Tls.Key).To(Equal(key))
	} else {
		t.Error("failed matching manageddinosaur id with dinosaurrequest id")
	}
}

func TestDataPlaneEndpoints_GetAndUpdateManagedDinosaursWithServiceAccounts(t *testing.T) {
	startHook := func(keycloakConfig *keycloak.KeycloakConfig) {
		keycloakConfig.EnableAuthenticationOnDinosaur = true
	}
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.DinosaurRealm.ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"kas-fleetshard-operator-cluster-id": cid,
		}
	}, startHook)
	defer testServer.TearDown()
	bootstrapServerHost := "some-bootstrap⁻host"
	ssoClientID := "some-sso-client-id"
	ssoSecret := "some-sso-secret"
	canaryServiceAccountClientId := "canary-servie-account-client-id"
	canaryServiceAccountClientSecret := "canary-service-account-client-secret"

	testDinosaur := &dbapi.DinosaurRequest{
		ClusterID:                        testServer.ClusterID,
		MultiAZ:                          false,
		Name:                             mockDinosaurName1,
		Status:                           constants2.DinosaurRequestStatusReady.String(),
		BootstrapServerHost:              bootstrapServerHost,
		SsoClientID:                      ssoClientID,
		SsoClientSecret:                  ssoSecret,
		CanaryServiceAccountClientID:     canaryServiceAccountClientId,
		CanaryServiceAccountClientSecret: canaryServiceAccountClientSecret,
		PlacementId:                      "some-placement-id",
		DesiredDinosaurVersion:           "2.7.0",
	}

	db := test.TestServices.DBFactory.New()

	// create dummy dinosaur
	if err := db.Save(testDinosaur).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetPineapples(testServer.Ctx, testServer.ClusterID)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(list.Items)).To(Equal(1)) // we should have one managed dinosaur cr

	find := func(slice []private.ManagedPineapple, match func(dinosaur private.ManagedPineapple) bool) *private.ManagedPineapple {
		for _, item := range slice {
			if match(item) {
				return &item
			}
		}
		return nil
	}
	if mk := find(list.Items, func(item private.ManagedPineapple) bool { return item.Metadata.Annotations.MasId == testDinosaur.ID }); mk != nil {
		// check canary service account
		Expect(mk.Spec.ServiceAccounts).To(HaveLen(1))
		canaryServiceAccount := mk.Spec.ServiceAccounts[0]
		Expect(canaryServiceAccount.Name).To(Equal("canary"))
		Expect(canaryServiceAccount.Principal).To(Equal(canaryServiceAccountClientId))
		Expect(canaryServiceAccount.Password).To(Equal(canaryServiceAccountClientSecret))
	} else {
		t.Error("failed matching manageddinosaur id with dinosaurrequest id")
	}
}
func TestDataPlaneEndpoints_GetManagedDinosaursWithoutOAuthTLSCert(t *testing.T) {
	startHook := func(c *keycloak.KeycloakConfig) {
		c.TLSTrustedCertificatesValue = ""
	}
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.DinosaurRealm.ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"kas-fleetshard-operator-cluster-id": cid,
		}
	}, startHook)
	defer testServer.TearDown()
	bootstrapServerHost := "some-bootstrap⁻host"
	ssoClientID := "some-sso-client-id"
	ssoSecret := "some-sso-secret"

	testDinosaur := &dbapi.DinosaurRequest{
		ClusterID:              testServer.ClusterID,
		MultiAZ:                false,
		Name:                   mockDinosaurName1,
		Status:                 constants2.DinosaurRequestStatusReady.String(),
		BootstrapServerHost:    bootstrapServerHost,
		SsoClientID:            ssoClientID,
		SsoClientSecret:        ssoSecret,
		PlacementId:            "some-placement-id",
		DesiredDinosaurVersion: "2.7.0",
	}

	KeycloakConfig(testServer.Helper).EnableAuthenticationOnDinosaur = true

	db := test.TestServices.DBFactory.New()

	// create dummy dinosaur
	if err := db.Save(testDinosaur).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetPineapples(testServer.Ctx, testServer.ClusterID)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(list.Items)).To(Equal(1)) // we should have one managed dinosaur cr

	find := func(slice []private.ManagedPineapple, match func(dinosaur private.ManagedPineapple) bool) *private.ManagedPineapple {
		for _, item := range slice {
			if match(item) {
				return &item
			}
		}
		return nil
	}
	if mk := find(list.Items, func(item private.ManagedPineapple) bool { return item.Metadata.Annotations.MasId == testDinosaur.ID }); mk != nil {
		Expect(mk.Spec.Oauth.TlsTrustedCertificate).To(BeNil())
	} else {
		t.Error("failed matching manageddinosaur id with dinosaurrequest id")
	}
}

func TestDataPlaneEndpoints_UpdateManagedDinosaursWithRoutes(t *testing.T) {
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.DinosaurRealm.ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"kas-fleetshard-operator-cluster-id": cid,
		}
	}, nil)
	defer testServer.TearDown()
	db := test.TestServices.DBFactory.New()
	var cluster api.Cluster
	if err := db.Where("cluster_id = ?", testServer.ClusterID).First(&cluster).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	clusterDNS := strings.Replace(cluster.ClusterDNS, constants2.DefaultIngressDnsNamePrefix, constants2.ManagedDinosaurIngressDnsNamePrefix, 1)

	bootstrapServerHost := "prefix.some-bootstrap⁻host"
	ssoClientID := "some-sso-client-id"
	ssoSecret := "some-sso-secret"

	var testDinosaurs = []*dbapi.DinosaurRequest{
		{
			ClusterID:              testServer.ClusterID,
			MultiAZ:                false,
			Name:                   mockDinosaurName2,
			Status:                 constants2.DinosaurRequestStatusProvisioning.String(),
			BootstrapServerHost:    bootstrapServerHost,
			SsoClientID:            ssoClientID,
			SsoClientSecret:        ssoSecret,
			DesiredDinosaurVersion: "2.6.0",
		},
	}

	// create dummy dinosaurs
	if err := db.Create(&testDinosaurs).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetPineapples(testServer.Ctx, testServer.ClusterID)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(list.Items)).To(Equal(1)) // only count valid Managed Dinosaur CR

	var readyClusters []string
	updates := map[string]private.DataPlanePineappleStatus{}
	for _, item := range list.Items {
		updates[item.Metadata.Annotations.MasId] = private.DataPlanePineappleStatus{
			Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{{
				Type:   "Ready",
				Status: "True",
			}},
			Routes: &[]private.DataPlanePineappleStatusRoutes{
				{
					Name:   "admin-api",
					Prefix: "admin-api",
					Router: fmt.Sprintf("router.%s", clusterDNS),
				},
				{
					Name:   "bootstrap",
					Prefix: "",
					Router: fmt.Sprintf("router.%s", clusterDNS),
				},
			},
		}
		readyClusters = append(readyClusters, item.Metadata.Annotations.MasId)
	}

	// routes will be stored the first time status are updated
	_, err = testServer.PrivateClient.AgentClustersApi.UpdatePineappleClusterStatus(testServer.Ctx, testServer.ClusterID, updates)
	Expect(err).NotTo(HaveOccurred())

	// wait for the CNAMEs for routes to be created
	waitErr := common.NewPollerBuilder(test.TestServices.DBFactory).
		IntervalAndTimeout(1*time.Second, 1*time.Minute).
		RetryLogMessage("waiting for Dinosaur routes to be created").
		OnRetry(func(attempt int, maxRetries int) (done bool, err error) {
			c := &dbapi.DinosaurRequest{}
			if err := db.First(c, "routes IS NOT NULL").Error; err != nil {
				return false, err
			}
			routes, err := c.GetRoutes()
			if err != nil {
				return false, err
			}
			if len(routes) != 2 {
				return false, errors.Errorf("expected length of routes array to be 1")
			}
			wantDomain1 := "admin-api-prefix.some-bootstrap⁻host"
			if routes[0].Domain != wantDomain1 {
				return false, errors.Errorf("route domain value is %s but want %s", routes[0].Domain, wantDomain1)
			}
			wantDomain2 := "prefix.some-bootstrap⁻host"
			if routes[1].Domain != wantDomain2 {
				return false, errors.Errorf("route domain value is %s but want %s", routes[1].Domain, wantDomain2)
			}

			// if one route is created, it is safe to assume all routes are created
			return c.RoutesCreated, nil
		}).Build().Poll()

	Expect(waitErr).NotTo(HaveOccurred())

	// Send the requests again, this time the instances should be ready because routes are created
	_, err = testServer.PrivateClient.AgentClustersApi.UpdatePineappleClusterStatus(testServer.Ctx, testServer.ClusterID, updates)
	Expect(err).NotTo(HaveOccurred())

	for _, cid := range readyClusters {
		c := &dbapi.DinosaurRequest{}
		if err := db.First(c, "id = ?", cid).Error; err != nil {
			t.Errorf("failed to find dinosaur cluster with id %s due to error: %v", cid, err)
		}
		Expect(c.Status).To(Equal(constants2.DinosaurRequestStatusReady.String()))
	}
}

func TestDataPlaneEndpoints_GetManagedDinosaursWithOAuthTLSCert(t *testing.T) {
	cert := "some-fake-cert"
	startHook := func(c *keycloak.KeycloakConfig) {
		c.TLSTrustedCertificatesValue = cert
		c.EnableAuthenticationOnDinosaur = true
	}
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.DinosaurRealm.ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"kas-fleetshard-operator-cluster-id": cid,
		}
	}, startHook)
	defer testServer.TearDown()
	bootstrapServerHost := "some-bootstrap⁻host"
	ssoClientID := "some-sso-client-id"
	ssoSecret := "some-sso-secret"

	testDinosaur := &dbapi.DinosaurRequest{
		ClusterID:              testServer.ClusterID,
		MultiAZ:                false,
		Name:                   mockDinosaurName1,
		Status:                 constants2.DinosaurRequestStatusReady.String(),
		BootstrapServerHost:    bootstrapServerHost,
		SsoClientID:            ssoClientID,
		SsoClientSecret:        ssoSecret,
		PlacementId:            "some-placement-id",
		DesiredDinosaurVersion: "2.7.0",
	}

	KeycloakConfig(testServer.Helper).EnableAuthenticationOnDinosaur = true

	db := test.TestServices.DBFactory.New()

	// create dummy dinosaur
	if err := db.Save(testDinosaur).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetPineapples(testServer.Ctx, testServer.ClusterID)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(list.Items)).To(Equal(1)) // we should have one managed dinosaur cr

	find := func(slice []private.ManagedPineapple, match func(dinosaur private.ManagedPineapple) bool) *private.ManagedPineapple {
		for _, item := range slice {
			if match(item) {
				return &item
			}
		}
		return nil
	}
	if mk := find(list.Items, func(item private.ManagedPineapple) bool { return item.Metadata.Annotations.MasId == testDinosaur.ID }); mk != nil {
		Expect(mk.Spec.Oauth.TlsTrustedCertificate).ToNot(BeNil())
	} else {
		t.Error("failed matching manageddinosaur id with dinosaurrequest id")
	}

}

func KeycloakConfig(helper *coreTest.Helper) (c *keycloak.KeycloakConfig) {
	helper.Env.MustResolveAll(&c)
	return
}

func TestDataPlaneEndpoints_UpdateManagedDinosaurWithErrorStatus(t *testing.T) {
	testServer := setup(t, func(account *v1.Account, cid string, h *coreTest.Helper) jwt.MapClaims {
		username, _ := account.GetUsername()
		return jwt.MapClaims{
			"username": username,
			"iss":      test.TestServices.KeycloakConfig.DinosaurRealm.ValidIssuerURI,
			"realm_access": map[string][]string{
				"roles": {"kas_fleetshard_operator"},
			},
			"kas-fleetshard-operator-cluster-id": cid,
		}
	}, nil)
	defer testServer.TearDown()
	bootstrapServerHost := "some-bootstrap⁻host"
	ssoClientID := "some-sso-client-id"
	ssoSecret := "some-sso-secret"

	db := test.TestServices.DBFactory.New()

	testDinosaur := dbapi.DinosaurRequest{
		ClusterID:              testServer.ClusterID,
		MultiAZ:                false,
		Name:                   mockDinosaurName1,
		Status:                 constants2.DinosaurRequestStatusReady.String(),
		BootstrapServerHost:    bootstrapServerHost,
		SsoClientID:            ssoClientID,
		SsoClientSecret:        ssoSecret,
		DesiredDinosaurVersion: "2.7.0",
	}

	// create dummy dinosaurs
	if err := db.Create(&testDinosaur).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
		return
	}

	list, resp, err := testServer.PrivateClient.AgentClustersApi.GetPineapples(testServer.Ctx, testServer.ClusterID)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(list.Items)).To(Equal(1)) // we should have one managed dinosaur cr
	dinosaurReqID := list.Items[0].Metadata.Annotations.MasId

	errMessage := "test-err-message"
	updateReq := map[string]private.DataPlanePineappleStatus{
		dinosaurReqID: kasfleetshardsync.GetErrorWithCustomMessageDinosaurStatusResponse(errMessage),
	}
	_, err = testServer.PrivateClient.AgentClustersApi.UpdatePineappleClusterStatus(testServer.Ctx, testServer.ClusterID, updateReq)
	Expect(err).NotTo(HaveOccurred())

	c := &dbapi.DinosaurRequest{}
	if err := db.First(c, "id = ?", dinosaurReqID).Error; err != nil {
		t.Errorf("failed to find dinosaur cluster with id %s due to error: %v", dinosaurReqID, err)
	}
	Expect(c.Status).To(Equal(constants2.DinosaurRequestStatusFailed.String()))
	Expect(c.FailedReason).To(ContainSubstring(errMessage))
}
