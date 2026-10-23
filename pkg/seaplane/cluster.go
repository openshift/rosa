// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package seaplane

import (
	"context"
	"fmt"

	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	seaplaneclientset "github.com/openshift-online/rosa-seaplane-api/clientset"
	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
)

// ResolveCluster looks up a cluster by metadata.name or UID.
func ResolveCluster(
	ctx context.Context, client seaplaneclientset.Interface, clusterKey string,
) (*v1alpha1.Cluster, error) {
	list, err := client.SeaplaneV1alpha1().Clusters().List(ctx, platform.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	for _, c := range list.Items {
		if c.Name == clusterKey || string(c.UID) == clusterKey {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("cluster '%s' not found", clusterKey)
}

// ResolveClusterUID looks up a cluster by name or UID and returns its UID.
func ResolveClusterUID(ctx context.Context, client seaplaneclientset.Interface, clusterKey string) (string, error) {
	cluster, err := ResolveCluster(ctx, client, clusterKey)
	if err != nil {
		return "", err
	}
	return string(cluster.UID), nil
}

// ResolveClusterName looks up a cluster by name or UID and returns its metadata.name
// for account-scoped Platform API routes.
func ResolveClusterName(ctx context.Context, client seaplaneclientset.Interface, clusterKey string) (string, error) {
	cluster, err := ResolveCluster(ctx, client, clusterKey)
	if err != nil {
		return "", err
	}
	return cluster.Name, nil
}

// GetCluster looks up a cluster by name or UID and returns the cluster object.
// Returns nil if the cluster is not found.
func GetCluster(
	ctx context.Context, client seaplaneclientset.Interface, clusterKey string,
) (*v1alpha1.Cluster, error) {
	list, err := client.SeaplaneV1alpha1().Clusters().List(ctx, platform.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	for _, c := range list.Items {
		if c.Name == clusterKey || string(c.UID) == clusterKey {
			return &c, nil
		}
	}
	return nil, nil
}

// HasClusterUsingOperatorRolesPrefix reports whether any Platform API cluster's
// RolesRef was created with the given operator-roles prefix.
func HasClusterUsingOperatorRolesPrefix(
	ctx context.Context, client seaplaneclientset.Interface, prefix string,
) (bool, error) {
	if prefix == "" {
		return false, nil
	}
	list, err := client.SeaplaneV1alpha1().Clusters().List(ctx, platform.ListOptions{})
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
	ctx context.Context, client seaplaneclientset.Interface, oidcConfigID string,
) (bool, error) {
	if oidcConfigID == "" {
		return false, nil
	}
	list, err := client.SeaplaneV1alpha1().Clusters().List(ctx, platform.ListOptions{})
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

// ClusterIDsByDNSBaseDomain maps each in-use DNS base domain to its cluster UID.
// The DNS-domain endpoint does not return cluster ownership, so list output uses
// this cluster-side association to show why a reservation cannot be deleted.
func ClusterIDsByDNSBaseDomain(
	ctx context.Context, client seaplaneclientset.Interface,
) (map[string]string, error) {
	list, err := client.SeaplaneV1alpha1().Clusters().List(ctx, platform.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	clusterIDs := make(map[string]string)
	for _, cluster := range list.Items {
		baseDomain := cluster.Spec.HostedCluster.DNS.BaseDomain
		if baseDomain == "" {
			continue
		}
		if _, found := clusterIDs[baseDomain]; !found {
			clusterIDs[baseDomain] = string(cluster.UID)
		}
	}
	return clusterIDs, nil
}
