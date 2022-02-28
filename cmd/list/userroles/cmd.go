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

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:     "user-roles",
	Aliases: []string{"userrole", "user-role", "userroles", "user-roles"},
	Short:   "List user roles",
	Long:    "List user roles for current AWS account",
	Example: `# List all user roles
rosa list user-roles`,
	Run:    run,
	Hidden: true,
}

func init() {
	output.AddFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)
	awsClient := aws.CreateNewClientOrExit(logger, reporter)
	ocmClient := ocm.CreateNewClientOrExit(logger, reporter)
	defer func() {
		err := ocmClient.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()

	var spin *spinner.Spinner
	if reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		reporter.Infof("Fetching user roles")
		spin.Start()
	}

	userRoles, err := listUserRoles(awsClient, ocmClient)

	if spin != nil {
		spin.Stop()
	}

	if err != nil {
		reporter.Errorf("Failed to get user roles: %v", err)
		os.Exit(1)
	}

	if len(userRoles) == 0 {
		reporter.Infof("No user roles available")
		os.Exit(0)
	}
	if output.HasFlag() {
		err = output.Print(userRoles)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
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

func listUserRoles(awsClient aws.Client, ocmClient *ocm.Client) ([]aws.Role, error) {
	userRoles, err := awsClient.ListUserRoles()
	if err != nil {
		return nil, err
	}

	// Check if roles are linked to account
	account, err := ocmClient.GetCurrentAccount()
	if err != nil {
		return nil, fmt.Errorf("Failed to get Redhat User Account: %v", err)
	}
	linkedRoles, err := ocmClient.GetAccountLinkedUserRoles(account.ID())
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

	return userRoles, nil
}
