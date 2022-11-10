package types

import (
	"testing"

	"github.com/onsi/gomega"
)

func Test_ValidKafkaInstanceTypes(t *testing.T) {
	g := gomega.NewWithT(t)
	g.Expect(ValidKafkaInstanceTypes).To(gomega.Equal([]string{DEVELOPER.String(), STANDARD.String()}))
}
