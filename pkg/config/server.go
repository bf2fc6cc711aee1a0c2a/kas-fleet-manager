package config

import (
	"time"

	"github.com/spf13/pflag"
)

type ServerConfig struct {
	Hostname      string        `json:"hostname"`
	BindAddress   string        `json:"bind_address"`
	ReadTimeout   time.Duration `json:"read_timeout"`
	WriteTimeout  time.Duration `json:"write_timeout"`
	HTTPSCertFile string        `json:"https_cert_file"`
	HTTPSKeyFile  string        `json:"https_key_file"`
	EnableHTTPS   bool          `json:"enable_https"`
	EnableJWT     bool          `json:"enable_jwt"`
	EnableAuthz   bool          `json:"enable_authz"`
	JwkCertURL    string        `json:"jwk_cert_url"`
	JwkCertCA     string        `json:"jwk_cert_ca"`
	JwkCertCAFile string        `json:"jwk_cert_ca_file"`
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{
		Hostname:      "",
		BindAddress:   "localhost:8000",
		ReadTimeout:   5 * time.Second,
		WriteTimeout:  30 * time.Second,
		EnableHTTPS:   false,
		EnableJWT:     true,
		EnableAuthz:   true,
		JwkCertURL:    "https://api.openshift.com/.well-known/jwks.json",
		JwkCertCA:     "",
		JwkCertCAFile: "secrets/rhsm.ca",
		HTTPSCertFile: "",
		HTTPSKeyFile:  "",
	}
}

func (s *ServerConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.BindAddress, "api-server-bindaddress", s.BindAddress, "API server bind adddress")
	fs.StringVar(&s.Hostname, "api-server-hostname", s.Hostname, "Server's public hostname")
	fs.DurationVar(&s.ReadTimeout, "http-read-timeout", s.ReadTimeout, "HTTP server read timeout")
	fs.DurationVar(&s.WriteTimeout, "http-write-timeout", 30*time.Second, "HTTP server write timeout")
	fs.StringVar(&s.HTTPSCertFile, "https-cert-file", s.HTTPSCertFile, "The path to the tls.crt file.")
	fs.StringVar(&s.HTTPSKeyFile, "https-key-file", s.HTTPSKeyFile, "The path to the tls.key file.")
	fs.BoolVar(&s.EnableHTTPS, "enable-https", s.EnableHTTPS, "Enable HTTPS rather than HTTP")
	fs.BoolVar(&s.EnableJWT, "enable-jwt", s.EnableJWT, "Enable JWT authentication validation")
	fs.BoolVar(&s.EnableAuthz, "enable-authz", s.EnableAuthz, "Enable Authorization on endpoints, should only be disabled for debug")
	fs.StringVar(&s.JwkCertURL, "jwk-cert-url", s.JwkCertURL, "JWK Certificate URL")
	fs.StringVar(&s.JwkCertCAFile, "jwk-cert-ca-file", s.JwkCertCAFile, "JWK Certificate CA file")
}

func (s *ServerConfig) ReadFiles() error {
	return readFileValueString(s.JwkCertCAFile, &s.JwkCertCA)
}
