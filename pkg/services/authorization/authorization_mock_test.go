package authorization

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_NewMockAuthorization(t *testing.T) {
	g := gomega.NewWithT(t)

	auth := NewMockAuthorization()
	_, isExpectedType := auth.(*mock)
	g.Expect(isExpectedType).To(gomega.BeTrue())
}

func Test_MockAuthorization_CheckUserValid(t *testing.T) {
	g := gomega.NewWithT(t)
	auth := NewMockAuthorization()
	res, err := auth.CheckUserValid("testuser", "testorg")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(res).To(gomega.BeTrue())
}
