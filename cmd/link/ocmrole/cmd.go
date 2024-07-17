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
	Short:   "Link OCM role to specific OCM organization.",
	Long:    "Link OCM role to specific OCM organization before you create your cluster.",
	Example: ` # Link OCM role
  rosa link ocm-role --role-arn arn:aws:iam::123456789012:role/ManagedOpenshift-OCM-Role`,
	Run:  run,
	Args: cobra.MaximumNArgs(1),
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

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	if len(argv) > 0 {
		args.roleArn = argv[0]
	}

	orgAccount, _, err := r.OCMClient.GetCurrentOrganization()
	if err != nil {
		r.Reporter.Errorf("Error getting organization account: %v", err)
		os.Exit(1)
	}

	if args.organizationID != "" && orgAccount != args.organizationID {
		r.Reporter.Errorf("Invalid organization ID '%s'. "+
			"It doesn't match with the user session '%s'.", args.organizationID, orgAccount)
		os.Exit(1)
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
		r.Reporter.Errorf("The role with '%s' cannot be found", roleArn)
		os.Exit(1)
	}

	if !confirm.Prompt(true, "Link the '%s' role with organization '%s'?", roleArn, orgAccount) {
		os.Exit(0)
	}

	linked, err := r.OCMClient.LinkOrgToRole(orgAccount, roleArn)
	if err != nil {
		if errors.GetType(err) == errors.Forbidden || strings.Contains(err.Error(), "ACCT-MGMT-11") {
			var errMessage string
			ocmAccount, localErr := r.OCMClient.GetCurrentAccount()
			if localErr != nil {
				r.Reporter.Warnf("Error getting Red Hat account: %v", localErr)
			} else {
				errMessage = fmt.Sprintf(
					"Your Red Hat Account '%s' has no permission for this command.\n", ocmAccount.Username())
			}

			r.Reporter.Errorf("%s"+
				"Only organization member can run this command. "+
				"Please ask someone with the organization member role to run the following command \n\n"+
				"\t rosa link ocm-role --role-arn %s --organization-id %s", errMessage, roleArn, orgAccount)
			os.Exit(1)
		}
		r.Reporter.Errorf("Unable to link role arn '%s' with the organization id : '%s' : %v",
			roleArn, orgAccount, err)
		os.Exit(1)
	}
	if !linked {
		r.Reporter.Infof("Role-arn '%s' is already linked with the organization account '%s'", roleArn, orgAccount)
		os.Exit(0)
	}
	r.Reporter.Infof("Successfully linked role-arn '%s' with organization account '%s'", roleArn, orgAccount)
}
