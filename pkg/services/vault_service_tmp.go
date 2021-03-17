package services

import (
	"fmt"
	"sync"
)

var _ VaultService = &tmpVaultService{}
var NotFound = fmt.Errorf("Not Found")

type tmpSecret struct {
	name           string
	value          string
	owningResource string
}

type tmpVaultService struct {
	secrets map[string]tmpSecret
	mu      sync.Mutex
}

func NewTmpVaultService() (*tmpVaultService, error) {
	return &tmpVaultService{
		secrets: map[string]tmpSecret{},
	}, nil
}

func (k *tmpVaultService) Kind() string {
	return "tmp"
}

func (k *tmpVaultService) SetSecretString(name string, value string, owningResource string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.secrets[name] = tmpSecret{
		name:           name,
		value:          value,
		owningResource: owningResource,
	}
	return nil
}

func (k *tmpVaultService) GetSecretString(name string) (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if entry, ok := k.secrets[name]; ok {
		return entry.value, nil
	}
	return "", NotFound
}

func (k *tmpVaultService) DeleteSecretString(name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	delete(k.secrets, name)
	return nil
}

func (k *tmpVaultService) ForEachSecret(f func(name string, owningResource string) bool) error {

	// Copy the secrets to an array...
	k.mu.Lock()
	secrets := []tmpSecret{}
	for _, s := range k.secrets {
		secrets = append(secrets, s)
	}
	k.mu.Unlock()

	l := len(secrets)
	for i := 0; i < l; i++ {
		if !f(secrets[i].name, secrets[i].owningResource) {
			return nil
		}
	}
	return nil

}
