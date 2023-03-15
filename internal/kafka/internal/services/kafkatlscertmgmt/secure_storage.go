package kafkatlscertmgmt

import (
	"context"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-secretsmanager-caching-go/secretcache"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/caddyserver/certmagic"
)

var _ certmagic.Storage = &secureStorage{}

type secureStorage struct {
	secretPrefix     string
	lock             sync.Mutex
	secretCache      *secretcache.Cache
	storageItemLocks map[string]*sync.Mutex
	secretClient     *secretsmanager.SecretsManager
}

func newSecureStorage(config *config.AWSConfig, automaticCertificateManagementConfig config.AutomaticCertificateManagementConfig) (*secureStorage, error) {
	awsConfig := &aws.Config{
		Credentials: credentials.NewStaticCredentials(
			config.SecretManager.AccessKey,
			config.SecretManager.SecretAccessKey,
			""),
		Region:  aws.String(config.SecretManager.Region),
		Retryer: client.DefaultRetryer{NumMaxRetries: 2},
	}
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}

	secretClient := secretsmanager.New(sess)
	secretCache, err := secretcache.New(func(cache *secretcache.Cache) {
		cache.Client = secretClient
		cache.CacheItemTTL = automaticCertificateManagementConfig.CertificateCacheTTL.Nanoseconds()
	})

	if err != nil {
		return nil, err
	}
	return &secureStorage{
		secretClient:     secretClient,
		secretCache:      secretCache,
		lock:             sync.Mutex{},
		storageItemLocks: map[string]*sync.Mutex{},
		secretPrefix:     config.SecretManager.SecretPrefix,
	}, nil
}

// Lock blocks until it can acquire the name-specific lock.
// The process of obtaining a name-specific lock is atomic, as required per the certmagic.Locker
// interface documentation: https://github.com/caddyserver/certmagic/blob/master/storage.go#L87.
// NOTE: The intent here is to not implement a distributed locking mechanism but an in-process locking mechanism good enough for
// our certificate management logic which happens within one reconciliater leader.
func (storage *secureStorage) Lock(ctx context.Context, key string) error {
	storage.lock.Lock()
	mu, ok := storage.storageItemLocks[key]
	if !ok {
		// initialize the storage item lock the first time if it does not exist yet
		mu = &sync.Mutex{}
		storage.storageItemLocks[key] = mu
	}
	storage.lock.Unlock()
	mu.Lock()
	return nil

}

func (storage *secureStorage) Unlock(ctx context.Context, key string) error {
	mu, ok := storage.storageItemLocks[key]
	if !ok {
		return fs.ErrNotExist
	}

	mu.Unlock()
	return nil
}

func (storage *secureStorage) Store(ctx context.Context, key string, value []byte) error {
	name := storage.constructSecretName(key)
	_, err := storage.secretClient.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:         &name,
		SecretBinary: value,
	})

	if err == nil {
		return nil
	}

	switch err.(type) {
	case *secretsmanager.ResourceExistsException:
		_, err = storage.secretClient.UpdateSecret(&secretsmanager.UpdateSecretInput{
			SecretId:     &name,
			SecretBinary: value,
		})

		return err
	default:
		return err
	}

}

func (storage *secureStorage) Load(ctx context.Context, key string) ([]byte, error) {
	name := storage.constructSecretName(key)
	result, err := storage.secretCache.GetSecretBinary(name)

	if err == nil {
		return []byte(result), nil
	}

	switch err.(type) {
	case *secretsmanager.ResourceNotFoundException:
		return nil, fs.ErrNotExist
	default:
		return nil, err
	}
}

func (storage *secureStorage) Delete(ctx context.Context, key string) error {
	name := storage.constructSecretName(key)
	force := true
	_, err := storage.secretClient.DeleteSecret(&secretsmanager.DeleteSecretInput{
		SecretId:                   &name,
		ForceDeleteWithoutRecovery: &force, // force deletion to allow for a new secret with the sanem name to be stored
	})

	return err
}

func (storage *secureStorage) Exists(ctx context.Context, key string) bool {
	_, err := storage.Load(ctx, key)
	return err == nil
}

func (storage *secureStorage) List(ctx context.Context, prefix string, recursive bool) ([]string, error) {
	filterKey := "name"
	filterValue := storage.constructSecretName(prefix)

	input := &secretsmanager.ListSecretsInput{
		Filters: []*secretsmanager.Filter{
			{Key: &filterKey, Values: []*string{&filterValue}}},
	}

	keys := []string{}
	err := storage.secretClient.ListSecretsPages(input, func(output *secretsmanager.ListSecretsOutput, lastPage bool) bool {
		for _, entry := range output.SecretList {
			if entry.Name != nil {
				name := strings.TrimPrefix(*entry.Name, filterValue)
				keys = append(keys, name)
			}
		}
		return !lastPage
	})

	return keys, err
}

func (storage *secureStorage) Stat(ctx context.Context, key string) (certmagic.KeyInfo, error) {
	result, err := storage.Load(ctx, key)

	if err != nil {
		return certmagic.KeyInfo{}, err
	}
	name := storage.constructSecretName(key)
	describeOutput, err := storage.secretClient.DescribeSecret(&secretsmanager.DescribeSecretInput{
		SecretId: &name,
	})
	if err != nil {
		return certmagic.KeyInfo{}, err
	}
	return certmagic.KeyInfo{
		Key:        key,
		Size:       int64(len(result)),
		IsTerminal: strings.HasSuffix(key, "/"),
		Modified:   *describeOutput.LastChangedDate,
	}, nil
}

func (storage *secureStorage) String() string {
	return "SecureStorage"
}

func (storage *secureStorage) constructSecretName(key string) string {
	return fmt.Sprintf("%s/%s", storage.secretPrefix, key)
}
