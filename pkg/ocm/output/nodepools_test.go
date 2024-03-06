package output

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("Create node drain grace period builder validations", func() {
	zeroValue, _ := cmv1.NewValue().Value(0).Build()
	oneValue, _ := cmv1.NewValue().Value(1).Build()
	twoValue, _ := cmv1.NewValue().Value(2).Build()

	DescribeTable("Create node drain grace period builder validations",
		func(period *cmv1.Value, expectedOutput string) {
			output := PrintNodeDrainGracePeriod(period)
			Expect(output).To(Equal(expectedOutput))
		},
		Entry(nil,
			"",
		),
		Entry(zeroValue,
			"",
		),
		Entry(oneValue,
			"1 minute",
		),
		Entry(twoValue,
			"2 minutes",
		),
	)
})
