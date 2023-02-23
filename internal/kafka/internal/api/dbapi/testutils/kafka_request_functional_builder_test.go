package testutils_test

import (
	. "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi/testutils"
	"github.com/onsi/gomega"
	"testing"
)

func TestKafkaRequest_FunctionalBuilder(t *testing.T) {
	g := gomega.NewWithT(t)

	k := NewKafkaRequest(WithName("test123"))
	g.Expect(k.Name).To(gomega.Equal("test123"))

	k = NewKafkaRequest(WithActualBillingModel("testbillingmodel"))
	g.Expect(k.ActualKafkaBillingModel).To(gomega.Equal("testbillingmodel"))

	k = NewKafkaRequest(WithDesiredBillingModel("testbillingmodel"))
	g.Expect(k.DesiredKafkaBillingModel).To(gomega.Equal("testbillingmodel"))

	k = NewKafkaRequest(WithID("test-123"))
	g.Expect(k.ID).To(gomega.Equal("test-123"))
}
