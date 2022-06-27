/*
Copyright (c) 2021 Red Hat, Inc.

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

package addon

import (
	"fmt"
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "addon ID",
	Aliases: []string{"addons", "add-on", "add-ons"},
	Short:   "Uninstall add-on from cluster",
	Long:    "Uninstall Red Hat managed add-on from a cluster",
	Example: `  # Remove the CodeReady Workspaces add-on installation from the cluster
  rosa uninstall addon --cluster=mycluster codeready-workspaces`,
	Run: run,
	Args: func(_ *cobra.Command, argv []string) error {
		if len(argv) != 1 {
			return fmt.Errorf("Expected exactly one command line parameter containing the id of the add-on")
		}
		return nil
	},
}

func init() {
	flags := Cmd.Flags()
	confirm.AddFlag(flags)
	ocm.AddClusterFlag(Cmd)
}

func run(_ *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	addOnID := argv[0]

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

	addOn, _ := r.OCMClient.GetAddOnInstallation(cluster.ID(), addOnID)
	if addOn == nil {
		r.Reporter.Warnf("Addon '%s' is not installed on cluster '%s'", addOnID, clusterKey)
		os.Exit(0)
	}

	if !confirm.Confirm("uninstall add-on '%s' from cluster '%s'", addOnID, clusterKey) {
		os.Exit(0)
	}

	r.Reporter.Debugf("Uninstalling add-on '%s' from cluster '%s'", addOnID, clusterKey)
	err = r.OCMClient.UninstallAddOn(cluster.ID(), addOnID)
	if err != nil {
		r.Reporter.Errorf("Failed to remove add-on installation '%s' from cluster '%s': %s", addOnID, clusterKey, err)
		os.Exit(1)
	}
	r.Reporter.Infof("Add-on '%s' is now uninstalling. To check the status run 'rosa list addons -c %s'",
		addOnID, clusterKey)
}
