package config

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"io/ioutil"
	"log"
	"os"
	"testing"

	. "github.com/onsi/gomega"
)

func TestConfigReadStringFile(t *testing.T) {
	RegisterTestingT(t)

	stringFile, err := createConfigFile("string", "example\n")
	defer os.Remove(stringFile.Name())
	if err != nil {
		log.Fatal(err)
	}

	var stringConfig string
	err = shared.ReadFileValueString(stringFile.Name(), &stringConfig)
	Expect(err).NotTo(HaveOccurred())
	Expect(stringConfig).To(Equal("example"))
}

func TestConfigReadEmptyFile(t *testing.T) {
	RegisterTestingT(t)
	res, err := shared.ReadFile("")
	Expect(err).NotTo(HaveOccurred())
	Expect(res).To(Equal(""))
}

func TestConfigReadIntFile(t *testing.T) {
	RegisterTestingT(t)

	intFile, err := createConfigFile("int", "123")
	defer os.Remove(intFile.Name())
	if err != nil {
		log.Fatal(err)
	}

	var intConfig int
	err = shared.ReadFileValueInt(intFile.Name(), &intConfig)
	Expect(err).NotTo(HaveOccurred())
	Expect(intConfig).To(Equal(123))
}

func TestConfigReadBoolFile(t *testing.T) {
	RegisterTestingT(t)

	boolFile, err := createConfigFile("bool", "true")
	defer os.Remove(boolFile.Name())
	if err != nil {
		log.Fatal(err)
	}

	var boolConfig bool = false
	err = shared.ReadFileValueBool(boolFile.Name(), &boolConfig)
	Expect(err).NotTo(HaveOccurred())
	Expect(boolConfig).To(Equal(true))
}

func TestConfigReadQuotedFile(t *testing.T) {
	RegisterTestingT(t)

	stringFile, err := createConfigFile("string", "example")
	defer os.Remove(stringFile.Name())
	if err != nil {
		log.Fatal(err)
	}

	quotedFileName := "\"" + stringFile.Name() + "\""
	val, err := shared.ReadFile(quotedFileName)
	Expect(err).NotTo(HaveOccurred())
	Expect(val).To(Equal("example"))
}
func createConfigFile(namePrefix, contents string) (*os.File, error) {
	configFile, err := ioutil.TempFile("", namePrefix)
	if err != nil {
		return nil, err
	}
	if _, err = configFile.Write([]byte(contents)); err != nil {
		return configFile, err
	}
	err = configFile.Close()
	return configFile, err
}
