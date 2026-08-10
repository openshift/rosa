package cluster

import (
	"os"

	"github.com/openshift-online/rosa-hyperfleet-api/clientset/wrappers"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hfEnabled       = hyperfleet.Enabled
	exitFn          = func(code int) { os.Exit(code) }
	confirmFn       = confirm.Confirm
	hfDeleteCluster = func(cmd *cobra.Command) {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetDelete(r, cmd)
	}
)

// runHyperfleetDelete deletes an HCP cluster via the Platform API v2.
// It resolves the human-readable cluster name to its server-assigned UID
// (the Platform API routes mutations by UID, not by name).
func runHyperfleetDelete(r *rosa.Runtime, cmd *cobra.Command) {
	ctx := cmd.Context()

	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		exitFn(1)
	}

	clusterID, err := hyperfleet.ResolveClusterUID(ctx, r.HyperFleetClient, clusterKey)
	if err != nil {
		r.Reporter.Errorf("%v", err)
		exitFn(1)
	}

	if !confirmFn("delete cluster %s", clusterKey) {
		return
	}

	clusters := r.HyperFleetClient.HyperfleetV1alpha1().Clusters()
	if err = clusters.Delete(ctx, clusterID, wrappers.DeleteOptions{}); err != nil {
		r.Reporter.Errorf("Failed to delete cluster '%s': %v", clusterKey, err)
		exitFn(1)
	}

	r.Reporter.Infof("Cluster '%s' will start deleting now", clusterKey)
}
