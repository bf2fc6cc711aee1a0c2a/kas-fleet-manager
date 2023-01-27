package vault

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-secretsmanager-caching-go/secretcache"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/metrics"
)

var OwnerResourceTagKey = "owner-resource"

var _ VaultService = &awsVaultService{}

type awsVaultService struct {
	secretCache        *secretcache.Cache
	secretClient       *secretsmanager.SecretsManager
	secretPrefixEnable bool
	secretPrefix       string
}

func NewAwsVaultService(vaultConfig *Config) (*awsVaultService, error) {
	awsConfig := &aws.Config{
		Credentials: credentials.NewStaticCredentials(
			vaultConfig.AccessKey,
			vaultConfig.SecretAccessKey,
			""),
		Region:  aws.String(vaultConfig.Region),
		Retryer: client.DefaultRetryer{NumMaxRetries: 2},
	}
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}

	secretClient := secretsmanager.New(sess)
	secretCache, err := secretcache.New(func(cache *secretcache.Cache) {
		cache.Client = secretClient
	})
	if err != nil {
		return nil, err
	}
	return &awsVaultService{
		secretClient:       secretClient,
		secretCache:        secretCache,
		secretPrefixEnable: vaultConfig.SecretPrefixEnable,
		secretPrefix:       vaultConfig.SecretPrefix + "/",
	}, nil
}

func (k *awsVaultService) Kind() string {
	return KindAws
}

func (k *awsVaultService) GetSecretString(name string) (string, error) {

	name = k.getVaultSecretName(name)
	metrics.IncreaseVaultServiceTotalCount("get")
	result, err := k.secretCache.GetSecretString(name)
	if err != nil {
		switch err.(type) {
		case *secretsmanager.ResourceNotFoundException:
			metrics.IncreaseVaultServiceErrorsCount("get")
		default:
			metrics.IncreaseVaultServiceFailureCount("get")
		}
	} else {
		metrics.IncreaseVaultServiceSuccessCount("get")
	}
	return result, err
}

func (k *awsVaultService) SetSecretString(name string, value string, owningResource string) error {

	name = k.getVaultSecretName(name)
	var tags []*secretsmanager.Tag
	if owningResource != "" {
		tags = append(tags,
			&secretsmanager.Tag{
				Key:   &OwnerResourceTagKey,
				Value: &owningResource,
			})
	}

	metrics.IncreaseVaultServiceTotalCount("set")
	_, err := k.secretClient.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:         &name,
		SecretString: &value,
		Tags:         tags,
	})
	if err != nil {
		metrics.IncreaseVaultServiceFailureCount("set")
		return err
	} else {
		metrics.IncreaseVaultServiceSuccessCount("set")
	}

	return nil
}

func (k *awsVaultService) ForEachSecret(f func(name string, owningResource string) bool) error {
	filterKey := `name`
	var paging *secretsmanager.ListSecretsInput
	if k.secretPrefixEnable {
		paging = &secretsmanager.ListSecretsInput{
			Filters: []*secretsmanager.Filter{
				{Key: &filterKey, Values: []*string{&k.secretPrefix}}, // filter secret keys with prefix
			},
		}
	} else {
		paging = &secretsmanager.ListSecretsInput{}
	}
	err := k.secretClient.ListSecretsPages(paging, func(output *secretsmanager.ListSecretsOutput, lastPage bool) bool {
		for _, entry := range output.SecretList {
			metrics.IncreaseVaultServiceTotalCount("get")
			owner := getTag(entry.Tags, OwnerResourceTagKey)
			name := ""
			if entry.Name != nil {
				name = *entry.Name
			}
			metrics.IncreaseVaultServiceSuccessCount("get")
			if !f(name, owner) {
				return false
			}
		}
		return true
	})
	if err != nil {
		metrics.IncreaseVaultServiceFailureCount("get")
		return err
	}
	return nil
}

func getTag(tags []*secretsmanager.Tag, key string) string {
	for _, tag := range tags {
		if *tag.Key == key {
			return *tag.Value
		}
	}
	return ""
}

func (k *awsVaultService) DeleteSecretString(name string) error {
	name = k.getVaultSecretName(name)
	metrics.IncreaseVaultServiceTotalCount("delete")
	_, err := k.secretClient.DeleteSecret(&secretsmanager.DeleteSecretInput{
		SecretId: &name,
	})
	if err != nil {
		switch err.(type) {
		case *secretsmanager.ResourceNotFoundException:
			metrics.IncreaseVaultServiceErrorsCount("delete")
		default:
			metrics.IncreaseVaultServiceFailureCount("delete")
		}
	} else {
		metrics.IncreaseVaultServiceSuccessCount("delete")
	}
	return err
}

func (k *awsVaultService) getVaultSecretName(name string) string {
	if k.secretPrefixEnable {
		return k.secretPrefix + name
	}
	return name
}
