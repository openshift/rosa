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

package userrole

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
	roleArn   string
	accountID string
}

var Cmd = &cobra.Command{
	Use:     "user-role",
	Aliases: []string{"userrole"},
	Short:   "Unlink user role from a specific OCM account",
	Long:    "Unlink user role from a specific OCM account",
	Example: ` # Unlink user role
rosa unlink user-role --role-arn arn:aws:iam::{accountid}:role/{prefix}-User-{username}-Role`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.roleArn,
		"role-arn",
		"",
		"Role ARN to identify the user-role to be unlinked from the OCM account",
	)
	flags.StringVar(
		&args.accountID,
		"account-id",
		"",
		"OCM account id to unlink the user role ARN",
	)

	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	var err error
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	if len(argv) > 0 {
		args.roleArn = argv[0]
	}

	accountID := args.accountID
	if accountID == "" {
		currentAccount, err := r.OCMClient.GetCurrentAccount()
		if err != nil {
			r.Reporter.Errorf("Error getting current account: %v", err)
			os.Exit(1)
		}
		accountID = currentAccount.ID()
	}

	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("Unlinking user role")
	}

	roleArn := args.roleArn

	// Determine if interactive mode is needed
	if !interactive.Enabled() && roleArn == "" {
		interactive.Enable()
	}

	if interactive.Enabled() && roleArn == "" {
		roleArn, err = interactive.GetString(interactive.Input{
			Question: "User Role ARN",
			Help:     cmd.Flags().Lookup("role-arn").Usage,
			Default:  roleArn,
			Required: true,
			Validators: []interactive.Validator{
				aws.ARNValidator,
			},
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid user role ARN to unlink from the current account: %s", err)
			os.Exit(1)
		}
	}
	if roleArn != "" {
		err = aws.ARNValidator(roleArn)
		if err != nil {
			r.Reporter.Errorf("Expected a valid user role ARN to unlink from the current account: %s", err)
			os.Exit(1)
		}
	}
	if !confirm.Prompt(true, "Unlink the '%s' role from the current account '%s'?", roleArn, accountID) {
		os.Exit(0)
	}

	err = r.OCMClient.UnlinkUserRoleFromAccount(accountID, roleArn)
	if err != nil {
		if errors.GetType(err) == errors.Forbidden || strings.Contains(err.Error(), "ACCT-MGMT-11") {
			r.Reporter.Errorf("Only organization admin or the user that owns this account '%s' can run this command. "+
				"Please ask someone with adequate permissions to run the following command \n\n"+
				"\t rosa unlink user-role --role-arn %s --account-id %s", accountID, roleArn, accountID)
			os.Exit(1)
		}
		r.Reporter.Errorf("Unable to unlink role ARN '%s' from the account id : '%s' : %v",
			roleArn, accountID, err)
		os.Exit(1)
	}
	r.Reporter.Infof("Successfully unlinked role ARN '%s' from account '%s'", roleArn, accountID)
}
