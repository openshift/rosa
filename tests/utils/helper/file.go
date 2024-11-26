package helper

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func GetCurrentWorkingDir() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	exPath := filepath.Dir(ex)
	return exPath, err
}

func CreateTemplateDirForNetworkResources(templateName string, fileContent string) (string, error) {
	err := os.Mkdir(templateName, 0744)
	if err != nil {
		return "", err
	}
	exPath, err := GetCurrentWorkingDir()
	if err != nil {
		return "", err
	}
	dirpath := filepath.Join(exPath + "/" + templateName)
	outputPath := filepath.Join(dirpath, "cloudformation.yaml")

	f, err := os.Create(outputPath)
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

// Read file content to an object
func ReadFileContentToObject(fileAbsPath string, obj interface{}) error {
	content, err := ReadFileContent(fileAbsPath)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(content), obj)
	if err != nil {
		return err
	}
	return nil
}
