package shared

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

var projectRootDirectory = GetProjectRootDir()

// Read the contents of file into integer value
func ReadFileValueInt(file string, val *int) error {
	fileContents, err := ReadFile(file)
	if err != nil {
		return err
	}

	*val, err = strconv.Atoi(fileContents)
	return err
}

// Read the contents of file into string value
func ReadFileValueString(file string, val *string) error {
	fileContents, err := ReadFile(file)
	if err != nil {
		return err
	}

	*val = strings.TrimSuffix(fileContents, "\n")
	return err
}

// Read the contents of file into boolean value
func ReadFileValueBool(file string, val *bool) error {
	fileContents, err := ReadFile(file)
	if err != nil {
		return err
	}

	*val, err = strconv.ParseBool(fileContents)
	return err
}

func ReadFile(file string) (string, error) {
	absFilePath := BuildFullFilePath(file)

	// If no file is provided then we don't try to read it
	if absFilePath == "" {
		return "", nil
	}

	// Read the file
	buf, err := os.ReadFile(absFilePath)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func BuildFullFilePath(filename string) string {
	// If the value is in quotes, unquote it
	unquotedFile, err := strconv.Unquote(filename)
	if err != nil {
		// values without quotes will raise an error, ignore it.
		unquotedFile = filename
	}

	// If no file is provided, leave val unchanged.
	if unquotedFile == "" {
		return ""
	}

	// Ensure the absolute file path is used
	absFilePath := unquotedFile
	if !filepath.IsAbs(unquotedFile) {
		absFilePath = filepath.Join(projectRootDirectory, unquotedFile)
	}
	return absFilePath
}

func CreateTempFileFromStringData(namePrefix string, contents string) (string, error) {
	configFile, err := os.CreateTemp("", namePrefix)
	if err != nil {
		return "", err
	}
	if _, err = configFile.Write([]byte(contents)); err != nil {
		// don't forget to close the file
		_ = configFile.Close()
		return configFile.Name(), err
	}
	err = configFile.Close()
	return configFile.Name(), err
}

func ReadYamlFile(filename string, out interface{}) (err error) {
	fileContents, err := ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal([]byte(fileContents), out)
}

// ReadJSONFile reads a JSON file located in `filename` path and
// leaves the result in the `out` argument. If the file
// exists but it is empty or only contain blanks the method
// succeeds leaving the `out` argument unmodified
func ReadJSONFile(filename string, out interface{}) error {
	fileContents, err := ReadFile(filename)
	if err != nil {
		return err
	}

	trimmedString := strings.TrimSpace(fileContents)
	if trimmedString == "" {
		return nil
	}

	return json.Unmarshal([]byte(fileContents), out)
}
