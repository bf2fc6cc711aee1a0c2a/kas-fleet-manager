package integration

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test"

	api "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test/mocks"
	. "github.com/onsi/gomega"
)

func TestClusterCreate_InvalidAwsCredentials(t *testing.T) {
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServerBuilder.SetClustersPostResponse(nil, errors.Validation("The provided AWS credentials are not valid"))
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	_, _, teardown := test.NewDinosaurHelperWithHooks(t, ocmServer, func(aws *config.AWSConfig) {
		// setting AWS.AccountID to invalid value
		aws.AccountID = "123456789012"
	})
	defer teardown()

	cluster, err := test.TestServices.ClusterService.Create(&api.Cluster{
		CloudProvider:      "aws",
		Region:             "us-east-1",
		MultiAZ:            true,
		IdentityProviderID: "some-identity-provider-id",
	})
	Expect(err).To(HaveOccurred())
	Expect(cluster).To(BeNil())
}
