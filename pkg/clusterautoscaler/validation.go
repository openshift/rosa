package clusterautoscaler

import (
	"fmt"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/rosa"
)

const (
	NoHCPAutoscalerSupportMessage = "Hosted Control Plane clusters do not support cluster-autoscaler configuration"
	ClusterNotReadyMessage        = "Cluster '%s' is not yet ready. Current state is '%s'"
)

func IsAutoscalerSupported(runtime *rosa.Runtime, cluster *cmv1.Cluster) error {

	if cluster.State() != cmv1.ClusterStateReady {
		return fmt.Errorf(ClusterNotReadyMessage, runtime.ClusterKey, cluster.State())
	}

	return nil
}
