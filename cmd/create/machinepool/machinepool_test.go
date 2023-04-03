package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MachinePool", func() {
	DescribeTable("ParseTaints validation",
		func(taint, expectedError string, numberOfTaints int) {
			taints, err := parseTaints(taint)
			if expectedError == "" {
				Expect(err).ToNot(HaveOccurred())
				Expect(len(taints)).To(Equal(numberOfTaints))
			} else {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expectedError))
			}
		},
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
			"Invalid label value 'node-role.kubernetes.io/infra': at key: 'key'", 0),
	)
})
