/*
  Copyright (c) 2022 Red Hat, Inc.

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

package ocmrole

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	unlinkocmrole "github.com/openshift/rosa/cmd/unlink/ocmrole"
	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	roleARN string
}

var Cmd = &cobra.Command{
	Use:     "ocm-role",
	Aliases: []string{"ocmrole"},
	Short:   "Delete OCM role",
	Long:    "Delete OCM role from the current AWS organization",
	Example: ` # Delete OCM role
rosa delete ocm-role --role-arn arn:aws:iam::123456789012:role/xxx-OCM-Role-1223456778`,
	Args: cobra.MaximumNArgs(1),
	Run:  run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.roleARN,
		"role-arn",
		"",
		"Role ARN to delete from the OCM organization account")

	aws.AddModeFlag(Cmd)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	mode, err := aws.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	orgID, _, err := r.OCMClient.GetCurrentOrganization()
	if err != nil {
		r.Reporter.Errorf("Error getting organization account: %v", err)
		os.Exit(1)
	}

	if len(argv) > 0 {
		args.roleARN = argv[0]
	}

	// Determine if interactive mode is needed
	if !interactive.Enabled() && (!cmd.Flags().Changed("mode")) {
		interactive.Enable()
	}

	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("Deleting OCM role")
	}

	roleARN := args.roleARN

	if !interactive.Enabled() && roleARN == "" {
		interactive.Enable()
	}

	if interactive.Enabled() {
		roleARN, err = interactive.GetString(interactive.Input{
			Question: "OCM Role ARN",
			Help:     cmd.Flags().Lookup("role-arn").Usage,
			Default:  roleARN,
			Required: true,
			Validators: []interactive.Validator{
				aws.ARNValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid ocm role ARN to delete from the current organization: %s", err)
			os.Exit(1)
		}
	}

	err = aws.ARNValidator(roleARN)
	if err != nil {
		r.Reporter.Errorf("Expected a valid ocm role ARN to delete from the current organization: %s", err)
		os.Exit(1)
	}

	err = r.AWSClient.ValidateRoleARNAccountIDMatchCallerAccountID(roleARN)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	managedPolicies, err := r.AWSClient.HasManagedPolicies(roleARN)
	if err != nil {
		r.Reporter.Errorf("Failed to determine if cluster has managed policies: %v", err)
		os.Exit(1)
	}

	if !confirm.Prompt(true, "Delete '%s' ocm role?", roleARN) {
		os.Exit(0)
	}

	linkedRoles, err := r.OCMClient.GetOrganizationLinkedOCMRoles(orgID)
	if err != nil {
		r.Reporter.Errorf("An error occurred while trying to get the organization linked roles: %s", err)
		os.Exit(1)
	}
	isLinked := helper.Contains(linkedRoles, roleARN)

	if interactive.Enabled() && !cmd.Flags().Changed("mode") {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "OCM role deletion mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  aws.ModeAuto,
			Options:  aws.Modes,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid OCM role deletion mode: %s", err)
			os.Exit(1)
		}
	}

	roleName, err := aws.GetResourceIdFromARN(roleARN)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if !aws.IsOCMRole(&roleName) {
		r.Reporter.Errorf("Role '%s' is not an OCM role", roleName)
		os.Exit(1)
	}

	roleExistOnAWS, existingRoleARN, err := r.AWSClient.CheckRoleExists(roleName)
	if err != nil {
		r.Reporter.Errorf("%v", err)
	}
	if !roleExistOnAWS {
		r.Reporter.Warnf("the ARN %s does not exist. Nothing to delete", roleARN)
	} else if existingRoleARN != roleARN {
		r.Reporter.Warnf("role with same name but different ARN exists. Existing role ARN: %s", existingRoleARN)
		os.Exit(1)
	}

	switch mode {
	case aws.ModeAuto:
		r.OCMClient.LogEvent("ROSADeleteOCMRoleModeAuto", nil)
		if isLinked {
			r.Reporter.Warnf("Role ARN '%s' is linked to organization '%s'", roleARN, orgID)
			unlinkocmrole.Cmd.Run(unlinkocmrole.Cmd, []string{roleARN})
		}
		if roleExistOnAWS {
			err := r.AWSClient.DeleteOCMRole(roleName, managedPolicies)
			if err != nil {
				r.Reporter.Errorf("There was an error deleting the OCM role: %s", err)
				os.Exit(1)
			}
			r.Reporter.Infof("Successfully deleted the OCM role")
		}
	case aws.ModeManual:
		r.OCMClient.LogEvent("ROSADeleteOCMRoleModeManual", nil)
		commands, err := buildCommands(roleName, roleARN, isLinked, r.AWSClient, roleExistOnAWS, managedPolicies)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		if r.Reporter.IsTerminal() {
			if roleExistOnAWS {
				r.Reporter.Infof("Run the following commands to delete the OCM role:\n")
			} else if isLinked {
				r.Reporter.Infof("Run the following commands to unlink the OCM role:\n")
			}
		}
		fmt.Println(commands)
	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
}

func buildCommands(roleName string, roleARN string, isLinked bool, awsClient aws.Client,
	roleExistOnAWS bool, managedPolicies bool) (string, error) {
	var commands []string

	if isLinked {
		unlinkRole := fmt.Sprintf("rosa unlink ocm-role \\\n"+
			"\t--role-arn %s", roleARN)
		commands = append(commands, unlinkRole)
	}

	if roleExistOnAWS {
		policies, err := awsClient.GetAttachedPolicy(&roleName)
		if err != nil {
			return "", err
		}
		for _, policy := range policies {
			detachPolicy := awscb.NewIAMCommandBuilder().
				SetCommand(awscb.DetachRolePolicy).
				AddParam(awscb.RoleName, roleName).
				AddParam(awscb.PolicyArn, policy.PolicyArn).
				Build()
			commands = append(commands, detachPolicy)

			if !managedPolicies {
				deletePolicy := awscb.NewIAMCommandBuilder().
					SetCommand(awscb.DeletePolicy).
					AddParam(awscb.PolicyArn, policy.PolicyArn).
					Build()
				commands = append(commands, deletePolicy)
			}
		}

		hasPermissionBoundary, err := awsClient.HasPermissionsBoundary(roleName)
		if err != nil {
			return "", err
		}
		if hasPermissionBoundary {
			deletePermissionBoundary := awscb.NewIAMCommandBuilder().
				SetCommand(awscb.DeleteRolePermissionsBoundary).
				AddParam(awscb.RoleName, roleName).
				Build()
			commands = append(commands, deletePermissionBoundary)
		}

		deleteRole := awscb.NewIAMCommandBuilder().
			SetCommand(awscb.DeleteRole).
			AddParam(awscb.RoleName, roleName).
			Build()
		commands = append(commands, deleteRole)
	}

	return awscb.JoinCommands(commands), nil
}
