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

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
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
	RunE: run,
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

func run(cmd *cobra.Command, argv []string) (err error) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)
	ocmClient := ocm.CreateNewClientOrExit(logger, reporter)
	defer func() {
		err = ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	if len(argv) > 0 {
		args.roleArn = argv[0]
	}

	accountID := args.accountID
	if accountID == "" {
		currentAccount, err := ocmClient.GetCurrentAccount()
		if err != nil {
			reporter.Errorf("Error getting current account: %v", err)
			os.Exit(1)
		}
		accountID = currentAccount.ID()
	}

	if reporter.IsTerminal() {
		reporter.Infof("Unlinking user role")
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
			reporter.Errorf("Expected a valid user role ARN to unlink from the current account: %s", err)
			os.Exit(1)
		}
	}
	if roleArn != "" {
		_, err := arn.Parse(roleArn)
		if err != nil {
			reporter.Errorf("Expected a valid user role ARN to unlink from the current account: %s", err)
			os.Exit(1)
		}
	}
	if !confirm.Prompt(true, "Unlink the '%s' role from the current account '%s'?", roleArn, accountID) {
		os.Exit(0)
	}

	err = ocmClient.UnlinkUserRoleFromAccount(accountID, roleArn)
	if err != nil {
		if errors.GetType(err) == errors.Forbidden || strings.Contains(err.Error(), "ACCT-MGMT-11") {
			reporter.Errorf("Only organization admin or the user that owns this account can run this command. "+
				"Please ask someone with adequate permissions to run the following command \n\n"+
				"\t rosa unlink user-role --role-arn %s --account-id %s", roleArn, accountID)
			os.Exit(1)
		}
		reporter.Errorf("Unable to unlink role ARN '%s' from the account id : '%s' : %v",
			roleArn, accountID, err)
		os.Exit(1)
	}
	reporter.Infof("Successfully unlinked role ARN '%s' from account '%s'", roleArn, accountID)
	return nil
}
