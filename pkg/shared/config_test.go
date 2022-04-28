package shared

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	. "github.com/onsi/gomega"
)

func Test_Config_ReadStringFile(t *testing.T) {
	RegisterTestingT(t)

	stringFile, err := createConfigFile("string", "example\n")
	defer os.Remove(stringFile.Name())
	if err != nil {
		log.Fatal(err)
	}

	var stringConfig string
	err = ReadFileValueString(stringFile.Name(), &stringConfig)
	Expect(err).NotTo(HaveOccurred())
	Expect(stringConfig).To(Equal("example"))
}

func Test_Config_ReadEmptyFile(t *testing.T) {
	RegisterTestingT(t)
	res, err := ReadFile("")
	Expect(err).NotTo(HaveOccurred())
	Expect(res).To(Equal(""))
}

func Test_Config_ReadIntFile(t *testing.T) {
	RegisterTestingT(t)

	intFile, err := createConfigFile("int", "123")
	defer os.Remove(intFile.Name())
	if err != nil {
		log.Fatal(err)
	}

	var intConfig int
	err = ReadFileValueInt(intFile.Name(), &intConfig)
	Expect(err).NotTo(HaveOccurred())
	Expect(intConfig).To(Equal(123))
}

func Test_Config_ReadBoolFile(t *testing.T) {
	RegisterTestingT(t)

	boolFile, err := createConfigFile("bool", "true")
	defer os.Remove(boolFile.Name())
	if err != nil {
		log.Fatal(err)
	}

	var boolConfig bool = false
	err = ReadFileValueBool(boolFile.Name(), &boolConfig)
	Expect(err).NotTo(HaveOccurred())
	Expect(boolConfig).To(Equal(true))
}

func Test_Config_ReadQuotedFile(t *testing.T) {
	RegisterTestingT(t)

	stringFile, err := createConfigFile("string", "example")
	defer os.Remove(stringFile.Name())
	if err != nil {
		log.Fatal(err)
	}

	quotedFileName := "\"" + stringFile.Name() + "\""
	val, err := ReadFile(quotedFileName)
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

func createYamlFilefromStringData(namePrefix string, contents string) (*os.File, error) {
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

func Test_ReadYamlFile(t *testing.T) {
	RegisterTestingT(t)

	yamlFile, err := createYamlFilefromStringData("skiplist.yaml", "---\n- 01234\n- 56789")
	defer os.Remove(yamlFile.Name())
	if err != nil {
		log.Fatal(err)
	}

	var skiplist []string
	expectedSkipList := []string{"01234", "56789"}
	quotedFileName := "\"" + yamlFile.Name() + "\""
	err = ReadYamlFile(quotedFileName, &skiplist)
	Expect(err).NotTo(HaveOccurred())
	Expect(expectedSkipList).To(Equal(skiplist))
}
