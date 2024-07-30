package machinepool

import (
	"github.com/AlecAivazis/survey/v2/core"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MachinePool validation", func() {
	Context("KubeletConfigs", func() {

		It("Fails if customer requests more than 1 kubelet config via []string", func() {
			kubeletConfigs := []string{"foo", "bar"}
			err := ValidateKubeletConfig(kubeletConfigs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Only a single kubelet config is supported for Machine Pools"))
		})

		It("Fails if customer requests more than 1 kubelet config via []core.OptionAnswer", func() {
			kubeletConfigs := []core.OptionAnswer{
				{
					Value: "foo",
					Index: 0,
				},
				{
					Value: "bar",
					Index: 1,
				},
			}
			err := ValidateKubeletConfig(kubeletConfigs)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Only a single kubelet config is supported for Machine Pools"))
		})

		It("Passes if a customer selects only a single kubelet config via []core.OptionAnswer", func() {
			kubeletConfigs := []core.OptionAnswer{
				{
					Value: "foo",
					Index: 0,
				},
			}
			err := ValidateKubeletConfig(kubeletConfigs)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Passes if a customer selects only a single kubelet config via []string", func() {
			kubeletConfigs := []string{"foo"}
			err := ValidateKubeletConfig(kubeletConfigs)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Passes with empty selection via []string", func() {
			err := ValidateKubeletConfig([]string{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("Passes with empty selection via []core.OptionAnswer", func() {
			err := ValidateKubeletConfig([]core.OptionAnswer{})
			Expect(err).NotTo(HaveOccurred())
		})

		It("Fails if the input is not a []string or []core.OptionAnswer", func() {
			err := ValidateKubeletConfig("foo")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Input for kubelet config flag is not valid"))
		})
	})
	Context("Validate edit machinepool options", func() {
		It("Fails with autoscaling + replicas set", func() {
			Expect(validateEditInput("machine", true, 1,
				2, 1, true, true, true,
				true, "test")).ToNot(Succeed())
		})
		It("Fails with max, min, and replicas < 0", func() {
			Expect(validateEditInput("machine", true, -1,
				1, 0, false, true, true,
				true, "test")).ToNot(Succeed())
			Expect(validateEditInput("machine", true, 1,
				-1, 0, false, true, true,
				true, "test")).ToNot(Succeed())
			Expect(validateEditInput("machine", false, 0,
				0, -1, true, true, false,
				false, "test")).ToNot(Succeed())
		})
		It("Fails with max < min replicas", func() {
			Expect(validateEditInput("machine", true, 5,
				4, 0, false, true, true,
				true, "test")).ToNot(Succeed())
		})
		It("Fails when no autoscaling, but min/max are set", func() {
			Expect(validateEditInput("machine", false, 1,
				0, 1, true, false, true,
				false, "test")).ToNot(Succeed())
			Expect(validateEditInput("machine", false, 0,
				1, 1, true, false, false,
				true, "test")).ToNot(Succeed())
			Expect(validateEditInput("machine", false, 1,
				1, 1, true, false, true,
				true, "test")).ToNot(Succeed())
		})
		It("Passes (autoscaling)", func() {
			Expect(validateEditInput("machine", true, 1,
				2, 0, false, true, true,
				true, "test")).To(Succeed())
		})
		It("Passes (not autoscaling)", func() {
			Expect(validateEditInput("machine", false, 0,
				0, 2, true, false, false,
				false, "test")).To(Succeed())
		})
	})
})
