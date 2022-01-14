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

package gates

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	semver "github.com/hashicorp/go-version"

	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	version string
	gate    string
}

var Cmd = &cobra.Command{
	Use:     "gates",
	Aliases: []string{"gates"},
	Short:   "List available OCP Gates",
	Long:    "List available OCP Gates.",
	Example: `  # List all available OCP Gates 
  rosa list gates --version 4.9`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()
	flags.StringVar(
		&args.version,
		"version",
		"",
		"OCP version",
	)
	flags.StringVar(
		&args.gate,
		"gate",
		"",
		"Gate type",
	)

	Cmd.MarkFlagRequired("version")

	output.AddFlag(Cmd)
}

const (
	GateSTS = "sts"
	GateOCP = "ocp"
)

var Gates = []string{GateSTS, GateOCP}

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

	var versionGates []*v1.VersionGate

	version, err := parseMajorMinor(args.version)
	if err != nil {
		reporter.Errorf("Unable to parse version %s: %v", version, err)
	}

	// Query OCM for available OCP gates
	reporter.Debugf("Fetching available gates")
	switch args.gate {
	case GateSTS:
		versionGates, err = ocmClient.ListStsGates(version)
		if err != nil {
			reporter.Errorf("Failed to fetch available %s gates for OCP version %s: %v", args.gate, args.version, err)
			os.Exit(1)
		}
	case GateOCP:
		versionGates, err = ocmClient.ListOcpGates(version)
		if err != nil {
			reporter.Errorf("Failed to fetch available %s gates for OCP version %s: %v", args.gate, args.version, err)
			os.Exit(1)
		}
	case "":
		versionGates, err = ocmClient.ListAllOcpGates(version)
		if err != nil {
			reporter.Errorf("Failed to fetch available %s gates for OCP version %s: %v", args.gate, args.version, err)
			os.Exit(1)
		}
	default:
		reporter.Errorf("Invalid gate. Allowed values are %s and \"\" for all", strings.Join(Gates, ","))
		os.Exit(1)
	}

	if err != nil {
		reporter.Errorf("Failed to fetch available OCP gates for OCP version %s: %v", err, args.version)
		os.Exit(1)
	}

	if len(versionGates) == 0 {
		reporter.Warnf("There are no gates for OCP version %s", args.version)
		os.Exit(1)
	}

	if output.HasFlag() {
		err = output.Print(versionGates)
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "Gate Description\t\tSTS\t\tOCP Version\t\tDocumentation URL\n")

	for _, gate := range versionGates {
		fmt.Fprintf(writer,
			"%s\t\t%t\t\t%s\t\t%s\n",
			gate.Description(),
			gate.STSOnly(),
			gate.VersionRawIDPrefix(),
			gate.DocumentationURL(),
		)
	}
	writer.Flush()
}

func parseMajorMinor(version string) (string, error) {
	parsedVersion, err := semver.NewVersion(version)
	if err != nil {
		return "", err
	}
	versionSplit := parsedVersion.Segments64()
	return fmt.Sprintf("%d.%d",
		versionSplit[0], versionSplit[1]), err
}
