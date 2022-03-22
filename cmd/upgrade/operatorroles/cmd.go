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

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:     "operator-roles",
	Aliases: []string{"operator-role", "operatorroles"},
	Short:   "Upgrade operator IAM roles for a cluster.",
	Long:    "Upgrade cluster-specific operator IAM roles to latest version.",
	Example: `  # Upgrade cluster-specific operator IAM roles
  rosa upgrade operators-roles`,
	RunE: runE,
}

var args struct {
	upgradeVersion string
}

func init() {
	flags := Cmd.Flags()

	aws.AddModeFlag(Cmd)
	ocm.AddClusterFlag(Cmd)
	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func runE(cmd *cobra.Command, argv []string) error {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Allow the command to be called programmatically
	skipInteractive := false
	isProgrammaticallyCalled := false
	if len(argv) >= 2 && !cmd.Flag("cluster").Changed {
		ocm.SetClusterKey(argv[0])
		aws.SetModeKey(argv[1])

		if argv[1] != "" {
			skipInteractive = true
		}
		if len(argv) > 2 && argv[2] != "" {
			args.upgradeVersion = argv[2]
		}
		isProgrammaticallyCalled = true
	}

	mode, err := aws.GetMode()
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}
	fmt.Println(mode)

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
	if !hasOperatorRoles || len(operatorRoles) == 0 {
		reporter.Errorf("Cluster '%s' doesnt have any operator roles associated with it",
			clusterKey)
	}
	prefix, err := aws.GetPrefixFromAccountRole(cluster)
	if err != nil {
		reporter.Errorf("Error getting account role prefix for the cluster '%s'",
			clusterKey)
	}

	// If the command was invoked from upgrade cluster, we already performed the policies upgrade
	var isAccountRoleUpgradeNeed bool
	if !isProgrammaticallyCalled {
		//Check if account roles are up-to-date
		isAccountRoleUpgradeNeed, err = awsClient.IsUpgradedNeededForAccountRolePolicies(
			prefix, aws.DefaultPolicyVersion)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
		if isAccountRoleUpgradeNeed && !isProgrammaticallyCalled {
			reporter.Infof("Account roles with prefix '%s' need to be upgraded before operator roles. "+
				"Roles can be upgraded with the following command :"+
				"\n\n\trosa upgrade account-roles --prefix %s\n", prefix, prefix)
			os.Exit(1)
		}

		isUpgradeNeeded, err := awsClient.IsUpgradedNeededForOperatorRolePolicies(cluster,
			creator.AccountID, aws.DefaultPolicyVersion)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}

		if !isUpgradeNeeded {
			reporter.Infof("Operator roles associated with the cluster '%s' is already up-to-date.", cluster.ID())
			os.Exit(1)
		}
	}

	missingRoles, err := awsClient.FindMissingOperatorRolesForUpgrade(cluster, ocm.GetVersionMinor(args.upgradeVersion))
	if err != nil {
		reporter.Errorf("Failed to find missing roles for upgrade: %v", err)
		os.Exit(1)
	}
	if len(missingRoles) == 0 {
		reporter.Infof("Operator roles associated with the cluster '%s' is already up-to-date.", cluster.ID())
		return nil
	}
	reporter.Infof("Starting to upgrade the policies")

	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
		interactive.Enable()
	}

	if interactive.Enabled() && !skipInteractive {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "Operator IAM role upgrade mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  aws.ModeAuto,
			Options:  aws.Modes,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid operator IAM role upgrade mode: %s", err)
			os.Exit(1)
		}
	}

	if !isProgrammaticallyCalled {
		err = upgradeOperatorPolicies(mode, reporter, awsClient, creator, cluster, prefix, isProgrammaticallyCalled)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	//If missing roles length is greater than 0
	//iterate the missing roles and find if it is present in the aws or not . If not then proceed with auto and manual.
	//Else call ocm api
	//If user runs manual mode exits and come back or if role is created in aws alone and there was an error when updating the ocm
	//this will handle it effectively. anything we missed?
	if len(missingRoles) > 0 {
		reporter.Infof("Detected missing Operator IAM roles")
		for _, operator := range missingRoles {
			//operator := aws.CredentialRequests[missingRole]
			roleName := getRoleName(cluster, operator)
			exists, _, _ := awsClient.CheckRoleExists(roleName)
			//Handle error
			if !exists {
				err = createOperatorRole(mode, reporter, awsClient, creator, cluster, prefix, missingRoles)
				if err != nil {
					reporter.Errorf("%s", err)
					os.Exit(1)
				}
			}
			// todo add to OCM
		}
	}

	return nil
}

func upgradeOperatorPolicies(mode string, reporter *rprtr.Object, awsClient aws.Client, creator *aws.Creator,
	cluster *cmv1.Cluster, prefix string, isAccountRoleUpgradeNeed bool) error {
	switch mode {
	case aws.ModeAuto:
		err := upgradeOperatorRolePolicies(reporter, awsClient, creator.AccountID, cluster, prefix)
		if err != nil {
			return reporter.Errorf("Error upgrading the role polices: %s", err)
		}
	case aws.ModeManual:
		err := aws.GenerateOperatorPolicyFiles(reporter)
		if err != nil {
			return reporter.Errorf("There was an error generating the policy files: %s", err)
		}
		if reporter.IsTerminal() {
			reporter.Infof("All policy files saved to the current directory")
			reporter.Infof("Run the following commands to upgrade the operator IAM policies:\n")
			if isAccountRoleUpgradeNeed {
				reporter.Warnf("Operator role policies MUST only be upgraded after " +
					"Account Role policies upgrade has completed.\n")
			}
		}
		commands, err := buildCommands(prefix, creator.AccountID, cluster, awsClient)
		if err != nil {
			return reporter.Errorf("There was an error building the commands %s", err)

		}
		fmt.Println(commands)
	default:
		return reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
	}
	return nil
}

/**
reused from create operator roles. Need to check permission boundary and add it
Need to effectively return error
Need to clean up
*/

func createOperatorRole(mode string, reporter *rprtr.Object, awsClient aws.Client, creator *aws.Creator,
	cluster *cmv1.Cluster, prefix string, missingRoles map[string]aws.Operator) error {
	accountID := creator.AccountID
	switch mode {
	case aws.ModeAuto:
		for _, operator := range missingRoles {
			roleName := getRoleName(cluster, operator)
			if !confirm.Prompt(true, "Create the '%s' role?", roleName) {
				continue
			}
			policyARN := aws.GetOperatorPolicyARN(accountID, prefix, operator.Namespace, operator.Name)
			policy, err := aws.GenerateRolePolicyDoc(cluster, accountID, operator)
			if err != nil {
				return err
			}
			reporter.Debugf("Creating role '%s'", roleName)
			roleARN, err := awsClient.EnsureRole(roleName, policy, "", "",
				map[string]string{
					tags.ClusterID:       cluster.ID(),
					"operator_namespace": operator.Namespace,
					"operator_name":      operator.Name,
				})
			if err != nil {
				return err
			}
			reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)
			reporter.Debugf("Attaching permission policy '%s' to role '%s'", policyARN, roleName)
			err = awsClient.AttachRolePolicy(roleName, policyARN)
			if err != nil {
				return fmt.Errorf("Failed to attach role policy. Check your prefix or run "+
					"'rosa create account-roles' to create the necessary policies: %s", err)
			}
		}
	case aws.ModeManual:
		commands := []string{}
		for credRequest, operator := range missingRoles {
			roleName := getRoleName(cluster, operator)
			policyARN := aws.GetOperatorPolicyARN(accountID, prefix, operator.Namespace, operator.Name)
			policy, err := aws.GenerateRolePolicyDoc(cluster, accountID, operator)
			if err != nil {
				return err
			}
			filename := fmt.Sprintf("operator_%s_policy.json", credRequest)
			reporter.Debugf("Saving '%s' to the current directory", filename)
			err = ocm.SaveDocument(policy, filename)
			if err != nil {
				return err
			}
			iamTags := fmt.Sprintf(
				"Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s",
				tags.ClusterID, cluster.ID(),
				tags.RolePrefix, prefix,
				"operator_namespace", operator.Namespace,
				"operator_name", operator.Name,
			)
			permBoundaryFlag := ""

			createRole := fmt.Sprintf("aws iam create-role \\\n"+
				"\t--role-name %s \\\n"+
				"\t--assume-role-policy-document file://%s \\\n"+
				"%s"+
				"\t--tags %s",
				roleName, filename, permBoundaryFlag, iamTags)
			attachRolePolicy := fmt.Sprintf("aws iam attach-role-policy \\\n"+
				"\t--role-name %s \\\n"+
				"\t--policy-arn %s",
				roleName, policyARN)
			commands = append(commands, createRole, attachRolePolicy)

		}
		if reporter.IsTerminal() {
			reporter.Infof("Run the following commands to create the operator roles:\n")
		}
		fmt.Println(commands)

	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
	return nil
}

/**
Need to rewrite in a effective way
*/
func getRoleName(cluster *cmv1.Cluster, missingOperator aws.Operator) string {
	operatorIAMRoles := cluster.AWS().STS().OperatorIAMRoles()
	rolePrefix := ""

	for _, operatorIAMRole := range operatorIAMRoles {
		//Currently we are not persisting the operator role prefix so to determine we need to
		//Split based on the name and namespace. We truncate the role name to 64 length so
		//prefix length cannot be greater than 32 so probability of finding the prefix with the name
		//that is less than 32 is more. This logic will not work if we change the name of all the
		//operators to be greater than 32 which is unlikely in immediate term
		if len(operatorIAMRole.Namespace()) <= 32 {
			roleName := strings.SplitN(operatorIAMRole.RoleARN(), "/", 2)[1]
			rolePrefix = strings.Split(roleName, fmt.Sprintf("-%s", operatorIAMRole.Namespace()))[0]
			break
		}
	}
	//This is unneccessary. Just a safer code if something changes in the operator role and
	//developer forgot to change this
	if rolePrefix == "" {
		rolePrefix = fmt.Sprintf("%s-%s", cluster.Name(), ocm.RandomLabel(4))
	}
	role := fmt.Sprintf("%s-%s-%s", rolePrefix, missingOperator.Namespace, missingOperator.Name)
	if len(role) > 64 {
		role = role[0:64]
	}
	return role
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
