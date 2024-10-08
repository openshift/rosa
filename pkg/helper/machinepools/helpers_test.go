package machinepools

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MachinePool", func() {
	DescribeTable("ParseTaints validation",
		func(taint, expectedError string, numberOfTaints int) {
			taints, err := ParseTaints(taint)
			if expectedError == "" {
				Expect(err).ToNot(HaveOccurred())
				Expect(len(taints)).To(Equal(numberOfTaints))
			} else {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expectedError))
			}
		},
		Entry("Empty taints are parsed correctly",
			"", "", 0,
		),
		Entry("Resetting taints in interactive mode is parsed correctly",
			`""`, "", 0,
		),
		Entry(
			"Well formed taint",
			"node-role.kubernetes.io/infra=val:NoSchedule", "", 1),
		Entry(
			"Well formed taint",
			"foo=bar:NoExecute", "", 1),
		Entry(
			"2 well formed taints",
			"node-role.kubernetes.io/infra=bar:NoSchedule,node-role.kubernetes.io/master=val:NoSchedule", "", 2),
		Entry(
			"Trailing ',' is parsed correctly",
			"node-role.kubernetes.io/infra=bar:NoSchedule,node-role.kubernetes.io/master=val:NoSchedule,", "", 2),
		Entry(
			"Empty value taint bad format",
			"node-role.kubernetes.io/infraNoSchedule",
			"Expected key=value:scheduleType format", 0),
		Entry(
			"Empty value taint good format",
			"node-role.kubernetes.io/infra=:NoSchedule",
			"", 1),
		Entry(
			"Empty value taint good format",
			"node-role.kubernetes.io/infra=val:NoSchedule,node-role.kubernetes.io/infra=:NoSchedule",
			"", 2),
		Entry(
			"Empty effect taint -> KO",
			"node-role.kubernetes.io/infra=:",
			"Expected a not empty effect", 0),
		Entry(
			"Bad value -> KO",
			"key=node-role.kubernetes.io/infra:NoEffect",
			"Invalid taint value 'node-role.kubernetes.io/infra': at key: 'key'", 0),
	)

	DescribeTable("Parse Labels", func(userLabels, expectedError string, numberOfLabels int) {
		labels, err := ParseLabels(userLabels)
		if expectedError == "" {
			Expect(err).ToNot(HaveOccurred())
			Expect(len(labels)).To(Equal(numberOfLabels))
		} else {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(expectedError))
		}
	},
		Entry("Empty Labels are parsed correctly",
			"", "", 0,
		),
		Entry("Resetting labels in interactive mode is parsed correctly",
			`""`, "", 0,
		),
		Entry("Single label is parsed correctly",
			"com.example.foo=bar", "", 1,
		),
		Entry("Multiple labels are parsed correctly",
			"com.example.foo=bar,com.example.baz=bob", "", 2,
		),
		Entry("Trailing ',' is parsed correctly",
			"com.example.foo=bar,com.example.baz=bob,", "", 2,
		),
		Entry("Labels with no value are parsed correctly",
			"com.example.foo=,com.example.baz=bob", "", 2,
		),
		Entry("Duplicate labels are not supported",
			"com.example.foo=bar,com.example.foo=bob", "Duplicated label key 'com.example.foo' used", 0,
		),
		Entry("Malformed labels are not supported",
			"com.example.foo,com.example.bar=bob", "Expected key=value format for labels", 0,
		),
	)

})

var _ = Describe("Label validations", func() {
	DescribeTable("Label validation",
		func(key string, value string, hasError bool) {
			err := ValidateLabelKeyValuePair(key, value)
			if hasError {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("label"))
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
		},
		Entry("Should not error with key of 'mykey', value 'myvalue'",
			"mykey",
			"myvalue",
			false,
		),
		Entry("Should error with key of 'bad key', value 'myvalue'",
			"bad key",
			"myvalue",
			true,
		),
		Entry("Should error with key of 'mykey', value 'bad value'",
			"mykey",
			"bad value",
			true,
		),
		Entry("Should not error with key of 'xyz/mykey', value 'myvalue'",
			"xyz/mykey",
			"myvalue",
			false,
		),
		Entry("Should error with key of '/mykey', value 'myvalue'",
			"/mykey",
			"myvalue",
			true,
		),
		Entry("Should error with key of '/', value 'myvalue'",
			"/",
			"myvalue",
			true,
		),
	)
})

var _ = Describe("Taint validations", func() {
	DescribeTable("Taint validation",
		func(key string, value string, hasError bool) {
			err := ValidateTaintKeyValuePair(key, value)
			if hasError {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("taint"))
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
		},
		Entry("Should not error with key of 'mykey', value 'myvalue'",
			"mykey",
			"myvalue",
			false,
		),
		Entry("Should error with key of 'bad key', value 'myvalue'",
			"bad key",
			"myvalue",
			true,
		),
		Entry("Should error with key of 'mykey', value 'bad value'",
			"mykey",
			"bad value",
			true,
		),
		Entry("Should not error with key of 'xyz/mykey', value 'myvalue'",
			"xyz/mykey",
			"myvalue",
			false,
		),
		Entry("Should error with key of '/mykey', value 'myvalue'",
			"/mykey",
			"myvalue",
			true,
		),
		Entry("Should error with key of '/', value 'myvalue'",
			"/",
			"myvalue",
			true,
		),
	)
})

var _ = Describe("Create node drain grace period builder validations", func() {
	DescribeTable("Create node drain grace period builder validations",
		func(period string, errMsg string) {
			_, err := CreateNodeDrainGracePeriodBuilder(period)
			if errMsg != "" {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errMsg))
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
		},
		Entry("Should not error with empty value",
			"",
			"",
		),
		Entry("Should not error with 0 value",
			"0",
			"",
		),
		Entry("Should not error with lower limit value",
			"1 minute",
			"",
		),
		Entry("Should not error with hour unit",
			"1 hour",
			"",
		),
		Entry("Should error if the time is not a numeric value",
			"hour",
			"Invalid time for the node drain grace period",
		),
	)
})

var _ = Describe("Validate node drain grace period", func() {
	DescribeTable("Validate node drain grace period",
		func(period interface{}, errMsg string) {
			err := ValidateNodeDrainGracePeriod(period)
			if errMsg != "" {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errMsg))
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
		},
		Entry("Should not error with empty value",
			"",
			"",
		),
		Entry("Should not error with 0 value",
			"0",
			"",
		),
		Entry("Should not error with lower limit value",
			"1 minute",
			"",
		),
		Entry("Should not error with upper limit value",
			"10080 minutes",
			"",
		),
		Entry("Should not error with hour unit",
			"1 hour",
			"",
		),
		Entry("Should not error with hours unit",
			"168 hours",
			"",
		),
		Entry("Should error with invalid number of tokens",
			"1 minute later",
			"Expected format to include the duration",
		),
		Entry("Should error with invalid unit",
			"1 day",
			"Invalid unit",
		),
		Entry("Should error with float value",
			"1.1",
			"duration must be an integer",
		),
		Entry("Should error with float value",
			"-1 minute",
			"cannot be negative",
		),
		Entry("Should error above upper limit minutes",
			"10081 minutes",
			"cannot exceed the maximum of 10080 minutes",
		),
		Entry("Should error above upper limit hours",
			"169 hours",
			"cannot exceed the maximum of 168 hours",
		),
	)
})

var _ = Describe("ValidateMachinePoolTaintEffect", func() {
	It("should return nil for recognized taint effects", func() {
		Expect(validateMachinePoolTaintEffect("key=value:NoSchedule")).To(Succeed())
		Expect(validateMachinePoolTaintEffect("key=value:NoExecute")).To(Succeed())
		Expect(validateMachinePoolTaintEffect("key=value:PreferNoSchedule")).To(Succeed())
	})

	It("should return an error for unrecognized taint effects", func() {
		Expect(validateMachinePoolTaintEffect("key=value:unrecognized")).To(MatchError(
			MatchRegexp("Invalid taint effect 'unrecognized', only the following" +
				" effects are supported: 'NoExecute', 'NoSchedule', 'PreferNoSchedule'")))
		Expect(validateMachinePoolTaintEffect("key=value:unrecognized:")).To(MatchError(
			MatchRegexp("Invalid taint format: 'key=value:unrecognized:'. Expected format" +
				" is '<key>=<value>:<effect>'")))
	})
})

var _ = Describe("Validate MaxSurge and MaxUnavailable", func() {
	DescribeTable("Validate MaxSurge and MaxUnavailable",
		func(value interface{}, errMsg string) {
			err := ValidateUpgradeMaxSurgeUnavailable(value)
			if errMsg != "" {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errMsg))
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
		},
		Entry("Should not error with empty value",
			"",
			"",
		),
		Entry("Should not error with 0 percent",
			"0%",
			"",
		),
		Entry("Should not error with 100 percent",
			"100%",
			"",
		),
		Entry("Should error with negative percentage",
			"-1%",
			"Percentage value -1 must be between 0 and 100",
		),
		Entry("Should error with 101% percent",
			"101%",
			"Percentage value 101 must be between 0 and 100",
		),
		Entry("Should error with non-integer percent",
			"1.1%",
			"Percentage value '1.1' must be an integer",
		),
		Entry("Should not error with 0",
			"0",
			"",
		),
		Entry("Should not error with positive integer",
			"1",
			"",
		),
		Entry("Should error with negative number",
			"-1",
			"Value -1 cannot be negative",
		),
		Entry("Should error with non-integer",
			"1.1",
			"Value '1.1' must be an integer",
		),
	)
})
