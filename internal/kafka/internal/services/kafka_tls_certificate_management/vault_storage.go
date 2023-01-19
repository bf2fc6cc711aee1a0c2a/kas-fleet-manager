package kafka_tls_certificate_management

import (
	"context"
	"fmt"
	"io/fs"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-secretsmanager-caching-go/secretcache"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/caddyserver/certmagic"
)

type vaultStorage struct {
	secretCache  *secretcache.Cache
	secretClient *secretsmanager.SecretsManager
	secretPrefix string
}

func newVaultStorage(config *config.AWSConfig) (*vaultStorage, error) {
	awsConfig := &aws.Config{
		Credentials: credentials.NewStaticCredentials(
			config.SecretManagerAccessKey,
			config.SecretManagerSecretAccessKey,
			""),
		Region:  aws.String(config.SecretManagerRegion),
		Retryer: client.DefaultRetryer{NumMaxRetries: 2},
	}
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}

	// cache the keys for a day
	oneDayInNanoSeconds := int64(86400000000000)
	secretClient := secretsmanager.New(sess)
	secretCache, err := secretcache.New(func(cache *secretcache.Cache) {
		cache.Client = secretClient
		cache.CacheItemTTL = oneDayInNanoSeconds
	})

	if err != nil {
		return nil, err
	}
	return &vaultStorage{
		secretClient: secretClient,
		secretCache:  secretCache,
		secretPrefix: config.SecretManagerSecretPrefix,
	}, nil
}

func (storage *vaultStorage) Lock(ctx context.Context, key string) error {
	return nil // NOOP as AWS Secret manager uses versioning mechanism to achieve optimistic locking
}

func (storage *vaultStorage) Unlock(ctx context.Context, key string) error {
	return nil // NOOP as AWS Secret manager uses versioning mechanism to achieve optimistic locking
}

func (storage *vaultStorage) Store(ctx context.Context, key string, value []byte) error {
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

func (storage *vaultStorage) Load(ctx context.Context, key string) ([]byte, error) {
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

func (storage *vaultStorage) Delete(ctx context.Context, key string) error {
	name := storage.constructSecretName(key)
	force := true
	_, err := storage.secretClient.DeleteSecret(&secretsmanager.DeleteSecretInput{
		SecretId:                   &name,
		ForceDeleteWithoutRecovery: &force, // force deletion to allow for a new secret with the sanem name to be stored
	})

	return err
}

func (storage *vaultStorage) Exists(ctx context.Context, key string) bool {
	_, err := storage.Load(ctx, key)
	return err == nil
}

func (storage *vaultStorage) List(ctx context.Context, prefix string, recursive bool) ([]string, error) {
	filterKey := `name`
	filterValue := fmt.Sprintf("%s/", storage.secretPrefix)

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

func (storage *vaultStorage) Stat(ctx context.Context, key string) (certmagic.KeyInfo, error) {
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
		Modified:   *describeOutput.LastChangedDate,
		Size:       int64(len(result)),
		IsTerminal: strings.HasSuffix(key, "/"),
	}, nil
}

func (storage *vaultStorage) String() string {
	return "VaultStorage"
}

func (storage *vaultStorage) constructSecretName(key string) string {
	return fmt.Sprintf("%s/%s", storage.secretPrefix, key)
}
