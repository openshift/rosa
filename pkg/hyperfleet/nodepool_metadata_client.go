package hyperfleet

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/client-go/rest"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

// WithNodePoolMetadataUpdates preserves explicit empty labels and taints in PUT
// requests. The SDK omits empty collections, which the API treats as unchanged.
func WithNodePoolMetadataUpdates(client hyperfleetclientset.Interface,
	labelsChanged, taintsChanged bool) hyperfleetclientset.Interface {
	return &nodePoolMetadataClient{Interface: client, labelsChanged: labelsChanged, taintsChanged: taintsChanged}
}

type nodePoolMetadataClient struct {
	hyperfleetclientset.Interface
	labelsChanged, taintsChanged bool
}

func (c *nodePoolMetadataClient) HyperfleetV1alpha1() platform.V1alpha1PublicInterface {
	return &nodePoolMetadataPublicClient{
		V1alpha1PublicInterface: c.Interface.HyperfleetV1alpha1(),
		labelsChanged:           c.labelsChanged, taintsChanged: c.taintsChanged,
	}
}

type nodePoolMetadataPublicClient struct {
	platform.V1alpha1PublicInterface
	labelsChanged, taintsChanged bool
}

func (c *nodePoolMetadataPublicClient) NodePools(namespace string) platform.NodePoolInterface {
	return &nodePoolMetadataUpdateClient{
		NodePoolInterface: c.V1alpha1PublicInterface.NodePools(namespace),
		restClient:        c.RESTClient(), namespace: namespace,
		labelsChanged: c.labelsChanged, taintsChanged: c.taintsChanged,
	}
}

type nodePoolMetadataUpdateClient struct {
	platform.NodePoolInterface
	restClient                   rest.Interface
	namespace                    string
	labelsChanged, taintsChanged bool
}

type nodePoolMetadataUpdateRequest struct {
	*v1alpha1.NodePool
	Spec nodePoolMetadataUpdateSpec `json:"spec"`
}

type nodePoolMetadataUpdateSpec struct {
	v1alpha1.NodePoolSpec
	Labels   *map[string]string          `json:"labels,omitempty"`
	NodePool nodePoolMetadataPassthrough `json:"nodePool"`
}

type nodePoolMetadataPassthrough struct {
	v1alpha1.NodePoolSpecPassthrough
	Taints *[]hypershiftv1beta1.Taint `json:"taints,omitempty"`
}

func (c *nodePoolMetadataUpdateClient) Update(ctx context.Context, obj *v1alpha1.NodePool,
	_ platform.UpdateOptions) (*v1alpha1.NodePool, error) {
	if obj == nil || obj.UID == "" {
		return nil, fmt.Errorf("node pool UID is required")
	}
	request := nodePoolMetadataUpdateRequest{
		NodePool: obj,
		Spec: nodePoolMetadataUpdateSpec{
			NodePoolSpec: obj.Spec,
			NodePool:     nodePoolMetadataPassthrough{NodePoolSpecPassthrough: obj.Spec.NodePool},
		},
	}
	if c.labelsChanged || len(obj.Spec.Labels) > 0 {
		labels := obj.Spec.Labels
		if len(labels) == 0 {
			// null resets a Go map; {} would retain existing keys when decoded.
			labels = nil
		}
		request.Spec.Labels = &labels
	}
	if c.taintsChanged || len(obj.Spec.NodePool.Taints) > 0 {
		request.Spec.NodePool.Taints = &obj.Spec.NodePool.Taints
	}
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to encode node pool update: %w", err)
	}
	result := c.restClient.Put().Namespace(c.namespace).Resource("nodepools").
		Name(string(obj.UID)).Body(data).Do(ctx)
	if err := result.Error(); err != nil {
		return nil, err
	}
	response, err := result.Raw()
	if err != nil {
		return nil, err
	}
	updated := &v1alpha1.NodePool{}
	if err := json.Unmarshal(response, updated); err != nil {
		return nil, fmt.Errorf("failed to decode node pool update: %w", err)
	}
	return updated, nil
}
