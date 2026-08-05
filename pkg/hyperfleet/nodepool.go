package hyperfleet

import (
	"context"
	"fmt"

	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/wrappers"
)

// ResolveNodePoolUID looks up a node pool by human-readable name within the
// given cluster namespace (cluster UID) and returns the node pool's UID.
func ResolveNodePoolUID(
	ctx context.Context, client hyperfleetclientset.Interface, clusterUID, nodePoolName string,
) (string, error) {
	list, err := client.HyperfleetV1alpha1().NodePools(clusterUID).List(ctx, wrappers.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list node pools: %w", err)
	}
	for _, np := range list.Items {
		if np.Name == nodePoolName {
			return string(np.UID), nil
		}
	}
	return "", fmt.Errorf("node pool '%s' not found", nodePoolName)
}
