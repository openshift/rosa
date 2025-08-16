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
	consolesize "github.com/nathan-fiscaletti/consolesize-go"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
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
	Run:  run,
	Args: cobra.NoArgs,
}

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

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	version, err := parseMajorMinor(args.version)
	if err != nil {
		r.Reporter.Errorf("Unable to parse version %s: %v", version, err)
	}

	if args.clusterKey != "" {
		r = r.WithAWS()
		ocm.SetClusterKey(args.clusterKey)

		clusterKey := r.GetClusterKey()
		cluster := r.FetchCluster()

		if cluster.State() != v1.ClusterStateReady {
			r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
			os.Exit(1)
		}

		upgradePolicyBuilder := v1.NewUpgradePolicy().
			ScheduleType(v1.ScheduleTypeManual).
			Version(args.version)

		upgradePolicy, err := upgradePolicyBuilder.Build()
		if err != nil {
			r.Reporter.Errorf("Failed to schedule upgrade for cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}

		// check if the cluster upgrade requires gate agreements
		versionGates, err = r.OCMClient.GetMissingGateAgreementsClassic(cluster.ID(), upgradePolicy)
		if err != nil {
			r.Reporter.Errorf("Failed to check for missing gate agreements upgrade for "+
				"cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
	} else {
		// Query OCM for available OCP gates
		r.Reporter.Debugf("Fetching available gates")
		switch args.gate {
		case GateSTS:
			versionGates, err = r.OCMClient.ListStsGates(version)
			if err != nil {
				r.Reporter.Errorf("Failed to fetch available %s gates for OCP version %s: %v", args.gate, args.version, err)
				os.Exit(1)
			}
		case GateOCP:
			versionGates, err = r.OCMClient.ListOcpGates(version)
			if err != nil {
				r.Reporter.Errorf("Failed to fetch available %s gates for OCP version %s: %v", args.gate, args.version, err)
				os.Exit(1)
			}
		case "":
			versionGates, err = r.OCMClient.ListAllOcpGates(version)
			if err != nil {
				r.Reporter.Errorf("Failed to fetch available %s gates for OCP version %s: %v", args.gate, args.version, err)
				os.Exit(1)
			}
		default:
			r.Reporter.Errorf("Invalid gate. Allowed values are %s and \"\" for all", strings.Join(Gates, ","))
			os.Exit(1)
		}

		if err != nil {
			r.Reporter.Errorf("Failed to fetch available OCP gates for OCP version %s: %v", err, args.version)
			os.Exit(1)
		}

		if len(versionGates) == 0 {
			r.Reporter.Warnf("There are no gates for OCP version %s", args.version)
			os.Exit(1)
		}
	}

	if output.HasFlag() {
		err = output.Print(versionGates)
		if err != nil {
			r.Reporter.Errorf("%s", err)
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
