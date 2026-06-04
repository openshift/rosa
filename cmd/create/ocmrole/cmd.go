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
	"errors"
	"fmt"
	"os"

	common "github.com/openshift-online/ocm-common/pkg/aws/validations"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	linkocmrole "github.com/openshift/rosa/cmd/link/ocmrole"
	internalocmrole "github.com/openshift/rosa/internal/ocmrole"
	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	awscb "github.com/openshift/rosa/pkg/aws/commandbuilder"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/roles"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	prefix              string
	permissionsBoundary string
	admin               bool
	path                string
	managed             bool
	noConsole           bool
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

	flags.BoolVar(
		&args.noConsole,
		"no-console",
		false,
		"Create OCM role with minimal permissions (cannot be used with console.redhat.com)",
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

	Cmd.MarkFlagsMutuallyExclusive("admin", "no-console")

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
	if roles.ClassicManagedPoliciesUnsupportedInEnv(isManagedSet, args.managed, env) {
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

	profile := internalocmrole.DetermineProfile(args.admin, args.noConsole)

	if interactive.Enabled() && profile == internalocmrole.ProfileStandard {
		profile, err = promptProfile(cmd, profile)
		if err != nil {
			r.Reporter.Errorf("Expected a valid --admin value: %s", err)
			os.Exit(1)
		}
	}

	if profile == internalocmrole.ProfileNoConsole {
		r.Reporter.Warnf("This OCM role cannot be used to provision clusters via console.redhat.com")
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

	// Validate no-console policy availability before any operations
	if profile == internalocmrole.ProfileNoConsole {
		filename := fmt.Sprintf("sts_%s_permission_policy", aws.OCMNoConsoleRolePolicyFile)
		policy, ok := policies[filename]
		// For managed policies, validate ARN exists
		// For customer-managed policies, validate Details exists (ARN is constructed later)
		if !ok || (managedPolicies && policy.ARN() == "") || (!managedPolicies && policy.Details() == "") {
			r.Reporter.Errorf("There was an error creating the ocm role: " +
				"the no-console OCM role profile is not yet enabled for your Organization")
			os.Exit(1)
		}
	}

	switch mode {
	case interactive.ModeAuto:
		r.Reporter.Infof("Creating role using '%s'", r.Creator.ARN)
		roleARN, err := createRoles(r, prefix, roleNameRequested, path, permissionsBoundary,
			orgID, env, profile, policies, managedPolicies)
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
		_, _, err = checkRoleExists(r, roleNameRequested, profile, interactive.ModeManual, path)
		if err != nil {
			r.Reporter.Warnf("Creating ocm role '%s' should fail: %s", roleNameRequested, err)
		}
		err = generateOcmRolePolicyFiles(r, env, orgID, profile, policies)
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
			profile,
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

func promptProfile(cmd *cobra.Command, curr internalocmrole.RoleProfile) (internalocmrole.RoleProfile, error) {
	isAdmin, err := interactive.GetBool(interactive.Input{
		Question: "Enable admin capabilities for the OCM role",
		Help:     cmd.Flags().Lookup("admin").Usage,
		Default:  curr == internalocmrole.ProfileAdmin,
		Required: false,
	})
	if err != nil {
		return internalocmrole.ProfileStandard, fmt.Errorf("expected a valid --admin value: %s", err)
	}
	if isAdmin {
		return internalocmrole.ProfileAdmin, nil
	}

	isNoConsole, err := interactive.GetBool(interactive.Input{
		Question: "Create OCM role with minimal permissions (no console access)",
		Help:     cmd.Flags().Lookup("no-console").Usage,
		Default:  curr == internalocmrole.ProfileNoConsole,
		Required: false,
	})
	if err != nil {
		return internalocmrole.ProfileStandard, fmt.Errorf("expected a valid --no-console value: %s", err)
	}
	if isNoConsole {
		return internalocmrole.ProfileNoConsole, nil
	}

	return internalocmrole.ProfileStandard, nil
}

func buildCommands(prefix string, roleName string, rolePath string, permissionsBoundary string,
	creator *aws.Creator, env string, profile internalocmrole.RoleProfile, managedPolicies bool, autoConfirmLink bool,
	policies map[string]*cmv1.AWSSTSPolicy,
) (string, error) {
	commands := []string{}
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

	noConsoleTags := map[string]string{
		tags.NoConsoleRole: tags.True,
	}

	builder := awscb.NewIAMCommandBuilder().
		SetCommand(awscb.CreateRole).
		AddParam(awscb.RoleName, roleName).
		AddParam(awscb.AssumeRolePolicyDocument, fmt.Sprintf("file://sts_%s_trust_policy.json", aws.OCMRolePolicyFile)).
		AddParam(awscb.PermissionsBoundary, permissionsBoundary).
		AddTags(iamTags).
		AddParam(awscb.Path, rolePath)

	var policyFile string
	var policyName string

	switch profile {
	case internalocmrole.ProfileAdmin:
		builder.AddTags(adminTags)

		policyFile = aws.OCMRolePolicyFile
		policyName = aws.GetPolicyName(roleName)
	case internalocmrole.ProfileNoConsole:
		builder.AddTags(noConsoleTags)

		policyFile = aws.OCMNoConsoleRolePolicyFile
		policyName = aws.GetNoConsolePolicyName(roleName)
	case internalocmrole.ProfileStandard:
		// No additional tags

		policyFile = aws.OCMRolePolicyFile
		policyName = aws.GetPolicyName(roleName)
	default:
		return "", fmt.Errorf("invalid profile: %s", profile)
	}

	createRole := builder.Build()

	var createPolicy string
	if !managedPolicies {
		createPolicy = awscb.NewIAMCommandBuilder().
			SetCommand(awscb.CreatePolicy).
			AddParam(awscb.PolicyName, policyName).
			AddParam(awscb.PolicyDocument, fmt.Sprintf("file://sts_%s_permission_policy.json", policyFile)).
			AddTags(iamTags).
			AddParam(awscb.Path, rolePath).
			Build()
	}

	var policyARN string
	var err error
	policyKey := fmt.Sprintf("sts_%s_permission_policy", policyFile)
	if managedPolicies {
		policyARN, err = aws.GetManagedPolicyARN(policies, policyKey)
		if err != nil {
			return "", err
		}
	} else {
		switch profile {
		case internalocmrole.ProfileNoConsole:
			policyARN = aws.GetNoConsolePolicyARN(creator.Partition, creator.AccountID, roleName, rolePath)
		case internalocmrole.ProfileAdmin, internalocmrole.ProfileStandard:
			policyARN = aws.GetPolicyArnWithSuffix(creator.Partition, creator.AccountID, roleName, rolePath)
		default:
			return "", fmt.Errorf("invalid profile: %s", profile)
		}
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

	if profile == internalocmrole.ProfileAdmin {
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
	permissionsBoundary string, orgID string, env string, profile internalocmrole.RoleProfile,
	policies map[string]*cmv1.AWSSTSPolicy, managedPolicies bool,
) (string, error) {
	if !confirm.Prompt(true, "Create the '%s' role?", roleName) {
		os.Exit(0)
	}

	roleARN, exists, err := checkRoleExists(r, roleName, profile, interactive.ModeAuto, rolePath)
	if err != nil {
		return "", err
	}
	if exists {
		return roleARN, nil
	}

	return internalocmrole.CreateRolesInternal(r, prefix, roleName, rolePath, permissionsBoundary,
		orgID, env, profile, policies, managedPolicies)
}

func generateOcmRolePolicyFiles(r *rosa.Runtime, env string, orgID string, profile internalocmrole.RoleProfile,
	policies map[string]*cmv1.AWSSTSPolicy,
) error {
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

	var policyFile string
	switch profile {
	case internalocmrole.ProfileNoConsole:
		policyFile = aws.OCMNoConsoleRolePolicyFile
	case internalocmrole.ProfileAdmin, internalocmrole.ProfileStandard:
		policyFile = aws.OCMRolePolicyFile
	default:
		return fmt.Errorf("invalid profile: %s", profile)
	}

	filename = fmt.Sprintf("sts_%s_permission_policy", policyFile)

	policyDetail = aws.GetPolicyDetails(policies, filename)
	filename = aws.GetFormattedFileName(filename)
	r.Reporter.Debugf("Saving '%s' to the current directory", filename)
	err = helper.SaveDocument(policyDetail, filename)
	if err != nil {
		return err
	}

	if profile == internalocmrole.ProfileAdmin {
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

func checkRoleExists(r *rosa.Runtime, roleName string, profile internalocmrole.RoleProfile,
	mode string, rolePath string,
) (string, bool, error) {
	roleARN, exists, err := internalocmrole.CheckRoleExistsInternal(r, roleName, profile, mode, rolePath)

	if err != nil && errors.Is(err, internalocmrole.ErrRoleExistsWrongProfile) {
		if mode == interactive.ModeAuto {
			if !confirm.Prompt(true, "Add admin policies to '%s' role?", roleName) {
				return roleARN, true, nil
			}

			// User accepted - return false to trigger upgrade/creation
			return "", false, nil
		}
	}

	return roleARN, exists, err
}
