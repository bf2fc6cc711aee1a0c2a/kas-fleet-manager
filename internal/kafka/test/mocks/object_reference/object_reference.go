package mocks

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/compat"
)

const (
	TestId             = "test-id"
	KindKafka          = "Kafka"
	KindError          = "Error"
	KindServiceAccount = "ServiceAccount"
	BasePath           = "/api/kafkas_mgmt/v1"
)

func GetObjectReferenceMockId(id string) string {
	if id != "" {
		return id
	}
	return TestId
}

func GetEmptyObjectReference() compat.ObjectReference {
	return compat.ObjectReference{
		Id: TestId,
	}
}

func GetKafkaObjectReference() compat.ObjectReference {
	return compat.ObjectReference{
		Id:   TestId,
		Kind: KindKafka,
		Href: fmt.Sprintf("%s/kafkas/%s", BasePath, TestId),
	}
}

func GetServiceAccountObjectReference() compat.ObjectReference {
	return compat.ObjectReference{
		Id:   TestId,
		Kind: KindServiceAccount,
		Href: fmt.Sprintf("%s/service_accounts/%s", BasePath, TestId),
	}
}

func GetErrorObjectReference() compat.ObjectReference {
	return compat.ObjectReference{
		Id:   TestId,
		Kind: KindError,
		Href: fmt.Sprintf("%s/errors/%s", BasePath, TestId),
	}
}
