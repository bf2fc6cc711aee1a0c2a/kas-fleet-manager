package account

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/onsi/gomega"
)

func TestNewAccount(t *testing.T) {
	g := gomega.NewWithT(t)
	ocmConfig := ocm.NewOCMConfig()

	ocmConfig.EnableMock = true
	account := NewAccount(ocmConfig)
	var isExpectedType bool
	_, isExpectedType = account.(*mock)
	g.Expect(isExpectedType).To(gomega.BeTrue())

	ocmConfig.EnableMock = false
	ocmConfig.ClientID = "dummyclientid"
	ocmConfig.ClientSecret = "dummyclientsecret"
	account = NewAccount(ocmConfig)
	_, isExpectedType = account.(*accountService)
	g.Expect(isExpectedType).To(gomega.BeTrue())
}
