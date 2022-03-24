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
	"strings"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/spf13/cobra"

	linkocmrole "github.com/openshift/rosa/cmd/link/ocmrole"
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
	admin               bool
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
	Run:    run,
	Hidden: true,
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
		reporter.Infof("Creating ocm role")
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

	isAdmin := args.admin

	if interactive.Enabled() && !isAdmin {
		isAdmin, err = interactive.GetBool(interactive.Input{
			Question: "Enable admin capabilities for the OCM role",
			Help:     cmd.Flags().Lookup("admin").Usage,
			Default:  isAdmin,
			Required: false,
		})
		if err != nil {
			reporter.Errorf("Expected a valid --admin value: %s", err)
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

	// Get current OCM org account:
	orgID, externalID, err := ocmClient.GetCurrentOrganization()
	if err != nil {
		reporter.Errorf("Failed to get organization account: %v", err)
		os.Exit(1)
	}

	roleNameRequested := aws.GetOCMRoleName(prefix, aws.OCMRole, externalID)

	existsOnOCM, existingRole, selectedARN, err := ocmClient.CheckRoleExists(orgID, roleNameRequested, creator.AccountID)

	if err != nil {
		reporter.Errorf("Error checking existing ocm-role: %v", err)
		os.Exit(1)
	}
	if existsOnOCM {
		reporter.Errorf("User organization '%s' has ocm-role '%s' for aws account %s. "+
			"Only one role can be created per AWS account per organization.\n"+
			"run the following command to unlink the role from the organization \n\n"+
			"\t rosa unlink ocm-role --role-arn %s\n",
			orgID, existingRole, creator.AccountID, selectedARN)

		existOnAWS, _, err := awsClient.CheckRoleExists(existingRole)
		if err != nil {
			reporter.Errorf("%v", err)
		}
		if !existOnAWS {
			reporter.Warnf("ocm-role '%s' doesn't exist on the aws account %s", existingRole, creator.AccountID)
		}
		os.Exit(1)
	}
	switch mode {
	case aws.ModeAuto:
		reporter.Infof("Creating role using '%s'", creator.ARN)
		roleARN, err := createRoles(reporter, awsClient, prefix, roleNameRequested, permissionsBoundary, creator.AccountID,
			orgID, env, isAdmin)
		if err != nil {
			reporter.Errorf("There was an error creating the ocm role: %s", err)
			ocmClient.LogEvent("ROSACreateOCMRoleModeAuto", map[string]string{
				ocm.Response: ocm.Failure,
			})
			os.Exit(1)
		}
		ocmClient.LogEvent("ROSACreateOCMRoleModeAuto", map[string]string{
			ocm.Response: ocm.Success,
		})
		err = linkocmrole.Cmd.RunE(linkocmrole.Cmd, []string{roleARN})
		if err != nil {
			reporter.Errorf("Unable to link role arn '%s' with the organization account id : '%s' : %v",
				roleARN, orgID, err)
		}
	case aws.ModeManual:
		ocmClient.LogEvent("ROSACreateOCMRoleModeManual", map[string]string{})
		_, _, err = checkRoleExists(reporter, awsClient, roleNameRequested, isAdmin, aws.ModeManual)
		if err != nil {
			reporter.Warnf("Creating ocm role '%s' should fail: %s", roleNameRequested, err)
		}
		err = generateOcmRolePolicyFiles(reporter, env, orgID, isAdmin)
		if err != nil {
			reporter.Errorf("There was an error generating the policy files: %s", err)
			ocmClient.LogEvent("ROSACreateOCMRoleModeManual", map[string]string{
				ocm.Response: ocm.Failure,
			})
			os.Exit(1)
		}
		if reporter.IsTerminal() {
			reporter.Infof("All policy files saved to the current directory")
			reporter.Infof("Run the following commands to create the ocm role and policies:\n")
		}
		commands := buildCommands(prefix, roleNameRequested, permissionsBoundary, creator.AccountID, env, isAdmin)
		fmt.Println(commands)
	default:
		reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
}

func buildCommands(prefix string, roleName string, permissionsBoundary string,
	accountID string, env string, isAdmin bool) string {
	commands := []string{}
	policyName := fmt.Sprintf("%s-Policy", roleName)
	iamTags := fmt.Sprintf(
		"Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s",
		tags.RolePrefix, prefix,
		tags.RoleType, aws.OCMRole,
		tags.Environment, env,
	)

	adminTags := ""
	if isAdmin {
		adminTags += fmt.Sprintf(" Key=%s,Value=true", tags.AdminRole)
	}

	permBoundaryFlag := ""
	if permissionsBoundary != "" {
		permBoundaryFlag = fmt.Sprintf("\t--permissions-boundary %s \\\n", permissionsBoundary)
	}
	createRole := fmt.Sprintf("aws iam create-role \\\n"+
		"\t--role-name %s \\\n"+
		"\t--assume-role-policy-document file://sts_%s_trust_policy.json \\\n"+
		"%s"+
		"\t--tags %s%s",
		roleName, aws.OCMRolePolicyFile, permBoundaryFlag, iamTags, adminTags)
	createPolicy := fmt.Sprintf("aws iam create-policy \\\n"+
		"\t--policy-name %s \\\n"+
		"\t--policy-document file://sts_%s_permission_policy.json \\\n"+
		"\t--tags %s",
		policyName, aws.OCMRolePolicyFile, iamTags)
	attachRolePolicy := fmt.Sprintf("aws iam attach-role-policy \\\n"+
		"\t--role-name %s \\\n"+
		"\t--policy-arn %s",
		roleName, aws.GetPolicyARN(accountID, policyName))

	commands = append(commands, createRole, createPolicy, attachRolePolicy)
	if isAdmin {
		adminTags := fmt.Sprintf("Key=%s,Value=%v", tags.AdminRole, true)
		policyName := fmt.Sprintf("%s-Admin-Policy", roleName)

		createAdminPolicy := fmt.Sprintf("aws iam create-policy \\\n"+
			"\t--policy-name %s \\\n"+
			"\t--policy-document file://sts_%s_permission_policy.json \\\n"+
			"\t--tags %s",
			policyName, aws.OCMAdminRolePolicyFile, adminTags)
		attachRoleAdminPolicy := fmt.Sprintf("aws iam attach-role-policy \\\n"+
			"\t--role-name %s \\\n"+
			"\t--policy-arn %s",
			roleName, aws.GetPolicyARN(accountID, policyName))

		commands = append(commands, createAdminPolicy, attachRoleAdminPolicy)
	}

	linkRole := fmt.Sprintf("rosa link ocm-role --role-arn %s",
		aws.GetRoleARN(accountID, roleName))
	commands = append(commands, linkRole)

	return strings.Join(commands, "\n\n")
}

func createRoles(reporter *rprtr.Object, awsClient aws.Client, prefix string, roleName string,
	permissionsBoundary string, accountID string, orgID string, env string, isAdmin bool) (string, error) {
	policyARN := aws.GetPolicyARN(accountID, fmt.Sprintf("%s-Policy", roleName))
	if !confirm.Prompt(true, "Create the '%s' role?", roleName) {
		os.Exit(0)
	}

	filename := fmt.Sprintf("sts_%s_trust_policy.json", aws.OCMRolePolicyFile)
	path := fmt.Sprintf("templates/policies/%s", filename)

	policy, err := aws.ReadPolicyDocument(path, map[string]string{
		"aws_account_id":      aws.JumpAccounts[env],
		"ocm_organization_id": orgID,
	})
	if err != nil {
		return "", err
	}

	roleARN, exists, err := checkRoleExists(reporter, awsClient, roleName, isAdmin, aws.ModeAuto)
	if err != nil {
		return "", err
	}
	if exists {
		return roleARN, nil
	}

	iamTags := map[string]string{
		tags.RolePrefix:  prefix,
		tags.RoleType:    aws.OCMRole,
		tags.Environment: env,
	}

	if !exists {
		reporter.Debugf("Creating role '%s'", roleName)

		roleARN, err = awsClient.EnsureRole(roleName, string(policy), permissionsBoundary,
			"", iamTags)
		if err != nil {
			return "", err
		}
		reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)

		// create and attach the permission policy to the role
		filename = fmt.Sprintf("sts_%s_permission_policy.json", aws.OCMRolePolicyFile)
		err = createPermissionPolicy(reporter, awsClient, policyARN, iamTags, roleName, filename)
		if err != nil {
			return "", err
		}
	}

	if isAdmin {
		// tag role with admin tag
		err = awsClient.AddRoleTag(roleName, tags.AdminRole, "true")
		if err != nil {
			return "", err
		}

		// create and attach the admin policy to the role
		policyARN := aws.GetPolicyARN(accountID, fmt.Sprintf("%s-Admin-Policy", roleName))
		filename = fmt.Sprintf("sts_%s_permission_policy.json", aws.OCMAdminRolePolicyFile)
		iamTags[tags.AdminRole] = "true"
		err = createPermissionPolicy(reporter, awsClient, policyARN, iamTags, roleName, filename)
		if err != nil {
			return "", err
		}
	}

	return roleARN, nil
}

func generateOcmRolePolicyFiles(reporter *rprtr.Object, env string, orgID string, isAdmin bool) error {
	filename := fmt.Sprintf("sts_%s_trust_policy.json", aws.OCMRolePolicyFile)
	path := fmt.Sprintf("templates/policies/%s", filename)
	policy, err := aws.ReadPolicyDocument(path, map[string]string{
		"aws_account_id":      aws.JumpAccounts[env],
		"ocm_organization_id": orgID,
	})
	if err != nil {
		return err
	}
	reporter.Debugf("Saving '%s' to the current directory", filename)
	err = aws.SaveDocument(policy, filename)
	if err != nil {
		return err
	}

	filename = fmt.Sprintf("sts_%s_permission_policy.json", aws.OCMRolePolicyFile)
	path = fmt.Sprintf("templates/policies/%s", filename)
	policy, err = aws.ReadPolicyDocument(path)
	if err != nil {
		return err
	}
	reporter.Debugf("Saving '%s' to the current directory", filename)
	err = aws.SaveDocument(policy, filename)
	if err != nil {
		return err
	}

	if isAdmin {
		filename = fmt.Sprintf("sts_%s_admin_permission_policy.json", aws.OCMRolePolicyFile)
		path = fmt.Sprintf("templates/policies/%s", filename)
		policy, err = aws.ReadPolicyDocument(path)
		if err != nil {
			return err
		}
		reporter.Debugf("Saving '%s' to the current directory", filename)
		err = aws.SaveDocument(policy, filename)
		if err != nil {
			return err
		}
	}
	return nil
}

func createPermissionPolicy(reporter *rprtr.Object, awsClient aws.Client, policyARN string,
	iamTags map[string]string, roleName string, filename string) error {
	path := fmt.Sprintf("templates/policies/%s", filename)

	policy, err := aws.ReadPolicyDocument(path)
	if err != nil {
		return err
	}

	reporter.Debugf("Creating permission policy '%s'", policyARN)
	policyARN, err = awsClient.EnsurePolicy(policyARN, string(policy),
		"", iamTags)
	if err != nil {
		return err
	}

	reporter.Debugf("Attaching permission policy to role '%s'", filename)
	err = awsClient.AttachRolePolicy(roleName, policyARN)
	if err != nil {
		return err
	}

	return nil
}

func checkRoleExists(reporter *rprtr.Object, awsClient aws.Client, roleName string, isAdmin bool,
	mode string) (string, bool, error) {
	exists, roleARN, err := awsClient.CheckRoleExists(roleName)
	if err != nil {
		return "", false, err
	}
	if exists {
		isExistingRoleAdmin, err := awsClient.IsAdminRole(roleName)
		if err != nil {
			return "", true, err
		}
		reporter.Warnf("Role '%s' already exists", roleName)

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

		if mode == aws.ModeAuto && !confirm.Prompt(true, "Add admin policies to '%s' role?", roleName) {
			return roleARN, true, nil
		}
	}

	return "", false, nil
}
