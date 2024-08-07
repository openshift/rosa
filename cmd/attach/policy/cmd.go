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
	short   = "Attach AWS IAM Policies to an AWS IAM Role"
	long    = "Attach existing AWS IAM Policies to an AWS IAM Role in the authenticated AWS Account"
	example = `  # Attach policy <policy_arn_1> and <policy_arn_2> to role <role_name>
  rosa attach policy --role-name=<role_name> --policy-arns=<policy_arn_1>,<policy_arn_2>`
)

type RosaAttachPolicyOptions struct {
	policyArns string
	roleName   string
}

func NewRosaAttachPolicyOptions() RosaAttachPolicyOptions {
	return RosaAttachPolicyOptions{}
}

func NewAttachPolicyCommand() *cobra.Command {
	options := NewRosaAttachPolicyOptions()
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Example: example,
		Args:    cobra.NoArgs,
		Hidden:  true,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCMAndAWS(), AttachPolicyRunner(&options)),
	}

	flags := cmd.Flags()
	flags.StringVarP(
		&options.policyArns,
		"policy-arns",
		"p",
		"",
		"Policy arn of the policies to be attached to the specified role."+
			" Format should be a comma-separated list. (required).",
	)
	flags.StringVarP(
		&options.roleName,
		"role-name",
		"r",
		"",
		"Role name of the role to attach the specified policy (required).",
	)
	cmd.MarkFlagRequired("policy-arns")
	cmd.MarkFlagRequired("role-name")
	interactive.AddModeFlag(cmd)
	return cmd
}

func AttachPolicyRunner(userOptions *RosaAttachPolicyOptions) rosa.CommandRunner {
	return func(_ context.Context, r *rosa.Runtime, cmd *cobra.Command, _ []string) error {
		options := NewRosaAttachPolicyOptions()
		options.BindAndValidate(*userOptions)
		policySvc := policy.NewPolicyService(r.OCMClient, r.AWSClient)
		policyArns := strings.Split(options.policyArns, ",")
		slices.Sort(policyArns)
		policyArns = slices.Compact(policyArns)
		err := policySvc.ValidateAttachOptions(options.roleName, policyArns)
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
			mode, err = interactive.GetOptionMode(cmd, mode, "Attach policy mode")
			if err != nil {
				return fmt.Errorf("Expected a valid attach policy mode: %s", err)
			}
		}

		orgID, _, err := r.OCMClient.GetCurrentOrganization()
		if err != nil {
			return fmt.Errorf("Failed to get current organization: %s", err)
		}
		switch mode {
		case interactive.ModeAuto:
			err := policySvc.AutoAttachArbitraryPolicy(r.Reporter, options.roleName, policyArns,
				r.Creator.AccountID, orgID)
			if err != nil {
				return err
			}
		case interactive.ModeManual:
			r.Reporter.Infof("Run the following command to attach the policy:")
			fmt.Print(policySvc.ManualAttachArbitraryPolicy(options.roleName, policyArns,
				r.Creator.AccountID, orgID))
		default:
			return fmt.Errorf("Invalid mode. Allowed values are %s", interactive.Modes)
		}
		return nil
	}
}

func (o *RosaAttachPolicyOptions) BindAndValidate(options RosaAttachPolicyOptions) {
	o.policyArns = options.policyArns
	o.roleName = options.roleName
}
