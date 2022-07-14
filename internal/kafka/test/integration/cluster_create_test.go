package integration

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"

	api "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/onsi/gomega"
)

func TestClusterCreate_InvalidAwsCredentials(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServerBuilder.SetClustersPostResponse(nil, errors.Validation("The provided AWS credentials are not valid"))
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	_, _, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(aws *config.AWSConfig) {
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
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(cluster).To(gomega.BeNil())
}
