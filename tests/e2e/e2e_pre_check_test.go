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
		author, err := rosacli.GetCommitAuthor()
		Expect(err).ToNot(HaveOccurred())

		focus, err := rosacli.GetCommitFoucs()
		Expect(err).ToNot(HaveOccurred())
		fmt.Printf("[%s] Focus: %v\n", author, focus)
	})
})
