package arguments

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

var _ = Describe("Normalize Argument Flags Tests", func() {
	When("when flags have been deprecated", func() {
		It("should normalize --installer-role-arn to --role-arn", func() {
			f := pflag.NewFlagSet("test", pflag.ContinueOnError)
			flagName := deprecatedInstallerRoleArnFlag

			normalized := NormalizeFlags(f, flagName)

			Expect(string(normalized)).To(Equal(newInstallerRoleArnFlag))
		})
	})

	When("Flags that have not been deprecated", func() {
		It("should return unchanged flag name if not deprecatedInstallerRoleArnFlag", func() {
			f := pflag.NewFlagSet("test", pflag.ContinueOnError)
			flagName := "some_other_flag"

			normalized := NormalizeFlags(f, flagName)

			Expect(string(normalized)).To(Equal(flagName))
		})
	})
})
