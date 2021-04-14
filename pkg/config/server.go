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
	JwksURL       string        `json:"jwks_url"`
	JwksFile      string        `json:"jwks_file"`
	JwkCertCA     string        `json:"jwk_cert_ca"`
	JwkCertCAFile string        `json:"jwk_cert_ca_file"`
	// The public http host URL to access the service
	// For staging it is "https://api.stage.openshift.com"
	// For production it is "https://api.openshift.com"
	PublicHostURL         string `json:"public_url"`
	EnableTermsAcceptance bool   `json:"enable_terms_acceptance"`
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
		JwksURL:       "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/certs",
		JwksFile:      "config/jwks-file.json",
		JwkCertCA:     "",
		JwkCertCAFile: "secrets/rhsm.ca",
		HTTPSCertFile: "",
		HTTPSKeyFile:  "",
		PublicHostURL: "http://localhost",
	}
}

func (s *ServerConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.BindAddress, "api-server-bindaddress", s.BindAddress, "API server bind adddress")
	fs.StringVar(&s.Hostname, "api-server-hostname", s.Hostname, "Server's public hostname")
	fs.DurationVar(&s.ReadTimeout, "http-read-timeout", s.ReadTimeout, "HTTP server read timeout")
	fs.DurationVar(&s.WriteTimeout, "http-write-timeout", s.WriteTimeout, "HTTP server write timeout")
	fs.StringVar(&s.HTTPSCertFile, "https-cert-file", s.HTTPSCertFile, "The path to the tls.crt file.")
	fs.StringVar(&s.HTTPSKeyFile, "https-key-file", s.HTTPSKeyFile, "The path to the tls.key file.")
	fs.BoolVar(&s.EnableHTTPS, "enable-https", s.EnableHTTPS, "Enable HTTPS rather than HTTP")
	fs.BoolVar(&s.EnableJWT, "enable-jwt", s.EnableJWT, "Enable JWT authentication validation")
	fs.BoolVar(&s.EnableTermsAcceptance, "enable-terms-acceptance", s.EnableTermsAcceptance, "Enable terms acceptance check")
	fs.BoolVar(&s.EnableAuthz, "enable-authz", s.EnableAuthz, "Enable Authorization on endpoints, should only be disabled for debug")
	fs.StringVar(&s.JwksURL, "jwks-url", s.JwksURL, "The URL of the JSON web token signing certificates.")
	fs.StringVar(&s.JwksFile, "jwks-file", s.JwksFile, "File containing the the JSON web token signing certificates.")
	fs.StringVar(&s.JwkCertCAFile, "jwk-cert-ca-file", s.JwkCertCAFile, "JWK Certificate CA file")
	fs.StringVar(&s.PublicHostURL, "public-host-url", s.PublicHostURL, "Public http host URL of the service")
}

func (s *ServerConfig) ReadFiles() error {
	err := readFileValueString(s.JwkCertCAFile, &s.JwkCertCA)
	if err != nil {
		return err
	}

	s.JwksFile = BuildFullFilePath(s.JwksFile)

	return nil
}
