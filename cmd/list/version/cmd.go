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

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	channelGroup string
	hostedCp     bool
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
	flags.BoolVar(
		&args.hostedCp,
		"hosted-cp",
		false,
		"Lists only versions that are hosted-cp enabled")
	output.AddFlag(Cmd)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	// Try to find the cluster:
	r.Reporter.Debugf("Fetching versions")
	versions, err := r.OCMClient.GetVersions(args.channelGroup)
	if err != nil {
		r.Reporter.Errorf("Failed to fetch versions: %v", err)
		os.Exit(1)
	}

	var (
		hcpVersions       []*cmv1.Version
		availableVersions []*cmv1.Version
	)

	// Create a separate slice of only hcp-enabled versions
	for _, version := range versions {
		if !version.HostedControlPlaneEnabled() {
			continue
		}
		hcpVersions = append(hcpVersions, version)
	}

	// If hosted-cp arg is supplied, use the hcp enabled versions
	if args.hostedCp {
		versions = hcpVersions
	}

	// Remove disabled versions
	for _, version := range versions {
		if !version.Enabled() {
			continue
		}
		availableVersions = append(availableVersions, version)
	}

	if len(availableVersions) == 0 {
		r.Reporter.Warnf("There are no OpenShift versions available")
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(availableVersions)
		if err != nil {
			r.Reporter.Errorf("%s", err)
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
