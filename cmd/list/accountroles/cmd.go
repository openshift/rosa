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

package accountroles

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	version string
}

var Cmd = &cobra.Command{
	Use:     "account-roles",
	Aliases: []string{"accountrole", "account-role", "accountroles"},
	Short:   "List account roles and policies",
	Long:    "List account roles and policies for the current AWS account.",
	Example: `  # List all account roles
  rosa list account-roles`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.SortFlags = false
	flags.StringVar(
		&args.version,
		"version",
		"",
		"List only account-roles that are associated with the given version.",
	)
	output.AddFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.NewLogger()

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}

	// Create the client for the OCM API:
	ocmClient, err := ocm.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	versionList, err := ocm.GetVersionMinorList(ocmClient)
	if err != nil {
		reporter.Errorf("%s", err)
		os.Exit(1)
	}

	_, err = ocm.ValidateVersion(args.version, versionList)
	if err != nil {
		reporter.Errorf("Version '%s' is invalid", args.version)
		os.Exit(1)
	}

	var spin *spinner.Spinner
	if reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		reporter.Infof("Fetching account roles")
		spin.Start()
	}

	accountRoles, err := awsClient.ListAccountRoles(args.version)

	if spin != nil {
		spin.Stop()
	}

	if err != nil {
		reporter.Errorf("Failed to get account roles: %v", err)
		os.Exit(1)
	}

	if len(accountRoles) == 0 {
		reporter.Infof("No account roles available")
		os.Exit(0)
	}
	if output.HasFlag() {
		err = output.Print(accountRoles)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "ROLE NAME\tROLE TYPE\tROLE ARN\tOPENSHIFT VERSION\n")
	for _, accountRoles := range accountRoles {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\n",
			accountRoles.RoleName,
			accountRoles.RoleType,
			accountRoles.RoleARN,
			accountRoles.Version,
		)
	}
	writer.Flush()
}
