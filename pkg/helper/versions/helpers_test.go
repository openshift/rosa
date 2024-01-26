package versions

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("Version Helpers", Ordered, func() {
	Context("when creating a hosted machine pool ", func() {
		DescribeTable("Filtered versions",
			func(versionList []string, minVersion string, maxVersion string, expectedVersionList []string) {
				filteredVersionList := GetFilteredVersionList(versionList, minVersion, maxVersion)
				Expect(filteredVersionList).To(BeEquivalentTo(expectedVersionList))
			},
			Entry("machinepool create",
				[]string{
					"4.12.0-rc.8",
					"4.12.1",
					"4.12.2",
					"4.12.3",
					"4.12.4",
					"4.12.5",
					"4.13.0-0.nightly-2023-02-22-192922",
				},
				"4.12.2",
				"4.12.5",
				[]string{
					"4.12.2",
					"4.12.3",
					"4.12.4",
					"4.12.5",
				},
			),
			Entry("machinepool update",
				[]string{
					"4.12.0-rc.8",
					"4.12.1",
					"4.12.2",
					"4.12.3",
					"4.12.4",
					"4.12.5",
					"4.13.0-0.nightly-2023-02-22-192922",
				},
				"4.12.4",
				"4.12.5",
				[]string{
					"4.12.4",
					"4.12.5",
				},
			),
		)

		DescribeTable("Minimal hosted machinepool version",
			func(controlPlaneVersion string, expected string) {
				minimalVersion, err := GetMinimalHostedMachinePoolVersion(controlPlaneVersion)
				Expect(err).ToNot(HaveOccurred())
				Expect(minimalVersion).To(Equal(expected))
			},
			Entry("Future control plane",
				"4.15.0",
				"4.13.0",
			),
			Entry("Nightly control plane",
				"4.14.0-0.nightly-2023-02-27-084419",
				"4.12.0",
			),
			Entry("Current control plane",
				"4.12.5",
				"4.12.0-0.a",
			),
		)
	})
})

var _ = Describe("Validates Format Major Minor Patch", func() {
	DescribeTable("Validates entries",
		func(val string, expected string) {
			formatted, err := FormatMajorMinorPatch(val)
			Expect(err).ToNot(HaveOccurred())
			Expect(formatted).To(Equal(expected))
		},
		Entry("Nightly", "4.14.0-0.nightly-2023-10-24-225235", "4.14.0"),
		Entry("General Availability", "4.14.1", "4.14.1"),
		Entry("Candidate", "4.14.0-rc.4-candidate", "4.14.0"),
	)
})

var _ = Describe("Get default version", func() {
	versionHostedDefault, err := v1.NewVersion().ROSAEnabled(true).
		RawID("4.14.9").Enabled(true).ChannelGroup("stable").
		HostedControlPlaneDefault(true).HostedControlPlaneEnabled(true).Build()
	Expect(err).NotTo(HaveOccurred())

	versionClassicDefault, err := v1.NewVersion().ROSAEnabled(true).
		RawID("4.14.8").Enabled(true).ChannelGroup("stable").Default(true).Build()
	Expect(err).NotTo(HaveOccurred())

	notDefault, err := v1.NewVersion().ROSAEnabled(true).
		RawID("4.14.0").Enabled(true).ChannelGroup("stable").Build()
	Expect(err).NotTo(HaveOccurred())

	DescribeTable("Validates entries",
		func(val *v1.Version, expected string) {
			isHostedCP := true
			result := getDefaultVersion(val, isHostedCP)
			Expect(result).To(Equal(expected))
		},
		Entry("Hosted default", versionHostedDefault,
			"4.14.9"),
		Entry("Classic default", versionClassicDefault,
			"4.14.8"),
		Entry("Not default", notDefault,
			""),
	)
})
