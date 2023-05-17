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

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/login"
	"github.com/openshift/rosa/cmd/verify/oc"
	"github.com/openshift/rosa/cmd/verify/quota"
	"github.com/openshift/rosa/pkg/aws"
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
	managed             bool
	forcePolicyCreation bool
	hostedCP            bool
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

	flags.StringVar(
		&args.channelGroup,
		"channel-group",
		ocm.DefaultChannelGroup,
		"Channel group is the name of the channel where this image belongs, for example \"stable\" or \"fast\".",
	)
	flags.MarkHidden("channel-group")

	// TODO: add `legacy-policies` once AWS managed policies are in place (managed-policies will be the default)
	flags.BoolVar(
		&args.managed,
		"managed-policies",
		false,
		"Attach AWS managed policies to the account roles",
	)
	flags.MarkHidden("managed-policies")
	flags.BoolVar(
		&args.managed,
		"mp",
		false,
		"Attach AWS managed policies to the account roles. This is an alias for --managed-policies")
	flags.MarkHidden("mp")

	flags.BoolVarP(
		&args.forcePolicyCreation,
		"force-policy-creation",
		"f",
		false,
		"Forces creation of policies skipping compatibility check",
	)

	flags.BoolVar(
		&args.hostedCP,
		"hosted-cp",
		false,
		"Enable the use of Hosted Control Planes",
	)
	flags.MarkHidden("hosted-cp")

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

	// Determine if managed policies are enabled
	isManagedSet := cmd.Flags().Changed("managed-policies") || cmd.Flags().Changed("mp")
	if isManagedSet && env == ocm.Production {
		r.Reporter.Errorf("Managed policies are not supported in this environment")
		os.Exit(1)
	}
	managedPolicies := args.managed

	if args.forcePolicyCreation && managedPolicies {
		r.Reporter.Warnf("Forcing creation of policies only works for unmanaged policies")
		os.Exit(1)
	}

	if args.hostedCP && cmd.Flags().Changed("version") {
		r.Reporter.Warnf("Setting `version` flag for hosted CP managed policies has no effect, " +
			"any supported ROSA version can be installed with managed policies")
	}

	// Hosted cluster roles always use managed policies
	if cmd.Flags().Changed("hosted-cp") && cmd.Flags().Changed("managed-policies") && !args.managed {
		r.Reporter.Errorf("Setting `hosted-cp` as unmanaged policies is not supported")
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
	policyVersion, err := r.OCMClient.GetPolicyVersion(version, channelGroup)
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
	if !args.hostedCP && strings.HasSuffix(prefix, "-HCP") {
		r.Reporter.Errorf("The '-HCP' suffix is reserved for hosted CP managed policies")
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
		err = aws.ARNValidator(permissionsBoundary)
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

	if path != "" && !aws.ARNPath.MatchString(path) {
		r.Reporter.Errorf("The specified value for path is invalid. " +
			"It must begin and end with '/' and contain only alphanumeric characters and/or '/' characters.")
		os.Exit(1)
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

	if args.forcePolicyCreation && mode != aws.ModeAuto {
		r.Reporter.Warnf("Forcing creation of policies only works in auto mode")
		os.Exit(1)
	}

	policies, err := r.OCMClient.GetPolicies("AccountRole")
	if err != nil {
		r.Reporter.Errorf("Expected a valid role creation mode: %s", err)
		os.Exit(1)
	}

	createHostedCP := args.hostedCP
	if env != ocm.Production { // TODO: remove env check once AWS publishes all the hcp Managed Policies
		if interactive.Enabled() && !cmd.Flags().Changed("hosted-cp") {
			createHostedCP, err = interactive.GetBool(interactive.Input{
				Question: "Create Hosted CP account roles",
				Help:     cmd.Flags().Lookup("hosted-cp").Usage,
				Default:  false,
				Required: false,
			})
			if err != nil {
				r.Reporter.Errorf("Expected a valid value: %s", err)
				os.Exit(1)
			}
		}
	}

	rolesCreator := initCreator(managedPolicies, createHostedCP)
	input := buildRolesCreationInput(prefix, permissionsBoundary, r.Creator.AccountID, env, policies,
		policyVersion, path)

	switch mode {
	case aws.ModeAuto:
		r.Reporter.Infof("Creating roles using '%s'", r.Creator.ARN)

		err = rolesCreator.createRoles(r, input)
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
		var createClusterFlag string
		if args.hostedCP {
			createClusterFlag = "--hosted-cp"
		} else {
			createClusterFlag = "--sts"
		}
		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("To create an OIDC Config, run the following command:\n" +
				"\trosa create oidc-config")
			r.Reporter.Infof(fmt.Sprintf("To create a cluster with these roles, run the following command:\n"+
				"\trosa create cluster %s", createClusterFlag))
		}
		r.OCMClient.LogEvent("ROSACreateAccountRolesModeAuto", map[string]string{
			ocm.Response: ocm.Success,
			ocm.Version:  policyVersion,
		})
	case aws.ModeManual:
		err = aws.GeneratePolicyFiles(r.Reporter, env, true, false, policies, nil, managedPolicies)
		if err != nil {
			r.Reporter.Errorf("There was an error generating the policy files: %s", err)
			r.OCMClient.LogEvent("ROSACreateAccountRolesModeManual", map[string]string{
				ocm.Response: ocm.Failure,
			})
			os.Exit(1)
		}
		commands, err := rolesCreator.buildCommands(input)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
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
