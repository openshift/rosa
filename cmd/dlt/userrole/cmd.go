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

package userrole

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/spf13/cobra"

	unlinkuserrole "github.com/openshift/rosa/cmd/unlink/userrole"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	roleARN string
}

var Cmd = &cobra.Command{
	Use:     "user-role",
	Aliases: []string{"userrole"},
	Short:   "Delete user role",
	Long:    "Delete user role from the current AWS account",
	Example: ` # Delete user role
rosa delete user-role --role-arn {prefix}-User-{username}-Role`,
	Run:    run,
	Hidden: true,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.roleARN,
		"role-arn",
		"",
		"Role ARN to delete from the user role from the AWS account")

	aws.AddModeFlag(Cmd)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)
	awsClient := aws.CreateNewClientOrExit(logger, reporter)
	ocmClient := ocm.CreateNewClientOrExit(logger, reporter)
	defer func() {
		err := ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	mode, err := aws.GetMode()
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if len(argv) > 0 {
		args.roleARN = argv[0]
	}

	// Determine if interactive mode is needed
	if !interactive.Enabled() && (!cmd.Flags().Changed("mode")) {
		interactive.Enable()
	}

	if reporter.IsTerminal() {
		reporter.Infof("Deleting user role")
	}

	roleARN := args.roleARN
	if interactive.Enabled() {
		roleARN, err = interactive.GetString(interactive.Input{
			Question: "User Role ARN",
			Help:     cmd.Flags().Lookup("role-arn").Usage,
			Default:  roleARN,
			Required: true,
			Validators: []interactive.Validator{
				aws.ARNValidator,
			},
		})
		if err != nil {
			reporter.Errorf("Expected a valid user role ARN to delete from the current AWS account: %s", err)
			os.Exit(1)
		}
	}
	if roleARN != "" {
		_, err := arn.Parse(roleARN)
		if err != nil {
			reporter.Errorf("Expected a valid user role ARN to delete from the current AWS account: %s", err)
			os.Exit(1)
		}
	}
	if !confirm.Prompt(true, "Delete the '%s' role from the AWS account?", roleARN) {
		os.Exit(0)
	}

	currentAccount, err := ocmClient.GetCurrentAccount()
	if err != nil {
		reporter.Errorf("Error getting current account: %v", err)
		os.Exit(1)
	}

	linkedRoles, err := ocmClient.GetAccountLinkedUserRoles(currentAccount.ID())
	if err != nil {
		reporter.Errorf("An error occurred while trying to get the account linked roles")
		os.Exit(1)
	}
	isLinked := helper.Contains(linkedRoles, roleARN)

	if interactive.Enabled() {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "Role creation mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  aws.ModeAuto,
			Options:  aws.Modes,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid role deletion mode: %s", err)
			os.Exit(1)
		}
	}

	roleName, err := aws.RoleARNToRoleName(roleARN)
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}
	switch mode {
	case aws.ModeAuto:
		ocmClient.LogEvent("ROSADeleteUserMRoleModeAuto", nil)
		if isLinked {
			reporter.Warnf("Role ARN '%s' is linked to account '%s'",
				roleARN, currentAccount.ID())
			err = unlinkuserrole.Cmd.RunE(unlinkuserrole.Cmd, []string{roleARN})
			if err != nil {
				reporter.Errorf("Unable to unlink role ARN '%s' from account : '%s' : '%v'",
					roleARN, currentAccount.ID(), err)
				os.Exit(1)
			}
		}
		err := awsClient.DeleteUserRole(roleName)
		if err != nil {
			reporter.Errorf("There was an error deleting the user role: %s", err)
			os.Exit(1)
		}
		reporter.Infof("Successfully deleted the user role")
	case aws.ModeManual:
		ocmClient.LogEvent("ROSADeleteUserMRoleModeManual", nil)
		commands, err := buildCommands(roleName, roleARN, isLinked, awsClient)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
		if reporter.IsTerminal() {
			reporter.Infof("Run the following commands to delete the user role:\n")
		}
		fmt.Println(commands)
	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
}

func buildCommands(roleName string, roleARN string, isLinked bool, awsClient aws.Client) (string, error) {
	var commands []string

	if isLinked {
		unlinkRole := fmt.Sprintf("rosa unlink user-role \\\n"+
			"\t--role-arn %s", roleARN)
		commands = append(commands, unlinkRole)
	}

	policies, err := awsClient.GetAttachedPolicy(&roleName)
	if err != nil {
		return "", err
	}
	for _, policy := range policies {
		detachPolicy := fmt.Sprintf("aws iam detach-role-policy \\\n"+
			"\t--role-name %s \\\n"+
			"\t--policy-arn %s",
			roleName, policy.PolicyArn)
		commands = append(commands, detachPolicy)
	}

	hasPermissionBoundary, err := awsClient.HasPermissionsBoundary(roleName)
	if err != nil {
		return "", err
	}
	if hasPermissionBoundary {
		deletePermissionBoundary := fmt.Sprintf("aws iam delete-role-permissions-boundary \\\n"+
			"\t--role-name %s",
			roleName)
		commands = append(commands, deletePermissionBoundary)
	}

	deleteRole := fmt.Sprintf("aws iam delete-role \\\n"+
		"\t--role-name %s", roleName)
	commands = append(commands, deleteRole)

	return strings.Join(commands, "\n"), nil
}
