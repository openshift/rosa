package roles

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/openshift/rosa/pkg/aws"
	. "github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
)

func GetInstallerRoleArn(r *rosa.Runtime, cmd *cobra.Command,
	defaultInstallerRoleArn string, minMinorVersion string) string {
	spin := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	spin.Start()
	awsClient := r.AWSClient
	role := aws.AccountRoles[aws.InstallerAccountRole]
	roleARN := defaultInstallerRoleArn
	// Find all installer roles in the current account using AWS resource tags
	roleARNs, err := awsClient.FindRoleARNs(aws.InstallerAccountRole, minMinorVersion)
	if err != nil {
		r.Reporter.Errorf("Failed to find %s role: %s", role.Name, err)
		os.Exit(1)
	}
	spin.Stop()

	if len(roleARNs) > 1 {
		defaultRoleARN := roleARNs[0]
		// Prioritize roles with the default prefix
		for _, rARN := range roleARNs {
			roleName, err := aws.GetResourceIdFromARN(rARN)
			if err != nil {
				continue
			}
			if roleName == fmt.Sprintf("%s-%s-Role", aws.DefaultPrefix, role.Name) {
				defaultRoleARN = rARN
			}
		}
		r.Reporter.Warnf("More than one %s role found", role.Name)
		if !Enabled() && confirm.Yes() {
			roleARN = defaultRoleARN
		} else {
			if roleARN != "" {
				defaultRoleARN = roleARN
			}
			roleARN, err = GetOption(Input{
				Question: fmt.Sprintf("%s role ARN", role.Name),
				Help:     cmd.Flags().Lookup("installer-role-arn").Usage,
				Options:  roleARNs,
				Default:  defaultRoleARN,
				Required: true,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid role ARN: %s", err)
				os.Exit(1)
			}
		}
	} else if len(roleARNs) == 1 {
		if !output.HasFlag() || r.Reporter.IsTerminal() {
			r.Reporter.Infof("Using %s for the %s role", roleARNs[0], role.Name)
		}
		roleARN = roleARNs[0]
	} else {
		createAccountRolesCommand := "rosa create account-roles"
		r.Reporter.Warnf(fmt.Sprintf("No account roles found. You will need to manually set them in the "+
			"next steps or run '%s' to create them first.", createAccountRolesCommand))
		os.Exit(1)
	}
	return roleARN
}
