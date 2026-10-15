package hyperfleet

import (
	"context"
	"fmt"
	"strings"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
)

// ListNodePoolsForCluster lists NodePools with the parent cluster UID selector.
// The Platform API is account-scoped, so the namespace is not passed in the URL.
func ListNodePoolsForCluster(
	ctx context.Context, client hyperfleetclientset.Interface, clusterUID string,
) (*v1alpha1.NodePoolList, error) {
	return client.HyperfleetV1alpha1().NodePools().List(ctx, platform.NodePoolListOptions{ClusterUID: clusterUID})
}

// ResolveNodePoolName looks up a node pool by the CLI child name or its full
// metadata.name and returns the account-scoped metadata.name used for API routes.
func ResolveNodePoolName(
	ctx context.Context, client hyperfleetclientset.Interface, clusterUID, nodePoolKey string,
) (string, error) {
	list, err := ListNodePoolsForCluster(ctx, client, clusterUID)
	if err != nil {
		return "", fmt.Errorf("failed to list node pools: %w", err)
	}
	for _, np := range list.Items {
		if nodePoolMatchesKey(np.Name, nodePoolKey) {
			return np.Name, nil
		}
	}
	return "", fmt.Errorf("machine pool '%s' does not exist", nodePoolKey)
}

// ResolveNodePoolUID looks up a node pool by human-readable name within the
// given cluster UID selector and returns its server-assigned UID.
func ResolveNodePoolUID(
	ctx context.Context, client hyperfleetclientset.Interface, clusterUID, nodePoolName string,
) (string, error) {
	list, err := ListNodePoolsForCluster(ctx, client, clusterUID)
	if err != nil {
		return "", fmt.Errorf("failed to list node pools: %w", err)
	}
	for _, np := range list.Items {
		if nodePoolMatchesKey(np.Name, nodePoolName) {
			return string(np.UID), nil
		}
	}
	return "", fmt.Errorf("machine pool '%s' does not exist", nodePoolName)
}

func nodePoolMatchesKey(fullName, key string) bool {
	if fullName == key {
		return true
	}
	_, childName, found := strings.Cut(fullName, ".")
	return found && childName == key
}

// NodePoolDisplayName removes the parent cluster prefix from a NodePool's
// account-scoped metadata.name for ROSA CLI output.
func NodePoolDisplayName(clusterName, metadataName string) string {
	return strings.TrimPrefix(metadataName, clusterName+".")
}

// FormatNodePoolDiskSize returns the disk size column for list/describe output.
func FormatNodePoolDiskSize(aws *v1alpha1.AWSNodePoolPlatform) string {
	if aws == nil || aws.RootVolume == nil || aws.RootVolume.Size <= 0 {
		return ""
	}
	return fmt.Sprintf("%d GiB", aws.RootVolume.Size)
}
