package vault

import (
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"sync"
)

var _ VaultService = &TmpVaultService{}
var NotFound = fmt.Errorf("Not Found")

type tmpSecret struct {
	name           string
	value          string
	owningResource string
}

type TmpVaultService struct {
	mu            sync.Mutex
	secrets       map[string]tmpSecret
	deleteCounter int64
	insertCounter int64
	updateCounter int64
	getCounter    int64
	missCounter   int64
}

type Counters struct {
	Deletes int64
	Inserts int64
	Updates int64
	Gets    int64
	Misses  int64
}

func NewTmpVaultService() (*TmpVaultService, error) {
	return &TmpVaultService{
		secrets: map[string]tmpSecret{},
	}, nil
}

func (k *TmpVaultService) Kind() string {
	return "tmp"
}

func (k *TmpVaultService) ResetCounters() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.deleteCounter = 0
	k.insertCounter = 0
	k.updateCounter = 0
	k.getCounter = 0
	k.missCounter = 0
}

func (k *TmpVaultService) Counters() Counters {
	k.mu.Lock()
	defer k.mu.Unlock()
	return Counters{
		Deletes: k.deleteCounter,
		Inserts: k.insertCounter,
		Updates: k.updateCounter,
		Gets:    k.getCounter,
		Misses:  k.missCounter,
	}
}

func (k *TmpVaultService) SetSecretString(name string, value string, owningResource string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	metrics.IncreaseVaultServiceTotalCount("set")

	if _, found := k.secrets[name]; found {
		k.updateCounter += 1
	} else {
		k.insertCounter += 1
	}
	k.secrets[name] = tmpSecret{
		name:           name,
		value:          value,
		owningResource: owningResource,
	}
	metrics.IncreaseVaultServiceSuccessCount("set")
	return nil
}

func (k *TmpVaultService) GetSecretString(name string) (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	metrics.IncreaseVaultServiceTotalCount("get")

	entry, found := k.secrets[name]
	if found {
		metrics.IncreaseVaultServiceSuccessCount("get")
		k.getCounter += 1
		return entry.value, nil
	} else {
		metrics.IncreaseVaultServiceErrorsCount("get")
		k.missCounter += 1
		return "", NotFound
	}
}

func (k *TmpVaultService) DeleteSecretString(name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	metrics.IncreaseVaultServiceTotalCount("delete")
	if _, ok := k.secrets[name]; ok {
		metrics.IncreaseVaultServiceSuccessCount("delete")
		k.deleteCounter += 1
	} else {
		metrics.IncreaseVaultServiceErrorsCount("delete")
		return NotFound
	}

	delete(k.secrets, name)
	return nil
}

func (k *TmpVaultService) ForEachSecret(f func(name string, owningResource string) bool) error {

	// Copy the secrets to an array...
	k.mu.Lock()
	secrets := []tmpSecret{}
	for _, s := range k.secrets {
		metrics.IncreaseVaultServiceTotalCount("get")
		secrets = append(secrets, s)
	}
	k.mu.Unlock()

	l := len(secrets)
	for i := 0; i < l; i++ {
		metrics.IncreaseVaultServiceSuccessCount("get")
		if !f(secrets[i].name, secrets[i].owningResource) {
			return nil
		}
	}
	return nil

}
