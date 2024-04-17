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

package userroles

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
	Use:     "user-roles",
	Aliases: []string{"userrole", "user-role", "userroles", "user-roles"},
	Short:   "List user roles",
	Long:    "List user roles for current AWS account",
	Example: `# List all user roles
rosa list user-roles`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	output.AddFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	var spin *spinner.Spinner
	if r.Reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		r.Reporter.Infof("Fetching user roles")
		spin.Start()
	}

	userRoles, err := listUserRoles(r)

	if spin != nil {
		spin.Stop()
	}

	if err != nil {
		r.Reporter.Errorf("Failed to get user roles: %v", err)
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(userRoles)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(userRoles) == 0 {
		r.Reporter.Infof("No user roles available")
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprint(writer, "ROLE NAME\tROLE ARN\tLINKED\n")
	for _, userRole := range userRoles {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", userRole.RoleName, userRole.RoleARN, userRole.Linked)
	}
	writer.Flush()
}

func listUserRoles(r *rosa.Runtime) ([]aws.Role, error) {
	userRoles, err := r.AWSClient.ListUserRoles()
	if err != nil {
		return nil, err
	}

	// If no roles available, return empty slice to avoid further work
	if len(userRoles) == 0 {
		return []aws.Role{}, nil
	}

	// Check if roles are linked to account
	account, err := r.OCMClient.GetCurrentAccount()
	if err != nil {
		return nil, fmt.Errorf("Failed to get Redhat User Account: %v", err)
	}
	linkedRoles, err := r.OCMClient.GetAccountLinkedUserRoles(account.ID())
	if err != nil {
		return nil, err
	}

	linkedRolesMap := helper.SliceToMap(linkedRoles)
	for i := range userRoles {
		_, exist := linkedRolesMap[userRoles[i].RoleARN]
		if exist {
			userRoles[i].Linked = "Yes"
		} else {
			userRoles[i].Linked = "No"
		}
	}

	aws.SortRolesByLinkedRole(userRoles)

	return userRoles, nil
}
