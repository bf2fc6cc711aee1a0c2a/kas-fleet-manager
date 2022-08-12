package shared

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/onsi/gomega"
)

func Test_Config_ReadStringFile(t *testing.T) {
	g := gomega.NewWithT(t)

	stringFile, err := createConfigFile("string", "example\n")
	defer os.Remove(stringFile.Name())
	if err != nil {
		log.Fatal(err)
	}

	var stringConfig string
	err = ReadFileValueString(stringFile.Name(), &stringConfig)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(stringConfig).To(gomega.Equal("example"))
}

func Test_Config_ReadEmptyFile(t *testing.T) {
	g := gomega.NewWithT(t)

	res, err := ReadFile("")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(res).To(gomega.Equal(""))
}

func Test_Config_ReadIntFile(t *testing.T) {
	g := gomega.NewWithT(t)
	intFile, err := createConfigFile("int", "123")
	defer os.Remove(intFile.Name())
	if err != nil {
		log.Fatal(err)
	}

	var intConfig int
	err = ReadFileValueInt(intFile.Name(), &intConfig)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(intConfig).To(gomega.Equal(123))
}

func Test_Config_ReadBoolFile(t *testing.T) {
	g := gomega.NewWithT(t)

	boolFile, err := createConfigFile("bool", "true")
	defer os.Remove(boolFile.Name())
	if err != nil {
		log.Fatal(err)
	}

	var boolConfig bool = false
	err = ReadFileValueBool(boolFile.Name(), &boolConfig)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(boolConfig).To(gomega.Equal(true))
}

func Test_Config_ReadQuotedFile(t *testing.T) {
	g := gomega.NewWithT(t)
	stringFile, err := createConfigFile("string", "example")
	defer os.Remove(stringFile.Name())
	if err != nil {
		log.Fatal(err)
	}

	quotedFileName := "\"" + stringFile.Name() + "\""
	val, err := ReadFile(quotedFileName)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(val).To(gomega.Equal("example"))
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

func Test_ReadYamlFile(t *testing.T) {
	g := gomega.NewWithT(t)

	yamlFile, err := CreateTempFileFromStringData("skiplist.yaml", "---\n- 01234\n- 56789")
	defer os.Remove(yamlFile)
	if err != nil {
		log.Fatal(err)
	}

	var skiplist []string
	expectedSkipList := []string{"01234", "56789"}
	quotedFileName := "\"" + yamlFile + "\""
	err = ReadYamlFile(quotedFileName, &skiplist)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(expectedSkipList).To(gomega.Equal(skiplist))
}

func Test_ReadJSONFile(t *testing.T) {
	g := gomega.NewWithT(t)

	testJSONContent := "{\"test_key\": \"test_value\"}"
	jsonFile, err := CreateTempFileFromStringData("testjsonfile.json", testJSONContent)
	defer os.Remove(jsonFile)
	if err != nil {
		log.Fatal(err)
	}

	type testStruct struct {
		TestKey string `json:"test_key"`
	}

	var res testStruct
	expectedResult := testStruct{
		TestKey: "test_value",
	}

	quotedFileName := "\"" + jsonFile + "\""
	err = ReadJSONFile(quotedFileName, &res)
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(res).To(gomega.Equal(expectedResult))
}
