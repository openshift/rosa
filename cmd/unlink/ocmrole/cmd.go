/*
Copyright (c) 2022 Red Hat, Inc.
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
	Short:   "Unlink ocm role from a specific OCM organization",
	Long:    "Unlink ocm role from a specific OCM organization",
	Example: ` #Unlink ocm role
rosa unlink ocm-role --role-arn arn:aws:iam::123456789012:role/ManagedOpenshift-OCM-Role`,
	RunE: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.roleArn,
		"role-arn",
		"",
		"Role ARN to identify the ocm-role to be unlinked from the OCM organization",
	)
	flags.StringVar(
		&args.organizationID,
		"organization-id",
		"",
		"OCM organization id to unlink the ocm role ARN",
	)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) (err error) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	if len(argv) > 0 {
		args.roleArn = argv[0]
	}

	orgID, _, err := r.OCMClient.GetCurrentOrganization()
	if err != nil {
		r.Reporter.Errorf("Error getting organization account: %v", err)
		os.Exit(1)
	}
	if args.organizationID != "" && orgID != args.organizationID {
		r.Reporter.Errorf("Invalid organization ID '%s'. "+
			"It doesn't match with the user session '%s'.", args.organizationID, orgID)
		os.Exit(1)
	}

	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("Unlinking OCM role")
	}

	roleArn := args.roleArn

	// Determine if interactive mode is needed
	if !interactive.Enabled() && roleArn == "" {
		interactive.Enable()
	}

	if interactive.Enabled() && roleArn == "" {
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
			r.Reporter.Errorf("Expected a valid ocm role ARN to unlink from the current organization: %s", err)
			os.Exit(1)
		}
	}
	if roleArn != "" {
		err = aws.ARNValidator(roleArn)
		if err != nil {
			r.Reporter.Errorf("Expected a valid ocm role ARN to unlink from the current organization: %s", err)
			os.Exit(1)
		}
	}
	if !confirm.Prompt(true, "Unlink the '%s' role from organization '%s'?", roleArn, orgID) {
		os.Exit(0)
	}

	err = r.OCMClient.UnlinkOCMRoleFromOrg(orgID, roleArn)
	if err != nil {
		if errors.GetType(err) == errors.Forbidden || strings.Contains(err.Error(), "ACCT-MGMT-11") {
			r.Reporter.Errorf("Only organization admin can run this command. "+
				"Please ask someone with the organization admin role to run the following command \n\n"+
				"\t rosa unlink ocm-role --role-arn %s --organization-id %s", roleArn, orgID)
			os.Exit(1)
		}
		r.Reporter.Errorf("Unable to unlink role arn '%s' from the organization id : '%s' : %v",
			roleArn, orgID, err)
		os.Exit(1)
	}
	r.Reporter.Infof("Successfully unlinked role-arn '%s' from organization account '%s'", roleArn, orgID)

	return nil
}
