package server

import (
	"crypto/tls"

	"github.com/spf13/pflag"
)

type HealthCheckConfig struct {
	BindAddress string `json:"bind_address"`
	EnableHTTPS bool   `json:"enable_https"`
	// Minimum TLS version accepted by the healthcheck server. Only takes effect when
	// EnableHTTPS is true. The data type is uint16 due to golang's
	// tls package accepts the versions in uint16 format, whose values
	// are available as constants in that same package
	MinTLSVersion uint16
}

func NewHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		BindAddress:   "localhost:8083",
		EnableHTTPS:   false,
		MinTLSVersion: tls.VersionTLS12,
	}
}

func (c *HealthCheckConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.BindAddress, "health-check-server-bindaddress", c.BindAddress, "Health check server bind address")
	fs.BoolVar(&c.EnableHTTPS, "enable-health-check-https", c.EnableHTTPS, "Enable HTTPS for health check server")
}

func (c *HealthCheckConfig) ReadFiles() error {
	return nil
}
