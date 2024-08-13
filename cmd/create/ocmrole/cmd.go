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

package ocmrole

import (
	"fmt"
	"os"

	common "github.com/openshift-online/ocm-common/pkg/aws/validations"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	linkocmrole "github.com/openshift/rosa/cmd/link/ocmrole"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	prefix              string
	permissionsBoundary string
	admin               bool
	path                string
	managed             bool
}

var Cmd = &cobra.Command{
	Use:     "ocm-role",
	Aliases: []string{"ocmrole"},
	Short:   "Create role used by OCM",
	Long:    "Create role used by OCM to verify necessary roles and OIDC providers are in place.",
	Example: `  # Create default ocm role for ROSA clusters using STS
  rosa create ocm-role

  # Create ocm role with a specific permissions boundary
  rosa create ocm-role --permissions-boundary arn:aws:iam::123456789012:policy/perm-boundary`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.prefix,
		"prefix",
		aws.DefaultPrefix,
		"User-defined prefix for all generated AWS resources",
	)

	flags.StringVar(
		&args.permissionsBoundary,
		"permissions-boundary",
		"",
		"The ARN of the policy that is used to set the permissions boundary for the OCM role.",
	)
	flags.BoolVar(
		&args.admin,
		"admin",
		false,
		"Enable admin capabilities for the role",
	)

	flags.StringVar(
		&args.path,
		"path",
		"",
		"The arn path for the ocm role and policies",
	)

	// TODO: add `legacy-policies` once AWS managed policies are in place (managed will be the default)
	flags.BoolVar(
		&args.managed,
		"managed-policies",
		false,
		"Attach Classic ROSA AWS managed policies to the account roles",
	)
	flags.MarkHidden("managed-policies")
	flags.BoolVar(
		&args.managed,
		"mp",
		false,
		"Attach Classic ROSA AWS managed policies to the account roles. This is an alias for --managed-policies")
	flags.MarkHidden("mp")

	interactive.AddModeFlag(Cmd)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	mode, err := interactive.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	env, err := ocm.GetEnv()
	if err != nil {
		r.Reporter.Errorf("Failed to determine OCM environment: %v", err)
		os.Exit(1)
	}

	// Determine if Classic ROSA managed policies are enabled
	isManagedSet := cmd.Flags().Changed("managed-policies") || cmd.Flags().Changed("mp")
	if isManagedSet && env == ocm.Production {
		r.Reporter.Errorf("Classic ROSA managed policies are not supported in this environment")
		os.Exit(1)
	}
	managedPolicies := args.managed

	// Determine if interactive mode is needed
	if !interactive.Enabled() && (!cmd.Flags().Changed("mode")) {
		interactive.Enable()
	}

	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("Creating ocm role")
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

	isAdmin := args.admin

	if interactive.Enabled() && !isAdmin {
		isAdmin, err = interactive.GetBool(interactive.Input{
			Question: "Enable admin capabilities for the OCM role",
			Help:     cmd.Flags().Lookup("admin").Usage,
			Default:  isAdmin,
			Required: false,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid --admin value: %s", err)
			os.Exit(1)
		}
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
			r.Reporter.Errorf("Expected a valid policy ARN for permissions boundary: %s", err)
			os.Exit(1)
		}
	}

	if permissionsBoundary != "" {
		err = aws.ARNValidator(permissionsBoundary)
		if err != nil {
			r.Reporter.Errorf("Expected a valid policy ARN for permissions boundary: %s", err)
			os.Exit(1)
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
			r.Reporter.Errorf("Expected a valid path: %s", err)
			os.Exit(1)
		}
	}

	if path != "" && !aws.ARNPath.MatchString(path) {
		r.Reporter.Errorf("The specified value for path is invalid. " +
			"It must begin and end with '/' and contain only alphanumeric characters and/or '/' characters.")
		os.Exit(1)
	}

	if interactive.Enabled() {
		mode, err = interactive.GetOptionMode(cmd, mode, "Role creation mode")
		if err != nil {
			r.Reporter.Errorf("Expected a valid role creation mode: %s", err)
			os.Exit(1)
		}
	}

	// Get current OCM org account:
	orgID, externalID, err := r.OCMClient.GetCurrentOrganization()
	if err != nil {
		r.Reporter.Errorf("Failed to get organization account: %v", err)
		os.Exit(1)
	}

	roleNameRequested := aws.GetOCMRoleName(prefix, aws.OCMRole, externalID)

	existsOnOCM, _, selectedARN, err := r.OCMClient.CheckRoleExists(orgID, roleNameRequested, r.Creator.AccountID)

	if err != nil {
		r.Reporter.Errorf("Error checking existing ocm-role: %v", err)
		os.Exit(1)
	}
	if existsOnOCM {
		r.Reporter.Errorf("Only one ocm-role can be created per AWS account '%s' per organization '%s'.\n"+
			"In order to create a new ocm-role, you have to unlink the ocm-role '%s'.\n",
			r.Creator.AccountID, orgID, selectedARN)
		os.Exit(1)
	}

	policies, err := r.OCMClient.GetPolicies("OCMRole")
	if err != nil {
		r.Reporter.Errorf("Expected a valid role creation mode: %s", err)
		os.Exit(1)
	}

	switch mode {
	case interactive.ModeAuto:
		r.Reporter.Infof("Creating role using '%s'", r.Creator.ARN)
		roleARN, err := createRoles(r, prefix, roleNameRequested, path, permissionsBoundary,
			orgID, env, isAdmin, policies, managedPolicies)
		if err != nil {
			r.Reporter.Errorf("There was an error creating the ocm role: %s", err)
			r.OCMClient.LogEvent("ROSACreateOCMRoleModeAuto", map[string]string{
				ocm.Response: ocm.Failure,
			})
			os.Exit(1)
		}
		r.OCMClient.LogEvent("ROSACreateOCMRoleModeAuto", map[string]string{
			ocm.Response: ocm.Success,
		})
		arguments.DisableRegionDeprecationWarning = true // disable region deprecation warning
		linkocmrole.Cmd.Run(linkocmrole.Cmd, []string{roleARN})
		arguments.DisableRegionDeprecationWarning = false // enable region deprecation again
	case interactive.ModeManual:
		r.OCMClient.LogEvent("ROSACreateOCMRoleModeManual", map[string]string{})
		_, _, err = checkRoleExists(r, roleNameRequested, isAdmin, interactive.ModeManual)
		if err != nil {
			r.Reporter.Warnf("Creating ocm role '%s' should fail: %s", roleNameRequested, err)
		}
		err = generateOcmRolePolicyFiles(r, env, orgID, isAdmin, policies)
		if err != nil {
			r.Reporter.Errorf("There was an error generating the policy files: %s", err)
			r.OCMClient.LogEvent("ROSACreateOCMRoleModeManual", map[string]string{
				ocm.Response: ocm.Failure,
			})
			os.Exit(1)
		}
		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("All policy files saved to the current directory")
			r.Reporter.Infof("Run the following commands to create the ocm role and policies:\n")
		}
		var commands string
		commands, err = buildCommands(
			prefix,
			roleNameRequested,
			path,
			permissionsBoundary,
			r.Creator,
			env,
			isAdmin,
			managedPolicies,
			confirm.Yes(),
			policies,
		)
		if err != nil {
			r.Reporter.Errorf("Failed to generate commands for manual mode: %v", err)
			os.Exit(1)
		}

		fmt.Println(commands)
	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", interactive.Modes)
		os.Exit(1)
	}
}

func buildCommands(prefix string, roleName string, rolePath string, permissionsBoundary string,
	creator *aws.Creator, env string, isAdmin bool, managedPolicies bool, autoConfirmLink bool,
	policies map[string]*cmv1.AWSSTSPolicy) (string, error) {
	commands := []string{}
	policyName := aws.GetPolicyName(roleName)
	iamTags := map[string]string{
		tags.RolePrefix:    prefix,
		tags.RoleType:      aws.OCMRole,
		tags.Environment:   env,
		tags.RedHatManaged: tags.True,
	}
	if managedPolicies {
		iamTags[common.ManagedPolicies] = tags.True
	}

	adminTags := map[string]string{
		tags.AdminRole: tags.True,
	}

	builder := awscb.NewIAMCommandBuilder().
		SetCommand(awscb.CreateRole).
		AddParam(awscb.RoleName, roleName).
		AddParam(awscb.AssumeRolePolicyDocument, fmt.Sprintf("file://sts_%s_trust_policy.json", aws.OCMRolePolicyFile)).
		AddParam(awscb.PermissionsBoundary, permissionsBoundary).
		AddTags(iamTags).
		AddParam(awscb.Path, rolePath)
	if isAdmin {
		builder.AddTags(adminTags)
	}
	createRole := builder.Build()

	var createPolicy string
	if !managedPolicies {
		createPolicy = awscb.NewIAMCommandBuilder().
			SetCommand(awscb.CreatePolicy).
			AddParam(awscb.PolicyName, policyName).
			AddParam(awscb.PolicyDocument, fmt.Sprintf("file://sts_%s_permission_policy.json", aws.OCMRolePolicyFile)).
			AddTags(iamTags).
			AddParam(awscb.Path, rolePath).
			Build()
	}

	var policyARN string
	var err error
	if managedPolicies {
		policyARN, err = aws.GetManagedPolicyARN(policies, "sts_ocm_permission_policy")
		if err != nil {
			return "", err
		}
	} else {
		policyARN = aws.GetPolicyARN(creator.Partition, creator.AccountID, roleName, rolePath)
	}
	attachRolePolicy := awscb.NewIAMCommandBuilder().
		SetCommand(awscb.AttachRolePolicy).
		AddParam(awscb.RoleName, roleName).
		AddParam(awscb.PolicyArn, policyARN).
		Build()

	if managedPolicies {
		commands = append(commands, createRole, attachRolePolicy)
	} else {
		commands = append(commands, createRole, createPolicy, attachRolePolicy)
	}
	if isAdmin {
		policyName := aws.GetAdminPolicyName(roleName)

		var createAdminPolicy string
		if !managedPolicies {
			createAdminPolicy = awscb.NewIAMCommandBuilder().
				SetCommand(awscb.CreatePolicy).
				AddParam(awscb.PolicyName, policyName).
				AddParam(awscb.PolicyDocument, fmt.Sprintf("file://sts_%s_permission_policy.json", aws.OCMAdminRolePolicyFile)).
				AddTags(iamTags).
				AddTags(adminTags).
				AddParam(awscb.Path, rolePath).
				Build()
		}

		if managedPolicies {
			policyARN, err = aws.GetManagedPolicyARN(policies,
				fmt.Sprintf("sts_%s_permission_policy", aws.OCMAdminRolePolicyFile))
			if err != nil {
				return "", err
			}
		} else {
			policyARN = aws.GetAdminPolicyARN(creator.Partition, creator.AccountID, roleName, rolePath)
		}
		attachRoleAdminPolicy := awscb.NewIAMCommandBuilder().
			SetCommand(awscb.AttachRolePolicy).
			AddParam(awscb.RoleName, roleName).
			AddParam(awscb.PolicyArn, policyARN).
			Build()

		if managedPolicies {
			commands = append(commands, attachRoleAdminPolicy)
		} else {
			commands = append(commands, createAdminPolicy, attachRoleAdminPolicy)
		}
	}

	linkRole := fmt.Sprintf("rosa link ocm-role --role-arn %s",
		aws.GetRoleARN(creator.AccountID, roleName, rolePath, creator.Partition))
	if autoConfirmLink {
		linkRole += " -y"
	}
	commands = append(commands, linkRole)

	return awscb.JoinCommands(commands), nil
}

func createRoles(r *rosa.Runtime, prefix string, roleName string, rolePath string,
	permissionsBoundary string, orgID string, env string, isAdmin bool,
	policies map[string]*cmv1.AWSSTSPolicy, managedPolicies bool) (string, error) {
	var policyARN string
	var err error

	if managedPolicies {
		policyARN, err = aws.GetManagedPolicyARN(policies, "sts_ocm_permission_policy")
		if err != nil {
			return "", err
		}
	} else {
		policyARN = aws.GetPolicyARN(r.Creator.Partition, r.Creator.AccountID, roleName, rolePath)
	}
	if !confirm.Prompt(true, "Create the '%s' role?", roleName) {
		os.Exit(0)
	}
	filename := fmt.Sprintf("sts_%s_trust_policy", aws.OCMRolePolicyFile)
	policyDetail := aws.GetPolicyDetails(policies, filename)
	policy := aws.InterpolatePolicyDocument(r.Creator.Partition, policyDetail, map[string]string{
		"partition":           r.Creator.Partition,
		"aws_account_id":      aws.GetJumpAccount(env),
		"ocm_organization_id": orgID,
	})

	roleARN, exists, err := checkRoleExists(r, roleName, isAdmin, interactive.ModeAuto)
	if err != nil {
		return "", err
	}
	if exists {
		return roleARN, nil
	}

	iamTags := map[string]string{
		tags.RolePrefix:    prefix,
		tags.RoleType:      aws.OCMRole,
		tags.Environment:   env,
		tags.RedHatManaged: tags.True,
	}
	if managedPolicies {
		iamTags[common.ManagedPolicies] = tags.True
	}

	if !exists {
		r.Reporter.Debugf("Creating role '%s'", roleName)

		roleARN, err = r.AWSClient.EnsureRole(roleName, policy, permissionsBoundary,
			"", iamTags, rolePath, false)
		if err != nil {
			return "", err
		}
		r.Reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)

		// create and attach the permission policy to the role
		filename = fmt.Sprintf("sts_%s_permission_policy", aws.OCMRolePolicyFile)
		policyDetail = aws.GetPolicyDetails(policies, filename)
		err = createPermissionPolicy(r, policyARN, iamTags, roleName, rolePath, policyDetail, managedPolicies)
		if err != nil {
			return "", err
		}
	}

	if isAdmin {
		// tag role with admin tag
		err = r.AWSClient.AddRoleTag(roleName, tags.AdminRole, tags.True)
		if err != nil {
			return "", err
		}

		// create and attach the admin policy to the role
		filename = fmt.Sprintf("sts_%s_permission_policy", aws.OCMAdminRolePolicyFile)
		if managedPolicies {
			policyARN, err = aws.GetManagedPolicyARN(policies, filename)
			if err != nil {
				return "", err
			}
		} else {
			policyARN = aws.GetAdminPolicyARN(r.Creator.Partition, r.Creator.AccountID, roleName, "")
		}
		iamTags[tags.AdminRole] = tags.True
		policyDetail = aws.GetPolicyDetails(policies, filename)
		err = createPermissionPolicy(r, policyARN, iamTags, roleName, rolePath, policyDetail, managedPolicies)
		if err != nil {
			return "", err
		}
	}

	return roleARN, nil
}

func generateOcmRolePolicyFiles(r *rosa.Runtime, env string, orgID string, isAdmin bool,
	policies map[string]*cmv1.AWSSTSPolicy) error {
	filename := fmt.Sprintf("sts_%s_trust_policy", aws.OCMRolePolicyFile)

	policyDetail := aws.GetPolicyDetails(policies, filename)
	policy := aws.InterpolatePolicyDocument(r.Creator.Partition, policyDetail, map[string]string{
		"partition":           r.Creator.Partition,
		"aws_account_id":      aws.GetJumpAccount(env),
		"ocm_organization_id": orgID,
	})
	r.Reporter.Debugf("Saving '%s' to the current directory", filename)
	filename = aws.GetFormattedFileName(filename)
	err := helper.SaveDocument(policy, filename)
	if err != nil {
		return err
	}
	filename = fmt.Sprintf("sts_%s_permission_policy", aws.OCMRolePolicyFile)
	policyDetail = aws.GetPolicyDetails(policies, filename)
	filename = aws.GetFormattedFileName(filename)
	r.Reporter.Debugf("Saving '%s' to the current directory", filename)
	err = helper.SaveDocument(policyDetail, filename)
	if err != nil {
		return err
	}

	if isAdmin {
		filename = fmt.Sprintf("sts_%s_admin_permission_policy", aws.OCMRolePolicyFile)
		policyDetail = aws.GetPolicyDetails(policies, filename)
		filename = aws.GetFormattedFileName(filename)
		r.Reporter.Debugf("Saving '%s' to the current directory", filename)
		err = helper.SaveDocument(policyDetail, filename)
		if err != nil {
			return err
		}
	}
	return nil
}

func createPermissionPolicy(r *rosa.Runtime, policyARN string,
	iamTags map[string]string, roleName string, rolePath string, policyDetail string, managedPolicies bool) error {

	r.Reporter.Debugf("Creating permission policy '%s'", policyARN)
	if !managedPolicies {
		var err error
		policyARN, err = r.AWSClient.EnsurePolicy(policyARN, policyDetail, "", iamTags, rolePath)
		if err != nil {
			return err
		}
	}

	r.Reporter.Debugf("Attaching permission policy to role '%s'", roleName)
	err := r.AWSClient.AttachRolePolicy(r.Reporter, roleName, policyARN)
	if err != nil {
		return err
	}

	return nil
}

func checkRoleExists(r *rosa.Runtime, roleName string, isAdmin bool,
	mode string) (string, bool, error) {
	exists, roleARN, err := r.AWSClient.CheckRoleExists(roleName)
	if err != nil {
		return "", false, err
	}
	if exists {
		isExistingRoleAdmin, err := r.AWSClient.IsAdminRole(roleName)
		if err != nil {
			return "", true, err
		}
		r.Reporter.Warnf("Role '%s' already exists", roleName)

		if !isAdmin {
			if isExistingRoleAdmin {
				return "", true, fmt.Errorf("the existing role is an admin role."+
					" To remove admin capabilities please delete the admin policy and the '%s' tag",
					tags.AdminRole)
			}
			return roleARN, true, nil
		}

		if isExistingRoleAdmin {
			return roleARN, true, nil
		}

		if mode == interactive.ModeAuto && !confirm.Prompt(true, "Add admin policies to '%s' role?", roleName) {
			return roleARN, true, nil
		}
	}

	return "", false, nil
}
