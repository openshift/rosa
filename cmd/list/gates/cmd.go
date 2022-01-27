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

	"github.com/nathan-fiscaletti/consolesize-go"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	version    string
	gate       string
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:     "gates",
	Aliases: []string{"gates"},
	Short:   "List available OCP Gates",
	Long:    "List available OCP Gates for a specific OCP release or by cluster upgrade version",
	Example: `  # List all OCP gates for OCP version
  rosa list gates --version 4.9

  # List all STS gates for OCP version
  rosa list gates --gate sts --version 4.9

  # List all OCP gates for OCP version
  rosa list gates --gate ocp --version 4.9

  # List available gates for cluster upgrade version
  rosa list gates -c <cluster_id> --version 4.9.15`,
	Run: run}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster.",
	)

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

var (
	Gates        = []string{GateSTS, GateOCP}
	versionGates = []*v1.VersionGate{}
)

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}

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

	version, err := parseMajorMinor(args.version)
	if err != nil {
		reporter.Errorf("Unable to parse version %s: %v", version, err)
	}

	if args.clusterKey != "" {
		awsCreator, err := awsClient.GetCreator()
		if err != nil {
			reporter.Errorf("Failed to get AWS creator: %v", err)
			os.Exit(1)
		}

		ocm.SetClusterKey(args.clusterKey)

		clusterKey, err := ocm.GetClusterKey()
		if err != nil {
			reporter.Errorf("%s", err)
			os.Exit(1)
		}

		// Try to find the cluster:
		reporter.Debugf("Loading cluster '%s'", clusterKey)
		cluster, err := ocmClient.GetCluster(clusterKey, awsCreator)
		if err != nil {
			reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}

		if cluster.State() != v1.ClusterStateReady {
			reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
			os.Exit(1)
		}

		upgradePolicyBuilder := v1.NewUpgradePolicy().
			ScheduleType("manual").
			Version(args.version)

		upgradePolicy, err := upgradePolicyBuilder.Build()
		if err != nil {
			reporter.Errorf("Failed to schedule upgrade for cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}

		// check if the cluster upgrade requires gate agreements
		versionGates, err = ocmClient.GetMissingGateAgreements(cluster.ID(), upgradePolicy)
		if err != nil {
			reporter.Errorf("Failed to check for missing gate agreements upgrade for "+
				"cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
	}

	if args.clusterKey == "" {
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
	cols, _ := consolesize.GetConsoleSize()
	descriptionSize := float64(cols) * 0.30
	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(writer, "Gate Description\tSTS\tOCP Version\tDocumentation URL\t")

	for _, gate := range versionGates {
		wrappedDescription := wordWrap(strings.TrimSuffix(gate.Description(), "\n"), int(descriptionSize))

		for i, line := range strings.Split(wrappedDescription, "\n") {
			if i == 0 {
				fmt.Fprintf(writer,
					"%s\t%t\t%s\t%s\t\n",
					line,
					gate.STSOnly(),
					gate.VersionRawIDPrefix(),
					gate.DocumentationURL(),
				)
			} else {
				fmt.Fprintf(writer,
					"%s\t \t \t \t\n",
					line,
				)
			}
		}
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

func wordWrap(text string, lineWidth int) (wrapped string) {
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return text
	}
	wrapped = words[0]
	spaceLeft := lineWidth - len(wrapped)
	for _, word := range words[1:] {
		if len(word)+1 > spaceLeft {
			wrapped += "\n" + word
			spaceLeft = lineWidth - len(word)
		} else {
			wrapped += " " + word
			spaceLeft -= 1 + len(word)
		}
	}
	return
}
