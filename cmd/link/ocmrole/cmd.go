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
	"os"
	"strings"

	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	roleArn        string
	organizationID string
}

var Cmd = &cobra.Command{
	Use:     "ocm-role",
	Aliases: []string{"ocmrole"},
	Short:   "link ocm role to specific OCM organization account.",
	Long:    "link ocm role to specific OCM organization account before you create your cluster.",
	Example: ` # Link ocm role
  rosa link ocm-role --role-arn arn:aws:iam::123456789012:role/ManagedOpenshift-OCM-Role`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.roleArn,
		"role-arn",
		"",
		"Role ARN to associate the OCM organization account to",
	)
	flags.StringVar(
		&args.organizationID,
		"organization-id",
		"",
		"OCM organization id to associate the ocm role ARN",
	)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) (err error) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	if len(argv) > 0 {
		args.roleArn = argv[0]
	}

	orgAccount, _, err := r.OCMClient.GetCurrentOrganization()
	if err != nil {
		r.Reporter.Errorf("Error getting organization account: %v", err)
		return err
	}

	if args.organizationID != "" && orgAccount != args.organizationID {
		r.Reporter.Errorf("Invalid organization ID '%s'. "+
			"It doesn't match with the user session '%s'.", args.organizationID, orgAccount)
		return err
	}

	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("Linking OCM role")
	}

	roleArn := args.roleArn

	// Determine if interactive mode is needed
	if !interactive.Enabled() && roleArn == "" {
		interactive.Enable()
	}

	if interactive.Enabled() {
		roleArn, err = interactive.GetString(interactive.Input{
			Question: "OCM Role ARN",
			Help:     cmd.Flags().Lookup("role-arn").Usage,
			Default:  roleArn,
			Required: true,
			Validators: []interactive.Validator{
				aws.ARNValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid ocm role ARN to link to a current organization: %s", err)
			os.Exit(1)
		}
	}
	if roleArn != "" {
		err = aws.ARNValidator(roleArn)
		if err != nil {
			r.Reporter.Errorf("Expected a valid ocm role ARN to link to a current organization: %s", err)
			os.Exit(1)
		}
	}

	role, err := r.AWSClient.GetRoleByARN(roleArn)
	if err != nil {
		r.Reporter.Errorf("There was a problem checking if role '%s' exists: %v", roleArn, err)
		os.Exit(1)
	}

	if *role.Arn != roleArn {
		r.Reporter.Errorf("ARN '%s' did not match the existing role ARN", *role.Arn)
		os.Exit(1)
	}

	if !confirm.Prompt(true, "Link the '%s' role with organization '%s'?", roleArn, orgAccount) {
		os.Exit(0)
	}

	linked, err := r.OCMClient.LinkOrgToRole(orgAccount, roleArn)
	if err != nil {
		if errors.GetType(err) == errors.Forbidden || strings.Contains(err.Error(), "ACCT-MGMT-11") {
			r.Reporter.Errorf("Only organization admin can run this command. "+
				"Please ask someone with the organization admin role to run the following command \n\n"+
				"\t rosa link ocm-role --role-arn %s --organization-id %s", roleArn, orgAccount)
			return err
		}
		r.Reporter.Errorf("Unable to link role arn '%s' with the organization id : '%s' : %v",
			roleArn, orgAccount, err)
		return err
	}
	if !linked {
		r.Reporter.Infof("Role-arn '%s' is already linked with the organization account '%s'", roleArn, orgAccount)
		os.Exit(0)
	}
	r.Reporter.Infof("Successfully linked role-arn '%s' with organization account '%s'", roleArn, orgAccount)
	return nil
}
