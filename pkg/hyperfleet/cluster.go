package hyperfleet

import (
	"context"
	"fmt"

	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/wrappers"
)

// ResolveClusterUID looks up a cluster by human-readable name and returns its UID.
func ResolveClusterUID(
	ctx context.Context, client hyperfleetclientset.Interface, clusterName string,
) (string, error) {
	list, err := client.HyperfleetV1alpha1().Clusters().List(ctx, wrappers.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list clusters: %w", err)
	}
	for _, c := range list.Items {
		if c.Name == clusterName {
			return string(c.UID), nil
		}
	}
	return "", fmt.Errorf("cluster '%s' not found", clusterName)
}
