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

package ocmroles

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:     "ocm-roles",
	Aliases: []string{"ocmrole", "ocm-role", "ocmroles", "ocm-roles"},
	Short:   "List ocm roles",
	Long:    "List ocm roles for the current AWS account.",
	Example: ` # List all ocm roles
rosa list ocm-roles`,
	Run:    run,
	Hidden: true,
}

func init() {
	output.AddFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

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

	var spin *spinner.Spinner
	if reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		reporter.Infof("Fetching ocm roles")
		spin.Start()
	}

	ocmRoles, err := listOCMRoles(awsClient, ocmClient)

	if spin != nil {
		spin.Stop()
	}

	if err != nil {
		reporter.Errorf("Failed to get ocm roles: %v", err)
		os.Exit(1)
	}

	if len(ocmRoles) == 0 {
		reporter.Infof("No ocm roles available")
		os.Exit(0)
	}
	if output.HasFlag() {
		err = output.Print(ocmRoles)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprint(writer, "ROLE NAME\tROLE ARN\tLINKED\tADMIN\n")
	for _, ocmRole := range ocmRoles {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", ocmRole.RoleName, ocmRole.RoleARN, ocmRole.Linked, ocmRole.Admin)
	}
	writer.Flush()
}

func listOCMRoles(awsClient aws.Client, ocmClient *ocm.Client) ([]aws.Role, error) {
	ocmRoles, err := awsClient.ListOCMRoles()
	if err != nil {
		return nil, err
	}

	// Check if roles are linked to organization
	orgID, _, err := ocmClient.GetCurrentOrganization()
	if err != nil {
		return nil, fmt.Errorf("failed to get organization account: %v", err)
	}
	linkedRoles, err := ocmClient.GetOrganizationLinkedOCMRoles(orgID)
	if err != nil {
		return nil, err
	}

	for i := range ocmRoles {
		if helper.Contains(linkedRoles, ocmRoles[i].RoleARN) {
			ocmRoles[i].Linked = "Yes"
		} else {
			ocmRoles[i].Linked = "No"
		}
	}

	aws.SortRolesByLinkedRole(ocmRoles)

	return ocmRoles, nil
}
