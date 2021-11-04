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

package operatorroles

import (
	"fmt"

	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var modes []string = []string{"auto", "manual"}

var args struct {
	mode string
}

var Cmd = &cobra.Command{
	Use:     "operator-roles",
	Aliases: []string{"operator-role", "operatorroles"},
	Short:   "Upgrade operator IAM roles for a cluster.",
	Long:    "Upgrade cluster-specific operator IAM roles to latest version.",
	Example: `  # Upgrade cluster-specific operator IAM roles
  rosa upgrade operators-roles`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.mode,
		"mode",
		modes[0],
		"How to perform the operation. Valid options are:\n"+
			"auto: Operator IAM roles will be upgraded automatically to the latest version\n"+
			"manual: Command to upgrade the operator IAM roles will be output which can be used to upgrade manually",
	)
	Cmd.RegisterFlagCompletionFunc("mode", modeCompletion)

	ocm.AddClusterFlag(Cmd)
	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func modeCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return modes, cobra.ShellCompDirectiveDefault
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	if !arguments.IsValidMode(modes, args.mode) {
		reporter.Errorf("Invalid mode. Allowed values are %s", modes)
		os.Exit(1)
	}

	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		reporter.Errorf("%s", err)
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
	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}

	creator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get IAM credentials: %s", err)
		os.Exit(1)
	}
	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocmClient.GetCluster(clusterKey, creator)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	operatorRoles, hasOperatorRoles := cluster.AWS().STS().GetOperatorIAMRoles()
	if !hasOperatorRoles || len(operatorRoles) < 0 {
		reporter.Errorf("Cluster '%s' doesnt have any operator roles associated with it",
			clusterKey)
	}
	prefix, err := aws.GetPrefixFromAccountRole(cluster)
	if err != nil {
		reporter.Errorf("Error getting account role prefix for the cluster '%s'",
			clusterKey)
	}
	//Check if account roles are up-to-date
	isAccountRoleUpgradeNeed, err := awsClient.IsUpgradedNeededForRole(prefix, creator.AccountID)
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}
	if isAccountRoleUpgradeNeed {
		reporter.Infof("Account roles with prefix '%s' need to be upgraded before operator roles. "+
			"Use rosa upgrade account-roles --prefix %s", prefix, prefix)
		os.Exit(1)
	}

	isUpgradeNeeded, err := awsClient.IsUpgradedNeededForOperatorRole(cluster, aws.DefaultPolicyVersion,
		creator.AccountID)
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}
	if !isUpgradeNeeded {
		reporter.Infof("Operator roles associated with the cluster '%s' is already up-to-date.", cluster.ID())
		os.Exit(1)
	}
	reporter.Infof("Starting to upgrade the policies")

	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
		interactive.Enable()
	}
	mode := args.mode
	if interactive.Enabled() {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "Operator IAM role upgrade mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  mode,
			Options:  modes,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid operator IAM role upgrade mode: %s", err)
			os.Exit(1)
		}
	}
	switch mode {
	case "auto":
		err = upgradeOperatorRolePolicies(reporter, awsClient, creator.AccountID, cluster, prefix)
		if err != nil {
			reporter.Errorf("Error upgrading the role polices: %s", err)
			os.Exit(1)
		}
	case "manual":
		err = aws.GenerateOperatorPolicyFiles(reporter)
		if err != nil {
			reporter.Errorf("There was an error generating the policy files: %s", err)
			os.Exit(1)
		}
		if reporter.IsTerminal() {
			reporter.Infof("All policy files saved to the current directory")
			reporter.Infof("Run the following commands to upgrade the operator IAM policies:\n")
		}
		commands, err := buildCommands(prefix, creator.AccountID, cluster, awsClient)
		if err != nil {
			reporter.Errorf("There was an error building the commands %s", err)
			os.Exit(1)
		}
		fmt.Println(commands)
	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", modes)
		os.Exit(1)
	}
}

func upgradeOperatorRolePolicies(reporter *rprtr.Object, awsClient aws.Client, accountID string,
	cluster *cmv1.Cluster, prefix string) error {
	for credrequest, operator := range aws.CredentialRequests {
		roleName := aws.GetOperatorRoleName(cluster, operator)
		if !confirm.Prompt(true, "Upgrade the '%s' operator role policy to version %s?", roleName,
			aws.DefaultPolicyVersion) {
			continue
		}
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
		err = awsClient.UpdateTag(roleName)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildCommands(prefix string, accountID string, cluster *cmv1.Cluster, awsClient aws.Client) (string, error) {
	commands := []string{}
	for credrequest, operator := range aws.CredentialRequests {
		roleName := aws.GetOperatorRoleName(cluster, operator)
		iamRoleTags := fmt.Sprintf(
			"Key=%s,Value=%s",
			tags.OpenShiftVersion, aws.DefaultPolicyVersion)
		tagRole := fmt.Sprintf("aws iam tag-role \\\n"+
			"\t--tags %s \\\n"+
			"\t--role-name %s",
			iamRoleTags, roleName)
		commands = append(commands, tagRole)

		policyARN := aws.GetOperatorPolicyARN(accountID, prefix, operator.Namespace, operator.Name)
		policyTags := fmt.Sprintf(
			"Key=%s,Value=%s",
			tags.OpenShiftVersion, aws.DefaultPolicyVersion,
		)
		isCompatible, err := awsClient.IsPolicyCompatible(policyARN, aws.DefaultPolicyVersion)
		if err != nil {
			return "", err
		}
		//We need because users might run it mutiple times and we dont want to create unnecessary versions
		if isCompatible {
			continue
		}
		createPolicy := fmt.Sprintf("aws iam create-policy-version \\\n"+
			"\t--policy-arn %s \\\n"+
			"\t--policy-document file://openshift_%s_policy.json \\\n"+
			"\t --set-as-default",
			policyARN, credrequest)
		tagPolicy := fmt.Sprintf("aws iam tag-policy \\\n"+
			"\t--tags %s \\\n"+
			"\t--policy-arn %s",
			policyTags, policyARN)

		commands = append(commands, createPolicy, tagPolicy, tagRole)
	}
	return strings.Join(commands, "\n\n"), nil
}
