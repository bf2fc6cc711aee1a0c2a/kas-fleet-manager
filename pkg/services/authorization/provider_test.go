package authorization

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"

	. "github.com/onsi/gomega"
)

func Test_NewAuthorization(t *testing.T) {
	RegisterTestingT(t)

	ocmConfig := ocm.NewOCMConfig()

	ocmConfig.EnableMock = true
	auth := NewAuthorization(ocmConfig)
	var isExpectedType bool
	_, isExpectedType = auth.(*mock)
	Expect(isExpectedType).To(BeTrue())

	ocmConfig.EnableMock = false
	ocmConfig.ClientID = "dummyclientid"
	ocmConfig.ClientSecret = "dummyclientsecret"
	auth = NewAuthorization(ocmConfig)
	_, isExpectedType = auth.(*authorization)
	Expect(isExpectedType).To(BeTrue())
}
