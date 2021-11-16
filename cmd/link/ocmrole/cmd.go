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
		"Role ARN to associate the ocm organization account to",
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
	orgAccount, err := ocmClient.GetCurrentOrganization()
	if err != nil {
		reporter.Errorf("Error getting organization account: %v", err)
	}
	if !confirm.Prompt(true, "Link the '%s' role with organization '%s'?", args.roleArn, orgAccount) {
		os.Exit(0)
	}

	err = ocmClient.LinkOrgToRole(orgAccount, args.roleArn)
	if err != nil {
		reporter.Errorf("Unable to link role arn '%s' with the organization id : '%s' : %v",
			args.roleArn, orgAccount, err)
		return err
	}
	reporter.Infof("Successfully linked role-arn '%s' with organization account '%s'", args.roleArn, orgAccount)
	return nil
}
