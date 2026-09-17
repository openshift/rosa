package hyperfleet

import (
	"context"
	"fmt"

	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

// ResolveNodePoolUID looks up a node pool by human-readable name within the
// given cluster namespace (cluster UID) and returns the node pool's UID.
func ResolveNodePoolUID(
	ctx context.Context, client hyperfleetclientset.Interface, clusterUID, nodePoolName string,
) (string, error) {
	list, err := client.HyperfleetV1alpha1().NodePools(clusterUID).List(ctx, platform.ListOptions{})
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

// FormatNodePoolDiskSize returns the disk size column for list/describe output.
func FormatNodePoolDiskSize(aws *hypershiftv1beta1.AWSNodePoolPlatform) string {
	if aws == nil || aws.RootVolume == nil || aws.RootVolume.Size <= 0 {
		return ""
	}
	return fmt.Sprintf("%d GiB", aws.RootVolume.Size)
}
