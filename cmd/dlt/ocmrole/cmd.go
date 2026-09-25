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
	"github.com/openshift/rosa/pkg/arguments"
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
	Short:   "Delete the Red Hat Hybrid Cloud Console role",
	Long:    "Delete the Red Hat Hybrid Cloud Console role from the current AWS organization",
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

	interactive.AddModeFlag(Cmd)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	if len(argv) > 0 {
		args.roleARN = argv[0]
	}

	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	if err := runWithRuntime(r, cmd); err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime, cmd *cobra.Command) error {
	mode, err := interactive.GetMode()
	if err != nil {
		return err
	}

	orgID, _, err := r.OCMClient.GetCurrentOrganization()
	if err != nil {
		return fmt.Errorf("error getting organization account: %v", err)
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
			return fmt.Errorf("expected a valid ocm role ARN to delete from the current organization: %s", err)
		}
	}

	err = aws.ARNValidator(roleARN)
	if err != nil {
		return fmt.Errorf("expected a valid ocm role ARN to delete from the current organization: %s", err)
	}

	err = r.AWSClient.ValidateRoleARNAccountIDMatchCallerAccountID(roleARN)
	if err != nil {
		return err
	}

	managedPolicies, err := r.AWSClient.HasManagedPolicies(roleARN)
	if err != nil {
		return fmt.Errorf("failed to determine if cluster has managed policies: %v", err)
	}

	if !confirm.Prompt(true, "Delete '%s' ocm role?", roleARN) {
		return nil
	}

	linkedRoles, err := r.OCMClient.GetOrganizationLinkedOCMRoles(orgID)
	if err != nil {
		return fmt.Errorf("an error occurred while trying to get the organization linked roles: %s", err)
	}
	isLinked := helper.Contains(linkedRoles, roleARN)

	if interactive.Enabled() && !cmd.Flags().Changed("mode") {
		mode, err = interactive.GetOptionMode(cmd, mode, "OCM role deletion mode")
		if err != nil {
			return fmt.Errorf("expected a valid OCM role deletion mode: %s", err)
		}
	}

	roleName, err := aws.GetResourceIdFromARN(roleARN)
	if err != nil {
		return err
	}

	if !aws.IsOCMRole(&roleName) {
		return fmt.Errorf("role '%s' is not an OCM role", roleName)
	}

	roleExistOnAWS, existingRoleARN, err := r.AWSClient.CheckRoleExists(roleName)
	if err != nil {
		r.Reporter.Errorf("%v", err)
	}
	if !roleExistOnAWS {
		r.Reporter.Warnf("the ARN %s does not exist. Nothing to delete", roleARN)
	} else if existingRoleARN != roleARN {
		r.Reporter.Warnf("role with same name but different ARN exists. Existing role ARN: %s", existingRoleARN)
		return fmt.Errorf("role with same name but different ARN exists. Existing role ARN: %s", existingRoleARN)
	}

	switch mode {
	case interactive.ModeAuto:
		r.OCMClient.LogEvent("ROSADeleteOCMRoleModeAuto", nil)
		if isLinked {
			r.Reporter.Warnf("Role ARN '%s' is linked to organization '%s'", roleARN, orgID)
			arguments.DisableRegionDeprecationWarning = true // disable region deprecation warning
			unlinkocmrole.Cmd.Run(unlinkocmrole.Cmd, []string{roleARN})
			arguments.DisableRegionDeprecationWarning = false // enable region deprecation again
		}
		if roleExistOnAWS {
			err := r.AWSClient.DeleteOCMRole(roleName, managedPolicies)
			if err != nil {
				return fmt.Errorf("there was an error deleting the OCM role: %s", err)
			}
			r.Reporter.Infof("Successfully deleted the OCM role")
		}
	case interactive.ModeManual:
		r.OCMClient.LogEvent("ROSADeleteOCMRoleModeManual", nil)
		commands, err := buildCommands(roleName, roleARN, isLinked, r.AWSClient, roleExistOnAWS, managedPolicies)
		if err != nil {
			return err
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
		return fmt.Errorf("invalid mode. Allowed values are %s", interactive.Modes)
	}
	return nil
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
