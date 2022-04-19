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
	"time"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/helper"
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
	upgradeVersion string
}

var Cmd = &cobra.Command{
	Use:     "operator-roles",
	Aliases: []string{"operator-role", "operatorroles"},
	Short:   "Upgrade operator IAM roles for a cluster.",
	Long:    "Upgrade cluster-specific operator IAM roles to latest version.",
	Example: `  # Upgrade cluster-specific operator IAM roles
  rosa upgrade operators-roles`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	aws.AddModeFlag(Cmd)
	ocm.AddClusterFlag(Cmd)

	flags.StringVar(
		&args.upgradeVersion,
		"version",
		"",
		"Version of OpenShift that the cluster will be upgraded to",
	)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) error {
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

	/**
	we dont want to give this option to the end-user. Adding this as a support for srep if needed.
	*/
	if args.upgradeVersion != "" && !isProgrammaticallyCalled {
		version := args.upgradeVersion
		availableUpgrades, err := ocmClient.GetAvailableUpgrades(ocm.GetVersionID(cluster))
		if err != nil {
			reporter.Errorf("Failed to find available upgrades: %v", err)
			os.Exit(1)
		}
		if len(availableUpgrades) == 0 {
			reporter.Warnf("There are no available upgrades")
			os.Exit(0)
		}
		// Check that the version is valid
		validVersion := false
		for _, v := range availableUpgrades {
			if v == version {
				validVersion = true
				break
			}
		}
		if !validVersion {
			reporter.Errorf("Expected a valid version to upgrade the cluster")
			os.Exit(1)
		}
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
	isOperatorPolicyUpgradeNeeded := false
	isAccountRoleUpgradeNeed := false
	//If this is invoked from the upgrade cluster we already performed the policies upgrade as
	//part of upgrade account roles that was called before this command. Refer to rosa upgrade cluster
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

		isOperatorPolicyUpgradeNeeded, err = awsClient.IsUpgradedNeededForOperatorRolePolicies(cluster,
			creator.AccountID, aws.DefaultPolicyVersion)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	version := args.upgradeVersion
	if version == "" {
		version = cluster.Version().RawID()
	}
	//Check if the upgrade is needed for the operators
	missingRolesInCS, err := aws.FindMissingOperatorRolesForUpgrade(cluster, version)
	if err != nil {
		return err
	}

	if len(missingRolesInCS) <= 0 && !isOperatorPolicyUpgradeNeeded {
		if !isProgrammaticallyCalled {
			reporter.Infof("Operator roles associated with the cluster '%s' is already up-to-date.", cluster.ID())
		}
		return nil
	}

	if len(missingRolesInCS) > 0 || isOperatorPolicyUpgradeNeeded {
		reporter.Infof("Starting to upgrade the operator IAM role and policies")
	}
	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
		interactive.Enable()
	}
	policies, err := ocmClient.GetPolicies("OperatorRole")
	if err != nil {
		reporter.Errorf("Expected a valid role creation mode: %s", err)
		os.Exit(1)
	}

	env, err := ocm.GetEnv()
	if err != nil {
		reporter.Errorf("Failed to determine OCM environment: %v", err)
		os.Exit(1)
	}

	//This might not be true when invoked from upgrade cluster
	if isOperatorPolicyUpgradeNeeded {
		mode, err = handleModeFlag(cmd, skipInteractive, mode, err, reporter)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
		err = upgradeOperatorPolicies(mode, reporter, awsClient, creator, prefix, isAccountRoleUpgradeNeed, policies, env)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	//If missing roles length is greater than 0
	//iterate the missing roles and find it is is present in the aws or not . If not then proceed with auto and manual.
	//Else call ocm api
	//If user runs manual mode exits and come back or if role is created in aws alone and there was an error when
	//updating the ocm. this will handle it effectively.
	if len(missingRolesInCS) > 0 {
		for _, operator := range missingRolesInCS {
			roleName := getRoleName(cluster, operator)
			exists, _, err := awsClient.CheckRoleExists(roleName)
			if err != nil {
				return reporter.Errorf("Error when detecting checking missing operator IAM roles %s", err)
			}
			if !exists {
				mode, err = handleModeFlag(cmd, skipInteractive, mode, err, reporter)
				if err != nil {
					reporter.Errorf("%s", err)
					os.Exit(1)
				}

				if err != nil {
					reporter.Errorf("Expected a valid role creation mode: %s", err)
					os.Exit(1)
				}

				err = createOperatorRole(mode, reporter, awsClient, creator, cluster, prefix, missingRolesInCS,
					isProgrammaticallyCalled, policies)
				if err != nil {
					reporter.Errorf("%s", err)
					os.Exit(1)
				}
			}
		}
	}
	return nil
}

func handleModeFlag(cmd *cobra.Command, skipInteractive bool, mode string, err error,
	reporter *rprtr.Object) (string, error) {
	if interactive.Enabled() && !skipInteractive {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "Operator IAM role/policy upgrade mode",
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
	aws.SetModeKey(mode)
	return mode, err
}

func upgradeOperatorPolicies(mode string, reporter *rprtr.Object, awsClient aws.Client, creator *aws.Creator,
	prefix string, isAccountRoleUpgradeNeed bool, policies map[string]string, env string) error {
	switch mode {
	case aws.ModeAuto:
		if !confirm.Prompt(true, "Upgrade the operator role policy to version %s?", aws.DefaultPolicyVersion) {
			return nil
		}
		err := aws.UpgradeOperatorPolicies(reporter, awsClient, creator.AccountID, prefix, policies)
		if err != nil {
			return reporter.Errorf("Error upgrading the role polices: %s", err)
		}
		return nil
	case aws.ModeManual:
		err := aws.GeneratePolicyFiles(reporter, env, false, true, policies)
		if err != nil {
			reporter.Errorf("There was an error generating the policy files: %s", err)
			os.Exit(1)
		}

		if reporter.IsTerminal() {
			reporter.Infof("All policy files saved to the current directory")
			reporter.Infof("Run the following commands to upgrade the operator IAM policies:\n")
			if isAccountRoleUpgradeNeed {
				reporter.Warnf("Operator role policies MUST only be upgraded after " +
					"Account Role policies upgrade has completed.\n")
			}
		}
		commands := aws.BuildOperatorRoleCommands(prefix, creator.AccountID, awsClient)
		fmt.Println(strings.Join(commands, "\n\n"))
	default:
		return reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
	}
	return nil
}

func createOperatorRole(mode string, reporter *rprtr.Object, awsClient aws.Client, creator *aws.Creator,
	cluster *cmv1.Cluster, prefix string, missingRoles map[string]aws.Operator, isProgrammaticallyCalled bool,
	policies map[string]string) error {
	accountID := creator.AccountID
	switch mode {
	case aws.ModeAuto:
		err := upgradeMissingOperatorRole(missingRoles, cluster, accountID, prefix, reporter, awsClient,
			isProgrammaticallyCalled, policies)
		if err != nil {
			return err
		}
		helper.DisplaySpinnerWithDelay(reporter, "Waiting for operator roles to reconcile", 5*time.Second)
	case aws.ModeManual:
		commands, err := buildMissingOperatorRoleCommand(missingRoles, cluster, accountID, prefix, reporter, policies)
		if err != nil {
			return err
		}
		if reporter.IsTerminal() {
			reporter.Infof("Run the following commands to create the operator roles:\n")
		}
		fmt.Println(commands)
		if isProgrammaticallyCalled {
			reporter.Infof("Run the following command to continue scheduling cluster upgrade"+
				" once account and operator roles have been upgraded : \n\n"+
				"\trosa upgrade cluster --cluster %s\n", cluster.ID())
			os.Exit(0)
		}
	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
	return nil
}

func buildMissingOperatorRoleCommand(missingRoles map[string]aws.Operator, cluster *cmv1.Cluster, accountID string,
	prefix string, reporter *rprtr.Object, policies map[string]string) (string, error) {
	commands := []string{}
	for missingRole, operator := range missingRoles {
		roleName := getRoleName(cluster, operator)
		policyARN := aws.GetOperatorPolicyARN(accountID, prefix, operator.Namespace, operator.Name)
		policyDetails := policies["operator_iam_role_policy"]
		policy, err := aws.GenerateRolePolicyDoc(cluster, accountID, operator, policyDetails)
		if err != nil {
			return "", err
		}
		filename := fmt.Sprintf("operator_%s_policy", missingRole)
		filename = aws.GetFormattedFileName(filename)
		reporter.Debugf("Saving '%s' to the current directory", filename)
		err = ocm.SaveDocument(policy, filename)
		if err != nil {
			return "", err
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
	return strings.Join(commands, "\n\n"), nil
}

func upgradeMissingOperatorRole(missingRoles map[string]aws.Operator, cluster *cmv1.Cluster,
	accountID string, prefix string,
	reporter *rprtr.Object, awsClient aws.Client, isProgrammaticallyCalled bool, policies map[string]string) error {
	for _, operator := range missingRoles {
		roleName := getRoleName(cluster, operator)
		if !confirm.Prompt(true, "Create the '%s' role?", roleName) {
			if isProgrammaticallyCalled {
				return fmt.Errorf("Operator roles need to be upgraded to proceed with cluster upgrade")
			}
			continue
		}
		policyDetails := policies["operator_iam_role_policy"]

		policyARN := aws.GetOperatorPolicyARN(accountID, prefix, operator.Namespace, operator.Name)
		policy, err := aws.GenerateRolePolicyDoc(cluster, accountID, operator, policyDetails)
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
	return nil
}

func getRoleName(cluster *cmv1.Cluster, missingOperator aws.Operator) string {
	operatorIAMRoles := cluster.AWS().STS().OperatorIAMRoles()
	rolePrefix := ""
	for _, operatorIAMRole := range operatorIAMRoles {
		roleName := strings.SplitN(operatorIAMRole.RoleARN(), "/", 2)[1]
		m := strings.LastIndex(roleName, "-openshift")
		rolePrefix = roleName[0:m]
		break
	}
	role := fmt.Sprintf("%s-%s-%s", rolePrefix, missingOperator.Namespace, missingOperator.Name)
	if len(role) > 64 {
		role = role[0:64]
	}
	return role
}
