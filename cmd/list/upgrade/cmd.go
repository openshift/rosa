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

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/ocm/upgrades"
	"github.com/openshift/rosa/pkg/ocm/versions"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:     "upgrades",
	Aliases: []string{"upgrade"},
	Short:   "List available cluster upgrades",
	Long:    "List available and scheduled cluster version upgrades",
	Run:     run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to list the upgrades of (required).",
	)
	Cmd.MarkFlagRequired("cluster")
}

func run(_ *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	clusterKey := args.clusterKey
	if !ocm.IsValidClusterKey(clusterKey) {
		reporter.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
		os.Exit(1)
	}

	// Create the client for the OCM API:
	ocmConnection, err := ocm.NewConnection().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create OCM connection: %v", err)
		os.Exit(1)
	}
	defer func() {
		err = ocmConnection.Close()
		if err != nil {
			reporter.Errorf("Failed to close OCM connection: %v", err)
		}
	}()
	ocmClient := ocmConnection.ClustersMgmt().V1()

	// Create the AWS client:
	awsClient, err := aws.NewClient().
		Logger(logger).
		Build()
	if err != nil {
		reporter.Errorf("Failed to create AWS client: %v", err)
		os.Exit(1)
	}

	awsCreator, err := awsClient.GetCreator()
	if err != nil {
		reporter.Errorf("Failed to get AWS creator: %v", err)
		os.Exit(1)
	}

	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocm.GetCluster(ocmClient.Clusters(), clusterKey, awsCreator.ARN)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateReady {
		reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	// Load available upgrades for this cluster
	reporter.Debugf("Loading available upgrades for cluster '%s'", clusterKey)
	availableUpgrades, err := versions.GetAvailableUpgrades(ocmClient, versions.GetVersionID(cluster))
	if err != nil {
		reporter.Errorf("Failed to get available upgrades for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if len(availableUpgrades) == 0 {
		reporter.Infof("There are no available upgrades for cluster '%s'", clusterKey)
		os.Exit(0)
	}

	latestRev := latestInCurrentMinor(versions.GetVersionID(cluster), availableUpgrades)

	reporter.Debugf("Loading scheduled upgrades for cluster '%s'", clusterKey)
	scheduledUpgrade, upgradeState, err := upgrades.GetScheduledUpgrade(ocmClient, cluster.ID())
	if err != nil {
		reporter.Errorf("Failed to get scheduled upgrades for cluster '%s': %v", clusterKey, err)
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
