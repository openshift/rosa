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

package uninstall

import (
	"fmt"
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/moactl/pkg/aws"
	clusterprovider "github.com/openshift/moactl/pkg/cluster"
	"github.com/openshift/moactl/pkg/logging"
	"github.com/openshift/moactl/pkg/ocm"
	rprtr "github.com/openshift/moactl/pkg/reporter"
)

var args struct {
	clusterKey string
	tail       int
	watch      bool
}

var Cmd = &cobra.Command{
	Use:   "uninstall [ID|NAME]",
	Short: "Show cluster uninstallation logs",
	Long:  "Show cluster uninstallation logs",
	Example: `  # Show last 100 uninstall log lines for a cluster named "mycluster"
  moactl logs uninstall mycluster --tail=100

  # Show uninstall logs for a cluster using the --cluster flag
  moactl logs uninstall --cluster=mycluster`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to get logs for.",
	)

	flags.IntVar(
		&args.tail,
		"tail",
		2000,
		"Number of lines to get from the end of the log.",
	)

	flags.BoolVarP(
		&args.watch,
		"watch",
		"w",
		false,
		"After getting the logs, watch for changes.",
	)
}

func run(cmd *cobra.Command, argv []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

	// Determine whether the user wants to watch logs streaming.
	// We check the flag value this way to allow other commands to watch logs
	watch := cmd.Flags().Lookup("watch").Value.String() == "true"

	// Check command line arguments:
	clusterKey := args.clusterKey
	if clusterKey == "" {
		if len(argv) != 1 {
			reporter.Errorf(
				"Expected exactly one command line argument or flag containing the name " +
					"or identifier of the cluster",
			)
			os.Exit(1)
		}
		clusterKey = argv[0]
	}

	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	if !clusterprovider.IsValidClusterKey(clusterKey) {
		reporter.Errorf(
			"Cluster name, identifier or external identifier '%s' isn't valid: it "+
				"must contain only letters, digits, dashes and underscores",
			clusterKey,
		)
		os.Exit(1)
	}

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

	// Get the client for the OCM collection of clusters:
	clustersCollection := ocmConnection.ClustersMgmt().V1().Clusters()

	// Try to find the cluster:
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := clusterprovider.GetCluster(clustersCollection, clusterKey, awsCreator.ARN)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateUninstalling && !watch {
		reporter.Warnf("Cluster '%s' is not currently uninstalling", clusterKey)
		os.Exit(1)
	}

	// Get logs from Hive
	logs, err := ocm.GetUninstallLogs(clustersCollection, cluster.ID(), args.tail)
	if err != nil {
		if errors.GetType(err) == errors.NotFound {
			reporter.Warnf("Logs for cluster '%s' are not available", clusterKey)
			if watch {
				reporter.Warnf("Waiting...")
			}
		} else {
			reporter.Errorf("Failed to get logs for cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
	}
	printLog(logs)

	if watch {
		// Poll for changing logs:
		response, err := ocm.PollUninstallLogs(clustersCollection, cluster.ID(), func(logResponse *cmv1.LogGetResponse) bool {
			state, err := ocm.GetClusterState(clustersCollection, cluster.ID())
			if err != nil || state == cmv1.ClusterState("") {
				return true
			}
			printLog(logResponse.Body())
			return false
		})
		if err != nil {
			if errors.GetType(err) != errors.NotFound {
				reporter.Errorf(fmt.Sprintf("Failed to watch logs for cluster '%s': %v", clusterKey, err))
				os.Exit(1)
			}
		}
		printLog(response)
	}
}

var lastLine string

// Print next log lines
func printLog(logs *cmv1.Log) {
	lines := findNextLines(logs)
	if lines != "" {
		fmt.Printf("%s\n", lines)
	}
}

// Remove duplicate lines from the log poll response
func findNextLines(logs *cmv1.Log) string {
	lines := strings.Split(logs.Content(), "\n")
	// Last element is always empty, remove it
	if len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}
	// Find where the new logs and the last line overlap
	for i, line := range lines {
		if lastLine != "" && line == lastLine {
			// Remove any duplicate lines
			lines = lines[i+1:]
			break
		}
	}
	// Store the last log lne
	if len(lines) > 0 {
		lastLine = lines[len(lines)-1]
	}
	return strings.Join(lines, "\n")
}
