package services

import (
	"errors"
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
)

// ConfigService is a service used for managing and accessing the various configurations used by the overall service
type ConfigService interface {
	// GetSupportedProviders returns the current supported providers in this service
	// Deprecated: should not be used anymore, use GetConfig() and then access the config object
	GetSupportedProviders() config.ProviderList
	// GetDefaultProvider returns the default provider in the supported providers configuration, if multiple providers
	// are specified as default then the first will be returned
	// Deprecated: should not be used anymore, use GetConfig() and then access the config object
	GetDefaultProvider() (config.Provider, error)
	// GetDefaultRegionForProvider returns the default region specified in the provider, if multiple regions are
	// specified as default then the first will be returned
	// Deprecated: should not be used anymore, use GetConfig() and then access the config object
	GetDefaultRegionForProvider(provider config.Provider) (config.Region, error)
	// IsProviderSupported returns true if the given provider is supported
	// Deprecated: should not be used anymore, use GetConfig() and then access the config object
	IsProviderSupported(providerName string) bool
	// IsRegionSupportedForProvider returns true if the provided region is supported in the given provider
	// Deprecated: should not be used anymore, use GetConfig() and then access the config object
	IsRegionSupportedForProvider(providerName, regionName string) bool
	// GetOrganisationById returns the organisaion by the given id
	// Deprecated: should not be used anymore, use GetConfig() and then access the config object
	GetOrganisationById(orgId string) (config.Organisation, bool)
	// GetAllowedAccountByUsernameAndOrgId returns the allowed user in a given organisation (if found organisation is found), else return user by from the global list
	// Deprecated: should not be used anymore, use GetConfig() and then access the config object
	GetAllowedAccountByUsernameAndOrgId(username string, orgId string) (config.AllowedAccount, bool)
	// GetServiceAccountByUsername returns allowed account by from the list of service accounts
	// Deprecated: should not be used anymore, use GetConfig() and then access the config object
	GetServiceAccountByUsername(username string) (config.AllowedAccount, bool)
	// Validate ensures all configuration managed by the service contains correct and valid values
	Validate() error
	// IsAutoCreateOSDEnabled returns true if the automatic creation of OSD cluster is enabled, false otherwise.
	// Deprecated: should not be used anymore, use GetConfig() and then access the config object
	IsAutoCreateOSDEnabled() bool
	// GetObservabilityConfiguration returns ObservabilityConfiguration.
	// Deprecated: should not be used anymore, use GetConfig() and then access the config object
	GetObservabilityConfiguration() config.ObservabilityConfiguration
	// GetConfig returns ApplicationConfig, which can then be used to access configurations
	GetConfig() config.ApplicationConfig
}

var _ ConfigService = &configService{}

// configService is an internal implementation of ConfigService
type configService struct {
	appConfig config.ApplicationConfig
}

// NewConfigService returns a new default implementation of ConfigService
func NewConfigService(appConfig config.ApplicationConfig) ConfigService {
	return &configService{
		appConfig: appConfig,
	}
}

func (c configService) GetSupportedProviders() config.ProviderList {
	return c.appConfig.SupportedProviders.ProvidersConfig.SupportedProviders
}

//TODO: move this to ProviderList
func (c configService) GetDefaultProvider() (config.Provider, error) {
	for _, p := range c.GetSupportedProviders() {
		if p.Default {
			return p, nil
		}
	}
	return config.Provider{}, errors.New("no default provider found in list of supported providers")
}

//TODO: move this to ProviderList
func (c configService) GetDefaultRegionForProvider(provider config.Provider) (config.Region, error) {
	for _, r := range provider.Regions {
		if r.Default {
			return r, nil
		}
	}
	return config.Region{}, fmt.Errorf("no default region found for provider %s", provider.Name)
}

//TODO: move this to ProviderList
func (c configService) IsProviderSupported(providerName string) bool {
	_, ok := c.GetSupportedProviders().GetByName(providerName)
	return ok
}

//TODO: move this to ProviderList
func (c configService) IsRegionSupportedForProvider(providerName, regionName string) bool {
	provider, ok := c.GetSupportedProviders().GetByName(providerName)
	if !ok {
		return false
	}
	_, ok = provider.Regions.GetByName(regionName)
	return ok
}

//TODO: move this to AllowList
func (c configService) GetOrganisationById(orgId string) (config.Organisation, bool) {
	return c.appConfig.AccessControlList.AllowList.Organisations.GetById(orgId)
}

//TODO: move this to AllowList
// GetServiceAccountByUsername returns allowed account by from the list of service accounts
func (c configService) GetServiceAccountByUsername(username string) (config.AllowedAccount, bool) {
	return c.appConfig.AccessControlList.AllowList.ServiceAccounts.GetByUsername(username)
}

//TODO: move this to AllowList
// GetAllowedAccountByUsernameAndOrgId returns the allowed user in a given organisation (if found organisation is found),
// else return user by from the global list
func (c configService) GetAllowedAccountByUsernameAndOrgId(username string, orgId string) (config.AllowedAccount, bool) {
	var user config.AllowedAccount
	var found bool
	org, _ := c.GetOrganisationById(orgId)
	user, found = org.AllowedAccounts.GetByUsername(username)
	if found {
		return user, found
	}

	return c.GetServiceAccountByUsername(username)
}

//TODO: move this to ProviderList
func (c configService) Validate() error {
	providerDefaultCount := 0
	for _, p := range c.GetSupportedProviders() {
		if err := c.validateProvider(p); err != nil {
			return err
		}
		if p.Default {
			providerDefaultCount++
		}
	}
	if providerDefaultCount != 1 {
		return fmt.Errorf("expected 1 default provider in provider list, got %d", providerDefaultCount)
	}
	return nil
}

//TODO: move this to ProviderList
// validateProvider ensures a given provider contains correct and valid values, including it's regions
func (c configService) validateProvider(provider config.Provider) error {
	defaultCount := 0
	for _, p := range provider.Regions {
		if p.Default {
			defaultCount++
		}
	}
	if defaultCount != 1 {
		return fmt.Errorf("expected 1 default region in provider %s, got %d", provider.Name, defaultCount)
	}
	return nil
}

//TODO: move this to ClusterCreationConfig
func (c configService) IsAutoCreateOSDEnabled() bool {
	return c.appConfig.ClusterCreationConfig.AutoOSDCreation
}

// GetObservabilityConfiguration returns ObservabilityConfiguration.
func (c configService) GetObservabilityConfiguration() config.ObservabilityConfiguration {
	return *c.appConfig.ObservabilityConfiguration
}

func (c configService) GetConfig() config.ApplicationConfig {
	return c.appConfig
}
