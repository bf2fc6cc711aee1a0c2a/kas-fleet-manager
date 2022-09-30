package integration

import (
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/onsi/gomega"
)

func TestKafkaCreate_Metadata(t *testing.T) {
	g := gomega.NewWithT(t)

	// setup ocm server
	ocmServerBuilder := mocks.NewMockConfigurableServerBuilder()
	ocmServer := ocmServerBuilder.Build()
	defer ocmServer.Close()

	// start servers
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	version, resp, err := client.DefaultApi.GetVersionMetadata(ctx)
	if resp != nil {
		resp.Body.Close()
	}

	g.Expect(err).NotTo(gomega.HaveOccurred(), "g.Unable to retrieve version metadata:  %v", err)
	g.Expect(resp.StatusCode).To(gomega.Equal(http.StatusOK))
	g.Expect(version.Href).To(gomega.ContainSubstring("/v1"))
	g.Expect(version.Collections).To(gomega.HaveLen(3))

	/*  Unable to test for a specific version as for now due to the debug package issue: https://github.com/golang/go/issues/33976,
	TODO when the issue is resolved.
	*/
}
