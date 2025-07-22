package kubeletconfig

import (
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("KubeletConfig Config", func() {

	Context("GetMaxPidsLimit", func() {

		var ctrl *gomock.Controller
		var capabilityChecker *MockCapabilityChecker

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			capabilityChecker = NewMockCapabilityChecker(ctrl)
		})

		It("Returns Correct Max Pids Limit When Org Has Capability", func() {
			capabilityChecker.EXPECT().IsCapabilityEnabled(ByPassPidsLimitCapability).Return(true, nil)

			max, err := GetMaxPidsLimit(capabilityChecker)
			Expect(err).NotTo(HaveOccurred())
			Expect(max).To(Equal(MaxUnsafePodPidsLimit))
		})

		It("Returns Correct Max Pids Limit When Org Does Not Have Capability", func() {
			capabilityChecker.EXPECT().IsCapabilityEnabled(ByPassPidsLimitCapability).Return(false, nil)

			max, err := GetMaxPidsLimit(capabilityChecker)
			Expect(err).NotTo(HaveOccurred())
			Expect(max).To(Equal(MaxPodPidsLimit))
		})
	})

	Context("GetInteractiveMaxPidsLimitHelp", func() {
		It("Correctly generates the Max Pids Limit Interactive Help", func() {
			help := GetInteractiveMaxPidsLimitHelp(5000)
			Expect(help).To(Equal("Set the Pod Pids Limit field to a value between 4096 and 5000"))
		})
	})

	Context("GetInteractiveInput", func() {
		It("Correctly generates Interactive Input for pre-existing KubeletConfig", func() {

			builder := v1.KubeletConfigBuilder{}
			kubeletConfig, err := builder.PodPidsLimit(10000).Build()

			Expect(err).NotTo(HaveOccurred())

			input := GetInteractiveInput(5000, kubeletConfig)
			Expect(input.Required).To(BeTrue())
			Expect(input.Question).To(Equal(InteractivePodPidsLimitPrompt))
			Expect(input.Help).To(Equal(GetInteractiveMaxPidsLimitHelp(5000)))
			Expect(len(input.Validators)).To(Equal(2))
			Expect(input.Default).To(Equal(kubeletConfig.PodPidsLimit()))
		})

		It("Correctly generates Interactive Input for new KubeletConfig", func() {
			input := GetInteractiveInput(5000, nil)
			Expect(input.Required).To(BeTrue())
			Expect(input.Question).To(Equal(InteractivePodPidsLimitPrompt))
			Expect(input.Help).To(Equal(GetInteractiveMaxPidsLimitHelp(5000)))
			Expect(len(input.Validators)).To(Equal(2))
			Expect(input.Default).To(Equal(MinPodPidsLimit))
		})
	})
})
