package e2e

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Report", func() {
	It("report-portal", labels.E2EReport, func() {
		rosacli.GenerateReportXMLFile()
		rosacli.GenerateReportLog()
	})
})
