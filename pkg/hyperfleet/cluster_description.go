package hyperfleet

import (
	"context"
	"encoding/json"
	"fmt"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
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
func GetClusterDescription(ctx context.Context, client hyperfleetclientset.Interface,
	clusterName string) (*ClusterDescription, error) {
	cluster, err := client.HyperfleetV1alpha1().Clusters().Get(ctx, clusterName, platform.GetOptions{})
	if err != nil {
		return nil, err
	}
	description := &ClusterDescription{Cluster: cluster}
	if cluster.Proxy != nil {
		description.Proxy = &ClusterProxy{
			HTTPProxy: cluster.Proxy.HTTPProxy, HTTPSProxy: cluster.Proxy.HTTPSProxy, NoProxy: cluster.Proxy.NoProxy,
		}
	} else {
		configuration := cluster.Spec.HostedCluster.Configuration
		if configuration != nil && configuration.Proxy != nil {
			description.Proxy = &ClusterProxy{
				HTTPProxy:  configuration.Proxy.HTTPProxy,
				HTTPSProxy: configuration.Proxy.HTTPSProxy,
				NoProxy:    configuration.Proxy.NoProxy,
			}
		}
	}
	if bundle := cluster.Spec.AdditionalTrustBundle; bundle != nil && *bundle != "" {
		// Never display PEM contents, even if an API response is not redacted.
		description.AdditionalTrustBundle = "REDACTED"
	}
	return description, nil
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
