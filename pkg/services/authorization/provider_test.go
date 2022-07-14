package authorization

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"

	"github.com/onsi/gomega"
)

func Test_NewAuthorization(t *testing.T) {
	g := gomega.NewWithT(t)

	ocmConfig := ocm.NewOCMConfig()

	ocmConfig.EnableMock = true
	auth := NewAuthorization(ocmConfig)
	var isExpectedType bool
	_, isExpectedType = auth.(*mock)
	g.Expect(isExpectedType).To(gomega.BeTrue())

	ocmConfig.EnableMock = false
	ocmConfig.ClientID = "dummyclientid"
	ocmConfig.ClientSecret = "dummyclientsecret"
	auth = NewAuthorization(ocmConfig)
	_, isExpectedType = auth.(*authorization)
	g.Expect(isExpectedType).To(gomega.BeTrue())
}
