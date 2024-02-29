package arguments

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

var _ = Describe("Normalize Argument Flags Tests", func() {
	When("when flags have been deprecated", func() {
		DescribeTable("should normalize flag names",
			func(flagName, expectedNormalized string) {
				f := pflag.NewFlagSet("test", pflag.ContinueOnError)
				normalized := NormalizeFlags(f, flagName)
				Expect(string(normalized)).To(Equal(expectedNormalized))
			},
			Entry("installer-role-arn to role-arn", deprecatedInstallerRoleArnFlag, newInstallerRoleArnFlag),
			Entry("default-mp-labels to worker-mp-labels", DeprecatedDefaultMPLabelsFlag, NewDefaultMPLabelsFlag),
			Entry("controlplane-iam-role to controlplane-iam-role-arn",
				DeprecatedControlPlaneIAMRole, NewControlPlaneIAMRole),
			Entry("worker-iam-role to worker-iam-role-arn", DeprecatedWorkerIAMRole, NewWorkerIAMRole),
		)
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
