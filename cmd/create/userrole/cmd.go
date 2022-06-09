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

package userrole

import (
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/spf13/cobra"

	linkuser "github.com/openshift/rosa/cmd/link/userrole"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	prefix              string
	permissionsBoundary string
}

var Cmd = &cobra.Command{
	Use:     "user-role",
	Aliases: []string{"userrole"},
	Short:   "Create user role to verify account association",
	Long: "Create user role that allows OCM to verify that users creating a cluster " +
		"have access to the current AWS account.",
	Example: `  # Create user roles
  rosa create user-role

  # Create user role with a specific permissions boundary
  rosa create user-role --permissions-boundary arn:aws:iam::123456789012:policy/perm-boundary`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.prefix,
		"prefix",
		aws.DefaultPrefix,
		"User-defined prefix for ocm-user role",
	)
	flags.StringVar(
		&args.permissionsBoundary,
		"permissions-boundary",
		"",
		"The ARN of the policy that is used to set the permissions boundary for the user role.",
	)

	aws.AddModeFlag(Cmd)
	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	mode, err := aws.GetMode()
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

	env, err := ocm.GetEnv()
	if err != nil {
		reporter.Errorf("Failed to determine OCM environment: %v", err)
		os.Exit(1)
	}

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
		reporter.Errorf("Unable to get IAM credentials: %s", err)
		os.Exit(1)
	}

	// Determine if interactive mode is needed
	if !interactive.Enabled() && (!cmd.Flags().Changed("mode")) {
		interactive.Enable()
	}

	if reporter.IsTerminal() {
		reporter.Infof("Creating User role")
	}

	prefix := args.prefix
	if interactive.Enabled() {
		prefix, err = interactive.GetString(interactive.Input{
			Question: "Role prefix",
			Help:     cmd.Flags().Lookup("prefix").Usage,
			Default:  prefix,
			Required: true,
			Validators: []interactive.Validator{
				interactive.RegExp(`[\w+=,.@-]+`),
				interactive.MaxLength(32),
			},
		})
		if err != nil {
			reporter.Errorf("Expected a valid role prefix: %s", err)
			os.Exit(1)
		}
	}
	if len(prefix) > 32 {
		reporter.Errorf("Expected a prefix with no more than 32 characters")
		os.Exit(1)
	}
	if !aws.RoleNameRE.MatchString(prefix) {
		reporter.Errorf("Expected a valid role prefix matching %s", aws.RoleNameRE.String())
		os.Exit(1)
	}
	permissionsBoundary := args.permissionsBoundary
	if interactive.Enabled() {
		permissionsBoundary, err = interactive.GetString(interactive.Input{
			Question: "Permissions boundary ARN",
			Help:     cmd.Flags().Lookup("permissions-boundary").Usage,
			Default:  permissionsBoundary,
			Validators: []interactive.Validator{
				aws.ARNValidator,
			},
		})
		if err != nil {
			reporter.Errorf("Expected a valid policy ARN for permissions boundary: %s", err)
			os.Exit(1)
		}
	}
	if permissionsBoundary != "" {
		_, err := arn.Parse(permissionsBoundary)
		if err != nil {
			reporter.Errorf("Expected a valid policy ARN for permissions boundary: %s", err)
			os.Exit(1)
		}
	}

	if interactive.Enabled() {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "Role creation mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  aws.ModeAuto,
			Options:  aws.Modes,
			Required: true,
		})
		if err != nil {
			reporter.Errorf("Expected a valid role creation mode: %s", err)
			os.Exit(1)
		}
	}

	// Get current OCM account:
	currentAccount, err := ocmClient.GetCurrentAccount()
	if err != nil {
		reporter.Errorf("Failed to get current account: %s", err)
		os.Exit(1)
	}

	policies, err := ocmClient.GetPolicies("")
	if err != nil {
		reporter.Errorf("Expected a valid role creation mode: %s", err)
		os.Exit(1)
	}

	switch mode {
	case aws.ModeAuto:
		reporter.Infof("Creating ocm user role using '%s'", creator.ARN)
		roleARN, err := createRoles(reporter, awsClient, prefix, currentAccount.Username(), env,
			currentAccount.ID(), permissionsBoundary, policies)
		if err != nil {
			reporter.Errorf("There was an error creating the ocm user role: %s", err)
			ocmClient.LogEvent("ROSACreateUserRoleModeAuto", map[string]string{
				ocm.Response: ocm.Failure,
			})
			os.Exit(1)
		}
		ocmClient.LogEvent("ROSACreateUserRoleModeAuto", map[string]string{
			ocm.Response: ocm.Success,
		})

		err = linkuser.Cmd.RunE(linkuser.Cmd, []string{roleARN})
		if err != nil {
			reporter.Errorf("Unable to link role arn '%s' with the account id : '%s' : %v",
				roleARN, currentAccount.ID(), err)
		}
	case aws.ModeManual:
		ocmClient.LogEvent("ROSACreateUserRoleModeManual", map[string]string{})
		err = generateUserRolePolicyFiles(reporter, env, currentAccount.ID(), policies)
		if err != nil {
			reporter.Errorf("There was an error generating the policy files: %s", err)
			os.Exit(1)
		}
		if reporter.IsTerminal() {
			reporter.Infof("All policy files saved to the current directory")
			reporter.Infof("Run the following commands to create the account roles and policies:\n")
		}
		commands := buildCommands(prefix, currentAccount.Username(), creator.AccountID, env, permissionsBoundary)
		fmt.Println(commands)

	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
}

func buildCommands(prefix string, userName string, accountID string, env string, permissionsBoundary string) string {
	commands := []string{}
	roleName := aws.GetUserRoleName(prefix, aws.OCMUserRole, userName)

	roleARN := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleName)
	iamTags := fmt.Sprintf(
		"Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s",
		tags.RolePrefix, prefix,
		tags.RoleType, aws.OCMUserRole,
		tags.Environment, env,
	)
	permBoundaryFlag := ""
	if permissionsBoundary != "" {
		permBoundaryFlag = fmt.Sprintf("\t--permissions-boundary %s \\\n", permissionsBoundary)
	}
	createRole := fmt.Sprintf("aws iam create-role \\\n"+
		"\t--role-name %s \\\n"+
		"\t--assume-role-policy-document file://sts_%s_trust_policy.json \\\n"+
		"%s"+
		"\t--tags %s",
		roleName, aws.OCMUserRolePolicyFile, permBoundaryFlag, iamTags)
	linkRole := fmt.Sprintf("rosa link user-role --role-arn %s", roleARN)
	commands = append(commands, createRole, linkRole)
	return strings.Join(commands, "\n\n")
}

func createRoles(reporter *rprtr.Object, awsClient aws.Client,
	prefix string, userName string, env string, accountID string, permissionsBoundary string,
	policies map[string]string) (string, error) {
	roleName := aws.GetUserRoleName(prefix, aws.OCMUserRole, userName)
	if !confirm.Prompt(true, "Create the '%s' role?", roleName) {
		os.Exit(0)
	}

	filename := fmt.Sprintf("sts_%s_trust_policy", aws.OCMUserRolePolicyFile)
	policyDetail := policies[filename]
	policy := aws.InterpolatePolicyDocument(policyDetail, map[string]string{
		"aws_account_id": aws.JumpAccounts[env],
		"ocm_account_id": accountID,
	})

	exists, roleARN, err := awsClient.CheckRoleExists(roleName)
	if err != nil {
		return "", err
	}
	if exists {
		reporter.Warnf("Role '%s' already exists", roleName)
		return roleARN, nil
	}
	reporter.Debugf("Creating role '%s'", roleName)
	roleARN, err = awsClient.EnsureRole(roleName, policy, permissionsBoundary,
		"", map[string]string{
			tags.RolePrefix:  prefix,
			tags.RoleType:    aws.OCMUserRole,
			tags.Environment: env,
		})
	if err != nil {
		return "", err
	}
	reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)

	return roleARN, nil
}

func generateUserRolePolicyFiles(reporter *rprtr.Object, env string, accountID string,
	policies map[string]string) error {
	filename := fmt.Sprintf("sts_%s_trust_policy", aws.OCMUserRolePolicyFile)
	policyDetail := policies[filename]
	policy := aws.InterpolatePolicyDocument(policyDetail, map[string]string{
		"aws_account_id": aws.JumpAccounts[env],
		"ocm_account_id": accountID,
	})

	filename = aws.GetFormattedFileName(filename)
	reporter.Debugf("Saving '%s' to the current directory", filename)
	err := aws.SaveDocument(policy, filename)
	if err != nil {
		return err
	}
	return nil
}
