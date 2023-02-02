package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/onsi/gomega"
	"testing"
)

func Test_StripProcessorSecretReferences(t *testing.T) {
	g := gomega.NewWithT(t)
	resource := dbapi.Processor{
		ServiceAccount: dbapi.ServiceAccount{
			ClientId:        "client-id",
			ClientSecret:    "client-secret",
			ClientSecretRef: "client-secret-ref",
		},
	}

	_ = stripProcessorSecretReferences(&resource)

	g.Expect(resource.ServiceAccount.ClientId).To(gomega.Equal("client-id"))
	g.Expect(resource.ServiceAccount.ClientSecret).To(gomega.Equal(""))
	g.Expect(resource.ServiceAccount.ClientSecretRef).To(gomega.Equal(""))
}

func Test_MoveProcessorSecretsToVault(t *testing.T) {
	g := gomega.NewWithT(t)
	vaultService, _ := vault.NewTmpVaultService()
	resource := dbapi.Processor{
		ServiceAccount: dbapi.ServiceAccount{
			ClientId:     "client-id",
			ClientSecret: "client-secret",
		},
	}

	_ = moveProcessorSecretsToVault(&resource, vaultService)

	vaultKey := resource.ServiceAccount.ClientSecretRef
	g.Expect(vaultKey).NotTo(gomega.BeEmpty())
	g.Expect(vaultService.GetSecretString(vaultKey)).To(gomega.Equal("client-secret"))
	g.Expect(resource.ServiceAccount.ClientSecret).To(gomega.Equal(""))
}

func Test_GetProcessorSecretRefs(t *testing.T) {
	g := gomega.NewWithT(t)
	resource := dbapi.Processor{
		ServiceAccount: dbapi.ServiceAccount{
			ClientSecretRef: "",
		},
	}

	results, _ := getProcessorSecretRefs(&resource)
	g.Expect(results).To(gomega.BeEmpty())

	resource = dbapi.Processor{
		ServiceAccount: dbapi.ServiceAccount{
			ClientSecretRef: "client-secret-ref",
		},
	}

	results, _ = getProcessorSecretRefs(&resource)
	g.Expect(len(results)).To(gomega.Equal(1))
	g.Expect(results[0]).To(gomega.Equal("client-secret-ref"))
}
