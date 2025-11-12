/*
Copyright (c) 2021 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package accountroles

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	common "github.com/openshift-online/ocm-common/pkg/aws/validations"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	awscbRoles "github.com/openshift/rosa/pkg/aws/commandbuilder/helper/roles"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/helper/roles"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	prefix       string
	version      string
	channelGroup string
	hostedCP     bool
}

var Cmd = &cobra.Command{
	Use:     "account-roles",
	Aliases: []string{"account-role", "accountroles", "policies"},
	Short:   "Upgrade account-wide IAM roles to the latest version.",
	Long:    "Upgrade account-wide IAM roles to the latest version before upgrading your cluster.",
	Example: `  # Upgrade account roles for ROSA STS clusters
  rosa upgrade account-roles`,
	Args: cobra.NoArgs,
	Run:  run,
}

func init() {
	flags := Cmd.Flags()

	interactive.AddModeFlag(Cmd)

	flags.StringVarP(
		&args.prefix,
		"prefix",
		"p",
		"",
		"User-defined prefix for all generated AWS resources",
	)
	Cmd.MarkFlagRequired("prefix")

	flags.StringVar(
		&args.version,
		"version",
		"",
		"Version of OpenShift that will be used to setup policy tag, for example \"4.11\"",
	)

	flags.StringVar(
		&args.channelGroup,
		"channel-group",
		ocm.DefaultChannelGroup,
		"Channel group is the name of the channel where this image belongs, for example \"stable\" or \"fast\".",
	)
	flags.MarkHidden("channel-group")

	flags.BoolVar(
		&args.hostedCP,
		"hosted-cp",
		false,
		"Enable the use of Hosted Control Planes",
	)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	reporter := r.Reporter
	awsClient := r.AWSClient
	ocmClient := r.OCMClient

	skipInteractive := false
	mode, err := interactive.GetMode()
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}
	prefix := args.prefix

	version := args.version
	isVersionChosen := version != ""
	channelGroup := args.channelGroup
	policyVersion, err := ocmClient.GetPolicyVersion(version, channelGroup)
	if err != nil {
		reporter.Errorf("Error getting version: %s", err)
		os.Exit(1)
	}

	env, err := ocm.GetEnv()
	if err != nil {
		reporter.Errorf("Failed to determine OCM environment: %v", err)
		os.Exit(1)
	}

	var role aws.AccountRole
	if args.hostedCP {
		role = aws.HCPAccountRoles[aws.InstallerAccountRole]
	} else {
		role = aws.AccountRoles[aws.InstallerAccountRole]
	}

	roleARN, err := awsClient.GetAccountRoleARN(prefix, role.Name)
	if err != nil {
		if args.hostedCP {
			r.Reporter.Errorf("Failed to get hosted CP account roles ARN: %v. "+
				"To upgrade classic account roles run the command without the '--hosted-cp' flag", err)
		} else {
			r.Reporter.Errorf("Failed to get classic account roles ARN: %v. "+
				"To upgrade hosted CP account roles use the '--hosted-cp' flag", err)
		}
		os.Exit(1)
	}

	managedPolicies, err := awsClient.HasManagedPolicies(roleARN)
	if err != nil {
		r.Reporter.Errorf("Failed to determine if the role has managed policies: %v", err)
		os.Exit(1)
	}

	if managedPolicies {
		hostedCPPolicies, err := awsClient.HasHostedCPPolicies(roleARN)
		if err != nil {
			r.Reporter.Errorf("Failed to determine if the role has hosted CP managed policies: %v", err)
			os.Exit(1)
		}

		if hostedCPPolicies && !args.hostedCP {
			r.Reporter.Errorf("Role with ARN '%s' has hosted CP managed policies, "+
				"please run the command with the flag '--hosted-cp'", roleARN)
			os.Exit(1)
		}

		err = roles.ValidateAccountRolesManagedPolicies(r, prefix, hostedCPPolicies)
		if err != nil {
			r.Reporter.Errorf("Failed while validating managed policies: %v", err)
			os.Exit(1)
		}

		r.Reporter.Infof("Account roles with the prefix '%s' have attached managed policies. "+
			"An upgrade isn't needed", prefix)
		return
	}

	creator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get IAM credentials: %s", err)
		os.Exit(1)
	}

	var spin *spinner.Spinner
	if reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		spin.Start()
	}
	reporter.Infof("Ensuring account role policies compatibility for upgrade")

	isUpgradeNeedForAccountRolePolicies, err := awsClient.IsUpgradedNeededForAccountRolePolicies(prefix, policyVersion)
	if err != nil {
		reporter.Errorf("%s", err)
		LogError(roles.RosaUpgradeAccRolesModeAuto, ocmClient, policyVersion, err, reporter)
		os.Exit(1)
	}

	if spin != nil {
		spin.Stop()
	}

	if !isUpgradeNeedForAccountRolePolicies {
		reporter.Infof("Account roles with the prefix '%s' are already up-to-date.", prefix)
		os.Exit(0)
	}

	policyPath, err := getAccountPolicyPath(awsClient, prefix)
	if err != nil {
		reporter.Errorf("Error trying to determine the path for the account policies. Error: %v", err)
		os.Exit(1)
	}

	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
		interactive.Enable()
	}

	if interactive.Enabled() && !skipInteractive {
		mode, err = interactive.GetOptionMode(cmd, mode, "Account role upgrade mode")
		if err != nil {
			reporter.Errorf("Expected a valid Account role upgrade mode: %s", err)
			os.Exit(1)
		}
		interactive.SetModeKey(mode)
	}
	policies, err := ocmClient.GetPolicies("")
	if err != nil {
		reporter.Errorf("Expected a valid role creation mode: %s", err)
		os.Exit(1)
	}

	switch mode {
	case interactive.ModeAuto:
		if isUpgradeNeedForAccountRolePolicies {
			reporter.Infof("Starting to upgrade the policies")
			err := upgradeAccountRolePolicies(reporter, awsClient, prefix, creator.Partition, creator.AccountID, policies,
				policyVersion, policyPath, isVersionChosen)
			if err != nil {
				LogError(roles.RosaUpgradeAccRolesModeAuto, ocmClient, policyVersion, err, reporter)
				reporter.Errorf("Error upgrading the role polices: %s", err)
				os.Exit(1)
			}
		}
	case interactive.ModeManual:
		if isUpgradeNeedForAccountRolePolicies {
			err = aws.GenerateAccountRolePolicyFiles(reporter, env, policies, false, aws.AccountRoles, creator.Partition)
			if err != nil {
				reporter.Errorf("There was an error generating the policy files: %s", err)
				os.Exit(1)
			}
		}
		if reporter.IsTerminal() {
			reporter.Infof("All policy files saved to the current directory")
			reporter.Infof("Run the following commands to upgrade the account role policies:\n")
		}

		commands := buildCommands(prefix, creator.Partition, creator.AccountID, isUpgradeNeedForAccountRolePolicies,
			awsClient, policyVersion, policyPath)
		fmt.Println(commands)

	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", interactive.Modes)
		os.Exit(1)
	}
}

func LogError(key string, ocmClient *ocm.Client, defaultPolicyVersion string, err error, reporter reporter.Logger) {
	reporter.Debugf("Logging throttle error")
	if strings.Contains(err.Error(), "Throttling") {
		ocmClient.LogEvent(key, map[string]string{
			ocm.Response:   ocm.Failure,
			ocm.Version:    defaultPolicyVersion,
			ocm.IsThrottle: "true",
		})
	}
}

func upgradeAccountRolePolicies(reporter reporter.Logger, awsClient aws.Client, prefix string, partition string,
	accountID string, policies map[string]*cmv1.AWSSTSPolicy, policyVersion string,
	policyPath string, isVersionChosen bool) error {
	for file, role := range aws.AccountRoles {
		roleName := common.GetRoleName(prefix, role.Name)
		promptString := fmt.Sprintf("Upgrade the '%s' role policy latest version ?", roleName)
		if isVersionChosen {
			promptString = fmt.Sprintf("Upgrade the '%s' role policy to version '%s' ?", roleName, policyVersion)
		}
		if !confirm.Prompt(true, "%s", promptString) {
			continue
		}
		filename := fmt.Sprintf("sts_%s_permission_policy", file)
		policyARN := aws.GetPolicyArnWithSuffix(partition, accountID, roleName, policyPath)

		policyDetails := aws.GetPolicyDetails(policies, filename)
		policyARN, err := awsClient.EnsurePolicy(policyARN, policyDetails,
			policyVersion, map[string]string{
				common.OpenShiftVersion: policyVersion,
				tags.RolePrefix:         prefix,
				tags.RoleType:           file,
				tags.RedHatManaged:      "true",
			}, policyPath)
		if err != nil {
			return err
		}

		err = awsClient.AttachRolePolicy(reporter, roleName, policyARN)
		if err != nil {
			return err
		}
		//Delete if present else continue
		err = awsClient.DeleteInlineRolePolicies(roleName)
		if err != nil {
			reporter.Debugf("Error deleting inline role policy %s : %s", policyARN, err)
		}
		reporterString := fmt.Sprintf("Upgraded policy with ARN '%s' to latest version", policyARN)
		if isVersionChosen {
			reporterString = fmt.Sprintf("Upgraded policy with ARN '%s' to version '%s'", policyARN, policyVersion)
		}
		reporter.Infof(reporterString)
		err = awsClient.UpdateTag(roleName, policyVersion)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildCommands(prefix string, partition string, accountID string, isUpgradeNeedForAccountRolePolicies bool,
	awsClient aws.Client, defaultPolicyVersion string, policyPath string) string {
	commands := []string{}
	if isUpgradeNeedForAccountRolePolicies {
		for file, role := range aws.AccountRoles {
			accRoleName := common.GetRoleName(prefix, role.Name)
			policyARN := aws.GetPolicyArnWithSuffix(partition, accountID, accRoleName, policyPath)
			_, err := awsClient.IsPolicyExists(policyARN)
			policyExists := err == nil
			policyName := aws.GetPolicyName(accRoleName)
			upgradeAccountPolicyCommands := awscbRoles.ManualCommandsForUpgradeAccountRolePolicy(
				awscbRoles.ManualCommandsForUpgradeAccountRolePolicyInput{
					DefaultPolicyVersion: defaultPolicyVersion,
					RoleName:             accRoleName,
					PolicyExists:         policyExists,
					Prefix:               prefix,
					File:                 file,
					PolicyName:           policyName,
					AccountPolicyPath:    policyPath,
					PolicyARN:            policyARN,
				},
			)
			commands = append(commands, upgradeAccountPolicyCommands...)
		}
	}
	return awscb.JoinCommands(commands)
}

func getAccountPolicyPath(awsClient aws.Client, prefix string) (string, error) {
	for _, accountRole := range aws.AccountRoles {
		accRoleName := common.GetRoleName(prefix, accountRole.Name)
		rolePolicies, err := awsClient.GetAttachedPolicy(&accRoleName)
		if err != nil {
			return "", err
		}
		policyName := aws.GetPolicyName(accRoleName)
		policyARN := ""
		policyType := ""
		for _, rolePolicy := range rolePolicies {
			if rolePolicy.PolicyName == policyName {
				policyARN = rolePolicy.PolicyArn
				policyType = rolePolicy.PolicyType
				break
			}
		}
		if policyARN != "" {
			return aws.GetPathFromARN(policyARN)
		}
		// Compatibility with old ROSA CLI
		if policyType == aws.Inline {
			return awsClient.GetRoleARNPath(prefix)
		}
	}
	return "", fmt.Errorf("could not find account policies that are attached to account roles." +
		"We need at least one in order to detect account policies path")
}
