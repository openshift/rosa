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

	"github.com/openshift/rosa/cmd/dlt/oidcprovider"
	"github.com/openshift/rosa/cmd/dlt/operatorrole"
	uninstallLogs "github.com/openshift/rosa/cmd/logs/uninstall"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var args struct {
	// Watch logs during cluster uninstallation
	watch      bool
	bestEffort bool
	mode       string
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
	flags.SortFlags = false

	ocm.AddClusterFlag(Cmd)

	flags.BoolVar(
		&args.bestEffort,
		"best-effort",
		false,
		"Deleting a cluster with \"best effort\" means that certain resources "+
			"may be left behind in the cloud account and will need manual cleanup. "+
			"This option should be used with care.",
	)

	flags.BoolVarP(
		&args.watch,
		"watch",
		"w",
		false,
		"Watch cluster uninstallation logs.",
	)
}

func run(cmd *cobra.Command, _ []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	clusterKey := r.GetClusterKey()

	if !confirm.Confirm("delete cluster %s", clusterKey) {
		os.Exit(0)
	}

	cluster := r.FetchCluster()

	err := handleClusterDelete(r, cluster, clusterKey, args.bestEffort)
	if err != nil {
		r.Reporter.Errorf("%s", err)
		os.Exit(1)
	}

	if cluster.AWS().STS().RoleARN() != "" {
		interactive.Enable()
		r.Reporter.Infof(
			"Your cluster '%s' will be deleted but the following objects may remain",
			clusterKey,
		)
		if len(cluster.AWS().STS().OperatorIAMRoles()) > 0 {
			str := "Operator IAM Roles:"
			for _, operatorIAMRole := range cluster.AWS().STS().OperatorIAMRoles() {
				str = fmt.Sprintf("%s"+
					" - %s\n", str,
					operatorIAMRole.RoleARN())
			}
			r.Reporter.Infof("%s", str)
		}
		r.Reporter.Infof("OIDC Provider : %s\n", cluster.AWS().STS().OIDCEndpointURL())
		r.Reporter.Infof("Once the cluster is uninstalled use the following commands to remove the " +
			"above aws resources.\n")
		commands := buildCommands(cluster)
		fmt.Print(commands, "\n")
	}
	if args.watch {
		uninstallLogs.Cmd.Run(uninstallLogs.Cmd, []string{clusterKey})
	} else {
		r.Reporter.Infof("To watch your cluster uninstallation logs, run 'rosa logs uninstall -c %s --watch'",
			clusterKey,
		)
	}
}

func handleClusterDelete(r *rosa.Runtime, cluster *cmv1.Cluster, clusterKey string, bestEffort bool) error {
	clusterState, err := r.OCMClient.GetClusterState(cluster.ID())
	if err != nil {
		return err
	}

	if clusterState == cmv1.ClusterStateUninstalling {
		r.Reporter.Infof("cluster '%s' is already uninstalling", clusterKey)
		return nil
	}

	r.Reporter.Debugf("Deleting cluster '%s'", clusterKey)
	_, err = r.OCMClient.DeleteCluster(clusterKey, bestEffort, r.Creator)
	if err != nil {
		return err
	}

	r.Reporter.Infof("Cluster '%s' will start uninstalling now", clusterKey)
	return nil
}

func buildCommands(cluster *cmv1.Cluster) string {
	commands := []string{}
	deleteOperatorRole := fmt.Sprintf("\trosa delete operator-roles -c %s", cluster.ID())
	deleteOIDCProvider := fmt.Sprintf("\trosa delete oidc-provider -c %s", cluster.ID())
	if ocm.IsOidcConfigReusable(cluster) {
		deleteOperatorRole = fmt.Sprintf("\trosa delete operator-roles --%s %s",
			operatorrole.PrefixFlag, cluster.AWS().STS().OperatorRolePrefix())
		deleteOIDCProvider = fmt.Sprintf("\trosa delete oidc-provider --%s %s",
			oidcprovider.OidcConfigIdFlag, cluster.AWS().STS().OidcConfig().ID())
	}
	commands = append(commands, deleteOperatorRole, deleteOIDCProvider)
	return strings.Join(commands, "\n")
}
