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

package cluster

import (
	"os"

	"github.com/spf13/cobra"

	uninstallLogs "github.com/openshift/rosa/cmd/logs/uninstall"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	// Watch logs during cluster uninstallation
	watch      bool
	clusterKey string
}

var Cmd = &cobra.Command{
	Use:   "cluster",
	Short: "Delete cluster",
	Long:  "Delete cluster.",
	Example: `  # Delete a cluster named "mycluster"
  rosa delete cluster --cluster=mycluster`,
	Run: run,
}

func init() {
	flags := Cmd.Flags()

	flags.StringVarP(
		&args.clusterKey,
		"cluster",
		"c",
		"",
		"Name or ID of the cluster to delete.",
	)
	Cmd.MarkFlagRequired("cluster")

	flags.BoolVar(
		&args.watch,
		"watch",
		false,
		"Watch cluster uninstallation logs.",
	)
}

func run(cmd *cobra.Command, _ []string) {
	reporter := rprtr.CreateReporterOrExit()
	logger := logging.CreateLoggerOrExit(reporter)

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

	if !confirm.Confirm("delete cluster %s", clusterKey) {
		os.Exit(0)
	}

	reporter.Debugf("Deleting cluster '%s'", clusterKey)
	_, err = ocmClient.DeleteCluster(clusterKey, awsCreator)
	if err != nil {
		reporter.Errorf("Failed to delete cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}
	reporter.Infof("Cluster '%s' will start uninstalling now", clusterKey)

	if args.watch {
		uninstallLogs.Cmd.Run(uninstallLogs.Cmd, []string{clusterKey})
	} else {
		reporter.Infof(
			"To watch your cluster uninstallation logs, run 'rosa logs uninstall -c %s --watch'",
			clusterKey,
		)
	}
}
