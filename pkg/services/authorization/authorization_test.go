package authorization

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test_NewOCMAuthorization(t *testing.T) {
	RegisterTestingT(t)

	auth := NewOCMAuthorization(nil)
	_, isExpectedType := auth.(*authorization)
	Expect(isExpectedType).To(BeTrue())
}
