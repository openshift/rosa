package kubeletconfig

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

var _ = Describe("KubeletConfigOptions", func() {

	It("Adds flags to command", func() {
		cmd := &cobra.Command{}
		flags := cmd.Flags()
		Expect(flags).NotTo(BeNil())
		Expect(flags.Lookup(PodPidsLimitOption)).To(BeNil())
		Expect(flags.Lookup(NameOption)).To(BeNil())

		options := NewKubeletConfigOptions()
		options.AddFlagsToCommand(cmd)

		flag := flags.Lookup(PodPidsLimitOption)
		assertFlag(flag, PodPidsLimitOption, PodPidsLimitOptionUsage)

		flag = flags.Lookup(NameOption)
		assertFlag(flag, NameOption, NameOptionUsage)
	})
})

func assertFlag(flag *flag.Flag, name string, usage string) {
	Expect(flag).NotTo(BeNil())
	Expect(flag.Name).To(Equal(name))
	Expect(flag.Usage).To(Equal(usage))
}
