package rosacli

import (
	"fmt"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"

	"github.com/openshift/rosa/tests/ci/config"
	"github.com/openshift/rosa/tests/ci/labels"
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
func GetCommitMessage() (string, error) {
	command := "git log -n 1 --no-merges --pretty=format:%s"
	runner := NewRunner()

	output, err := runner.RunCMD(strings.Split(command, " "))
	if err != nil {
		return "", err
	}
	fmt.Printf("\nThe last commit is: %s\n", output.String())
	return output.String(), err
}

func GetFocusCaseIDs(commitMessage string) (string, error) {
	reg := regexp.MustCompile(`ids?:([0-9,\s,]*)`)
	idsMatched := reg.FindAllStringSubmatch(commitMessage, -1)
	focus := ""
	var ids = []string{}
	for _, matched := range idsMatched {
		ids = append(ids, common.ParseCommaSeparatedStrings(matched[1])...)
	}

	focus = strings.Join(ids, "|")
	_, err := common.CreateFileWithContent(config.Test.TestFocusFile, focus)
	return focus, err
}

func GetFeatureLabelFilter(commitMessage string) (*Labels, error) {
	featureFilterMap := map[*Labels]regexp.Regexp{
		&labels.Feature.Machinepool: *regexp.MustCompile(`[Ff]eature\s?:\s?[mM]achine[\s-]?[pP]ools?`),
		// TODO more other feature labels mapping will be added once this can work well
	}
	for label, regexpMatch := range featureFilterMap {
		if regexpMatch.MatchString(commitMessage) {
			_, err := common.CreateFileWithContent(config.Test.TestLabelFilterFile, *label)
			return label, err
		}
	}

	return nil, nil
}
