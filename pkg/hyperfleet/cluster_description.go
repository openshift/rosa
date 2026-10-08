package hyperfleet

import (
	"context"
	"encoding/json"
	"fmt"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
)

// ClusterDescription preserves response fields not yet present in the pinned SDK.
type ClusterDescription struct {
	Cluster               *v1alpha1.Cluster
	Proxy                 *ClusterProxy
	AdditionalTrustBundle string
}

// ClusterProxy models proxy fields absent from the pinned SDK.
type ClusterProxy struct {
	HTTPProxy  string `json:"httpProxy,omitempty"`
	HTTPSProxy string `json:"httpsProxy,omitempty"`
	NoProxy    string `json:"noProxy,omitempty"`
}

// GetClusterDescription reads the typed cluster and its trust-bundle projection
// from the same response. The pinned SDK predates spec.additionalTrustBundle.
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
	var response struct {
		Proxy *struct {
			HTTPProxy  string `json:"http_proxy"`
			HTTPSProxy string `json:"https_proxy"`
			NoProxy    string `json:"no_proxy"`
		} `json:"proxy"`
		Spec struct {
			AdditionalTrustBundle *string `json:"additionalTrustBundle"`
			HostedCluster         struct {
				Configuration struct {
					Proxy *ClusterProxy `json:"proxy"`
				} `json:"configuration"`
			} `json:"hostedCluster"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to decode cluster proxy or trust bundle: %w", err)
	}
	description := &ClusterDescription{Cluster: cluster, Proxy: response.Spec.HostedCluster.Configuration.Proxy}
	if response.Proxy != nil {
		description.Proxy = &ClusterProxy{
			HTTPProxy: response.Proxy.HTTPProxy, HTTPSProxy: response.Proxy.HTTPSProxy, NoProxy: response.Proxy.NoProxy,
		}
	}
	if response.Spec.AdditionalTrustBundle != nil && *response.Spec.AdditionalTrustBundle != "" {
		// Never display PEM contents, even if an API response is not redacted.
		description.AdditionalTrustBundle = "REDACTED"
	}
	return description, nil
}
