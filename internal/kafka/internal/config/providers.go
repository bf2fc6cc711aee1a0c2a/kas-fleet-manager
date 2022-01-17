package config

import (
	"errors"
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

type InstanceType types.KafkaInstanceType
type InstanceTypeMap map[string]InstanceTypeConfig

type InstanceTypeConfig struct {
	Limit *int `yaml:"limit,omitempty"`
}

// Returns a region's supported instance type list as a slice
func (itl InstanceTypeMap) AsSlice() []string {
	instanceTypeList := []string{}

	for k := range itl {
		instanceTypeList = append(instanceTypeList, k)
	}

	return instanceTypeList
}

type Region struct {
	Name                   string          `yaml:"name"`
	Default                bool            `yaml:"default"`
	SupportedInstanceTypes InstanceTypeMap `yaml:"supported_instance_type"`
}

func (r Region) IsInstanceTypeSupported(instanceType InstanceType) bool {
	for k := range r.SupportedInstanceTypes {
		if k == string(instanceType) {
			return true
		}
	}
	return false
}

func (r Region) getLimitSetForInstanceTypeInRegion(t string) (*int, error) {
	if it, found := r.SupportedInstanceTypes[t]; found {
		return it.Limit, nil
	}
	return nil, fmt.Errorf("Instance type: %s is not supported", t)
}

func (r Region) Validate(dataplaneClusterConfig *DataplaneClusterConfig) error {
	counter := 1
	totalCapacityUsed := 0
	regionCapacity := dataplaneClusterConfig.ClusterConfig.GetCapacityForRegion(r.Name)

	// verify that Limits set in this configuration matches the capacity of clusters listed in the data plane configuration
	for k, v := range r.SupportedInstanceTypes {
		// skip if limit is not set or is explicitly set to 0
		if v.Limit == nil || v.Limit != nil && *v.Limit == 0 {
			continue
		}

		if len(r.SupportedInstanceTypes) == 1 {
			capacity := dataplaneClusterConfig.ClusterConfig.GetCapacityForRegionAndInstanceType(r.Name, k, false)
			if *v.Limit != capacity {
				return fmt.Errorf("limit for instance type '%s'(%d) does not match the capacity in region %s(%d)", k, *v.Limit, r.Name, capacity)
			}
			return nil
		}

		// ensure that limit is within min and max capacity
		// min: the total capacity of clusters that support only this instance type
		// max: the total capacity of clusters that supports this instance type
		minCapacity := dataplaneClusterConfig.ClusterConfig.GetCapacityForRegionAndInstanceType(r.Name, k, true)
		maxCapacity := dataplaneClusterConfig.ClusterConfig.GetCapacityForRegionAndInstanceType(r.Name, k, false)
		if minCapacity > *v.Limit || maxCapacity < *v.Limit {
			return fmt.Errorf("limit for %s instance type (%d) does not match cluster capacity configuration in region '%s': min(%d), max(%d)", k, *v.Limit, r.Name, minCapacity, maxCapacity)
		}

		// when all limits are set, ensure its total adds up to total capacity of the region.
		if !r.RegionHasZeroOrNoLimitInstanceType() {
			totalCapacityUsed += *v.Limit

			// when we reach the last item, ensure limits for all instance types adds up to the total capacity of the region
			if counter == len(r.SupportedInstanceTypes) && totalCapacityUsed != regionCapacity {
				return fmt.Errorf("total limits set in region '%s' does not match cluster capacity configuration", r.Name)
			}
		}
		counter++
	}

	return nil
}

func (r Region) RegionHasZeroOrNoLimitInstanceType() bool {
	for _, it := range r.SupportedInstanceTypes {
		if it.Limit == nil || (it.Limit != nil && *it.Limit == 0) {
			return true
		}
	}
	return false
}

type RegionList []Region

func (rl RegionList) GetByName(regionName string) (Region, bool) {
	for _, r := range rl {
		if r.Name == regionName {
			return r, true
		}
	}
	return Region{}, false
}

func (rl RegionList) String() string {
	var names []string
	for _, r := range rl {
		names = append(names, r.Name)
	}
	return fmt.Sprint(names)
}

type Provider struct {
	Name    string     `json:"name"`
	Default bool       `json:"default"`
	Regions RegionList `json:"regions"`
}

type ProviderList []Provider

func (pl ProviderList) GetByName(providerName string) (Provider, bool) {
	for _, p := range pl {
		if p.Name == providerName {
			return p, true
		}
	}
	return Provider{}, false
}

func (pl ProviderList) String() string {
	var names []string
	for _, p := range pl {
		names = append(names, p.Name)
	}
	return fmt.Sprint(names)
}

type ProviderConfiguration struct {
	SupportedProviders ProviderList `yaml:"supported_providers"`
}

type ProviderConfig struct {
	ProvidersConfig     ProviderConfiguration `json:"providers"`
	ProvidersConfigFile string                `json:"providers_config_file"`
}

func NewSupportedProvidersConfig() *ProviderConfig {
	return &ProviderConfig{
		ProvidersConfigFile: "config/provider-configuration.yaml",
	}
}

var _ environments.ServiceValidator = &ProviderConfig{}

func (c *ProviderConfig) Validate(env *environments.Env) error {
	var dataplaneClusterConfig *DataplaneClusterConfig
	env.MustResolve(&dataplaneClusterConfig)

	providerDefaultCount := 0
	for _, p := range c.ProvidersConfig.SupportedProviders {
		if err := p.Validate(dataplaneClusterConfig); err != nil {
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

func (provider Provider) Validate(dataplaneClusterConfig *DataplaneClusterConfig) error {
	// verify that there is only one default region
	defaultCount := 0
	for _, p := range provider.Regions {
		if p.Default {
			defaultCount++
		}

		// validate instance type limits with the data plane cluster configuration when manual scaling is enabled
		if dataplaneClusterConfig.IsDataPlaneManualScalingEnabled() {
			if err := p.Validate(dataplaneClusterConfig); err != nil {
				return err
			}
		}
	}
	if defaultCount != 1 {
		return fmt.Errorf("expected 1 default region in provider %s, got %d", provider.Name, defaultCount)
	}
	return nil
}

func (c *ProviderConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.ProvidersConfigFile, "providers-config-file", c.ProvidersConfigFile, "SupportedProviders configuration file")
}

func (c *ProviderConfig) ReadFiles() error {
	return readFileProvidersConfig(c.ProvidersConfigFile, &c.ProvidersConfig)
}

func (c *ProviderConfig) GetInstanceLimit(region string, providerName string, instanceType string) (*int, error) {
	provider, ok := c.ProvidersConfig.SupportedProviders.GetByName(providerName)
	if !ok {
		return nil, fmt.Errorf("cloud provider '%s' is unsupported", providerName)
	}
	reg, ok := provider.Regions.GetByName(region)
	if !ok {
		return nil, fmt.Errorf("'%s' region in '%s' cloud provider is unsupported", region, providerName)
	}
	return reg.getLimitSetForInstanceTypeInRegion(instanceType)
}

// Read the contents of file into the providers config
func readFileProvidersConfig(file string, val *ProviderConfiguration) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return err
	}
	return yaml.UnmarshalStrict([]byte(fileContents), val)
}

func (c ProviderList) GetDefault() (Provider, error) {
	for _, p := range c {
		if p.Default {
			return p, nil
		}
	}
	return Provider{}, errors.New("no default provider found in list of supported providers")
}

func (provider Provider) GetDefaultRegion() (Region, error) {
	for _, r := range provider.Regions {
		if r.Default {
			return r, nil
		}
	}
	return Region{}, fmt.Errorf("no default region found for provider %s", provider.Name)
}

func (provider Provider) IsRegionSupported(regionName string) bool {
	_, ok := provider.Regions.GetByName(regionName)
	return ok
}
