package machinepool

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("Checking Version", Ordered, func() {

	Context("when creating machinepools", func() {
		DescribeTable("Check whether local zone is supported by checking cluster version correctly",
			checkLocalZoneSupported,
			Entry("vesion less than 4.12",
				"4.11",
				true,
				nil,
			),
			Entry("version 4.12",
				"4.12",
				false,
				nil,
			),
			Entry("version greater than 4.12",
				"4.13",
				false,
				nil,
			),
			Entry("empty version",
				"",
				false,
				nil,
			),
		)
	})
})

func checkLocalZoneSupported(version string, expectedResult bool, expectedErr error) {
	cluster, err := cmv1.NewCluster().Version(cmv1.NewVersion().RawID(version)).Build()
	Expect(err).NotTo(HaveOccurred())

	result, err := isCheckLocalZoneRequired(cluster)
	if expectedErr != nil {
		Expect(err).To(BeEquivalentTo(expectedErr))
	}
	Expect(result).To(BeIdenticalTo(expectedResult))
}
