package input

import (
	"io"
	"os"

	"github.com/ghodss/yaml"
)

// UnmarshalInputFile is a generic unmarshaller from an input path
func UnmarshalInputFile(specPath string) (map[string]interface{}, error) {
	var result map[string]interface{}
	specFile, err := os.Open(specPath)
	if err != nil {
		return result, err
	}
	defer specFile.Close()
	byteValue, err := io.ReadAll(specFile)
	if err != nil {
		return result, err
	}

	// Unmarshall the spec file
	// Support both json and yaml formats
	err = yaml.Unmarshal(byteValue, &result)
	if err != nil {
		return result, err
	}
	return result, err
}
