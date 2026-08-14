package spotterminationqueue

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Spot termination queue command options", func() {
	It("exposes the expected flags and defaults", func() {
		cmd, options := BuildCreateSpotTerminationQueueCommandWithOptions()

		Expect(cmd.Flags().Lookup("name")).ToNot(BeNil())
		Expect(cmd.Flags().Lookup("nodepool-management-role-arn")).ToNot(BeNil())
		Expect(cmd.Flags().Lookup("mode")).ToNot(BeNil())
		Expect(options.Mode).To(Equal("auto"))
	})
})
