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

package version

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	channelGroup string
}

var Cmd = &cobra.Command{
	Use:     "versions",
	Aliases: []string{"version"},
	Short:   "List available versions",
	Long:    "List versions of OpenShift that are available for creating clusters.",
	Example: `  # List all OpenShift versions
  rosa list versions`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.StringVar(
		&args.channelGroup,
		"channel-group",
		ocm.DefaultChannelGroup,
		"List only versions from the specified channel group",
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
	reporter.Debugf("Fetching versions")
	versions, err := ocmClient.GetVersions(args.channelGroup)
	if err != nil {
		reporter.Errorf("Failed to fetch versions: %v", err)
		os.Exit(1)
	}

	var availableVersions []*cmv1.Version

	// Remove disabled versions
	for _, version := range versions {
		if !version.Enabled() {
			continue
		}
		availableVersions = append(availableVersions, version)
	}

	if len(availableVersions) == 0 {
		reporter.Warnf("There are no OpenShift versions available")
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(availableVersions)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "VERSION\t\tDEFAULT\t\tAVAILABLE UPGRADES\n")

	for _, version := range availableVersions {
		isDefault := "no"
		if version.Default() {
			isDefault = "yes"
		}
		fmt.Fprintf(writer,
			"%s\t\t%s\t\t%s\n",
			version.RawID(),
			isDefault,
			strings.Join(version.AvailableUpgrades(), ", "),
		)
	}
	writer.Flush()
}
