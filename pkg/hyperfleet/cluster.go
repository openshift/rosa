package hyperfleet

import (
	"context"
	"fmt"

	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
)

// ResolveClusterUID looks up a cluster by name or UID and returns its UID.
func ResolveClusterUID(
	ctx context.Context, client hyperfleetclientset.Interface, clusterKey string,
) (string, error) {
	list, err := client.HyperfleetV1alpha1().Clusters().List(ctx, platform.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list clusters: %w", err)
	}
	for _, c := range list.Items {
		if c.Name == clusterKey || string(c.UID) == clusterKey {
			return string(c.UID), nil
		}
	}
	return "", fmt.Errorf("cluster '%s' not found", clusterKey)
}
