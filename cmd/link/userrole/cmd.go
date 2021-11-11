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

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	roleArn string
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
	Cmd.MarkFlagRequired("role-arn")
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
	currentAccount, err := ocmClient.GetCurrentAccount()
	if err != nil {
		reporter.Errorf("Error getting current account: %v", err)
	}
	if !confirm.Prompt(true, "Link the '%s' role with account '%s'?", args.roleArn, currentAccount.ID()) {
		os.Exit(0)
	}

	err = ocmClient.LinkAccountRole(currentAccount.ID(), args.roleArn)
	if err != nil {
		reporter.Errorf("Unable to link role arn '%s' with the account id : '%s' : %v",
			args.roleArn, currentAccount.ID(), err)
		return err
	}
	reporter.Infof("Successfully linked role-arn '%s' with account '%s'", args.roleArn, currentAccount.ID())
	return nil
}
