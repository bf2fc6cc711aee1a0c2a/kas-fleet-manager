package handlers

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/onsi/gomega"
)

func Test_DefaultKafkaPromoteValidatorFactory_GetValidator(t *testing.T) {
	var kafkaConfig *config.KafkaConfig = &config.KafkaConfig{}
	validatorFactory := NewDefaultKafkaPromoteValidatorFactory(kafkaConfig)

	g := gomega.NewWithT(t)

	quotaType := api.AMSQuotaType
	res, err := validatorFactory.GetValidator(quotaType)
	_, ok := res.(*amsKafkaPromoteValidator)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(ok).To(gomega.BeTrue())
	_, ok = res.(*quotaManagementListKafkaPromoteValidator)
	g.Expect(ok).To(gomega.BeFalse())

	quotaType = api.QuotaManagementListQuotaType
	res, err = validatorFactory.GetValidator(quotaType)
	_, ok = res.(*quotaManagementListKafkaPromoteValidator)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(ok).To(gomega.BeTrue())
	_, ok = res.(*amsKafkaPromoteValidator)
	g.Expect(ok).To(gomega.BeFalse())

	quotaType = api.QuotaType("unexistingquotatype")
	_, err = validatorFactory.GetValidator(quotaType)
	g.Expect(err).To(gomega.HaveOccurred())
}
