package machinepools

import (
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

var _ = Describe("HyperFleet nodepool metadata validation", func() {
	DescribeTable("validates label input", func(input, expectedError string) {
		result, err := ParseHyperfleetLabels(input)
		if expectedError != "" {
			Expect(err).To(MatchError(ContainSubstring(expectedError)))
			return
		}
		Expect(err).NotTo(HaveOccurred())
		if input == "" || input == `""` {
			Expect(result).To(BeEmpty())
		} else {
			Expect(result).To(Equal(map[string]string{"team": "test", "empty": ""}))
		}
	},
		Entry("labels including empty values", "team=test,empty=", ""),
		Entry("trailing delimiter", "team=test,empty=,", ""),
		Entry("empty collection", "", ""), Entry("quoted clearing", `""`, ""),
		Entry("missing separator", "invalid", "expected key=value format"),
		Entry("extra separator", "team=a=b", "expected key=value format"),
		Entry("invalid key", "bad*=test", "invalid label key"),
		Entry("empty key", "=test", "invalid label key"),
		Entry("invalid value", "team=a/b", "invalid label value"),
		Entry("duplicate key", "team=test,team=other", "duplicated label key"),
	)
	DescribeTable("validates taint input", func(input, expectedError string) {
		result, err := ParseHyperfleetTaints(input)
		if expectedError != "" {
			Expect(err).To(MatchError(ContainSubstring(expectedError)))
			return
		}
		Expect(err).NotTo(HaveOccurred())
		if input == "" || input == `""` {
			Expect(result).To(BeEmpty())
		} else {
			Expect(result).To(Equal([]hypershiftv1beta1.Taint{{Key: "team", Value: "test", Effect: corev1.TaintEffectNoSchedule}, {Key: "empty", Effect: corev1.TaintEffectNoExecute}}))
		}
	},
		Entry("taints including empty values", "team=test:NoSchedule,empty=:NoExecute", ""),
		Entry("trailing delimiter", "team=test:NoSchedule,empty=:NoExecute,", ""),
		Entry("empty collection", "", ""), Entry("quoted clearing", `""`, ""),
		Entry("missing separator", "invalid", "expected key=value:scheduleType format"),
		Entry("missing effect", "team=test", "expected key=value:scheduleType format"),
		Entry("extra separator", "team=a=b:NoSchedule", "invalid taint format"),
		Entry("empty effect", "team=test:", "expected a not empty effect"),
		Entry("invalid effect", "team=test:Invalid", "invalid taint effect"),
		Entry("invalid key", "bad*=test:NoSchedule", "invalid taint key"),
	)

	DescribeTable("validates rolling update input", func(maxSurge, maxUnavailable, expectedError string) {
		err := ValidateHyperfleetUpgrade(maxSurge, maxUnavailable)
		if expectedError == "" {
			Expect(err).NotTo(HaveOccurred())
			return
		}
		Expect(err).To(MatchError(ContainSubstring(expectedError)))
	},
		Entry("absolute values", "2", "1", ""),
		Entry("percentages", "20%", "10%", ""),
		Entry("mixed units", "20%", "1", ""),
		Entry("zero max surge", "0", "1", ""),
		Entry("zero max unavailable", "1", "0", ""),
		Entry("unset values", "", "", ""),
		Entry("both zero", "0", "0", "only one of --max-surge and --max-unavailable can be zero"),
		Entry("both zero across units", "0%", "0", "only one of --max-surge and --max-unavailable can be zero"),
		Entry("negative max surge", "-1", "1", "invalid --max-surge"),
		Entry("negative max unavailable", "1", "-1", "invalid --max-unavailable"),
		Entry("percentage above 100", "101%", "1", "invalid --max-surge"),
		Entry("not a number", "two", "1", "invalid --max-surge"),
	)
})
