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

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	version string
}

var hfEnabled = hyperfleet.Enabled

var Cmd = &cobra.Command{
	Use:     "account-roles",
	Aliases: []string{"accountrole", "account-role", "accountroles"},
	Short:   "List ROSA account-role IAM resources",
	Long:    "List ROSA account-role IAM resources found in the current AWS account. Not available in HyperFleet mode.",
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
}

func run(_ *cobra.Command, _ []string) {
	if hfEnabled() {
		r := rosa.NewRuntime()
		r.Reporter.Errorf("Account roles are not supported in HyperFleet mode")
		os.Exit(1)
	}

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

	listAccountRoles(r)
}

func listAccountRoles(r *rosa.Runtime) {
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
		if err := output.Print(accountRoles); err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		return
	}

	if len(accountRoles) == 0 {
		r.Reporter.Infof("No account roles available")
		return
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "ROLE NAME\tROLE TYPE\tROLE ARN\tOPENSHIFT VERSION\tAWS Managed\n")
	for _, accountRole := range accountRoles {
		awsManaged := "No"
		if accountRole.ManagedPolicy {
			awsManaged = "Yes"
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
			accountRole.RoleName,
			accountRole.RoleType,
			accountRole.RoleARN,
			accountRole.Version,
			awsManaged,
		)
	}
	writer.Flush()
}
