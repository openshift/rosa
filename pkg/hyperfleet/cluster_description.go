package hyperfleet

import (
	"context"
	"encoding/json"
	"fmt"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
)

// ClusterDescription exposes normalized proxy settings and redacts trust-bundle contents.
type ClusterDescription struct {
	Cluster               *v1alpha1.Cluster
	Proxy                 *ClusterProxy
	AdditionalTrustBundle string
}

// ClusterProxy normalizes proxy fields from the Platform API response projections.
type ClusterProxy struct {
	HTTPProxy  string `json:"httpProxy,omitempty"`
	HTTPSProxy string `json:"httpsProxy,omitempty"`
	NoProxy    string `json:"noProxy,omitempty"`
}

// GetClusterDescription reads the cluster and its proxy and trust-bundle fields
// from the same Platform API response.
func GetClusterDescription(ctx context.Context, client hyperfleetclientset.Interface,
	clusterUID string) (*ClusterDescription, error) {
	result := client.HyperfleetV1alpha1().RESTClient().Get().Resource("clusters").Name(clusterUID).Do(ctx)
	if err := result.Error(); err != nil {
		return nil, err
	}
	data, err := result.Raw()
	if err != nil {
		return nil, err
	}
	return DecodeClusterDescription(data)
}

// DecodeClusterDescription preserves proxy settings and redacts trust bundles.
func DecodeClusterDescription(data []byte) (*ClusterDescription, error) {
	cluster := &v1alpha1.Cluster{}
	if err := json.Unmarshal(data, cluster); err != nil {
		return nil, fmt.Errorf("failed to decode cluster response: %w", err)
	}
	description := &ClusterDescription{Cluster: cluster}
	if configuration := cluster.Spec.HostedCluster.Configuration; configuration != nil && configuration.Proxy != nil {
		description.Proxy = &ClusterProxy{
			HTTPProxy:  configuration.Proxy.HTTPProxy,
			HTTPSProxy: configuration.Proxy.HTTPSProxy,
			NoProxy:    configuration.Proxy.NoProxy,
		}
	}
	if cluster.Proxy != nil {
		description.Proxy = &ClusterProxy{
			HTTPProxy:  cluster.Proxy.HTTPProxy,
			HTTPSProxy: cluster.Proxy.HTTPSProxy,
			NoProxy:    cluster.Proxy.NoProxy,
		}
	}
	if cluster.Spec.AdditionalTrustBundle != nil && *cluster.Spec.AdditionalTrustBundle != "" {
		// Never display PEM contents, even if an API response is not redacted.
		description.AdditionalTrustBundle = "REDACTED"
	}
	return description, nil
}
