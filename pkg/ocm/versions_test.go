package ocm

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("Versions", Ordered, func() {

	Context("when creating a HyperShift cluster", func() {
		DescribeTable("Should correctly validate the minimum version with a given channel group",
			validateVersion,
			Entry("OK: When the minimum version is provided",
				func() string { return LowestHostedCpSupport },
				func() string { return DefaultChannelGroup },
				true, true, nil),
			Entry("KO: Nightly channel group but too old",
				func() string { return "4.11.0-0.nightly-2022-10-17-040259-nightly" },
				func() string { return NightlyChannelGroup }, false, false, nil),
			Entry("OK: Nightly channel group and good version",
				func() string { return "4.12.0-0.nightly-2022-11-25-185455-nightly" },
				func() string { return NightlyChannelGroup }, true, true, nil),
			Entry("OK: When a greater version than the minimum is provided",
				func() string { return "4.13.0" },
				func() string { return DefaultChannelGroup }, true, true, nil),
			Entry("KO: When the minimum version requirement is not met",
				func() string { return "4.11.5" },
				func() string { return DefaultChannelGroup }, false, false, nil),
			Entry("OK: When a greater RC version than the minimum is provided",
				func() string { return "4.12.0-rc.1" },
				func() string { return "candidate" }, true, true, nil),
		)
	})

	Context("when listing machinepools versions", func() {
		DescribeTable("Parse correctly raw versions from version id",
			func(versionId string, expected string) {
				rawId := GetRawVersionId(versionId)
				Expect(rawId).To(Equal(expected))
			},
			Entry("stable channel",
				"openshift-v4.10.21",
				"4.10.21",
			),
			Entry("candidate channel",
				"openshift-v4.11.0-fc.0-candidate",
				"4.11.0-fc.0",
			),
			Entry("nightly channel",
				"openshift-v4.7.0-0.nightly-2021-05-21-224816-nightly",
				"4.7.0-0.nightly-2021-05-21-224816",
			),
		)
	})
})

var _ = Describe("Minimal http tokens required version", Ordered, func() {

	Context("validate http tokens required version", func() {
		DescribeTable("validate http tokens required version",
			validateMinimumHttpTokenRequiredVersion,
			Entry("required with lower version",
				"4.10",
				cmv1.Ec2MetadataHttpTokensRequired, fmt.Errorf("version '%s' is not supported with http tokens required, "+
					"minimum supported version is %s", "4.10", LowestHttpTokensRequiredSupport),
			),
			Entry("required with minimal version",
				LowestHttpTokensRequiredSupport, cmv1.Ec2MetadataHttpTokensRequired, nil,
			),
			Entry("optional with lower version",
				"4.10.21", cmv1.Ec2MetadataHttpTokensOptional, nil,
			),
			Entry("bad version",
				"bad version", cmv1.Ec2MetadataHttpTokensRequired, fmt.Errorf("version '%s' "+
					"is not supported: %v", "bad version", "Malformed version: bad version"),
			),
		)
	})
})

func validateMinimumHttpTokenRequiredVersion(version string, httpToken cmv1.Ec2MetadataHttpTokens, expectedErr error) {
	err := ValidateHttpTokensVersion(version, string(httpToken))
	if expectedErr != nil {
		Expect(err).To(BeEquivalentTo(expectedErr))
		return
	}
	Expect(err).NotTo(HaveOccurred())
}

func validateVersion(version func() string, channelGroup func() string, hypershiftEnabled bool,
	expectedValidation bool, expectedErr error) {

	v, err := cmv1.NewVersion().ID(version()).RawID(version()).HostedControlPlaneEnabled(hypershiftEnabled).Build()
	Expect(err).NotTo(HaveOccurred())

	b, err := HasHostedCPSupport(v)
	if expectedErr != nil {
		Expect(err).To(BeEquivalentTo(expectedErr))
	}
	Expect(b).To(BeIdenticalTo(expectedValidation))
}
