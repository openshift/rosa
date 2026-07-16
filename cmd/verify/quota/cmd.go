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
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:   "quota",
	Short: "Verify AWS quota is ok for cluster install",
	Long:  "Verify AWS quota needed to create a cluster is configured as expected",
	Example: `  # Verify AWS quotas are configured correctly
  rosa verify quota

  # Verify AWS quotas in a different region
  rosa verify quota --region=us-west-2`,
	Args: cobra.NoArgs,
	Run:  run,
}

func init() {
	flags := Cmd.Flags()

	arguments.AddRegionFlag(flags)
	arguments.AddProfileFlag(flags)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	err := runWithRuntime(r)
	r.Cleanup()
	if err != nil {
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime) error {
	region, err := aws.GetRegion(arguments.GetRegion())
	if err != nil {
		r.Reporter.Errorf("Error getting region: %v", err)
		return fmt.Errorf("getting AWS region for quota verification: %w", err)
	}

	if r.AWSClient == nil {
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
			return fmt.Errorf("building AWS client for quota verification: %w", err)
		}
	}

	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("Validating AWS quota...")
	}
	ok, err := r.AWSClient.ValidateQuota()
	if err != nil {
		r.OCMClient.LogEvent("ROSAVerifyQuotaInsufficient", nil)
		r.Reporter.Errorf("Insufficient AWS quotas")
		r.Reporter.Errorf("%v", err)
		return fmt.Errorf("validating AWS quotas: %w", err)
	}
	if !ok {
		r.OCMClient.LogEvent("ROSAVerifyQuotaInsufficient", nil)
		r.Reporter.Errorf("Insufficient AWS quotas")
		return fmt.Errorf("validating AWS quotas: insufficient AWS quotas")
	}
	if r.Reporter.IsTerminal() {
		r.Reporter.Infof("AWS quota ok. " +
			"If cluster installation fails, validate actual AWS resource usage against " +
			"https://docs.openshift.com/rosa/rosa_getting_started/rosa-required-aws-service-quotas.html")
	}
	return nil
}
