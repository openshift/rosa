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
	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) error {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	isInvokedFromClusterUpgrade := false
	skipInteractive := false
	if len(argv) == 2 && !cmd.Flag("prefix").Changed {
		args.prefix = argv[0]
		aws.SetModeKey(argv[1])

		if argv[1] != "" {
			skipInteractive = true
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

	isUpgradeNeedForAccountRolePolicies, err := awsClient.IsUpgradedNeededForAccountRolePolicies(prefix,
		aws.DefaultPolicyVersion)
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	isUpgradeNeedForOperatorRolePolicies, err := awsClient.IsUpgradedNeededForOperatorRolePoliciesUsingPrefix(prefix,
		creator.AccountID,
		aws.DefaultPolicyVersion)
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if !isUpgradeNeedForAccountRolePolicies && !isUpgradeNeedForOperatorRolePolicies {
		reporter.Infof("Account role with the prefix '%s' is already up-to-date.", prefix)
		os.Exit(0)
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
	}
	switch mode {
	case aws.ModeAuto:
		reporter.Infof("Starting to upgrade the policies")
		if isUpgradeNeedForAccountRolePolicies {
			err = upgradeAccountRolePolicies(reporter, awsClient, prefix, creator.AccountID)
			if err != nil {
				if args.isInvokedFromClusterUpgrade {
					return err
				}
				reporter.Errorf("Error upgrading the role polices: %s", err)
				os.Exit(1)
			}
		}
		if isUpgradeNeedForOperatorRolePolicies {
			err = upgradeOperatorRolePolicies(reporter, awsClient, creator.AccountID, prefix)
			if err != nil {
				if args.isInvokedFromClusterUpgrade {
					return err
				}
				reporter.Errorf("Error upgrading the operator role polices: %s", err)
				os.Exit(1)
			}
		}
	case aws.ModeManual:
		err = aws.GeneratePolicyFiles(reporter, env, isUpgradeNeedForAccountRolePolicies,
			isUpgradeNeedForOperatorRolePolicies)
		if err != nil {
			reporter.Errorf("There was an error generating the policy files: %s", err)
			os.Exit(1)
		}
		if reporter.IsTerminal() {
			reporter.Infof("All policy files saved to the current directory")
			reporter.Infof("Run the following commands to upgrade the account role policies:\n")
		}
		commands := buildCommands(prefix, creator.AccountID, isUpgradeNeedForAccountRolePolicies,
			isUpgradeNeedForOperatorRolePolicies)
		fmt.Println(commands)
	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
	return err
}

func upgradeAccountRolePolicies(reporter *rprtr.Object, awsClient aws.Client, prefix string, accountID string) error {
	for file, role := range aws.AccountRoles {
		name := aws.GetRoleName(prefix, role.Name)
		if !confirm.Prompt(true, "Upgrade the '%s' role polices to version %s?", name,
			aws.DefaultPolicyVersion) {
			if args.isInvokedFromClusterUpgrade {
				return reporter.Errorf("Account roles need to be upgraded to proceed" +
					"")
			}
			continue
		}
		filename := fmt.Sprintf("sts_%s_permission_policy.json", file)
		path := fmt.Sprintf("templates/policies/%s", filename)
		policyARN := aws.GetPolicyARN(accountID, fmt.Sprintf("%s-Policy", name))

		policy, err := aws.ReadPolicyDocument(path)
		if err != nil {
			return err
		}
		policyARN, err = awsClient.EnsurePolicy(policyARN, string(policy),
			aws.DefaultPolicyVersion, map[string]string{
				tags.OpenShiftVersion: aws.DefaultPolicyVersion,
				tags.RolePrefix:       prefix,
				tags.RoleType:         file,
			})
		if err != nil {
			return err
		}
		reporter.Infof("Upgraded policy with ARN '%s' to version '%s'", policyARN, aws.DefaultPolicyVersion)
		err = awsClient.UpdateTag(name)
		if err != nil {
			return err
		}
	}
	return nil
}

func upgradeOperatorRolePolicies(reporter *rprtr.Object, awsClient aws.Client, accountID string, prefix string) error {
	if !confirm.Prompt(true, "Upgrade the operator role policy to version %s?", aws.DefaultPolicyVersion) {
		if args.isInvokedFromClusterUpgrade {
			return reporter.Errorf("Operator roles need to be upgraded to proceed" +
				"")
		}
		return nil
	}
	for credrequest, operator := range aws.CredentialRequests {
		policyARN := aws.GetOperatorPolicyARN(accountID, prefix, operator.Namespace, operator.Name)
		filename := fmt.Sprintf("openshift_%s_policy.json", credrequest)
		path := fmt.Sprintf("templates/policies/%s", filename)

		policy, err := aws.ReadPolicyDocument(path)
		if err != nil {
			return err
		}
		policyARN, err = awsClient.EnsurePolicy(policyARN, string(policy),
			aws.DefaultPolicyVersion, map[string]string{
				tags.OpenShiftVersion: aws.DefaultPolicyVersion,
				tags.RolePrefix:       prefix,
				"operator_namespace":  operator.Namespace,
				"operator_name":       operator.Name,
			})
		if err != nil {
			return err
		}
		reporter.Infof("Upgraded policy with ARN '%s' to version '%s'", policyARN, aws.DefaultPolicyVersion)
	}
	return nil
}

func buildCommands(prefix string, accountID string, isUpgradeNeedForAccountRolePolicies bool,
	isUpgradeNeedForOperatorRolePolicies bool) string {
	commands := []string{}
	if isUpgradeNeedForAccountRolePolicies {
		for file, role := range aws.AccountRoles {
			name := aws.GetRoleName(prefix, role.Name)
			policyARN := aws.GetPolicyARN(accountID, fmt.Sprintf("%s-Policy", name))
			policyTags := fmt.Sprintf(
				"Key=%s,Value=%s",
				tags.OpenShiftVersion, aws.DefaultPolicyVersion,
			)
			createPolicyVersion := fmt.Sprintf("aws iam create-policy-version \\\n"+
				"\t--policy-arn %s \\\n"+
				"\t--policy-document file://sts_%s_permission_policy.json \\\n"+
				"\t--set-as-default",
				policyARN, file)
			tagPolicies := fmt.Sprintf("aws iam tag-policy \\\n"+
				"\t--tags %s \\\n"+
				"\t--policy-arn %s",
				policyTags, policyARN)
			iamRoleTags := fmt.Sprintf(
				"Key=%s,Value=%s",
				tags.OpenShiftVersion, aws.DefaultPolicyVersion)
			tagRole := fmt.Sprintf("aws iam tag-role \\\n"+
				"\t--tags %s \\\n"+
				"\t--role-name %s",
				iamRoleTags, name)
			commands = append(commands, createPolicyVersion, tagPolicies, tagRole)
		}
	}
	if isUpgradeNeedForOperatorRolePolicies {
		for credrequest, operator := range aws.CredentialRequests {
			policyARN := aws.GetOperatorPolicyARN(accountID, prefix, operator.Namespace, operator.Name)
			policTags := fmt.Sprintf(
				"Key=%s,Value=%s",
				tags.OpenShiftVersion, aws.DefaultPolicyVersion,
			)
			createPolicy := fmt.Sprintf("aws iam create-policy-version \\\n"+
				"\t--policy-arn %s \\\n"+
				"\t--policy-document file://openshift_%s_policy.json \\\n"+
				"\t--set-as-default",
				policyARN, credrequest)
			tagPolicy := fmt.Sprintf("aws iam tag-policy \\\n"+
				"\t--tags %s \\\n"+
				"\t--policy-arn %s",
				policTags, policyARN)
			commands = append(commands, createPolicy, tagPolicy)
		}
	}
	return strings.Join(commands, "\n\n")
}
