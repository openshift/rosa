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

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/login"
	"github.com/openshift/rosa/cmd/verify/oc"
	"github.com/openshift/rosa/cmd/verify/quota"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	prefix              string
	permissionsBoundary string
	path                string
	version             string
	channelGroup        string
}

var Cmd = &cobra.Command{
	Use:     "account-roles",
	Aliases: []string{"accountroles", "roles", "policies"},
	Short:   "Create account-wide IAM roles before creating your cluster.",
	Long:    "Create account-wide IAM roles before creating your cluster.",
	Example: `  # Create default account roles for ROSA clusters using STS
  rosa create account-roles

  # Create account roles with a specific permissions boundary
  rosa create account-roles --permissions-boundary arn:aws:iam::123456789012:policy/perm-boundary`,
	Run: run,
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
		"The ARN of the policy that is used to set the permissions boundary for the account roles.",
	)

	flags.StringVar(
		&args.path,
		"path",
		"",
		"The arn path for the account/operator roles as well as their policies",
	)

	flags.StringVar(
		&args.version,
		"version",
		"",
		"Version of OpenShift that will be used to setup policy tag, for example \"4.11\"",
	)
	flags.MarkHidden("version")

	flags.StringVar(
		&args.channelGroup,
		"channel-group",
		ocm.DefaultChannelGroup,
		"Channel group is the name of the channel where this image belongs, for example \"stable\" or \"fast\".",
	)
	flags.MarkHidden("channel-group")

	aws.AddModeFlag(Cmd)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime()

	mode, err := aws.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// If necessary, call `login` as part of `init`. We do this before
	// other validations to get the prompt out of the way before performing
	// longer checks.
	err = login.Call(cmd, argv, r.Reporter)
	if err != nil {
		r.Reporter.Errorf("Failed to login to OCM: %v", err)
		os.Exit(1)
	}
	r.WithOCM()
	defer r.Cleanup()

	env, err := ocm.GetEnv()
	if err != nil {
		r.Reporter.Errorf("Failed to determine OCM environment: %v", err)
		os.Exit(1)
	}

	r.WithAWS()
	// Validate AWS credentials for current user
	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("Validating AWS credentials...")
	}
	ok, err := r.AWSClient.ValidateCredentials()
	if err != nil {
		r.OCMClient.LogEvent("ROSAInitCredentialsFailed", nil)
		r.Reporter.Errorf("Error validating AWS credentials: %v", err)
		os.Exit(1)
	}
	if !ok {
		r.OCMClient.LogEvent("ROSAInitCredentialsInvalid", nil)
		r.Reporter.Errorf("AWS credentials are invalid")
		os.Exit(1)
	}
	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("AWS credentials are valid!")
	}

	// Validate AWS quota
	// Call `verify quota` as part of init
	err = quota.Cmd.RunE(cmd, argv)
	if err != nil {
		r.Reporter.Warnf("Insufficient AWS quotas. Cluster installation might fail.")
	}
	// Verify version of `oc`
	oc.Cmd.Run(cmd, argv)

	// Determine if interactive mode is needed
	if !interactive.Enabled() && (!cmd.Flags().Changed("mode")) {
		interactive.Enable()
	}

	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("Creating account roles")
	}

	version := args.version
	channelGroup := args.channelGroup
	policyVersion, err := r.OCMClient.GetVersion(version, channelGroup)
	if err != nil {
		r.Reporter.Errorf("Error getting version: %s", err)
		os.Exit(1)
	}

	r.Reporter.Debugf("Creating account roles compatible with OpenShift versions up to %s", policyVersion)

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
		_, err := arn.Parse(permissionsBoundary)
		if err != nil {
			r.Reporter.Errorf("Expected a valid policy ARN for permissions boundary: %s", err)
			os.Exit(1)
		}
	}

	path := args.path
	if interactive.Enabled() {
		path, err = interactive.GetString(interactive.Input{
			Question: "Path",
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

	if interactive.Enabled() {
		mode, err = interactive.GetOption(interactive.Input{
			Question: "Role creation mode",
			Help:     cmd.Flags().Lookup("mode").Usage,
			Default:  aws.ModeAuto,
			Options:  aws.Modes,
			Required: true,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid role creation mode: %s", err)
			os.Exit(1)
		}
	}
	policies, err := r.OCMClient.GetPolicies("")
	if err != nil {
		r.Reporter.Errorf("Expected a valid role creation mode: %s", err)
		os.Exit(1)
	}

	switch mode {
	case aws.ModeAuto:
		r.Reporter.Infof("Creating roles using '%s'", r.Creator.ARN)

		err = createRoles(r, prefix, permissionsBoundary, r.Creator.AccountID, env, policies,
			policyVersion, path)
		if err != nil {
			r.Reporter.Errorf("There was an error creating the account roles: %s", err)
			if strings.Contains(err.Error(), "Throttling") {
				r.OCMClient.LogEvent("ROSACreateAccountRolesModeAuto", map[string]string{
					ocm.Response:   ocm.Failure,
					ocm.Version:    policyVersion,
					ocm.IsThrottle: "true",
				})
				os.Exit(1)
			}
			r.OCMClient.LogEvent("ROSACreateAccountRolesModeAuto", map[string]string{
				ocm.Response: ocm.Failure,
			})
			os.Exit(1)
		}
		r.Reporter.Infof("To create a cluster with these roles, run the following command:\n" +
			"rosa create cluster --sts")
		r.OCMClient.LogEvent("ROSACreateAccountRolesModeAuto", map[string]string{
			ocm.Response: ocm.Success,
			ocm.Version:  policyVersion,
		})
	case aws.ModeManual:
		err = aws.GeneratePolicyFiles(r.Reporter, env, true, false, policies, nil)
		if err != nil {
			r.Reporter.Errorf("There was an error generating the policy files: %s", err)
			r.OCMClient.LogEvent("ROSACreateAccountRolesModeManual", map[string]string{
				ocm.Response: ocm.Failure,
			})
			os.Exit(1)
		}
		commands := buildCommands(prefix, permissionsBoundary, r.Creator.AccountID, policyVersion,
			path)
		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("All policy files saved to the current directory")
			r.Reporter.Infof("Run the following commands to create the account roles and policies:\n")
		}
		r.OCMClient.LogEvent("ROSACreateAccountRolesModeManual", map[string]string{
			ocm.Version: policyVersion,
		})
		fmt.Println(commands)
	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", aws.Modes)
		os.Exit(1)
	}
}

func buildCommands(prefix string, permissionsBoundary string, accountID string, defaultPolicyVersion string,
	path string) string {
	commands := []string{}

	for file, role := range aws.AccountRoles {
		name := aws.GetRoleName(prefix, role.Name)
		policyName := fmt.Sprintf("%s-Policy", name)
		iamTags := fmt.Sprintf(
			"Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s Key=%s,Value=%s",
			tags.OpenShiftVersion, defaultPolicyVersion,
			tags.RolePrefix, prefix,
			tags.RoleType, file,
			tags.RedHatManaged, "true",
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
			name, file, permBoundaryFlag, iamTags)

		if path != "" {
			createRole = fmt.Sprintf(createRole+"\t--path %s", path)
		}
		createPolicy := fmt.Sprintf("aws iam create-policy \\\n"+
			"\t--policy-name %s \\\n"+
			"\t--policy-document file://sts_%s_permission_policy.json"+
			"\t--tags %s",
			policyName, file, iamTags)
		if path != "" {
			createPolicy = fmt.Sprintf(createPolicy+"\t--path %s", path)
		}
		attachRolePolicy := fmt.Sprintf("aws iam attach-role-policy \\\n"+
			"\t--role-name %s \\\n"+
			"\t--policy-arn %s",
			name, aws.GetPolicyARN(accountID, policyName, path))
		commands = append(commands, createRole, createPolicy, attachRolePolicy)
	}

	return strings.Join(commands, "\n\n")
}

func createRoles(r *rosa.Runtime, prefix, permissionsBoundary, accountID, env string,
	policies map[string]string, defaultPolicyVersion string,
	path string) error {

	for file, role := range aws.AccountRoles {
		name := aws.GetRoleName(prefix, role.Name)
		policyARN := aws.GetPolicyARN(r.Creator.AccountID, fmt.Sprintf("%s-Policy", name), path)
		if !confirm.Prompt(true, "Create the '%s' role?", name) {
			continue
		}

		filename := fmt.Sprintf("sts_%s_trust_policy", file)
		policyDetail := policies[filename]

		policy := aws.InterpolatePolicyDocument(policyDetail, map[string]string{
			"partition":      aws.GetPartition(),
			"aws_account_id": aws.GetJumpAccount(env),
		})
		r.Reporter.Debugf("Creating role '%s'", name)
		roleARN, err := r.AWSClient.EnsureRole(name, policy, permissionsBoundary,
			defaultPolicyVersion, map[string]string{
				tags.OpenShiftVersion: defaultPolicyVersion,
				tags.RolePrefix:       prefix,
				tags.RoleType:         file,
				tags.RedHatManaged:    "true",
			}, path)
		if err != nil {
			return err
		}
		r.Reporter.Infof("Created role '%s' with ARN '%s'", name, roleARN)

		filename = fmt.Sprintf("sts_%s_permission_policy", file)
		policyDetail = policies[filename]

		r.Reporter.Debugf("Creating permission policy '%s'", policyARN)
		policyARN, err = r.AWSClient.EnsurePolicy(policyARN, policyDetail,
			defaultPolicyVersion, map[string]string{
				tags.OpenShiftVersion: defaultPolicyVersion,
				tags.RolePrefix:       prefix,
				tags.RoleType:         file,
				tags.RedHatManaged:    "true",
			}, path)
		if err != nil {
			return err
		}
		r.Reporter.Debugf("Attaching permission policy to role '%s'", filename)
		err = r.AWSClient.AttachRolePolicy(name, policyARN)
		if err != nil {
			return err
		}
	}

	return nil
}
