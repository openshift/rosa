package roles

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/rosa"
)

var _ = Describe("GetInstallerRoleArn", func() {
	var (
		runtime *rosa.Runtime
		cmd     *cobra.Command
	)

	BeforeEach(func() {
		runtime = rosa.NewRuntime()
		interactive.SetEnabled(false)
		cmd = &cobra.Command{
			Use: "test",
		}
		cmd.Flags().String("role-arn", "", "The installer role ARN")
		confirm.AddFlag(cmd.Flags())
		Expect(cmd.Flags().Set("yes", "false")).To(Succeed())
	})

	It("returns the single role ARN when exactly one role is found", func() {
		expectedArn := "arn:aws:iam::123456789012:role/my-Installer-Role"
		finder := func(roleType string, minVersion string) ([]string, error) {
			return []string{expectedArn}, nil
		}

		result := GetInstallerRoleArn(runtime, cmd, "", "", finder)
		Expect(result).To(Equal(expectedArn))
	})

	It("returns the found role ARN even when a default is provided", func() {
		defaultArn := "arn:aws:iam::123456789012:role/default-Installer-Role"
		foundArn := "arn:aws:iam::123456789012:role/found-Installer-Role"
		finder := func(roleType string, minVersion string) ([]string, error) {
			return []string{foundArn}, nil
		}

		result := GetInstallerRoleArn(runtime, cmd, defaultArn, "", finder)
		Expect(result).To(Equal(foundArn))
	})

	It("passes the correct role type and min version to findRoleARNs", func() {
		var capturedRoleType, capturedMinVersion string
		finder := func(roleType string, minVersion string) ([]string, error) {
			capturedRoleType = roleType
			capturedMinVersion = minVersion
			return []string{"arn:aws:iam::123456789012:role/test-Installer-Role"}, nil
		}

		GetInstallerRoleArn(runtime, cmd, "", "4.14", finder)
		Expect(capturedRoleType).To(Equal(aws.InstallerAccountRole))
		Expect(capturedMinVersion).To(Equal("4.14"))
	})

	It("prioritizes the role with default prefix when multiple roles and --yes", func() {
		role := aws.AccountRoles[aws.InstallerAccountRole]
		defaultPrefixArn := fmt.Sprintf(
			"arn:aws:iam::123456789012:role/%s-%s-Role", aws.DefaultPrefix, role.Name,
		)
		otherArn := "arn:aws:iam::123456789012:role/custom-Installer-Role"

		finder := func(roleType string, minVersion string) ([]string, error) {
			return []string{otherArn, defaultPrefixArn}, nil
		}

		Expect(cmd.Flags().Set("yes", "true")).To(Succeed())
		result := GetInstallerRoleArn(runtime, cmd, "", "", finder)
		Expect(result).To(Equal(defaultPrefixArn))
	})
})
