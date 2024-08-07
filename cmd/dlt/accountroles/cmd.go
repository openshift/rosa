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

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	prefix   string
	hostedCP bool
	classic  bool
}

var Cmd = &cobra.Command{
	Use:     "account-roles",
	Aliases: []string{"accountroles", "accountrole", "account-role"},
	Short:   "Delete Account Roles",
	Long:    "Cleans up account roles from the current AWS account.",
	Example: `  # Delete Account roles"
  rosa delete account-roles -p prefix`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.prefix,
		"prefix",
		"p",
		"",
		"Prefix of the account roles to be deleted.",
	)

	flags.BoolVar(
		&args.hostedCP,
		"hosted-cp",
		false,
		"Delete Hosted Control Planes roles",
	)

	flags.BoolVar(
		&args.classic,
		"classic",
		false,
		"Delete classic account roles",
	)

	interactive.AddModeFlag(Cmd)
	confirm.AddFlag(flags)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	// Determine if interactive mode is needed (if a prefix is not provided, fallback to interactive mode)
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") || args.prefix == "" {
		interactive.Enable()
	}

	mode, err := interactive.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	env, err := ocm.GetEnv()
	if err != nil {
		r.Reporter.Errorf("Error getting environment %s", err)
		os.Exit(1)
	}

	deleteClassic, deleteHostedCP := setDeleteRoles(cmd.Flags().Changed("classic"),
		cmd.Flags().Changed("hosted-cp"))

	clusters, err := r.OCMClient.GetAllClusters(r.Creator)
	if err != nil {
		r.Reporter.Errorf("Error getting clusters %s", err)
		os.Exit(1)
	}

	if cmd.Flags().Changed("hosted-cp") && r.Creator.IsGovcloud {
		r.Reporter.Errorf("Setting `hosted-cp` is not supported for Govcloud AWS accounts")
		os.Exit(1)
	}

	prefix := args.prefix
	if interactive.Enabled() && prefix == "" {
		prefix, err = interactive.GetString(interactive.Input{
			Question: "Role prefix",
			Help:     cmd.Flags().Lookup("prefix").Usage,
			Default:  "ManagedOpenShift",
			Required: true,
			Validators: []interactive.Validator{
				interactive.RegExp(`[\w+=,.@-]+`),
				interactive.MaxLength(32),
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid role prefix: %s", err)
			os.Exit(1)
		}
	}
	if len(prefix) > 32 {
		r.Reporter.Errorf("Expected a prefix with no more than 32 characters")
		os.Exit(1)
	}
	if !aws.RoleNameRE.MatchString(prefix) {
		r.Reporter.Errorf("Expected a valid role prefix matching %s", aws.RoleNameRE.String())
		os.Exit(1)
	}

	if interactive.Enabled() {
		mode, err = interactive.GetOptionMode(cmd, mode, "Account role deletion mode")
		if err != nil {
			r.Reporter.Errorf("Expected a valid Account role deletion mode: %s", err)
			os.Exit(1)
		}
	}

	if deleteClassic {
		err = deleteAccountRoles(r, env, prefix, clusters, mode, false)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}

	if r.Creator.IsGovcloud {
		deleteHostedCP = false
	}

	if deleteHostedCP {
		err = deleteAccountRoles(r, env, prefix, clusters, mode, true)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
	}
}

func setDeleteRoles(isClassicFlagSet bool, isHostedCPFlagSet bool) (bool, bool) {
	// Default flow deletes both classic and hosted-CP roles if the user didn't specify topology
	if !isClassicFlagSet && !isHostedCPFlagSet {
		return true, true
	}

	return isClassicFlagSet, isHostedCPFlagSet
}

func deleteAccountRoles(r *rosa.Runtime, env string, prefix string, clusters []*cmv1.Cluster, mode string,
	hostedCP bool) error {
	var accountRolesMap map[string]aws.AccountRole
	var roleTypeString string
	if hostedCP {
		accountRolesMap = aws.HCPAccountRoles
		roleTypeString = "hosted CP "
	} else {
		accountRolesMap = aws.AccountRoles
		roleTypeString = "classic "
	}

	finalRoleList, managedPolicies, err := getRoleListForDeletion(r, env, prefix, clusters, accountRolesMap)
	if err != nil {
		return err
	}
	if len(finalRoleList) == 0 {
		r.Reporter.Warnf("There are no %saccount roles to be deleted", roleTypeString)
		return nil
	}

	switch mode {
	case interactive.ModeAuto:
		r.Reporter.Infof(fmt.Sprintf("Deleting %saccount roles", roleTypeString))

		r.OCMClient.LogEvent("ROSADeleteAccountRoleModeAuto", nil)
		for _, role := range finalRoleList {
			if !confirm.Prompt(true, "Delete the account role '%s'?", role) {
				continue
			}
			r.Reporter.Infof("Deleting account role '%s'", role)
			err := r.AWSClient.DeleteAccountRole(role, prefix, managedPolicies)
			if err != nil {
				r.Reporter.Warnf("There was an error deleting the account roles or policies: %s", err)
				continue
			}
		}
		r.Reporter.Infof(fmt.Sprintf("Successfully deleted the %saccount roles", roleTypeString))
	case interactive.ModeManual:
		r.OCMClient.LogEvent("ROSADeleteAccountRoleModeManual", nil)
		policyMap, arbitraryPolicyMap, err := r.AWSClient.GetAccountRolePolicies(finalRoleList, prefix)
		if err != nil {
			return fmt.Errorf("There was an error getting the policy: %v", err)
		}
		commands := buildCommand(finalRoleList, policyMap, arbitraryPolicyMap, managedPolicies)

		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("Run the following commands to delete the account roles and policies:\n")
		}
		fmt.Println(commands)
	default:
		return fmt.Errorf("invalid mode. Allowed values are %s", interactive.Modes)
	}

	return nil
}

func getRoleListForDeletion(r *rosa.Runtime, env string, prefix string, clusters []*cmv1.Cluster,
	accountRolesMap map[string]aws.AccountRole) ([]string, bool, error) {
	finalRoleList := []string{}
	roles, err := r.AWSClient.GetAccountRoleForCurrentEnvWithPrefix(env, prefix, accountRolesMap)
	if err != nil {
		return finalRoleList, false, fmt.Errorf("Error getting role: %s", err)
	}
	if len(roles) == 0 {
		return finalRoleList, false, nil
	}

	for _, role := range roles {
		if role.RoleName == "" {
			continue
		}
		clusterID := checkIfRoleAssociated(clusters, role)
		if clusterID != "" {
			return finalRoleList, false, fmt.Errorf("Role %s is associated with the cluster %s", role.RoleName, clusterID)
		}
		finalRoleList = append(finalRoleList, role.RoleName)
	}

	if len(finalRoleList) == 0 {
		return finalRoleList, false, nil
	}
	for _, role := range finalRoleList {
		instanceProfiles, err := r.AWSClient.GetInstanceProfilesForRole(role)
		if err != nil {
			return finalRoleList, false, fmt.Errorf("Error checking for instance roles: %s", err)
		}
		if len(instanceProfiles) > 0 {
			return finalRoleList, false, fmt.Errorf(
				"Instance Profiles are attached to the role. Please make sure it is deleted: %s",
				strings.Join(instanceProfiles, ","))
		}
	}

	managedPolicies, err := r.AWSClient.HasManagedPolicies(roles[0].RoleARN)
	if err != nil {
		return finalRoleList, false, fmt.Errorf("Failed to determine if cluster has managed policies: %v", err)
	}

	return finalRoleList, managedPolicies, nil
}

func checkIfRoleAssociated(clusters []*cmv1.Cluster, role aws.Role) string {
	for _, cluster := range clusters {
		if cluster.AWS().STS().RoleARN() == role.RoleARN ||
			cluster.AWS().STS().SupportRoleARN() == role.RoleARN ||
			cluster.AWS().STS().InstanceIAMRoles().MasterRoleARN() == role.RoleARN ||
			cluster.AWS().STS().InstanceIAMRoles().WorkerRoleARN() == role.RoleARN {
			return cluster.Name()
		}
	}
	return ""
}

func buildCommand(roleNames []string, policyMap map[string][]aws.PolicyDetail,
	arbitraryPolicyMap map[string][]aws.PolicyDetail, managedPolicies bool) string {
	commands := []string{}
	for _, roleName := range roleNames {
		policyDetails := policyMap[roleName]
		excludedPolicyDetails := arbitraryPolicyMap[roleName]
		for _, policyDetail := range excludedPolicyDetails {
			if policyDetail.PolicyArn != "" {
				detachPolicy := awscb.NewIAMCommandBuilder().
					SetCommand(awscb.DetachRolePolicy).
					AddParam(awscb.RoleName, roleName).
					AddParam(awscb.PolicyArn, policyDetail.PolicyArn).
					Build()
				commands = append(commands, detachPolicy)
			}
		}
		for _, policyDetail := range policyDetails {
			if policyDetail.PolicyType == aws.Attached && policyDetail.PolicyArn != "" {
				detachPolicy := awscb.NewIAMCommandBuilder().
					SetCommand(awscb.DetachRolePolicy).
					AddParam(awscb.RoleName, roleName).
					AddParam(awscb.PolicyArn, policyDetail.PolicyArn).
					Build()
				commands = append(commands, detachPolicy)

				if !managedPolicies {
					deletePolicy := awscb.NewIAMCommandBuilder().
						SetCommand(awscb.DeletePolicy).
						AddParam(awscb.PolicyArn, policyDetail.PolicyArn).
						Build()
					commands = append(commands, deletePolicy)
				}
			}
			if policyDetail.PolicyType == aws.Inline && policyDetail.PolicyName != "" {
				deletePolicy := awscb.NewIAMCommandBuilder().
					SetCommand(awscb.DeleteRolePolicy).
					AddParam(awscb.RoleName, roleName).
					AddParam(awscb.PolicyName, policyDetail.PolicyName).
					Build()
				commands = append(commands, deletePolicy)
			}
		}
		deleteRole := awscb.NewIAMCommandBuilder().
			SetCommand(awscb.DeleteRole).
			AddParam(awscb.RoleName, roleName).
			Build()
		commands = append(commands, deleteRole)
	}
	return awscb.JoinCommands(commands)
}
