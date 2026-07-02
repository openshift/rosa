package machinepool

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	mpOpts "github.com/openshift/rosa/pkg/options/machinepool"
)

var _ = Describe("buildMachinePoolImageTypeInput", func() {
	It("builds a clearer hosted image type prompt", func() {
		cmd, _ := mpOpts.BuildMachinePoolCreateCommandWithOptions()

		input := buildMachinePoolImageTypeInput(cmd)

		Expect(input.Question).To(Equal("Machine pool image type"))
		Expect(input.Help).To(Equal(cmd.Flags().Lookup("type").Usage))
		Expect(input.Default).To(Equal(cmv1.ImageTypeDefault))
		Expect(input.Required).To(BeFalse())
		Expect(input.Options).To(Equal(mpHelpers.ImageTypes))
	})
})
