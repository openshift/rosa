package input

import (
	"os"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

// CheckIfHypershiftClusterOrExit will exit if the input cluster is not an Hypershift cluster
func CheckIfHypershiftClusterOrExit(r *rosa.Runtime, cluster *cmv1.Cluster) {
	if !ocm.IsHyperShiftCluster(cluster) {
		r.Reporter.Errorf("This command is only supported for Hosted Control Planes")
		os.Exit(1)
	}
}
