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
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	prefix                      string
	isInvokedFromClusterUpgrade bool
	clusterID                   string
	version                     string
	channelGroup                string
}

var Cmd = &cobra.Command{
	Use:     "account-roles",
	Aliases: []string{"accountroles", "roles", "policies"},
	Short:   "Upgrade account-wide IAM roles to the latest version.",
	Long:    "Upgrade account-wide IAM roles to the latest version before upgrading your cluster.",
	Example: `  # Upgrade account roles for ROSA STS clusters
  rosa upgrade account-roles`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	aws.AddModeFlag(Cmd)

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
		"Version of OpenShift that will be used to install the cluster, for example \"4.3.10\"",
	)
	flags.MarkHidden("version")

	flags.StringVar(
		&args.channelGroup,
		"channel-group",
		ocm.DefaultChannelGroup,
		"Channel group is the name of the group where this image belongs, for example \"stable\" or \"fast\".",
	)
	flags.MarkHidden("channel-group")

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) error {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.NewLogger()

	isInvokedFromClusterUpgrade := false
	skipInteractive := false
	if len(argv) >= 2 && !cmd.Flag("prefix").Changed {
		args.prefix = argv[0]
		aws.SetModeKey(argv[1])
		if argv[1] != "" {
			skipInteractive = true
		}
		if len(argv) > 2 && argv[2] != "" {
			args.clusterID = argv[2]
		}
		isInvokedFromClusterUpgrade = true
	}
	args.isInvokedFromClusterUpgrade = isInvokedFromClusterUpgrade
	mode, err := aws.GetMode()
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}
	prefix := args.prefix
	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}

	// Create the client for the OCM API:
	ocmClient, err := ocm.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	version := args.version
	channelGroup := args.channelGroup
	policyVersion, err := ocmClient.GetVersion(version, channelGroup)
	if err != nil {
		reporter.Errorf("Error getting version: %s", err)
		os.Exit(1)
	}

	env, err := ocm.GetEnv()
	if err != nil {
		reporter.Errorf("Failed to determine OCM environment: %v", err)
		os.Exit(1)
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
	if !args.isInvokedFromClusterUpgrade {
		reporter.Infof("Ensuring account role policies compatibility for upgrade")
	}

	isUpgradeNeedForAccountRolePolicies, err := awsClient.IsUpgradedNeededForAccountRolePolicies(prefix,
		policyVersion)
	if err != nil {
		reporter.Errorf("%s", err)
		LogError("ROSAUpgradeAccountRolesModeAuto", ocmClient, policyVersion, err, reporter)
		os.Exit(1)
	}

	if spin != nil {
		spin.Stop()
	}

	if !isUpgradeNeedForAccountRolePolicies {
		if args.isInvokedFromClusterUpgrade {
			return nil
		}
		reporter.Infof("Account role with the prefix '%s' is already up-to-date.", prefix)
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
		mode, err = interactive.GetOption(interactive.Input{
			Question: "Account role upgrade mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  aws.ModeAuto,
			Options:  aws.Modes,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid Account role upgrade mode: %s", err)
			os.Exit(1)
		}
		aws.SetModeKey(mode)
	}
	policies, err := ocmClient.GetPolicies("")
	if err != nil {
		reporter.Errorf("Expected a valid role creation mode: %s", err)
		os.Exit(1)
	}

	switch mode {
	case aws.ModeAuto:
		reporter.Infof("Starting to upgrade the policies")
		if isUpgradeNeedForAccountRolePolicies {
			err = upgradeAccountRolePolicies(reporter, awsClient, prefix, creator.AccountID, policies,
				policyVersion, policyPath)
			if err != nil {
				LogError("ROSAUpgradeAccountRolesModeAuto", ocmClient, policyVersion, err, reporter)
				if args.isInvokedFromClusterUpgrade {
					return err
				}
				reporter.Errorf("Error upgrading the role polices: %s", err)
				os.Exit(1)
			}
		}
	case aws.ModeManual:
		err = aws.GeneratePolicyFiles(reporter, env, isUpgradeNeedForAccountRolePolicies,
			false, policies, nil)
		if err != nil {
			reporter.Errorf("There was an error generating the policy files: %s", err)
			os.Exit(1)
		}
		if reporter.IsTerminal() {
			reporter.Infof("All policy files saved to the current directory")
			reporter.Infof("Run the following commands to upgrade the account role policies:\n")
		}
		commands := buildCommands(prefix, creator.AccountID, isUpgradeNeedForAccountRolePolicies,
			awsClient, policyVersion, policyPath)
		fmt.Println(commands)
		if args.isInvokedFromClusterUpgrade {
			reporter.Infof("Run the following command to continue scheduling cluster upgrade"+
				" once account and operator roles have been upgraded : \n\n"+
				"\trosa upgrade cluster --cluster %s\n", args.clusterID)
			os.Exit(0)
		}

	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
	return err
}

func LogError(key string, ocmClient *ocm.Client, defaultPolicyVersion string, err error, reporter *rprtr.Object) {
	reporter.Debugf("Logging throttle error")
	if strings.Contains(err.Error(), "Throttling") {
		ocmClient.LogEvent(key, map[string]string{
			ocm.Response:   ocm.Failure,
			ocm.Version:    defaultPolicyVersion,
			ocm.IsThrottle: "true",
		})
	}
}

func upgradeAccountRolePolicies(reporter *rprtr.Object, awsClient aws.Client, prefix string, accountID string,
	policies map[string]string, defaultPolicyVersion string, policyPath string) error {
	for file, role := range aws.AccountRoles {
		roleName := aws.GetRoleName(prefix, role.Name)
		if !confirm.Prompt(true, "Upgrade the '%s' role policy latest version ?", roleName) {
			if args.isInvokedFromClusterUpgrade {
				return reporter.Errorf("Account roles need to be upgraded to proceed" +
					"")
			}
			continue
		}
		filename := fmt.Sprintf("sts_%s_permission_policy", file)
		policyARN := aws.GetPolicyARN(accountID, fmt.Sprintf("%s-Policy", roleName), policyPath)

		policyDetails := policies[filename]
		policyARN, err := awsClient.EnsurePolicy(policyARN, policyDetails,
			defaultPolicyVersion, map[string]string{
				tags.OpenShiftVersion: defaultPolicyVersion,
				tags.RolePrefix:       prefix,
				tags.RoleType:         file,
				tags.RedHatManaged:    "true",
			}, policyPath)
		if err != nil {
			return err
		}

		err = awsClient.AttachRolePolicy(roleName, policyARN)
		if err != nil {
			return err
		}
		//Delete if present else continue
		err = awsClient.DeleteInlineRolePolicies(roleName)
		if err != nil {
			reporter.Debugf("Error deleting inline role policy %s : %s", policyARN, err)
		}
		reporter.Infof("Upgraded policy with ARN '%s' to latest version", policyARN)
		err = awsClient.UpdateTag(roleName, defaultPolicyVersion)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildCommands(prefix string, accountID string, isUpgradeNeedForAccountRolePolicies bool,
	awsClient aws.Client, defaultPolicyVersion string, policyPath string) string {
	commands := []string{}
	if isUpgradeNeedForAccountRolePolicies {
		for file, role := range aws.AccountRoles {
			name := aws.GetRoleName(prefix, role.Name)
			policyARN := aws.GetPolicyARN(accountID, fmt.Sprintf("%s-Policy", name), policyPath)
			policyTags := fmt.Sprintf(
				"Key=%s,Value=%s",
				tags.OpenShiftVersion, defaultPolicyVersion,
			)
			iamRoleTags := fmt.Sprintf(
				"Key=%s,Value=%s",
				tags.OpenShiftVersion, defaultPolicyVersion)
			tagRole := fmt.Sprintf("aws iam tag-role \\\n"+
				"\t--tags %s \\\n"+
				"\t--role-name %s",
				iamRoleTags, name)
			//check if the policy exists if not output create and attach
			_, err := awsClient.IsPolicyExists(policyARN)
			if err != nil {
				policyName := fmt.Sprintf("%s-Policy", name)
				iamTags := fmt.Sprintf(
					"Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s",
					tags.OpenShiftVersion, defaultPolicyVersion,
					tags.RolePrefix, prefix,
					tags.RoleType, file,
				)
				createPolicy := fmt.Sprintf("aws iam create-policy \\\n"+
					"\t--policy-name %s \\\n"+
					"\t--policy-document file://sts_%s_permission_policy.json"+
					"\t--tags %s"+
					"\t--path %s",
					policyName, file, iamTags, policyPath)
				attachRolePolicy := fmt.Sprintf("aws iam attach-role-policy \\\n"+
					"\t--role-name %s \\\n"+
					"\t--policy-arn %s",
					name, aws.GetPolicyARN(accountID, policyName, policyPath))

				_, _ = awsClient.IsRolePolicyExists(name, policyName)
				if err != nil {
					commands = append(commands, createPolicy, attachRolePolicy, tagRole)
				} else {
					deletePolicy := fmt.Sprintf("\taws iam delete-role-policy --role-name  %s  --policy-name  %s",
						name, policyName)
					commands = append(commands, createPolicy, attachRolePolicy, tagRole, deletePolicy)
				}
			} else {
				createPolicyVersion := fmt.Sprintf("aws iam create-policy-version \\\n"+
					"\t--policy-arn %s \\\n"+
					"\t--policy-document file://sts_%s_permission_policy.json \\\n"+
					"\t--set-as-default",
					policyARN, file)
				tagPolicies := fmt.Sprintf("aws iam tag-policy \\\n"+
					"\t--tags %s \\\n"+
					"\t--policy-arn %s",
					policyTags, policyARN)
				commands = append(commands, createPolicyVersion, tagPolicies, tagRole)
			}
		}
	}
	return strings.Join(commands, "\n\n")
}

func getAccountPolicyPath(awsClient aws.Client, prefix string) (string, error) {
	for _, accountRole := range aws.AccountRoles {
		roleName := aws.GetRoleName(prefix, accountRole.Name)
		rolePolicies, err := awsClient.GetAttachedPolicy(&roleName)
		if err != nil {
			return "", err
		}
		policyName := fmt.Sprintf("%s-Policy", roleName)
		policyARN := ""
		for _, rolePolicy := range rolePolicies {
			if rolePolicy.PolicyName == policyName {
				policyARN = rolePolicy.PolicyArn
				break
			}
		}
		if policyARN != "" {
			return aws.GetPathFromARN(policyARN)
		}
	}
	return "", fmt.Errorf("Could not find account policies that are attached to account roles." +
		"We need at least one in order to detect account policies path")
}
