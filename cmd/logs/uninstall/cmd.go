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

	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	tail  int
	watch bool
}

var Cmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Show cluster uninstallation logs",
	Long:  "Show cluster uninstallation logs",
	Example: `  # Show last 100 uninstall log lines for a cluster named "mycluster"
  rosa logs uninstall mycluster --tail=100

  # Show uninstall logs for a cluster using the --cluster flag
  rosa logs uninstall --cluster=mycluster`,
	Run:  run,
	Args: cobra.MaximumNArgs(1),
}

func init() {
	flags := Cmd.Flags()

	ocm.AddClusterFlag(Cmd)

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
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	// Determine whether the user wants to watch logs streaming.
	// We check the flag value this way to allow other commands to watch logs
	watch := cmd.Flags().Lookup("watch").Value.String() == "true"

	var err error

	// Allow the command to be called programmatically
	if len(argv) == 1 && !cmd.Flag("cluster").Changed {
		ocm.SetClusterKey(argv[0])
		watch = true
	}
	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateUninstalling && !watch {
		r.Reporter.Warnf("Cluster '%s' is not currently uninstalling", clusterKey)
		os.Exit(1)
	}

	if cluster.State() == cmv1.ClusterStateInstalling ||
		cluster.State() == cmv1.ClusterStatePending ||
		cluster.State() == cmv1.ClusterStateWaiting {
		r.Reporter.Errorf("Cluster '%s' is in '%s' state and no uninstallation logs are available",
			clusterKey, cluster.State(),
		)
		os.Exit(1)
	}

	// Get logs from Hive
	logs, err := r.OCMClient.GetUninstallLogs(cluster.ID(), args.tail)
	if err != nil {
		if errors.GetType(err) == errors.NotFound {
			r.Reporter.Warnf("Logs for cluster '%s' are not available", clusterKey)
		} else {
			r.Reporter.Errorf("Failed to get logs for cluster '%s': %v", clusterKey, err)
			os.Exit(1)
		}
	}
	printLog(logs, nil)

	if watch {
		var spin *spinner.Spinner
		if r.Reporter.IsTerminal() {
			spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
		}
		if spin != nil {
			spin.Start()
		}

		// Poll for changing logs:
		response, err := r.OCMClient.PollUninstallLogs(cluster.ID(), func(logResponse *cmv1.LogGetResponse) bool {
			state, err := r.OCMClient.GetClusterState(cluster.ID())
			if err != nil || state == cmv1.ClusterState("") {
				r.Reporter.Infof("Cluster '%s' completed uninstallation", clusterKey)
				os.Exit(0)
			}
			printLog(logResponse.Body(), spin)

			err = r.OCMClient.KeepTokensAlive()
			if err != nil {
				r.Reporter.Errorf(fmt.Sprintf("Failed to keep tokens alive for polling: %v", err))
				os.Exit(1)
			}

			return false
		})
		if err != nil {
			if errors.GetType(err) != errors.NotFound {
				r.Reporter.Errorf(fmt.Sprintf("Failed to watch logs for cluster '%s': %v", clusterKey, err))
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
