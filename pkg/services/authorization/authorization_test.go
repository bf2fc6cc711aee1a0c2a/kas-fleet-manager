package authorization

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_NewOCMAuthorization(t *testing.T) {
	g := gomega.NewWithT(t)

	auth := NewOCMAuthorization(nil)
	_, isExpectedType := auth.(*authorization)
	g.Expect(isExpectedType).To(gomega.BeTrue())
}
