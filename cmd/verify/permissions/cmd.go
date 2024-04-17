/*
Copyright (c) 2020 Red Hat, Inc.

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

package permissions

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "permissions",
	Aliases: []string{"scp"},
	Short:   "Verify AWS permissions are ok for non-STS cluster install",
	Long:    "Verify AWS permissions needed to create a non-STS cluster are configured as expected",
	Example: `  # Verify AWS permissions are configured correctly
  rosa verify permissions

  # Verify AWS permissions in a different region
  rosa verify permissions --region=us-west-2`,
	Run:  run,
	Args: cobra.NoArgs,
}

func init() {
	flags := Cmd.Flags()

	arguments.AddProfileFlag(flags)
	arguments.AddRegionFlag(flags)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	// Get AWS region
	region, err := aws.GetRegion(arguments.GetRegion())
	if err != nil {
		r.Reporter.Errorf("Error getting region: %v", err)
		os.Exit(1)
	}

	// Create the AWS client:
	r.AWSClient, err = aws.NewClient().
		Logger(r.Logger).
		Region(region).
		Build()
	if err != nil {
		// FIXME Hack to capture errors due to using STS accounts
		if strings.Contains(fmt.Sprintf("%s", err), "STS") {
			r.OCMClient.LogEvent("ROSAInitCredentialsSTS", nil)
		}
		r.Reporter.Errorf("Error creating AWS client: %v", err)
		os.Exit(1)
	}

	r.Reporter.Infof("Verifying permissions for non-STS clusters")
	r.Reporter.Infof("Validating SCP policies...")
	policies, err := r.OCMClient.GetPolicies("OSDSCPPolicy")
	if err != nil {
		r.Reporter.Errorf("Failed to get 'osdscppolicy' for '%s': %v", aws.AdminUserName, err)
		os.Exit(1)
	}
	ok, err := r.AWSClient.ValidateSCP(nil, policies)
	if err != nil {
		r.OCMClient.LogEvent("ROSAVerifyPermissionsSCPFailed", nil)
		r.Reporter.Errorf("Unable to validate SCP policies. Make sure that an organizational " +
			"SCP is not preventing this account from performing the required checks")
		if strings.Contains(err.Error(), "Throttling: Rate exceeded") {
			r.Reporter.Errorf("Throttling: Rate exceeded. Please wait 3-5 minutes before retrying.")
			os.Exit(1)
		}
		r.Reporter.Errorf("%v", err)
		os.Exit(1)
	}
	if !ok {
		r.OCMClient.LogEvent("ROSAVerifyPermissionsSCPInvalid", nil)
		r.Reporter.Warnf("Failed to validate SCP policies. Will try to continue anyway...")
	}
	r.Reporter.Infof("AWS SCP policies ok")
}
