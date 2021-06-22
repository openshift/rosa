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
	"time"

	"github.com/briandowns/spinner"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	clusterKey string
	tail       int
	watch      bool
}

var Cmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Show cluster uninstallation logs",
	Long:  "Show cluster uninstallation logs",
	Example: `  # Show last 100 uninstall log lines for a cluster named "mycluster"
  rosa logs uninstall mycluster --tail=100

  # Show uninstall logs for a cluster using the --cluster flag
  rosa logs uninstall --cluster=mycluster`,
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
	Cmd.MarkFlagRequired("cluster")

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

	// Allow the command to be called programmatically
	if len(argv) == 1 && !cmd.Flag("cluster").Changed {
		args.clusterKey = argv[0]
		watch = true
	}

	clusterKey := args.clusterKey
	// Check that the cluster key (name, identifier or external identifier) given by the user
	// is reasonably safe so that there is no risk of SQL injection:
	if !ocm.IsValidClusterKey(clusterKey) {
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
	reporter.Debugf("Loading cluster '%s'", clusterKey)
	cluster, err := ocmClient.GetCluster(clusterKey, awsCreator)
	if err != nil {
		reporter.Errorf("Failed to get cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if cluster.State() != cmv1.ClusterStateUninstalling && !watch {
		reporter.Warnf("Cluster '%s' is not currently uninstalling", clusterKey)
		os.Exit(1)
	}

	if cluster.State() == cmv1.ClusterStateInstalling ||
		cluster.State() == cmv1.ClusterStatePending {
		reporter.Errorf("Cluster '%s' is in '%s' state and no uninstallation logs are available",
			clusterKey, cluster.State(),
		)
		os.Exit(1)
	}

	// Get logs from Hive
	logs, err := ocmClient.GetUninstallLogs(cluster.ID(), args.tail)
	if err != nil {
		if errors.GetType(err) == errors.NotFound {
			reporter.Warnf("Logs for cluster '%s' are not available", clusterKey)
		} else {
			reporter.Errorf("Failed to get logs for cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
	}
	printLog(logs, nil)

	if watch {
		spin := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		spin.Start()

		// Poll for changing logs:
		response, err := ocmClient.PollUninstallLogs(cluster.ID(), func(logResponse *cmv1.LogGetResponse) bool {
			state, err := ocmClient.GetClusterState(cluster.ID())
			if err != nil || state == cmv1.ClusterState("") {
				return true
			}
			printLog(logResponse.Body(), spin)
			return false
		})
		if err != nil {
			if errors.GetType(err) != errors.NotFound {
				reporter.Errorf(fmt.Sprintf("Failed to watch logs for cluster '%s': %v", clusterKey, err))
				os.Exit(1)
			}
		}
		printLog(response, spin)
	}
}

var lastLine string

// Print next log lines
func printLog(logs *cmv1.Log, spin *spinner.Spinner) {
	lines := findNextLines(logs)
	if lines != "" {
		fmt.Printf("%s\n", lines)
		if spin != nil {
			spin.Stop()
		}
	} else if spin != nil {
		spin.Restart()
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
