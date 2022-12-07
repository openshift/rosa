package machinepool

import (
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/rosa"
)

func deleteNodePool(r *rosa.Runtime, nodePoolID string, clusterKey string, cluster *cmv1.Cluster) {
	// Try to find the machine pool:
	r.Reporter.Debugf("Loading machine pools for hosted cluster '%s'", clusterKey)
	nodePool, err := r.OCMClient.GetNodePool(cluster.ID(), nodePoolID)
	if err != nil {
		r.Reporter.Errorf("Failed to get machine pools for hosted cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	if confirm.Confirm("delete machine pool '%s' on hosted cluster '%s'", nodePoolID, clusterKey) {
		r.Reporter.Debugf("Deleting machine pool '%s' on hosted cluster '%s'", nodePool.ID(), clusterKey)
		err = r.OCMClient.DeleteNodePool(cluster.ID(), nodePool.ID())
		if err != nil {
			r.Reporter.Errorf("Failed to delete machine pool '%s' on hosted cluster '%s': %s",
				nodePool.ID(), clusterKey, err)
			os.Exit(1)
		}
		r.Reporter.Infof("Successfully deleted machine pool '%s' from hosted cluster '%s'", nodePoolID, clusterKey)
	}

}
