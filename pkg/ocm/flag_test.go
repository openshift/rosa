package ocm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var _ = Describe("Cluster Flag", func() {
	It("Adds cluster flag as optional to a command", func() {
		cmd := &cobra.Command{}
		Expect(cmd.Flag(clusterFlagName)).To(BeNil())

		AddOptionalClusterFlag(cmd)
		AssertClusterFlag(cmd.Flag(clusterFlagName), false)
	})

	It("Adds cluster flag as required to a command", func() {
		cmd := &cobra.Command{}
		Expect(cmd.Flag(clusterFlagName)).To(BeNil())

		AddClusterFlag(cmd)
		AssertClusterFlag(cmd.Flag(clusterFlagName), true)
	})

})

func AssertClusterFlag(flag *flag.Flag, required bool) {
	Expect(flag).NotTo(BeNil())
	Expect(flag.Name).To(Equal(clusterFlagName))
	Expect(flag.Shorthand).To(Equal(clusterFlagShortHand))
	Expect(flag.Usage).To(Equal(clusterFlagDescription))

	if required {
		// The cobra.BashCompOneRequiredFlag annotation is how Cobra marks flags as required
		Expect(flag.Annotations).To(HaveKey(cobra.BashCompOneRequiredFlag))
	} else {
		Expect(flag.Annotations).NotTo(HaveKey(cobra.BashCompOneRequiredFlag))
	}
}
