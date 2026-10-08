package hyperfleet

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"k8s.io/client-go/rest"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
)

// WithClusterProxy extends create requests through the SDK's authenticated REST
// client, retaining the generated workflow and delegating all other operations.
func WithClusterProxy(client hyperfleetclientset.Interface, proxy *ClusterProxy,
	trustBundleFile string) hyperfleetclientset.Interface {
	return &clusterProxyClient{Interface: client, proxy: proxy, trustBundleFile: trustBundleFile}
}

type clusterProxyClient struct {
	hyperfleetclientset.Interface
	proxy           *ClusterProxy
	trustBundleFile string
}

func (c *clusterProxyClient) HyperfleetV1alpha1() platform.V1alpha1PublicInterface {
	return &clusterProxyPublicClient{
		V1alpha1PublicInterface: c.Interface.HyperfleetV1alpha1(),
		proxy:                   c.proxy, trustBundleFile: c.trustBundleFile,
	}
}

type clusterProxyPublicClient struct {
	platform.V1alpha1PublicInterface
	proxy           *ClusterProxy
	trustBundleFile string
}

func (c *clusterProxyPublicClient) Clusters() platform.ClusterInterface {
	return &clusterProxyCreateClient{
		ClusterInterface: c.V1alpha1PublicInterface.Clusters(), restClient: c.RESTClient(),
		proxy: c.proxy, trustBundleFile: c.trustBundleFile,
	}
}

type clusterProxyCreateClient struct {
	platform.ClusterInterface
	restClient      rest.Interface
	proxy           *ClusterProxy
	trustBundleFile string
}

// The embedded SDK fields retain the normal create body; explicit fields extend
// only the portions whose proxy and trust-bundle properties the SDK cannot encode.
type clusterCreateRequest struct {
	*v1alpha1.Cluster
	Spec clusterCreateSpec `json:"spec"`
}

type clusterCreateSpec struct {
	v1alpha1.ClusterSpec
	HostedCluster         clusterCreateHostedSpec `json:"hostedCluster"`
	AdditionalTrustBundle *string                 `json:"additionalTrustBundle,omitempty"`
}

type clusterCreateHostedSpec struct {
	v1alpha1.HostedClusterSpecPassthrough
	Configuration *clusterCreateConfiguration `json:"configuration,omitempty"`
}

type clusterCreateConfiguration struct {
	*v1alpha1.ClusterConfiguration
	Proxy *ClusterProxy `json:"proxy,omitempty"`
}

func (c *clusterProxyCreateClient) Create(ctx context.Context, obj *v1alpha1.Cluster,
	_ platform.CreateOptions) (*v1alpha1.Cluster, error) {
	if obj == nil {
		return nil, fmt.Errorf("cluster is required")
	}
	request := clusterCreateRequest{
		Cluster: obj,
		Spec: clusterCreateSpec{
			ClusterSpec: obj.Spec,
			HostedCluster: clusterCreateHostedSpec{
				HostedClusterSpecPassthrough: obj.Spec.HostedCluster,
			},
		},
	}
	if configuration := obj.Spec.HostedCluster.Configuration; configuration != nil || c.proxy != nil {
		request.Spec.HostedCluster.Configuration = &clusterCreateConfiguration{
			ClusterConfiguration: configuration, Proxy: c.proxy,
		}
	}
	if c.trustBundleFile != "" {
		bundle, err := os.ReadFile(c.trustBundleFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read additional trust bundle file: %w", err)
		}
		contents := string(bundle)
		request.Spec.AdditionalTrustBundle = &contents
	}
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to encode cluster proxy request: %w", err)
	}
	result := c.restClient.Post().Resource("clusters").Body(data).Do(ctx)
	if err := result.Error(); err != nil {
		return nil, err
	}
	response, err := result.Raw()
	if err != nil {
		return nil, err
	}
	description, err := DecodeClusterDescription(response)
	if err != nil {
		return nil, err
	}
	return description.Cluster, nil
}
