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
	"fmt"
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	uninstallLogs "github.com/openshift/rosa/cmd/logs/uninstall"
	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	rprtr "github.com/openshift/rosa/pkg/reporter"
)

var args struct {
	// Watch logs during cluster uninstallation
	watch bool
	mode  string
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

	ocm.AddClusterFlag(Cmd)

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

	clusterKey, err := ocm.GetClusterKey()
	if err != nil {
		reporter.Errorf("%s", err)
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
	cluster, err := ocmClient.DeleteCluster(clusterKey, awsCreator)
	if err != nil {
		reporter.Errorf("Failed to delete cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}
	reporter.Infof("Cluster '%s' will start uninstalling now", clusterKey)

	if cluster.AWS().STS().RoleARN() != "" {
		interactive.Enable()
		reporter.Infof(
			"Your cluster '%s' will be deleted but the following object may remain",
			clusterKey,
		)
		if len(cluster.AWS().STS().OperatorIAMRoles()) > 0 {
			str := "Operator IAM Roles:"
			for _, operatorIAMRole := range cluster.AWS().STS().OperatorIAMRoles() {
				str = fmt.Sprintf("%s"+
					" - %s\n", str,
					operatorIAMRole.RoleARN())
			}
			reporter.Infof("%s", str)
		}
		reporter.Infof("OIDC Provider : %s\n", cluster.AWS().STS().OIDCEndpointURL())
		reporter.Infof("Once the cluster is uninstalled use the following commands to remove the " +
			"above aws resource.\n")
		commands := buildCommands(cluster)
		fmt.Print(commands, "\n")
	}
	if args.watch {
		uninstallLogs.Cmd.Run(uninstallLogs.Cmd, []string{clusterKey})
	} else {
		reporter.Infof("To watch your cluster uninstallation logs, run 'rosa logs uninstall -c %s --watch'",
			clusterKey,
		)
	}
}

func buildCommands(cluster *cmv1.Cluster) string {
	commands := []string{}
	deleteOperatorRole := fmt.Sprintf("\trosa delete operator-roles -c %s", cluster.ID())
	deleteOIDCProvider := fmt.Sprintf("\trosa delete oidc-provider -c %s", cluster.ID())
	commands = append(commands, deleteOperatorRole, deleteOIDCProvider)
	return strings.Join(commands, "\n")
}
