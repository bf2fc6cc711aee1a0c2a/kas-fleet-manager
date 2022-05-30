package authorization

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test_NewMockAuthorization(t *testing.T) {
	RegisterTestingT(t)

	auth := NewMockAuthorization()
	_, isExpectedType := auth.(*mock)
	Expect(isExpectedType).To(BeTrue())
}

func Test_MockAuthorization_CheckUserValid(t *testing.T) {
	RegisterTestingT(t)
	auth := NewMockAuthorization()
	res, err := auth.CheckUserValid("testuser", "testorg")
	Expect(err).NotTo(HaveOccurred())
	Expect(res).To(BeTrue())
}
