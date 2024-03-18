package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/openshift/rosa/tests/utils/log"
)

var _ = Describe("ROSA CLI Test", func() {
	Describe("Dummy test", func() {
		It("Dummy", func() {
			str := "dummy string"
			Expect(str).ToNot(BeEmpty())
			Logger.Infof("This is a dummy test to check everything is fine by executing jobs. Please remove me once other tests are added")
		})
	})
})
