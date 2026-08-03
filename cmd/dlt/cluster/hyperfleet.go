package cluster

import (
	"context"
	"os"

	"github.com/openshift-online/rosa-hyperfleet-api/clientset/wrappers"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var (
	hfEnabled       = hyperfleet.Enabled
	hfDeleteCluster = func() {
		r := rosa.NewRuntime().WithHyperFleet()
		defer r.Cleanup()
		runHyperfleetDelete(r)
	}
)

// runHyperfleetDelete deletes an HCP cluster via the Platform API v2.
// It resolves the human-readable cluster name to its server-assigned UID
// (the Platform API routes mutations by UID, not by name).
func runHyperfleetDelete(r *rosa.Runtime) {
	ctx := context.Background()

	clusterKey, err := ocm.GetClusterKey()
	if err != nil || clusterKey == "" {
		r.Reporter.Errorf("--cluster is required")
		os.Exit(1)
	}

	clusters := r.HyperFleetClient.HyperfleetV1alpha1().Clusters(r.Creator.AccountID)

	list, err := clusters.List(ctx, wrappers.ListOptions{})
	if err != nil {
		r.Reporter.Errorf("Failed to list clusters: %v", err)
		os.Exit(1)
	}

	var clusterID string
	for _, c := range list.Items {
		if c.Name == clusterKey {
			clusterID = string(c.UID)
			break
		}
	}
	if clusterID == "" {
		r.Reporter.Errorf("Cluster '%s' not found", clusterKey)
		os.Exit(1)
	}

	if err = clusters.Delete(ctx, clusterID, wrappers.DeleteOptions{}); err != nil {
		r.Reporter.Errorf("Failed to delete cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	r.Reporter.Infof("Cluster '%s' will start deleting now", clusterKey)
}
