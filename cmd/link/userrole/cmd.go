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
	"os"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/spf13/cobra"
)

var args struct {
	roleArn   string
	accountID string
}

var Cmd = &cobra.Command{
	Use:     "user-role",
	Aliases: []string{"userrole"},
	Short:   "link user role to specific OCM account.",
	Long:    "link user role to specific OCM account before create your cluster.",
	Example: ` # Link user roles 
  rosa link user-role --role-arn arn:aws:iam::{accountid}:role/{prefix}-ocm-user-{username}`,
	RunE: run,
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

func run(cmd *cobra.Command, argv []string) (err error) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	if len(argv) > 0 {
		args.roleArn = argv[0]
	}
	// Create the client for the OCM API:
	ocmClient, err := ocm.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		return err
	}
	defer func() {
		err = ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	accountID := args.accountID
	if accountID == "" {
		currentAccount, err := ocmClient.GetCurrentAccount()
		if err != nil {
			reporter.Errorf("Error getting current account: %v", err)
		}
		accountID = currentAccount.ID()
	}

	if reporter.IsTerminal() {
		reporter.Infof("Linking User role")
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
			reporter.Errorf("Expected a valid user role ARN to link to a current account: %s", err)
			os.Exit(1)
		}
	}
	if roleArn != "" {
		_, err := arn.Parse(roleArn)
		if err != nil {
			reporter.Errorf("Expected a valid user role ARN to link to a current account: %s", err)
			os.Exit(1)
		}
	}

	if !confirm.Prompt(true, "Link the '%s' role with account '%s'?", roleArn, accountID) {
		os.Exit(0)
	}

	err = ocmClient.LinkAccountRole(accountID, roleArn)
	if err != nil {
		reporter.Errorf("Unable to link role arn '%s' with the account id : '%s' : %v",
			args.roleArn, accountID, err)
		return err
	}
	reporter.Infof("Successfully linked role-arn '%s' with account '%s'", roleArn, accountID)
	return nil
}
