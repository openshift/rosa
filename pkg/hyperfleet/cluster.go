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

// HasClusterUsingOperatorRolesPrefix reports whether any Platform API cluster's
// RolesRef was created with the given operator-roles prefix.
func HasClusterUsingOperatorRolesPrefix(
	ctx context.Context, client hyperfleetclientset.Interface, prefix string,
) (bool, error) {
	if prefix == "" {
		return false, nil
	}
	list, err := client.HyperfleetV1alpha1().Clusters().List(ctx, platform.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to list clusters: %w", err)
	}
	for _, c := range list.Items {
		aws := c.Spec.HostedCluster.Platform.AWS
		if aws == nil {
			continue
		}
		if OperatorRolesPrefixFromRolesRef(aws.RolesRef) == prefix {
			return true, nil
		}
	}
	return false, nil
}

// HasClusterUsingOidcConfigID reports whether any Platform API cluster references
// the given OIDC config ID.
func HasClusterUsingOidcConfigID(
	ctx context.Context, client hyperfleetclientset.Interface, oidcConfigID string,
) (bool, error) {
	if oidcConfigID == "" {
		return false, nil
	}
	list, err := client.HyperfleetV1alpha1().Clusters().List(ctx, platform.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to list clusters: %w", err)
	}
	for _, c := range list.Items {
		if c.Spec.OidcConfigID == oidcConfigID {
			return true, nil
		}
	}
	return false, nil
}
