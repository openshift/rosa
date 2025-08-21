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
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "ocm-roles",
	Aliases: []string{"ocmrole", "ocm-role", "ocmroles", "ocm-roles"},
	Short:   "List ocm roles",
	Long:    "List ocm roles for the current AWS account.",
	Example: ` # List all ocm roles
rosa list ocm-roles`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	output.AddFlag(Cmd)
	output.AddHideEmptyColumnsFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	var spin *spinner.Spinner
	if r.Reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		r.Reporter.Infof("Fetching ocm roles")
		spin.Start()
	}

	ocmRoles, err := listOCMRoles(r)

	if spin != nil {
		spin.Stop()
	}

	if err != nil {
		r.Reporter.Errorf("Failed to get ocm roles: %v", err)
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(ocmRoles)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(ocmRoles) == 0 {
		r.Reporter.Infof("No ocm roles available")
		os.Exit(0)
	}

	headers := []string{"ROLE NAME", "ROLE ARN", "LINKED", "ADMIN", "AWS Managed"}
	var tableData [][]string
	for _, ocmRole := range ocmRoles {
		awsManaged := "No"
		if ocmRole.ManagedPolicy {
			awsManaged = "Yes"
		}
		row := []string{
			ocmRole.RoleName,
			ocmRole.RoleARN,
			ocmRole.Linked,
			ocmRole.Admin,
			awsManaged,
		}
		tableData = append(tableData, row)
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

func listOCMRoles(r *rosa.Runtime) ([]aws.Role, error) {
	ocmRoles, err := r.AWSClient.ListOCMRoles()

	if err != nil {
		return nil, err
	}

	// If there are no roles, return an empty slice to the caller and avoid additional work
	if len(ocmRoles) == 0 {
		return []aws.Role{}, nil
	}

	// Check if roles are linked to organization
	orgID, _, err := r.OCMClient.GetCurrentOrganization()
	if err != nil {
		return nil, fmt.Errorf("failed to get organization account: %v", err)
	}
	linkedRoles, err := r.OCMClient.GetOrganizationLinkedOCMRoles(orgID)
	if err != nil {
		return nil, err
	}

	linkedRolesMap := helper.SliceToMap(linkedRoles)
	for i := range ocmRoles {
		_, exist := linkedRolesMap[ocmRoles[i].RoleARN]
		if exist {
			ocmRoles[i].Linked = "Yes"
		} else {
			ocmRoles[i].Linked = "No"
		}
	}

	aws.SortRolesByLinkedRole(ocmRoles)

	return ocmRoles, nil
}
