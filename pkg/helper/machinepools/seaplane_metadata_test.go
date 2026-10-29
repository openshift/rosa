// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepools

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

var _ = Describe("Seaplane nodepool metadata validation", func() {
	DescribeTable("validates label input", func(input, expectedError string) {
		result, err := ParseSeaplaneLabels(input)
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
		result, err := ParseSeaplaneTaints(input)
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

	DescribeTable("parses and validates node drain grace period", func(input string, shouldErr bool, expectedDuration *time.Duration) {
		result, err := ParseSeaplaneNodeDrainGracePeriod(input)
		if shouldErr {
			Expect(err).To(HaveOccurred())
			return
		}
		Expect(err).NotTo(HaveOccurred())
		if expectedDuration == nil {
			Expect(result).To(BeNil())
		} else {
			Expect(result).NotTo(BeNil())
			Expect(result.Duration).To(Equal(*expectedDuration))
		}
	},
		Entry("empty input returns nil", "", false, nil),
		Entry("zero minutes", "0", false, ptrDuration(0)),
		Entry("zero with unit", "0 minutes", false, ptrDuration(0)),
		Entry("single minute", "1 minute", false, ptrDuration(1*time.Minute)),
		Entry("multiple minutes", "30 minutes", false, ptrDuration(30*time.Minute)),
		Entry("single hour", "1 hour", false, ptrDuration(1*time.Hour)),
		Entry("multiple hours", "2 hours", false, ptrDuration(2*time.Hour)),
		Entry("hour unit singular", "1 hour", false, ptrDuration(1*time.Hour)),
		Entry("minute unit singular", "1 minute", false, ptrDuration(1*time.Minute)),
		Entry("max allowed duration (168 hours)", "168 hours", false, ptrDuration(168*time.Hour)),
		Entry("max in minutes (10080)", "10080 minutes", false, ptrDuration(10080*time.Minute)),
		Entry("invalid - exceeds max", "169 hours", true, nil),
		Entry("invalid - exceeds max in minutes", "10081 minutes", true, nil),
		Entry("invalid - negative", "-1 hour", true, nil),
		Entry("invalid - invalid unit", "1 days", true, nil),
		Entry("invalid - malformed", "abc minutes", true, nil),
	)

	DescribeTable("formats node drain grace period for display", func(input *metav1.Duration, expected string) {
		result := FormatSeaplaneNodeDrainGracePeriod(input)
		Expect(result).To(Equal(expected))
	},
		Entry("nil duration", nil, ""),
		Entry("zero duration", &metav1.Duration{Duration: 0}, ""),
		Entry("30 minutes", &metav1.Duration{Duration: 30 * time.Minute}, "30m0s"),
		Entry("1 hour", &metav1.Duration{Duration: 1 * time.Hour}, "1h0m0s"),
		Entry("2 hours 30 minutes", &metav1.Duration{Duration: 2*time.Hour + 30*time.Minute}, "2h30m0s"),
	)

})

// Helper to create duration pointers for tests
func ptrDuration(d time.Duration) *time.Duration {
	return &d
}
