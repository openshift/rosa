package e2e

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/exec/reportportal"
)

var _ = Describe("Report", func() {
	It("report-portal", labels.E2EReport, func() {
		reportportal.GenerateReportXMLFile()
		reportportal.GenerateReportLog()
	})
})
