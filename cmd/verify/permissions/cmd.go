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

	"github.com/spf13/cobra"

	"github.com/openshift/moactl/pkg/aws"
	"github.com/openshift/moactl/pkg/logging"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:   "permissions",
	Short: "Verify AWS permissions are ok for cluster install",
	Long:  "Verify AWS permissions are ok for cluster install",
	Example: `  # Verify resources needed to create a cluster are configured as expected

  # Verify AWS permissions are configured correctly
  moactl verify permissions`,
	Run: run,
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Create the AWS client:
	client, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Error creating AWS client: %v", err)
		os.Exit(1)
	}

	reporter.Infof("Validating SCP policies...")
	ok, err := client.ValidateSCP()
	if err != nil {
		reporter.Errorf("Unable to validate SCP policies")
		reporter.Errorf("%v", err)
		os.Exit(1)
	}
	if !ok {
		reporter.Warnf("Failed to validate SCP policies. Will try to continue anyway...")
	}
	reporter.Infof("AWS SCP policies ok")
}
