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

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	linkuser "github.com/openshift/rosa/cmd/link/userrole"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	prefix              string
	permissionsBoundary string
	path                string
}

var Cmd = &cobra.Command{
	Use:     "user-role",
	Aliases: []string{"userrole"},
	Short:   "Create user role to verify account association",
	Long: "Create user role that allows Red Hat Hybrid Cloud Console to verify that users creating a cluster " +
		"have access to the current AWS account.",
	Example: `  # Create user roles
  rosa create user-role

  # Create user role with a specific permissions boundary
  rosa create user-role --permissions-boundary arn:aws:iam::123456789012:policy/perm-boundary`,
	Run:  run,
	Args: cobra.NoArgs,
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
	flags.StringVar(
		&args.path,
		"path",
		"",
		"The arn path for the user role and policies.",
	)

	interactive.AddModeFlag(Cmd)
	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, _ []string) {
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

	env, err := ocm.GetEnv()
	if err != nil {
		return fmt.Errorf("failed to determine OCM environment: %v", err)
	}

	// Determine if interactive mode is needed
	if !interactive.Enabled() && (!cmd.Flags().Changed("mode")) {
		interactive.Enable()
	}

	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("Creating User role")
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
			return fmt.Errorf("expected a valid role prefix: %s", err)
		}
	}
	if len(prefix) > 32 {
		return fmt.Errorf("expected a prefix with no more than 32 characters")
	}
	if !aws.RoleNameRE.MatchString(prefix) {
		return fmt.Errorf("expected a valid role prefix matching %s", aws.RoleNameRE.String())
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
			return fmt.Errorf("expected a valid policy ARN for permissions boundary: %s", err)
		}
	}

	if permissionsBoundary != "" {
		err = aws.ARNValidator(permissionsBoundary)
		if err != nil {
			return fmt.Errorf("expected a valid policy ARN for permissions boundary: %s", err)
		}
	}

	path := args.path
	if interactive.Enabled() {
		path, err = interactive.GetString(interactive.Input{
			Question: "Role Path",
			Help:     cmd.Flags().Lookup("path").Usage,
			Default:  path,
			Validators: []interactive.Validator{
				aws.ARNPathValidator,
			},
		})
		if err != nil {
			return fmt.Errorf("expected a valid path: %s", err)
		}
	}

	if path != "" && !aws.ARNPath.MatchString(path) {
		return fmt.Errorf("the specified value for path is invalid, " +
			"it must begin and end with '/' and contain only alphanumeric characters and/or '/' characters")
	}

	if interactive.Enabled() {
		mode, err = interactive.GetOptionMode(cmd, mode, "Role creation mode")
		if err != nil {
			return fmt.Errorf("expected a valid role creation mode: %s", err)
		}
	}

	// Get current OCM account:
	currentAccount, err := r.OCMClient.GetCurrentAccount()
	if err != nil {
		return fmt.Errorf("failed to get current account: %s", err)
	}

	policies, err := r.OCMClient.GetPolicies("")
	if err != nil {
		return fmt.Errorf("expected a valid role creation mode: %s", err)
	}

	switch mode {
	case interactive.ModeAuto:
		r.Reporter.Infof("Creating ocm user role using '%s'", r.Creator.ARN)
		roleARN, err := createRoles(r, prefix, path, currentAccount.Username(), env,
			currentAccount.ID(), permissionsBoundary, policies)
		if err != nil {
			r.OCMClient.LogEvent("ROSACreateUserRoleModeAuto", map[string]string{
				ocm.Response: ocm.Failure,
			})
			return fmt.Errorf("there was an error creating the ocm user role: %s", err)
		}
		if roleARN == "" {
			return nil
		}
		r.OCMClient.LogEvent("ROSACreateUserRoleModeAuto", map[string]string{
			ocm.Response: ocm.Success,
		})
		arguments.DisableRegionDeprecationWarning = true // disable region deprecation warning
		linkuser.Cmd.Run(linkuser.Cmd, []string{roleARN})
		arguments.DisableRegionDeprecationWarning = false // enable region deprecation again
	case interactive.ModeManual:
		r.OCMClient.LogEvent("ROSACreateUserRoleModeManual", map[string]string{})
		err = generateUserRolePolicyFiles(r.Reporter, env, r.Creator.Partition, currentAccount.ID(), policies)
		if err != nil {
			return fmt.Errorf("there was an error generating the policy files: %s", err)
		}
		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("All policy files saved to the current directory")
			r.Reporter.Infof("Run the following commands to create the account roles and policies:\n")
		}
		commands := buildCommands(
			prefix,
			path,
			currentAccount.Username(),
			r.Creator,
			env,
			permissionsBoundary,
		)
		fmt.Println(commands)

	default:
		return fmt.Errorf("invalid mode. Allowed values are %s", interactive.Modes)
	}
	return nil
}

func buildCommands(prefix string, path string, userName string,
	creator *aws.Creator, env string, permissionsBoundary string) string {
	commands := []string{}
	roleName := aws.GetUserRoleName(prefix, aws.OCMUserRole, userName)

	roleARN := aws.GetRoleARN(creator.AccountID, roleName, path, creator.Partition)
	iamTags := map[string]string{
		tags.RolePrefix:    prefix,
		tags.RoleType:      aws.OCMUserRole,
		tags.Environment:   env,
		tags.RedHatManaged: "true",
	}
	createRole := awscb.NewIAMCommandBuilder().
		SetCommand(awscb.CreateRole).
		AddParam(awscb.RoleName, roleName).
		AddParam(awscb.AssumeRolePolicyDocument,
			fmt.Sprintf("file://sts_%s_trust_policy.json", aws.OCMUserRolePolicyFile)).
		AddParam(awscb.PermissionsBoundary, permissionsBoundary).
		AddTags(iamTags).
		AddParam(awscb.Path, path).
		Build()
	linkRole := fmt.Sprintf("rosa link user-role --role-arn %s", roleARN)
	commands = append(commands, createRole, linkRole)
	return awscb.JoinCommands(commands)
}

func createRoles(r *rosa.Runtime,
	prefix string, path string, userName string, env string, accountID string, permissionsBoundary string,
	policies map[string]*cmv1.AWSSTSPolicy) (string, error) {
	roleName := aws.GetUserRoleName(prefix, aws.OCMUserRole, userName)
	if !confirm.Prompt(true, "Create the '%s' role?", roleName) {
		return "", nil
	}

	filename := fmt.Sprintf("sts_%s_trust_policy", aws.OCMUserRolePolicyFile)
	policyDetail := aws.GetPolicyDetails(policies, filename)
	policy := aws.InterpolatePolicyDocument(r.Creator.Partition, policyDetail, map[string]string{
		"partition":      r.Creator.Partition,
		"aws_account_id": aws.GetJumpAccount(env),
		"ocm_account_id": accountID,
	})

	exists, roleARN, err := r.AWSClient.CheckRoleExists(roleName)
	if err != nil {
		return "", err
	}
	if exists {
		r.Reporter.Warnf("Role '%s' already exists", roleName)
		return roleARN, nil
	}
	r.Reporter.Debugf("Creating role '%s'", roleName)
	roleARN, err = r.AWSClient.EnsureRole(r.Reporter, roleName, policy, permissionsBoundary,
		"", map[string]string{
			tags.RolePrefix:    prefix,
			tags.RoleType:      aws.OCMUserRole,
			tags.Environment:   env,
			tags.RedHatManaged: "true",
		}, path, false)
	if err != nil {
		return "", err
	}
	r.Reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)

	return roleARN, nil
}

func generateUserRolePolicyFiles(reporter reporter.Logger, env string, partition string, accountID string,
	policies map[string]*cmv1.AWSSTSPolicy) error {
	filename := fmt.Sprintf("sts_%s_trust_policy", aws.OCMUserRolePolicyFile)
	policyDetail := aws.GetPolicyDetails(policies, filename)
	policy := aws.InterpolatePolicyDocument(partition, policyDetail, map[string]string{
		"partition":      partition,
		"aws_account_id": aws.GetJumpAccount(env),
		"ocm_account_id": accountID,
	})

	filename = aws.GetFormattedFileName(filename)
	reporter.Debugf("Saving '%s' to the current directory", filename)
	err := helper.SaveDocument(policy, filename)
	if err != nil {
		return err
	}
	return nil
}
