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

package userrole

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
	roleArn   string
	accountID string
}

var Cmd = &cobra.Command{
	Use:     "user-role",
	Aliases: []string{"userrole"},
	Short:   "Link user role to specific OCM account.",
	Long:    "Link user role to specific OCM account before you create your cluster.",
	Example: ` # Link user roles
  rosa link user-role --role-arn arn:aws:iam::{accountid}:role/{prefix}-User-{username}-Role`,
	Run:  run,
	Args: cobra.MaximumNArgs(1),
}

func init() {
	flags := Cmd.Flags()

	flags.StringVar(
		&args.roleArn,
		"role-arn",
		"",
		"Role ARN to associate the OCM account to",
	)

	flags.StringVar(
		&args.accountID,
		"account-id",
		"",
		"OCM account id to associate the user role ARN",
	)
	confirm.AddFlag(flags)
	interactive.AddFlag(flags)
}

func run(cmd *cobra.Command, argv []string) {
	if len(argv) > 0 {
		args.roleArn = argv[0]
	}

	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()
	err := runWithRuntime(r, cmd)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime, cmd *cobra.Command) error {
	var err error

	accountID := args.accountID
	if accountID == "" {
		currentAccount, getAccountErr := r.OCMClient.GetCurrentAccount()
		if getAccountErr != nil {
			r.Reporter.Errorf("Error getting current account: %v", getAccountErr)
		}
		accountID = currentAccount.ID()
	}

	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("Linking User role")
	}

	roleArn := args.roleArn

	// Determine if interactive mode is needed
	if !interactive.Enabled() && roleArn == "" {
		interactive.Enable()
	}

	if interactive.Enabled() {
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
			return fmt.Errorf("expected a valid user role ARN to link to a current account: %s", err)
		}
	}
	if roleArn != "" {
		err = aws.ARNValidator(roleArn)
		if err != nil {
			return fmt.Errorf("expected a valid user role ARN to link to a current account: %s", err)
		}
	}

	role, err := r.AWSClient.GetRoleByARN(roleArn)
	if err != nil {
		return fmt.Errorf("there was a problem checking if role '%s' exists: %v", roleArn, err)
	}

	if *role.Arn != roleArn {
		return fmt.Errorf("the role with '%s' cannot be found", roleArn)
	}

	if !confirm.Prompt(true, "Link the '%s' role with account '%s'?", roleArn, accountID) {
		return nil
	}

	err = r.OCMClient.LinkAccountRole(accountID, roleArn)
	if err != nil {
		if errors.GetType(err) == errors.Forbidden || strings.Contains(err.Error(), "ACCT-MGMT-11") {
			return fmt.Errorf("only organization admin or the user that owns this account '%s' can run this command, "+
				"please ask someone with adequate permissions to run the following command: "+
				"rosa link user-role --role-arn %s --account-id %s", accountID, roleArn, accountID)
		}
		return fmt.Errorf("unable to link role ARN '%s' with the account id : '%s' : %v",
			args.roleArn, accountID, err)
	}
	r.Reporter.Infof("Successfully linked role ARN '%s' with account '%s'", roleArn, accountID)
	return nil
}
