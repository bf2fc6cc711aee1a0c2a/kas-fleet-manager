package config

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/spf13/pflag"
)

type ApplicationConfig struct {
	Server      *ServerConfig      `json:"server"`
	Metrics     *MetricsConfig     `json:"metrics"`
	HealthCheck *HealthCheckConfig `json:"health_check"`
	Database    *DatabaseConfig    `json:"database"`
	OCM         *OCM               `json:"ocm"`
	Sentry      *SentryConfig      `json:"sentry"`
}

func NewApplicationConfig() *ApplicationConfig {
	return &ApplicationConfig{
		Server:      NewServerConfig(),
		Metrics:     NewMetricsConfig(),
		HealthCheck: NewHealthCheckConfig(),
		Database:    NewDatabaseConfig(),
		OCM:         NewOCMConfig(),
		Sentry:      NewSentryConfig(),
	}
}

func (c *ApplicationConfig) AddFlags(flagset *pflag.FlagSet) {
	flagset.AddGoFlagSet(flag.CommandLine)
	c.Server.AddFlags(flagset)
	c.Metrics.AddFlags(flagset)
	c.HealthCheck.AddFlags(flagset)
	c.Database.AddFlags(flagset)
	c.OCM.AddFlags(flagset)
	c.Sentry.AddFlags(flagset)
}

func (c *ApplicationConfig) ReadFiles() error {
	err := c.Server.ReadFiles()
	if err != nil {
		return err
	}
	err = c.Metrics.ReadFiles()
	if err != nil {
		return err
	}
	err = c.HealthCheck.ReadFiles()
	if err != nil {
		return err
	}
	err = c.Database.ReadFiles()
	if err != nil {
		return err
	}
	err = c.OCM.ReadFiles()
	if err != nil {
		return err
	}
	err = c.Sentry.ReadFiles()
	return err
}

// Read the contents of file into integer value
func readFileValueInt(file string, val *int) error {
	fileContents, err := readFile(file)
	if err != nil {
		return err
	}

	*val, err = strconv.Atoi(fileContents)
	return err
}

// Read the contents of file into string value
func readFileValueString(file string, val *string) error {
	fileContents, err := readFile(file)
	if err != nil {
		return err
	}

	*val = strings.TrimSuffix(fileContents, "\n")
	return err
}

// Read the contents of file into boolean value
func readFileValueBool(file string, val *bool) error {
	fileContents, err := readFile(file)
	if err != nil {
		return err
	}

	*val, err = strconv.ParseBool(fileContents)
	return err
}

func readFile(file string) (string, error) {
	// If the value is in quotes, unquote it
	unquotedFile, err := strconv.Unquote(file)
	if err != nil {
		// values without quotes will raise an error, ignore it.
		unquotedFile = file
	}

	// If no file is provided, leave val unchanged.
	if unquotedFile == "" {
		return "", nil
	}

	// Ensure the absolute file path is used
	absFilePath := unquotedFile
	if !filepath.IsAbs(unquotedFile) {
		absFilePath = filepath.Join(GetProjectRootDir(), unquotedFile)
	}

	// Read the file
	buf, err := ioutil.ReadFile(absFilePath)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// TODO be sure to change this to the name of the root directory for this project
func GetProjectRootDir() string {
	wd, err := os.Getwd()
	if err != nil {
		glog.Fatal(err)
	}
	dirs := strings.Split(wd, "/")
	var rootPath string
	for _, d := range dirs {
		rootPath = rootPath + "/" + d
		if d == "ocm-example-service" {
			break
		}
	}

	return rootPath
}
