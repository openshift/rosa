package file

import (
	"fmt"
	"os"

	"github.com/openshift-online/ocm-common/pkg/log"
)

func WriteToFile(content string, fileName string, path ...string) (string, error) {
	KeyPath, _ := os.UserHomeDir()

	if len(path) != 0 {
		KeyPath = path[0]
	}

	filePath := fmt.Sprintf("%s/%s", KeyPath, fileName)
	if IfFileExists(filePath) {
		err := os.Remove(filePath)
		if err != nil {
			log.LogInfo("Delete file err:%v", err)
			return "", err
		}
	}
	err := os.WriteFile(filePath, []byte(content), 0600)
	if err != nil {
		log.LogInfo("Write to file err:%v", err)
		return "", err
	}
	return filePath, nil
}

func IfFileExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}
