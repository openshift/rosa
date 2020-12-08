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
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:   "permissions",
	Short: "Verify AWS permissions are ok for cluster install",
	Long:  "Verify AWS permissions needed to create a cluster are configured as expected",
	Example: `  # Verify AWS permissions are configured correctly
  rosa verify permissions

  # Verify AWS permissions in a different region
  rosa verify permissions --region=us-west-2`,
	Run: run,
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Get AWS region
	region, err := aws.GetRegion(cmd.Flags().Lookup("region").Value.String())
	if err != nil {
		reporter.Errorf("Error getting region: %v", err)
		os.Exit(1)
	}

	// Create the AWS client:
	client, err := aws.NewClient().
		Logger(logger).
		Region(region).
		Build()
	if err != nil {
		reporter.Errorf("Error creating AWS client: %v", err)
		os.Exit(1)
	}

	reporter.Infof("Validating SCP policies...")
	ok, err := client.ValidateSCP(nil)
	if err != nil {
		reporter.Errorf("Unable to validate SCP policies")
		if strings.Contains(err.Error(), "Throttling: Rate exceeded") {
			reporter.Errorf("Throttling: Rate exceeded. Please wait 3-5 minutes before retrying.")
			os.Exit(1)
		}
		reporter.Errorf("%v", err)
		os.Exit(1)
	}
	if !ok {
		reporter.Warnf("Failed to validate SCP policies. Will try to continue anyway...")
	}
	reporter.Infof("AWS SCP policies ok")
}
