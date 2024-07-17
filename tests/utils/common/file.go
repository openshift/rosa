package common

import (
	"encoding/json"
	"os"
	"strings"

	. "github.com/openshift/rosa/tests/utils/log"
)

func CreateTempFileWithContent(fileContent string) (string, error) {
	return CreateTempFileWithPrefixAndContent("tmpfile", fileContent)
}

func CreateTempFileWithPrefixAndContent(prefix string, fileContent string) (string, error) {
	f, err := os.CreateTemp("", prefix+"-")
	if err != nil {
		return "", err
	}
	return CreateFileWithContent(f.Name(), fileContent)
}

func CreateTempOCMConfig() (string, error) {
	// Create a tmp file
	tmpFile, err := os.CreateTemp("", "rosacli")
	if err != nil {
		return "", err
	}
	// Delete the tmp file, so that rosa will act as though it's logged out
	tmpFile.Close()
	os.Remove(tmpFile.Name())
	return tmpFile.Name(), nil
}

// Write string to a file
func CreateFileWithContent(fileAbsPath string, content interface{}) (string, error) {
	var err error
	switch content := content.(type) {
	case string:
		err = os.WriteFile(fileAbsPath, []byte(content), 0644) // #nosec G306
	case []byte:
		err = os.WriteFile(fileAbsPath, content, 0644) // #nosec G306
	case interface{}:
		var marshedContent []byte
		marshedContent, err = json.Marshal(content)
		if err != nil {
			return fileAbsPath, err
		}
		err = os.WriteFile(fileAbsPath, marshedContent, 0644) // #nosec G306
	}

	if err != nil {
		Logger.Errorf("Failed to write to file: %s", err)
		return "", err
	}
	return fileAbsPath, err
}

// Read file content to a string
func ReadFileContent(fileAbsPath string) (string, error) {
	output, err := os.ReadFile(fileAbsPath)
	if err != nil {
		Logger.Errorf("Failed to read file: %s", err)
		return "", err
	}
	content := strings.TrimSuffix(string(output), "\n")
	return content, err
}
