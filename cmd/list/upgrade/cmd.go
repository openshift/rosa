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

package upgrade

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "upgrades",
	Aliases: []string{"upgrade"},
	Short:   "List available cluster upgrades",
	Long:    "List available and scheduled cluster version upgrades",
	Run:     run,
}

func init() {
	ocm.AddClusterFlag(Cmd)
}

func run(_ *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	// Try to find the cluster:
	r.Reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := r.OCMClient.GetCluster(clusterKey, r.Creator)
	if err != nil {
		r.Reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	// Load available upgrades for this cluster
	r.Reporter.Debugf("Loading available upgrades for cluster '%s'", clusterKey)
	availableUpgrades, err := r.OCMClient.GetAvailableUpgrades(ocm.GetVersionID(cluster))
	if err != nil {
		r.Reporter.Errorf("Failed to get available upgrades for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if len(availableUpgrades) == 0 {
		r.Reporter.Infof("There are no available upgrades for cluster '%s'", clusterKey)
		os.Exit(0)
	}

	latestRev := latestInCurrentMinor(ocm.GetVersionID(cluster), availableUpgrades)

	r.Reporter.Debugf("Loading scheduled upgrades for cluster '%s'", clusterKey)
	scheduledUpgrade, upgradeState, err := r.OCMClient.GetScheduledUpgrade(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get scheduled upgrades for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(writer, "VERSION\tNOTES\n")
	for i, availableUpgrade := range availableUpgrades {
		notes := ""
		if notes == "" && (i == 0 || availableUpgrade == latestRev) {
			notes = "recommended"
		}
		if availableUpgrade == scheduledUpgrade.Version() {
			notes = fmt.Sprintf("%s for %s", upgradeState.Value(),
				scheduledUpgrade.NextRun().Format("2006-01-02 15:04 MST"))
		}
		fmt.Fprintf(writer, "%s\t%s\n", availableUpgrade, notes)
	}
	writer.Flush()
}

func latestInCurrentMinor(current string, versions []string) string {
	latestVersion := current
	currentParts := strings.Split(current, ".")
	currentPart1, _ := strconv.Atoi(currentParts[1])
	currentPart2, _ := strconv.Atoi(currentParts[2])
	currentRev := currentPart2
	latestRev := currentRev
	for _, version := range versions {
		versionParts := strings.Split(version, ".")
		versionPart1, _ := strconv.Atoi(versionParts[1])
		versionPart2, _ := strconv.Atoi(versionParts[2])
		if currentPart1 != versionPart1 {
			continue
		}
		if versionPart2 > latestRev {
			latestRev = versionPart2
			latestVersion = version
		}
	}
	return latestVersion
}
