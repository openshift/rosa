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

// NodePoolUpdateOverrides carries the PUT fields the pinned SDK cannot encode:
// explicit empty labels and taints, which the SDK omits and the API then treats
// as unchanged, and the rolling-update settings that live behind the
// service-set spec.nodePool.management.
type NodePoolUpdateOverrides struct {
	LabelsChanged bool
	TaintsChanged bool
	Upgrade       *NodePoolUpgrade
}

// Needed reports whether any override requires the hand-encoded PUT body.
func (o NodePoolUpdateOverrides) Needed() bool {
	return o.LabelsChanged || o.TaintsChanged || o.Upgrade != nil
}

// WithNodePoolUpdates encodes the node pool PUT body by hand so the overrides
// above survive the request, retaining the generated workflow and delegating
// all other operations.
func WithNodePoolUpdates(client hyperfleetclientset.Interface,
	overrides NodePoolUpdateOverrides) hyperfleetclientset.Interface {
	return &nodePoolUpdateClient{Interface: client, overrides: overrides}
}

type nodePoolUpdateClient struct {
	hyperfleetclientset.Interface
	overrides NodePoolUpdateOverrides
}

func (c *nodePoolUpdateClient) HyperfleetV1alpha1() platform.V1alpha1PublicInterface {
	return &nodePoolUpdatePublicClient{
		V1alpha1PublicInterface: c.Interface.HyperfleetV1alpha1(), overrides: c.overrides,
	}
}

type nodePoolUpdatePublicClient struct {
	platform.V1alpha1PublicInterface
	overrides NodePoolUpdateOverrides
}

func (c *nodePoolUpdatePublicClient) NodePools(namespace string) platform.NodePoolInterface {
	return &nodePoolUpdateRequestClient{
		NodePoolInterface: c.V1alpha1PublicInterface.NodePools(namespace),
		restClient:        c.RESTClient(), namespace: namespace, overrides: c.overrides,
	}
}

type nodePoolUpdateRequestClient struct {
	platform.NodePoolInterface
	restClient rest.Interface
	namespace  string
	overrides  NodePoolUpdateOverrides
}

type nodePoolUpdateRequest struct {
	*v1alpha1.NodePool
	Spec nodePoolUpdateSpec `json:"spec"`
}

type nodePoolUpdateSpec struct {
	v1alpha1.NodePoolSpec
	Labels         *map[string]string        `json:"labels,omitempty"`
	NodePool       nodePoolUpdatePassthrough `json:"nodePool"`
	MaxSurge       string                    `json:"maxSurge,omitempty"`
	MaxUnavailable string                    `json:"maxUnavailable,omitempty"`
}

type nodePoolUpdatePassthrough struct {
	v1alpha1.NodePoolSpecPassthrough
	Taints *[]hypershiftv1beta1.Taint `json:"taints,omitempty"`
}

func (c *nodePoolUpdateRequestClient) Update(ctx context.Context, obj *v1alpha1.NodePool,
	_ platform.UpdateOptions) (*v1alpha1.NodePool, error) {
	if obj == nil || obj.UID == "" {
		return nil, fmt.Errorf("node pool UID is required")
	}
	request := nodePoolUpdateRequest{
		NodePool: obj,
		Spec: nodePoolUpdateSpec{
			NodePoolSpec: obj.Spec,
			NodePool:     nodePoolUpdatePassthrough{NodePoolSpecPassthrough: obj.Spec.NodePool},
		},
	}
	if c.overrides.LabelsChanged || len(obj.Spec.Labels) > 0 {
		labels := obj.Spec.Labels
		if len(labels) == 0 {
			// null resets a Go map; {} would retain existing keys when decoded.
			labels = nil
		}
		request.Spec.Labels = &labels
	}
	if c.overrides.TaintsChanged || len(obj.Spec.NodePool.Taints) > 0 {
		request.Spec.NodePool.Taints = &obj.Spec.NodePool.Taints
	}
	if c.overrides.Upgrade != nil {
		request.Spec.MaxSurge = c.overrides.Upgrade.MaxSurge
		request.Spec.MaxUnavailable = c.overrides.Upgrade.MaxUnavailable
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
