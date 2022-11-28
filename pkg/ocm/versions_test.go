package ocm

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Versions", Ordered, func() {

	Context("when creating a HyperShift cluster", func() {
		DescribeTable("Should correctly validate the minimum version with a given channel group",
			validateVersion,
			Entry("OK: When the minimum version is provided",
				func() string { return LowestHostedCPSupport },
				func() string { return DefaultChannelGroup },
				true, nil),
			Entry("KO: Nightly channel group but too old",
				func() string { return "4.11.0-0.nightly-2022-10-17-040259-nightly" },
				func() string { return NightlyChannelGroup }, false, nil),
			Entry("OK: Nightly channel group and good version",
				func() string { return "4.12.0-0.nightly-2022-11-25-185455-nightly" },
				func() string { return NightlyChannelGroup }, true, nil),
			Entry("OK: When a greater version than the minimum is provided",
				func() string { return "4.13.0" },
				func() string { return DefaultChannelGroup }, true, nil),
			Entry("KO: When the minimum version requirement is not met",
				func() string { return "4.11.5" },
				func() string { return DefaultChannelGroup }, false, nil),
			Entry("OK: When a greater RC version than the minimum is provided",
				func() string { return "4.12.0-rc.1" },
				func() string { return "candidate" }, true, nil),
		)
	})
})

func validateVersion(version func() string, channelGroup func() string, expectedValidation bool, expectedErr error) {

	b, err := HasHostedCPSupport(version())
	if expectedErr != nil {
		Expect(err).To(BeEquivalentTo(expectedErr))
	}
	Expect(b).To(BeIdenticalTo(expectedValidation))
}
