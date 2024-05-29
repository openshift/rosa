package rosacli

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/openshift/rosa/tests/ci/config"
	common "github.com/openshift/rosa/tests/utils/common"
)

func GetCommitAuthor() (string, error) {
	command := "git log -n 1 --no-merges --pretty=format:%an"
	runner := NewRunner()

	output, err := runner.RunCMD(strings.Split(command, " "))
	if err != nil {
		return "", err
	}

	return output.String(), nil
}

func GetCommitFoucs() (string, error) {
	command := "git log -n 1 --no-merges --pretty=format:%s"
	runner := NewRunner()

	output, err := runner.RunCMD(strings.Split(command, " "))
	if err != nil {
		return "", err
	}
	fmt.Printf("\nThe last commit is: %s\n", output.String())
	theStrSlice := strings.Split(output.String(), " ")

	var tcIDs []string
	reg := regexp.MustCompile(`(id:.*)`)
	for _, theStr := range theStrSlice {
		m := reg.FindAllString(theStr, -1)
		if len(m) > 0 {
			for _, idStr := range m {
				idStr = strings.Split(idStr, "id:")[1]
				ids := strings.Split(idStr, ",")
				tcIDs = append(tcIDs, ids...)
			}
		}
	}

	focus := strings.Join(tcIDs, "|")
	_, err = common.CreateFileWithContent(config.Test.TestFocusFile, focus)
	return focus, err
}
