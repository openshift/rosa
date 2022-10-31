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
	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
)

var args struct {
	prefix string
}

var Cmd = &cobra.Command{
	Use:     "account-roles",
	Aliases: []string{"accountroles", "accountrole", "account-role"},
	Short:   "Delete Account Roles",
	Long:    "Cleans up account roles from the current AWS account.",
	Example: `  # Delete Account roles"
  rosa delete account-roles -p prefix`,
	Run: run,
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

	aws.AddModeFlag(Cmd)
	confirm.AddFlag(flags)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	// Determine if interactive mode is needed (if a prefix is not provided, fallback to interactive mode)
	if !interactive.Enabled() && !cmd.Flags().Changed("mode") || args.prefix == "" {
		interactive.Enable()
	}

	mode, err := aws.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	env, err := ocm.GetEnv()
	if err != nil {
		r.Reporter.Errorf("Error getting environment %s", err)
		os.Exit(1)
	}

	clusters, err := r.OCMClient.GetAllClusters(r.Creator)
	if err != nil {
		r.Reporter.Errorf("Error getting clusters %s", err)
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

	finalRoleList := []string{}
	roles, err := r.AWSClient.GetAccountRoleForCurrentEnvWithPrefix(env, prefix)
	if err != nil {
		r.Reporter.Errorf("Error getting role: %s", err)
		os.Exit(1)
	}
	if len(roles) == 0 {
		r.Reporter.Errorf("There are no roles to be deleted")
		os.Exit(1)
	}
	for _, role := range roles {
		if role.RoleName == "" {
			continue
		}
		clusterID := checkIfRoleAssociated(clusters, role)
		if clusterID != "" {
			r.Reporter.Errorf("Role %s is associated with the cluster %s", role.RoleName, clusterID)
			os.Exit(1)
		}
		finalRoleList = append(finalRoleList, role.RoleName)
	}

	if len(finalRoleList) == 0 {
		r.Reporter.Errorf("There are no roles to be deleted")
		os.Exit(1)
	}
	for _, role := range finalRoleList {
		instanceProfiles, err := r.AWSClient.GetInstanceProfilesForRole(role)
		if err != nil {
			r.Reporter.Errorf("Error checking for instance roles: %s", err)
			os.Exit(1)
		}
		if len(instanceProfiles) > 0 {
			r.Reporter.Errorf("Instance Profiles are attached to the role. Please make sure it is deleted: %s",
				strings.Join(instanceProfiles, ","))
			os.Exit(1)
		}
	}

	if interactive.Enabled() {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "Account role deletion mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  aws.ModeAuto,
			Options:  aws.Modes,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid Account role deletion mode: %s", err)
			os.Exit(1)
		}
	}
	switch mode {
	case aws.ModeAuto:
		r.OCMClient.LogEvent("ROSADeleteAccountRoleModeAuto", nil)
		for _, role := range finalRoleList {
			if !confirm.Prompt(true, "Delete the account role '%s'?", role) {
				continue
			}
			err := r.AWSClient.DeleteAccountRole(role)
			if err != nil {
				r.Reporter.Warnf("There was an error deleting the account roles or policies: %s", err)
				continue
			}
		}
		r.Reporter.Infof("Successfully deleted the account roles")
	case aws.ModeManual:
		r.OCMClient.LogEvent("ROSADeleteAccountRoleModeManual", nil)
		policyMap, err := r.AWSClient.GetAccountRolePolicies(finalRoleList)
		if err != nil {
			r.Reporter.Errorf("There was an error getting the policy: %v", err)
			os.Exit(1)
		}
		commands := buildCommand(finalRoleList, policyMap)

		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("Run the following commands to delete the account roles and policies:\n")
		}
		fmt.Println(commands)
	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
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

func buildCommand(roleNames []string, policyMap map[string][]aws.PolicyDetail) string {
	commands := []string{}
	for _, roleName := range roleNames {
		policyDetails := policyMap[roleName]
		for _, policyDetail := range policyDetails {
			if policyDetail.PolicType == aws.Attached && policyDetail.PolicyArn != "" {
				detachPolicy := awscb.NewIAMCommandBuilder().
					SetCommand(awscb.DetachRolePolicy).
					AddParam(awscb.RoleName, roleName).
					AddParam(awscb.PolicyArn, policyDetail.PolicyArn).
					Build()
				commands = append(commands, detachPolicy)

				deletePolicy := awscb.NewIAMCommandBuilder().
					SetCommand(awscb.DeletePolicy).
					AddParam(awscb.PolicyArn, policyDetail.PolicyArn).
					Build()
				commands = append(commands, deletePolicy)
			}
			if policyDetail.PolicType == aws.Inline && policyDetail.PolicyName != "" {
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
