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
	"os"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
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
	Run:  run,
	Args: cobra.NoArgs,
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
	output.AddHideEmptyColumnsFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	versionList, err := ocm.GetVersionMinorList(r.OCMClient)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	_, err = r.OCMClient.ValidateVersion(args.version, versionList,
		r.Cluster.Version().ChannelGroup(), r.Cluster.AWS().STS().RoleARN() == "", r.Cluster.Hypershift().Enabled())
	if err != nil {
		r.Reporter.Errorf("Version '%s' is invalid", args.version)
		os.Exit(1)
	}

	var spin *spinner.Spinner
	if r.Reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		r.Reporter.Infof("Fetching account roles")
		spin.Start()
	}

	accountRoles, err := r.AWSClient.ListAccountRoles(args.version)

	if spin != nil {
		spin.Stop()
	}

	if err != nil {
		r.Reporter.Errorf("Failed to get account roles: %v", err)
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(accountRoles)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(accountRoles) == 0 {
		r.Reporter.Infof("No account roles available")
		os.Exit(0)
	}

	headers := []string{"ROLE NAME", "ROLE TYPE", "ROLE ARN", "OPENSHIFT VERSION", "AWS Managed"}

	var tableData [][]string
	for _, accountRole := range accountRoles {
		awsManaged := "No"
		if accountRole.ManagedPolicy {
			awsManaged = "Yes"
		}
		tableData = append(tableData, []string{
			accountRole.RoleName,
			accountRole.RoleType,
			accountRole.RoleARN,
			accountRole.Version,
			awsManaged,
		})
	}

	if output.ShouldHideEmptyColumns() {
		tableData = output.RemoveEmptyColumns(headers, tableData)
	} else {
		tableData = append([][]string{headers}, tableData...)
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	output.BuildTable(writer, "\t", tableData)
	if err := writer.Flush(); err != nil {
		_ = r.Reporter.Errorf("Failed to flush output: %v", err)
		os.Exit(1)
	}

}
