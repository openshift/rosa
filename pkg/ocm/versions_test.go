package ocm

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Versions", Ordered, func() {

	var (
		lowestVersionWithoutChannelGroup string
	)
	_ = BeforeAll(func() {

		lowestVersionWithoutChannelGroup = strings.Split(LowestHostedCPSupport, "-")[0]
	})
	Context("when creating a HyperShift cluster", func() {
		DescribeTable("Should correctly validate the minimum version with a given channel group",
			validateVersion,
			Entry("OK: When the minimum version is provided",
				func() string { return LowestHostedCPSupport },
				true, nil),
			Entry("OK: When the minimum version with stable channel group",
				func() string { return fmt.Sprintf("%s-stable", lowestVersionWithoutChannelGroup) }, true, nil),
			Entry("OK: When a greater version than the minimum is provided",
				func() string { return "4.13.0" }, true, nil),
			Entry("KO: When the minimum version requirement is not met",
				func() string { return "4.11.5" }, false, nil),
			Entry("KO: When it contains an invalid version",
				func() string { return "foo.bar" }, false, fmt.Errorf("Malformed version: foo.bar")),
		)
	})
})

func validateVersion(version func() string, expectedValidation bool, expectedErr error) {

	b, err := HasHostedCPSupport(version())
	if expectedErr != nil {
		Expect(err).To(BeEquivalentTo(expectedErr))
	}
	Expect(b).To(BeIdenticalTo(expectedValidation))
}
