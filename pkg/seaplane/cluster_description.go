// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package seaplane

import (
	"context"
	"encoding/json"
	"fmt"

	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	seaplaneclientset "github.com/openshift-online/rosa-seaplane-api/clientset"
	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
)

// ClusterDescription provides the typed cluster with normalized proxy settings
// and a redacted view of the additional trust bundle for CLI output.
type ClusterDescription struct {
	Cluster               *v1alpha1.Cluster
	Proxy                 *ClusterProxy
	AdditionalTrustBundle string
}

// ClusterProxy is the CLI-facing proxy projection from the Platform API.
type ClusterProxy struct {
	HTTPProxy  string `json:"httpProxy,omitempty"`
	HTTPSProxy string `json:"httpsProxy,omitempty"`
	NoProxy    string `json:"noProxy,omitempty"`
}

// GetClusterDescription reads the typed cluster and its read-only projections
// from the account-scoped metadata.name route.
func GetClusterDescription(ctx context.Context, client seaplaneclientset.Interface,
	clusterName string) (*ClusterDescription, error) {
	cluster, err := client.SeaplaneV1alpha1().Clusters().Get(ctx, clusterName, platform.GetOptions{})
	if err != nil {
		return nil, err
	}
	return clusterDescriptionFromCluster(cluster), nil
}

func clusterDescriptionFromCluster(cluster *v1alpha1.Cluster) *ClusterDescription {
	description := &ClusterDescription{Cluster: cluster}
	if cluster.Proxy != nil {
		description.Proxy = &ClusterProxy{
			HTTPProxy:  cluster.Proxy.HTTPProxy,
			HTTPSProxy: cluster.Proxy.HTTPSProxy,
			NoProxy:    cluster.Proxy.NoProxy,
		}
	} else if configuration := cluster.Spec.HostedCluster.Configuration; configuration != nil &&
		configuration.Proxy != nil {
		description.Proxy = &ClusterProxy{
			HTTPProxy:  configuration.Proxy.HTTPProxy,
			HTTPSProxy: configuration.Proxy.HTTPSProxy,
			NoProxy:    configuration.Proxy.NoProxy,
		}
	}
	if bundle := cluster.Spec.AdditionalTrustBundle; bundle != nil && *bundle != "" {
		// Never display PEM contents, even if an API response is not redacted.
		description.AdditionalTrustBundle = "REDACTED"
	}
	return description
}

// DecodeClusterDescription preserves proxy settings and redacts trust bundles.
func DecodeClusterDescription(data []byte) (*ClusterDescription, error) {
	cluster := &v1alpha1.Cluster{}
	if err := json.Unmarshal(data, cluster); err != nil {
		return nil, fmt.Errorf("failed to decode cluster response: %w", err)
	}
	return clusterDescriptionFromCluster(cluster), nil
}
