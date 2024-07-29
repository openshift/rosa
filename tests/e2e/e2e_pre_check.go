package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("PreCheck", func() {
	It("commits-focus", labels.E2ECommit, func() {
		_, err := rosacli.GetCommitAuthor()
		Expect(err).ToNot(HaveOccurred())

		commitMessage, err := rosacli.GetCommitMessage()
		Expect(err).ToNot(HaveOccurred())

		focus, err := rosacli.GetFocusCaseIDs(commitMessage)
		Expect(err).ToNot(HaveOccurred())
		if focus != "" {
			fmt.Printf("\nThe latest commit updates the test cases: %s\n", focus)
		}

		label, err := rosacli.GetFeatureLabelFilter(commitMessage)
		Expect(err).ToNot(HaveOccurred())
		if focus != "" {
			fmt.Printf("\nThe latest commit updates the feature: %v\n", *label)
		}

	})
})
