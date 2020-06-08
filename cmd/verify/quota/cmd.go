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

package quota

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/moactl/pkg/aws"
	"github.com/openshift/moactl/pkg/logging"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

var Cmd = &cobra.Command{
	Use:   "quota",
	Short: "Verify AWS quota is ok for cluster install",
	Long:  "Verify AWS quota is ok for cluster install",
	Example: `  # Verify resources needed to create a cluster are configured as expected

  # Verify AWS quotas are configured correctly
  moactl verify quota`,
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

	reporter.Infof("Validating AWS quota...")
	_, err = client.ValidateQuota()
	if err != nil {
		reporter.Errorf("Insufficient AWS quotas")
		reporter.Errorf("%v", err)
		os.Exit(1)
	}
	reporter.Infof("AWS quota ok")
}
