package server

import (
	"crypto/tls"

	"github.com/spf13/pflag"
)

type MetricsConfig struct {
	BindAddress string `json:"bind_address"`
	EnableHTTPS bool   `json:"enable_https"`
	// Minimum TLS version accepted by the metrics server. Only takes effect when
	// EnableHTTPS is true. The data type is uint16 due to golang's
	// tls package accepts the versions in uint16 format, whose values
	// are available as constants in that same package
	MinTLSVersion uint16
}

func NewMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		BindAddress:   "localhost:8080",
		EnableHTTPS:   false,
		MinTLSVersion: tls.VersionTLS12,
	}
}

func (s *MetricsConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.BindAddress, "metrics-server-bindaddress", s.BindAddress, "Metrics server bind address")
	fs.BoolVar(&s.EnableHTTPS, "enable-metrics-https", s.EnableHTTPS, "Enable HTTPS for metrics server")
}

func (s *MetricsConfig) ReadFiles() error {
	return nil
}
