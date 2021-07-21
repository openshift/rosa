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

package region

import (
	"fmt"
	"os"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	multiAZ    bool
	roleARN    string
	externalID string
}

var Cmd = &cobra.Command{
	Use:     "regions",
	Aliases: []string{"region"},
	Short:   "List available regions",
	Long:    "List regions that are available for the current AWS account.",
	Example: `  # List all available regions
  rosa list regions`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.BoolVar(
		&args.multiAZ,
		"multi-az",
		false,
		"List only regions with support for multiple availability zones",
	)
	flags.StringVar(
		&args.roleARN,
		"role-arn",
		"",
		"The Amazon Resource Name of the role that the API will assume to fetch available regions.",
	)
	flags.StringVar(
		&args.externalID,
		"external-id",
		"",
		"A unique identifier that might be required when you assume a role in another account",
	)

	output.AddFlag(Cmd)
}

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

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

	// Try to find the cluster:
	reporter.Debugf("Fetching regions")
	regions, err := ocmClient.GetRegions(args.roleARN, args.externalID)
	if err != nil {
		reporter.Errorf("Failed to fetch regions: %v", err)
		os.Exit(1)
	}

	// Filter out unwanted regions
	var availableRegions []*cmv1.CloudRegion
	for _, region := range regions {
		if !region.Enabled() {
			continue
		}
		if cmd.Flags().Changed("multi-az") {
			if args.multiAZ != region.SupportsMultiAZ() {
				continue
			}
		}
		availableRegions = append(availableRegions, region)
	}

	if len(availableRegions) == 0 {
		reporter.Warnf("There are no regions available for this AWS account")
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(availableRegions)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "ID\t\tNAME\t\tMULTI-AZ SUPPORT\n")

	for _, region := range availableRegions {
		fmt.Fprintf(writer,
			"%s\t\t%s\t\t%t\n",
			region.ID(),
			region.DisplayName(),
			region.SupportsMultiAZ(),
		)
	}
	writer.Flush()
}
