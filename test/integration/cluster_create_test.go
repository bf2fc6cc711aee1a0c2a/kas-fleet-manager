package integration

import (
	"testing"

	. "github.com/onsi/gomega"
	api "gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	ocm "gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
)

func TestClusterCreate_InvalidAwsCredentials(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServerBuilder.SetClustersPostResponse(nil, errors.Validation("The provided AWS credentials are not valid"))
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	// setting AWS.AccountID to invalid value
	currentAWSAccountID := h.Env().Config.AWS.AccountID
	defer func(helper *test.Helper) {
		helper.Env().Config.AWS.AccountID = currentAWSAccountID
	}(h)
	h.Env().Config.AWS.AccountID = "123456789012"

	clusterService := services.NewClusterService(h.Env().DBFactory, ocm.NewClient(h.Env().Clients.OCM.Connection), h.Env().Config.AWS)

	cluster, err := clusterService.Create(&api.Cluster{
		CloudProvider: "aws",
		Region:        "eu-west-1",
	})
	Expect(err).To(HaveOccurred())
	Expect(cluster.ID()).To(Equal(""))
}

func TestClusterCreate_InvalidToken(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServerBuilder.SetClustersPostResponse(nil, errors.GeneralError("can't get access token: invalid_grant: Invalid refresh token"))
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	clusterService := services.NewClusterService(h.Env().DBFactory, ocm.NewClient(h.Env().Clients.OCM.Connection), h.Env().Config.AWS)
	// temporarily setting token to invalid value
	h.Env().Config.OCM.SelfTokenFile = "secrets/ocm-service.clientId"
	cluster, err := clusterService.Create(&api.Cluster{
		CloudProvider: "aws",
		Region:        "eu-west-1",
	})
	Expect(err).To(HaveOccurred())
	Expect(cluster.ID()).To(Equal(""))
}
