package versions

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/test"
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
				"4.17.0",
				"4.15.0",
			),
			Entry("Nightly control plane",
				"4.16.0-0.nightly-2023-02-27-084419",
				"4.14.0",
			),
			Entry("Current control plane",
				"4.14.5",
				"4.12.0",
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
		func(val *v1.Version, isHostedCP, expected bool) {
			result := isDefaultVersion(val, isHostedCP)
			Expect(result).To(Equal(expected))
		},
		Entry("Hosted default", versionHostedDefault, true,
			true),
		Entry("Classic default", versionClassicDefault, false,
			true),
		Entry("Not default", notDefault, false,
			false),
	)
})

var _ = Describe("computeVersionListAndDefault", func() {

	versionHostedDefault, err := v1.NewVersion().ROSAEnabled(true).
		RawID("4.14.9").Enabled(true).ChannelGroup("stable").
		HostedControlPlaneDefault(true).HostedControlPlaneEnabled(true).Build()
	Expect(err).NotTo(HaveOccurred())

	versionClassicDefault, err := v1.NewVersion().ROSAEnabled(true).
		RawID("4.14.8").Enabled(true).ChannelGroup("stable").Default(true).Build()
	Expect(err).NotTo(HaveOccurred())

	versionHostedNotDefault, err := v1.NewVersion().ROSAEnabled(true).
		RawID("4.14.7").Enabled(true).ChannelGroup("stable").
		HostedControlPlaneDefault(true).HostedControlPlaneEnabled(true).Build()
	Expect(err).NotTo(HaveOccurred())

	versionClassicNotDefault, err := v1.NewVersion().ROSAEnabled(true).
		RawID("4.14.6").Enabled(true).ChannelGroup("stable").Default(true).Build()
	Expect(err).NotTo(HaveOccurred())

	notROSAEnabled, err := v1.NewVersion().ROSAEnabled(false).
		RawID("4.14.0").Enabled(true).ChannelGroup("stable").Build()
	Expect(err).NotTo(HaveOccurred())

	// This is older than minimum supported
	notSTS, err := v1.NewVersion().ROSAEnabled(false).
		RawID("4.5.0").Enabled(true).ChannelGroup("stable").Build()
	Expect(err).NotTo(HaveOccurred())

	versionList := make([]*v1.Version, 3)
	versionList = append(versionList, versionHostedDefault, versionClassicDefault, versionHostedNotDefault,
		versionClassicNotDefault, notROSAEnabled, notSTS)

	DescribeTable("compute",
		func(versions []*v1.Version, isHostedCP, isSTS, filterHostedCP bool, expectedDefault string,
			expectedVersions []string) {
			defaultVersion, versionList, err := computeVersionListAndDefault(versions, isHostedCP,
				isSTS, filterHostedCP)
			Expect(err).ToNot(HaveOccurred())
			Expect(defaultVersion).To(Equal(expectedDefault))
			Expect(versionList).To(Equal(expectedVersions))
		},
		Entry("HCP", versionList, true, true, true, "4.14.9", []string{"4.14.9", "4.14.7"}),
		Entry("Classic", versionList, false, true, false, "4.14.8", []string{"4.14.9", "4.14.8",
			"4.14.7", "4.14.6", "4.14.0"}),
	)
})

var _ = Describe("GetVersionList", func() {
	var testRuntime test.TestingRuntime
	versionHCPDefault, err := v1.NewVersion().ID("4.14.1").RawID("openshift-4.14.1").
		HostedControlPlaneDefault(true).ROSAEnabled(true).Build()
	Expect(err).NotTo(HaveOccurred())
	versionHCPClassicDefault, err := v1.NewVersion().ID("4.14.2").RawID("openshift-4.14.2").
		HostedControlPlaneDefault(false).ROSAEnabled(true).Default(true).Build()
	Expect(err).NotTo(HaveOccurred())

	BeforeEach(func() {
		testRuntime.InitRuntime()
	})

	It("Expects no version found", func() {
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			test.FormatVersionList([]*v1.Version{})))
		_, vs, err := GetVersionList(testRuntime.RosaRuntime, "stable",
			true, true, true, false)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("could not find versions for the provided channel-group"))
		Expect(len(vs)).To(Equal(0))
	})

	It("Expects a single version", func() {
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			test.FormatVersionList([]*v1.Version{versionHCPDefault})))
		defaultVersion, vs, err := GetVersionList(testRuntime.RosaRuntime,
			"stable", true, true, false, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(vs)).To(Equal(1))
		Expect(defaultVersion).To(Equal("openshift-4.14.1"))
	})

	It("Expects a version list and a default", func() {
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			test.FormatVersionList([]*v1.Version{versionHCPDefault, versionHCPClassicDefault})))
		defaultVersion, vs, err := GetVersionList(testRuntime.RosaRuntime,
			"stable", true, true, false, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(vs)).To(Equal(2))
		Expect(defaultVersion).To(Equal("openshift-4.14.1"))
	})
})

var _ = DescribeTable("IsGreaterThanOrEqual", func(version1, version2 string, expectedResult bool,
	expectedError string) {
	result, err := IsGreaterThanOrEqual(version1, version2)
	if expectedError != "" {
		Expect(err.Error()).To(ContainSubstring(expectedError))
	}
	Expect(result).To(Equal(expectedResult))
},
	Entry("Not greater", "openshift-v4.14.1", "openshift-v4.14.2", false, ""),
	Entry("Equal", "openshift-v4.14.1", "openshift-v4.14.1", true, ""),
	Entry("Greater", "openshift-v4.14.2", "openshift-v4.14.1", true, ""),
	Entry("Invalid arg 1", "invalid", "openshift-v4.14.1", false, "Malformed version: invalid"),
	Entry("Invalid arg 2", "openshift-v4.14.2", "invalid2", false, "Malformed version: invalid2"),
)
