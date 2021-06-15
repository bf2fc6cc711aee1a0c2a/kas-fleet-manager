package integration

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"testing"

	api "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	. "github.com/onsi/gomega"
)

func TestClusterCreate_InvalidAwsCredentials(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServerBuilder.SetClustersPostResponse(nil, errors.Validation("The provided AWS credentials are not valid"))
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	h, _, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	// setting AWS.AccountID to invalid value
	currentAWSAccountID := h.Env.Config.AWS.AccountID
	defer func(helper *test.Helper) {
		helper.Env.Config.AWS.AccountID = currentAWSAccountID
	}(h)
	h.Env.Config.AWS.AccountID = "123456789012"

	var clusterService services.ClusterService
	h.Env.MustResolveAll(&clusterService)

	cluster, err := clusterService.Create(&api.Cluster{
		CloudProvider: "aws",
		Region:        "us-east-1",
		MultiAZ:       true,
	})
	Expect(err).To(HaveOccurred())
	Expect(cluster).To(BeNil())
}
