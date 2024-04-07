package common

import (
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

// Write string to a file
func CreateFileWithContent(fileAbsPath string, content string) (string, error) {
	err := os.WriteFile(fileAbsPath, []byte(content), 0644)
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
