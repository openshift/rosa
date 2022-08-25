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
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
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
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

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
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	clusterKey := r.GetClusterKey()

	defaultPolicyVersion, err := r.OCMClient.GetDefaultVersion()
	if err != nil {
		r.Reporter.Errorf("Error getting latest default version: %s", err)
		os.Exit(1)
	}

	cluster := r.FetchCluster()
	/**
	we dont want to give this option to the end-user. Adding this as a support for srep if needed.
	*/
	if args.upgradeVersion != "" && !isProgrammaticallyCalled {
		version := args.upgradeVersion
		availableUpgrades, err := r.OCMClient.GetAvailableUpgrades(ocm.GetVersionID(cluster))
		if err != nil {
			r.Reporter.Errorf("Failed to find available upgrades: %v", err)
			os.Exit(1)
		}
		if len(availableUpgrades) == 0 {
			r.Reporter.Warnf("There are no available upgrades")
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
			r.Reporter.Errorf("Expected a valid version to upgrade the cluster")
			os.Exit(1)
		}
	}

	operatorRoles, hasOperatorRoles := cluster.AWS().STS().GetOperatorIAMRoles()
	if !hasOperatorRoles || len(operatorRoles) == 0 {
		r.Reporter.Errorf("Cluster '%s' doesnt have any operator roles associated with it",
			clusterKey)
	}

	prefix, err := aws.GetPrefixFromAccountRole(cluster)
	if err != nil {
		r.Reporter.Errorf("Error getting account role prefix for the cluster '%s'",
			clusterKey)
	}
	rolePath, policyPath, err := getOperatorPaths(r.AWSClient, prefix, operatorRoles)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	isAccountRoleUpgradeNeed := false
	// If this is invoked from upgrade cluster then we already performed upgrade account roles
	if !isProgrammaticallyCalled {
		//Check if account roles are up-to-date
		isAccountRoleUpgradeNeed, err = r.AWSClient.IsUpgradedNeededForAccountRolePolicies(
			prefix, defaultPolicyVersion)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		if isAccountRoleUpgradeNeed {
			r.Reporter.Infof("Account roles with prefix '%s' need to be upgraded before operator roles. "+
				"Roles can be upgraded with the following command :"+
				"\n\n\trosa upgrade account-roles --prefix %s\n", prefix, prefix)
			os.Exit(1)
		}
	}

	credRequests, err := r.OCMClient.GetCredRequests()
	if err != nil {
		r.Reporter.Errorf("Error getting operator credential request from OCM %s", err)
		os.Exit(1)
	}

	isOperatorPolicyUpgradeNeeded := false
	isOperatorPolicyUpgradeNeeded, err = r.AWSClient.IsUpgradedNeededForOperatorRolePoliciesUsingPrefix(prefix,
		r.Creator.AccountID, defaultPolicyVersion, credRequests, policyPath)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	version := args.upgradeVersion
	if version == "" {
		version = cluster.Version().RawID()
	}

	//Check if the upgrade is needed for the operators
	missingRolesInCS, err := r.OCMClient.FindMissingOperatorRolesForUpgrade(cluster, version)
	if err != nil {
		return err
	}

	if len(missingRolesInCS) <= 0 && !isOperatorPolicyUpgradeNeeded {
		if !isProgrammaticallyCalled {
			r.Reporter.Infof("Operator roles associated with the cluster '%s' are already up-to-date.", cluster.ID())
		}
		return nil
	}

	if len(missingRolesInCS) > 0 || isOperatorPolicyUpgradeNeeded {
		r.Reporter.Infof("Starting to upgrade the operator IAM roles and policies")
	}
	// Determine if interactive mode is needed
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
		interactive.Enable()
	}
	policies, err := r.OCMClient.GetPolicies("OperatorRole")
	if err != nil {
		r.Reporter.Errorf("Expected a valid role creation mode: %s", err)
		os.Exit(1)
	}

	env, err := ocm.GetEnv()
	if err != nil {
		r.Reporter.Errorf("Failed to determine OCM environment: %v", err)
		os.Exit(1)
	}

	if isOperatorPolicyUpgradeNeeded {
		mode, err = handleModeFlag(cmd, skipInteractive, mode, err, r.Reporter)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		err = upgradeOperatorPolicies(isProgrammaticallyCalled, mode, r, prefix, isAccountRoleUpgradeNeed,
			policies, env, defaultPolicyVersion, credRequests, cluster, policyPath)
		if err != nil {
			r.Reporter.Errorf("%s", err)
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
			exists, _, err := r.AWSClient.CheckRoleExists(roleName)
			if err != nil {
				return r.Reporter.Errorf("Error when detecting checking missing operator IAM roles %s", err)
			}
			if !exists {
				mode, err = handleModeFlag(cmd, skipInteractive, mode, err, r.Reporter)
				if err != nil {
					r.Reporter.Errorf("%s", err)
					os.Exit(1)
				}

				if err != nil {
					r.Reporter.Errorf("Expected a valid role creation mode: %s", err)
					os.Exit(1)
				}

				err = createOperatorRole(mode, r, cluster, prefix, missingRolesInCS,
					isProgrammaticallyCalled, policies, rolePath, policyPath)
				if err != nil {
					r.Reporter.Errorf("%s", err)
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

func upgradeOperatorPolicies(isProgrammaticallyCalled bool, mode string, r *rosa.Runtime,
	prefix string, isAccountRoleUpgradeNeed bool, policies map[string]string, env string, defaultPolicyVersion string,
	credRequests map[string]*cmv1.STSOperator, cluster *cmv1.Cluster, policyPath string) error {
	switch mode {
	case aws.ModeAuto:
		if !confirm.Prompt(true, "Upgrade the operator role policy to version %s?", defaultPolicyVersion) {
			if isProgrammaticallyCalled {
				return r.Reporter.Errorf("Operator roles need to be upgraded to proceed")
			}
			return nil
		}
		err := aws.UpggradeOperatorRolePolicies(r.Reporter, r.AWSClient, r.Creator.AccountID, prefix, policies,
			defaultPolicyVersion, credRequests, policyPath)
		if err != nil {
			if strings.Contains(err.Error(), "Throttling") {
				r.OCMClient.LogEvent("ROSAUpgradeOperatorRolesModeAuto", map[string]string{
					ocm.Response:   ocm.Failure,
					ocm.Version:    defaultPolicyVersion,
					ocm.IsThrottle: "true",
				})
			}
			return r.Reporter.Errorf("Error upgrading the operator policies: %s", err)
		}
		return nil
	case aws.ModeManual:
		err := aws.GeneratePolicyFiles(r.Reporter, env, false,
			true, policies, credRequests)
		if err != nil {
			r.Reporter.Errorf("There was an error generating the policy files: %s", err)
			os.Exit(1)
		}

		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("All policy files saved to the current directory")
			r.Reporter.Infof("Run the following commands to upgrade the operator IAM policies:\n")
			if isAccountRoleUpgradeNeed {
				r.Reporter.Warnf("Operator role policies MUST only be upgraded after " +
					"Account Role policies upgrade has completed.\n")
			}
		}
		commands := aws.BuildOperatorRoleCommands(prefix, r.Creator.AccountID, r.AWSClient,
			defaultPolicyVersion, credRequests, policyPath)
		fmt.Println(strings.Join(commands, "\n\n"))
	default:
		return r.Reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
	}
	return nil
}

func createOperatorRole(mode string, r *rosa.Runtime, cluster *cmv1.Cluster, prefix string,
	missingRoles map[string]*cmv1.STSOperator, isProgrammaticallyCalled bool, policies map[string]string,
	rolePath string, policyPath string) error {
	accountID := r.Creator.AccountID
	switch mode {
	case aws.ModeAuto:
		err := upgradeMissingOperatorRole(missingRoles, cluster, accountID, prefix, r,
			isProgrammaticallyCalled, policies, rolePath, policyPath)
		if err != nil {
			return err
		}
		helper.DisplaySpinnerWithDelay(r.Reporter, "Waiting for operator roles to reconcile", 5*time.Second)
	case aws.ModeManual:
		commands, err := buildMissingOperatorRoleCommand(missingRoles, cluster, accountID, prefix, r, policies,
			rolePath, policyPath)
		if err != nil {
			return err
		}
		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("Run the following commands to create the operator roles:\n")
		}
		fmt.Println(commands)
		if isProgrammaticallyCalled {
			r.Reporter.Infof("Run the following command to continue scheduling cluster upgrade"+
				" once account and operator roles have been upgraded : \n\n"+
				"\trosa upgrade cluster --cluster %s\n", cluster.ID())
			os.Exit(0)
		}
	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
	return nil
}

func buildMissingOperatorRoleCommand(missingRoles map[string]*cmv1.STSOperator, cluster *cmv1.Cluster, accountID string,
	prefix string, r *rosa.Runtime, policies map[string]string,
	rolePath string, policyPath string) (string, error) {
	commands := []string{}
	for missingRole, operator := range missingRoles {
		roleName := getRoleName(cluster, operator)
		policyARN := aws.GetOperatorPolicyARN(accountID, prefix, operator.Namespace(), operator.Name(), policyPath)
		policyDetails := policies["operator_iam_role_policy"]
		policy, err := aws.GenerateOperatorRolePolicyDoc(cluster, accountID, operator, policyDetails)
		if err != nil {
			return "", err
		}
		filename := fmt.Sprintf("operator_%s_policy", missingRole)
		filename = aws.GetFormattedFileName(filename)
		r.Reporter.Debugf("Saving '%s' to the current directory", filename)
		err = helper.SaveDocument(policy, filename)
		if err != nil {
			return "", err
		}
		iamTags := fmt.Sprintf(
			"Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s",
			tags.ClusterID, cluster.ID(),
			tags.RolePrefix, prefix,
			"operator_namespace", operator.Namespace(),
			"operator_name", operator.Name(),
		)
		permBoundaryFlag := ""

		createRole := fmt.Sprintf("aws iam create-role \\\n"+
			"\t--role-name %s \\\n"+
			"\t--assume-role-policy-document file://%s \\\n"+
			"%s"+
			"\t--tags %s \\\n",
			roleName, filename, permBoundaryFlag, iamTags)
		if rolePath != "" {
			createRole = fmt.Sprintf(createRole+"\t--path %s", rolePath)
		}
		attachRolePolicy := fmt.Sprintf("aws iam attach-role-policy \\\n"+
			"\t--role-name %s \\\n"+
			"\t--policy-arn %s",
			roleName, policyARN)
		commands = append(commands, createRole, attachRolePolicy)

	}
	return strings.Join(commands, "\n\n"), nil
}

func upgradeMissingOperatorRole(missingRoles map[string]*cmv1.STSOperator, cluster *cmv1.Cluster,
	accountID string, prefix string, r *rosa.Runtime, isProgrammaticallyCalled bool, policies map[string]string,
	rolePath string, policyPath string) error {
	for _, operator := range missingRoles {
		roleName := getRoleName(cluster, operator)
		if !confirm.Prompt(true, "Create the '%s' role?", roleName) {
			if isProgrammaticallyCalled {
				return fmt.Errorf("Operator roles need to be upgraded to proceed with cluster upgrade")
			}
			continue
		}
		policyDetails := policies["operator_iam_role_policy"]

		policyARN := aws.GetOperatorPolicyARN(accountID, prefix, operator.Namespace(), operator.Name(), policyPath)
		policy, err := aws.GenerateOperatorRolePolicyDoc(cluster, accountID, operator, policyDetails)
		if err != nil {
			return err
		}
		r.Reporter.Debugf("Creating role '%s'", roleName)
		roleARN, err := r.AWSClient.EnsureRole(roleName, policy, "", "",
			map[string]string{
				tags.ClusterID:       cluster.ID(),
				"operator_namespace": operator.Namespace(),
				"operator_name":      operator.Name(),
			}, rolePath)
		if err != nil {
			return err
		}
		r.Reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)
		r.Reporter.Debugf("Attaching permission policy '%s' to role '%s'", policyARN, roleName)
		err = r.AWSClient.AttachRolePolicy(roleName, policyARN)
		if err != nil {
			return fmt.Errorf("Failed to attach role policy. Check your prefix or run "+
				"'rosa create account-roles' to create the necessary policies: %s", err)
		}
	}
	return nil
}

func getRoleName(cluster *cmv1.Cluster, missingOperator *cmv1.STSOperator) string {
	operatorIAMRoles := cluster.AWS().STS().OperatorIAMRoles()
	rolePrefix := ""
	if len(operatorIAMRoles) > 0 {
		roleARN := operatorIAMRoles[0].RoleARN()
		roleName, err := aws.GetResourceIdFromARN(roleARN)
		if err != nil {
			return ""
		}

		m := strings.LastIndex(roleName, "-openshift")
		rolePrefix = roleName[0:m]
	}
	role := fmt.Sprintf("%s-%s-%s", rolePrefix, missingOperator.Namespace(), missingOperator.Name())
	if len(role) > 64 {
		role = role[0:64]
	}
	return role
}

func getOperatorPaths(awsClient aws.Client, prefix string, operatorRoles []*cmv1.OperatorIAMRole) (
	string, string, error) {
	for _, operatorRole := range operatorRoles {
		roleName, err := aws.GetResourceIdFromARN(operatorRole.RoleARN())
		if err != nil {
			return "", "", err
		}
		rolePolicies, err := awsClient.GetAttachedPolicy(&roleName)
		if err != nil {
			return "", "", err
		}
		policyName := aws.GetPolicyName(prefix, operatorRole.Namespace(), operatorRole.Name())
		for _, rolePolicy := range rolePolicies {
			if rolePolicy.PolicyName == policyName {
				rolePath, err := aws.GetPathFromARN(operatorRole.RoleARN())
				if err != nil {
					return "", "", err
				}
				policyPath, err := aws.GetPathFromARN(rolePolicy.PolicyArn)
				if err != nil {
					return "", "", err
				}
				return rolePath, policyPath, nil
			}
		}
	}
	return "", "", fmt.Errorf("Can not detect operator policy path. " +
		"Existing operator roles do not have operator policies attached to them")
}
