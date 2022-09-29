package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	shared "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	arrayUtils "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

var _ environments.ConfigModule = (*AdminRoleAuthZConfig)(nil)

// RolesConfiguration is the configuration of required roles per HTTP method of the admin API.
type RolesConfiguration struct {
	HTTPMethod string   `yaml:"method"`
	RoleNames  []string `yaml:"roles"`
}

// RoleConfig represents the role configuration.
type RoleConfig []RolesConfiguration

// AdminRoleAuthZConfig is the configuration of the role authZ middleware.
type AdminRoleAuthZConfig struct {
	RolesConfigFile string
	RolesConfig     RoleConfig
}

// NewAdminAuthZConfig creates a default AdminRoleAuthZConfig which is enabled and uses the production configuration.
func NewAdminAuthZConfig() *AdminRoleAuthZConfig {
	return &AdminRoleAuthZConfig{
		RolesConfigFile: "config/admin-authz-configuration.yaml",
	}
}

// AddFlags adds required flags for the role authZ configuration.
func (c *AdminRoleAuthZConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.RolesConfigFile, "admin-authz-config-file", c.RolesConfigFile,
		"Admin API authZ configuration file containing list of required role per API method")
}

// ReadFiles will read and validate the contents of the configuration file.
func (c *AdminRoleAuthZConfig) ReadFiles() error {
	return readRoleAuthZConfigFile(c.RolesConfigFile, &c.RolesConfig)
}

// GetRoleMapping will create a map of the required roles. The key will be the HTTP method and value will be a list of
// allowed roles for that specific HTTP method.
func (c *AdminRoleAuthZConfig) GetRoleMapping() map[string][]string {
	roleMapping := make(map[string][]string, len(c.RolesConfig))

	for _, config := range c.RolesConfig {
		roleMapping[config.HTTPMethod] = config.RoleNames
	}

	return roleMapping
}

func readRoleAuthZConfigFile(file string, val *RoleConfig) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return errors.Wrap(err, "reading role authz config")
	}

	if err := yaml.UnmarshalStrict([]byte(fileContents), val); err != nil {
		return errors.Wrap(err, "unmarshalling role authz config")
	}

	return nil
}

func (c *AdminRoleAuthZConfig) Validate(env *environments.Env) error {
	return validateRolesConfiguration(c.RolesConfig)
}

var allowedHTTPMethods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

func validateRolesConfiguration(configs []RolesConfiguration) error {
	for _, config := range configs {
		if !arrayUtils.Contains(allowedHTTPMethods, config.HTTPMethod) {
			return fmt.Errorf("invalid http method used %q, expected to be one of [%s]",
				config.HTTPMethod, strings.Join(allowedHTTPMethods, ","))
		}
	}
	return nil
}
