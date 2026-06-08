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

package userrolebubble

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/bubbletea"
	bubbleteauserrole "github.com/openshift/rosa/pkg/interactive/bubbletea/userrole"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/userrole"
)

var args struct {
	prefix              string
	permissionsBoundary string
	path                string
}

var Cmd = &cobra.Command{
	Use:     "user-role-bubble",
	Aliases: []string{"userrolebubble"},
	Short:   "Create user role using Bubble Tea prompts (POC dry run)",
	Long: "Create user role that allows OCM to verify that users creating a cluster " +
		"have access to the current AWS account. This POC command uses Bubble Tea for " +
		"interactive prompts and does not create AWS or OCM resources in auto mode.",
	Example: `  # Create user role with Bubble Tea prompts (dry run)
  rosa create user-role-bubble

  # Create user role with a specific permissions boundary
  rosa create user-role-bubble --permissions-boundary arn:aws:iam::123456789012:policy/perm-boundary`,
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

	if !interactive.Enabled() && (!cmd.Flags().Changed("mode")) {
		interactive.Enable()
	}

	input := userrole.Input{
		Prefix:              args.prefix,
		PermissionsBoundary: args.permissionsBoundary,
		Path:                args.path,
		Mode:                mode,
	}

	if interactive.Enabled() {
		input, err = bubbleteauserrole.RunWizard(bubbleteauserrole.WizardInput{
			Prefix:                  args.prefix,
			PermissionsBoundary:     args.permissionsBoundary,
			Path:                    args.path,
			Mode:                    mode,
			PrefixHelp:              cmd.Flags().Lookup("prefix").Usage,
			PermissionsBoundaryHelp: cmd.Flags().Lookup("permissions-boundary").Usage,
			PathHelp:                cmd.Flags().Lookup("path").Usage,
			ModeHelp:                cmd.Flags().Lookup(interactive.Mode).Usage,
		})
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		mode = input.Mode
	} else if err = userrole.Validate(input); err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("Creating User role (Bubble Tea POC dry run)")
	}

	currentAccount, err := r.OCMClient.GetCurrentAccount()
	if err != nil {
		r.Reporter.Errorf("Failed to get current account: %s", err)
		os.Exit(1)
	}

	policies, err := r.OCMClient.GetPolicies("")
	if err != nil {
		r.Reporter.Errorf("Failed to get policies: %s", err)
		os.Exit(1)
	}

	switch mode {
	case interactive.ModeAuto:
		roleName := userrole.RoleName(input.Prefix, currentAccount.Username())
		confirmed, err := promptCreateRole(r, roleName)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		if !confirmed {
			os.Exit(0)
		}

		roleARN := userrole.RoleARN(input.Prefix, input.Path, currentAccount.Username(), r.Creator)
		r.Reporter.Infof("Created role '%s' with ARN '%s'", roleName, roleARN)
		r.Reporter.Infof("Run the following command to link the role to your OCM account:")
		fmt.Println(fmt.Sprintf("rosa link user-role --role-arn %s", roleARN))
	case interactive.ModeManual:
		err = userrole.GeneratePolicyFiles(r.Reporter, env, r.Creator.Partition, currentAccount.ID(), policies)
		if err != nil {
			r.Reporter.Errorf("There was an error generating the policy files: %s", err)
			os.Exit(1)
		}
		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("All policy files saved to the current directory")
			r.Reporter.Infof("Run the following commands to create the account roles and policies:\n")
		}
		fmt.Println(userrole.BuildCommands(
			input.Prefix,
			input.Path,
			currentAccount.Username(),
			r.Creator,
			env,
			input.PermissionsBoundary,
		))
	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", interactive.Modes)
		os.Exit(1)
	}
}

func promptCreateRole(r *rosa.Runtime, roleName string) (bool, error) {
	if confirm.Yes() {
		return true, nil
	}
	if !r.Reporter.IsTerminal() {
		return false, nil
	}
	return bubbletea.RunConfirm(fmt.Sprintf("Create the '%s' role?", roleName), true)
}
