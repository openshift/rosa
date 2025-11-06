/*
Copyright (c) 2024 Red Hat, Inc.

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

package policy

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/policy"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "policy"
	short   = "Detach AWS IAM Policies from an AWS IAM Role"
	long    = "Detach AWS IAM Policies from an AWS IAM Role in the authenticated AWS Account"
	example = `  # Detach policy <policy_arn_1> and <policy_arn_2> from role <role_name>
  rosa detach policy --role-name=<role_name> --policy-arns=<policy_arn_1>,<policy_arn_2>`
)

type RosaDetachPolicyOptions struct {
	policyArns string
	roleName   string
}

func NewRosaDetachPolicyOptions() RosaDetachPolicyOptions {
	return RosaDetachPolicyOptions{}
}

func NewDetachPolicyCommand() *cobra.Command {
	options := NewRosaDetachPolicyOptions()
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Example: example,
		Args:    cobra.NoArgs,
		Hidden:  true,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCMAndAWS(), DetachPolicyRunner(&options)),
	}

	flags := cmd.Flags()
	flags.StringVarP(
		&options.policyArns,
		"policy-arns",
		"p",
		"",
		"Policy arn of the policies to be detached from the specified role."+
			" Format should be a comma-separated list. (required).",
	)
	flags.StringVarP(
		&options.roleName,
		"role-name",
		"r",
		"",
		"Role name of the role to detach the specified policy (required).",
	)
	cmd.MarkFlagRequired("policy-arns")
	cmd.MarkFlagRequired("role-name")
	interactive.AddModeFlag(cmd)
	return cmd
}

func DetachPolicyRunner(userOptions *RosaDetachPolicyOptions) rosa.CommandRunner {
	return func(_ context.Context, r *rosa.Runtime, cmd *cobra.Command, _ []string) error {
		options := NewRosaDetachPolicyOptions()
		options.BindAndValidate(*userOptions)
		policySvc := policy.NewPolicyService(r.OCMClient, r.AWSClient)
		policyArns := strings.Split(options.policyArns, ",")
		slices.Sort(policyArns)
		policyArns = slices.Compact(policyArns)
		err := policySvc.ValidateDetachOptions(options.roleName, policyArns)
		if err != nil {
			return err
		}

		mode, err := interactive.GetMode()
		if err != nil {
			return err
		}
		// Determine if interactive mode is needed
		if !interactive.Enabled() && !cmd.Flags().Changed("mode") {
			interactive.Enable()
		}
		if interactive.Enabled() {
			mode, err = interactive.GetOptionMode(cmd, mode, "Detach policy mode")
			if err != nil {
				return fmt.Errorf("expected a valid detach policy mode: %s", err)
			}
		}

		orgID, _, err := r.OCMClient.GetCurrentOrganization()
		if err != nil {
			return fmt.Errorf("failed to get current organization: %s", err)
		}
		switch mode {
		case interactive.ModeAuto:
			output, err := policySvc.AutoDetachArbitraryPolicy(options.roleName, policyArns,
				r.Creator.AccountID, orgID)
			r.Reporter.Infof(output)
			if err != nil {
				return err
			}
		case interactive.ModeManual:
			output, warn, err := policySvc.ManualDetachArbitraryPolicy(options.roleName, policyArns,
				r.Creator.AccountID, orgID)
			if err != nil {
				return err
			}
			if len(warn) > 0 {
				fmt.Print(warn)
			}
			if len(output) > 0 {
				r.Reporter.Infof("Run the following command to detach the policy:")
				fmt.Print(output)
			}
		default:
			return fmt.Errorf("invalid mode. Allowed values are %s", interactive.Modes)
		}
		return nil
	}
}

func (o *RosaDetachPolicyOptions) BindAndValidate(options RosaDetachPolicyOptions) {
	o.policyArns = options.policyArns
	o.roleName = options.roleName
}
