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
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/spf13/cobra"

	unlinkocmrole "github.com/openshift/rosa/cmd/unlink/ocmrole"
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
	Use:     "ocm-role",
	Aliases: []string{"ocmrole"},
	Short:   "Delete ocm role",
	Long:    "Delete ocm role from the current AWS account",
	Example: ` # Delete ocm role
rosa delete ocm-role --role-arn arn:aws:iam::123456789012:role/xxx-OCM-Role-1223456778`,
	RunE: run,
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

func run(cmd *cobra.Command, argv []string) error {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.NewLogger()
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

	orgID, _, err := ocmClient.GetCurrentOrganization()
	if err != nil {
		reporter.Errorf("Error getting organization account: %v", err)
		return err
	}

	if len(argv) > 0 {
		args.roleARN = argv[0]
	}

	// Determine if interactive mode is needed
	if !interactive.Enabled() && (!cmd.Flags().Changed("mode")) {
		interactive.Enable()
	}

	if reporter.IsTerminal() {
		reporter.Infof("Deleting OCM role")
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
			reporter.Errorf("Expected a valid ocm role ARN to delete from the current organization: %s", err)
			os.Exit(1)
		}
	}
	if roleARN != "" {
		_, err := arn.Parse(roleARN)
		if err != nil {
			reporter.Errorf("Expected a valid ocm role ARN to delete from the current organization: %s", err)
			os.Exit(1)
		}
	}

	err = awsClient.ValidateRoleARNAccountIDMatchCallerAccountID(roleARN)
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	creator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Unable to get IAM credentials: %s", err)
		os.Exit(1)
	}

	if !confirm.Prompt(true, "Delete '%s' ocm role?", roleARN) {
		os.Exit(0)
	}

	linkedRoles, err := ocmClient.GetOrganizationLinkedOCMRoles(orgID)
	if err != nil {
		reporter.Errorf("An error occurred while trying to get the organization linked roles: %s", err)
		os.Exit(1)
	}
	isLinked := helper.Contains(linkedRoles, roleARN)

	if interactive.Enabled() {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "OCM role deletion mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  aws.ModeAuto,
			Options:  aws.Modes,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid OCM role deletion mode: %s", err)
			os.Exit(1)
		}
	}

	roleName, err := aws.RoleARNToRoleName(roleARN)
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if !aws.IsOCMRole(&roleName) {
		reporter.Errorf("Role '%s' is not an OCM role", roleName)
		os.Exit(1)
	}

	roleExistOnAWS, _, err := awsClient.CheckRoleExists(roleName)
	if err != nil {
		reporter.Errorf("%v", err)
	}
	if !roleExistOnAWS {
		reporter.Warnf("ocm-role '%s' doesn't exist on the aws account %s", roleName, creator.AccountID)
	}

	switch mode {
	case aws.ModeAuto:
		ocmClient.LogEvent("ROSADeleteOCMRoleModeAuto", nil)
		if isLinked {
			reporter.Warnf("Role ARN '%s' is linked to organization '%s'", roleARN, orgID)
			err = unlinkocmrole.Cmd.RunE(unlinkocmrole.Cmd, []string{roleARN})
			if err != nil {
				reporter.Errorf("Unable to unlink role ARN '%s' from organization : '%s' : %v",
					roleARN, orgID, err)
				os.Exit(1)
			}
		}
		if roleExistOnAWS {
			err := awsClient.DeleteOCMRole(roleName)
			if err != nil {
				reporter.Errorf("There was an error deleting the OCM role: %s", err)
				os.Exit(1)
			}
			reporter.Infof("Successfully deleted the OCM role")
		}
	case aws.ModeManual:
		ocmClient.LogEvent("ROSADeleteOCMRoleModeManual", nil)
		commands, err := buildCommands(roleName, roleARN, isLinked, awsClient, roleExistOnAWS)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
		if reporter.IsTerminal() {
			if roleExistOnAWS {
				reporter.Infof("Run the following commands to delete the OCM role:\n")
			} else if isLinked {
				reporter.Infof("Run the following commands to unlink the OCM role:\n")
			}
		}
		fmt.Println(commands)
	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}

	return nil
}

func buildCommands(roleName string, roleARN string, isLinked bool, awsClient aws.Client,
	roleExistOnAWS bool) (string, error) {
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
			detachPolicy := fmt.Sprintf("aws iam detach-role-policy \\\n"+
				"\t--role-name %s \\\n"+
				"\t--policy-arn %s",
				roleName, policy.PolicyArn)
			commands = append(commands, detachPolicy)
			deletePolicy := fmt.Sprintf("aws iam delete-policy \\\n"+
				"\t--policy-arn %s",
				policy.PolicyArn)
			commands = append(commands, deletePolicy)
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
	}

	return strings.Join(commands, "\n"), nil
}
