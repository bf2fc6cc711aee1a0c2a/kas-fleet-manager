package vault

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func Test_awsVaultService(t *testing.T) {
	RegisterTestingT(t)
	vc := NewConfig()

	// Enable testing against aws if the access keys are configured..
	if content, err := ioutil.ReadFile(shared.BuildFullFilePath(vc.AccessKeyFile)); err == nil && len(content) > 0 {
		vc.Kind = "aws"
	}
	Expect(vc.ReadFiles()).To(BeNil())

	// only run the tests below when aws vault service is configured
	if vc.Kind == "aws" {
		svc, err := NewVaultService(vc)
		Expect(err).To(BeNil())

		name := fmt.Sprintf("testkey%d", rand.Uint64())
		value := "testvalue"
		err = svc.SetSecretString(name, value, "/v1/connector/test")
		Expect(err).To(BeNil())

		// validate that aws secret has config prefix
		result, err := svc.(*awsVaultService).secretCache.GetSecretString(vc.SecretPrefix + "/" + name)
		Expect(err).To(BeNil())
		Expect(result).To(Equal(value))

		result, err = svc.GetSecretString(name)
		Expect(err).To(BeNil())
		Expect(result).To(Equal(value))

		err = svc.DeleteSecretString(name)
		Expect(err).To(BeNil())
	}
}
