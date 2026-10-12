package hyperfleet

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/client-go/rest"

	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hyperfleetclientset "github.com/openshift-online/rosa-hyperfleet-api/clientset"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
)

// WithNodePoolUpgrade extends node pool create requests with the rolling-update
// fields the pinned SDK cannot encode, through the SDK's authenticated REST
// client, retaining the generated workflow and delegating all other operations.
func WithNodePoolUpgrade(client hyperfleetclientset.Interface,
	upgrade *NodePoolUpgrade) hyperfleetclientset.Interface {
	return &nodePoolUpgradeClient{Interface: client, upgrade: upgrade}
}

type nodePoolUpgradeClient struct {
	hyperfleetclientset.Interface
	upgrade *NodePoolUpgrade
}

func (c *nodePoolUpgradeClient) HyperfleetV1alpha1() platform.V1alpha1PublicInterface {
	return &nodePoolUpgradePublicClient{
		V1alpha1PublicInterface: c.Interface.HyperfleetV1alpha1(), upgrade: c.upgrade,
	}
}

type nodePoolUpgradePublicClient struct {
	platform.V1alpha1PublicInterface
	upgrade *NodePoolUpgrade
}

func (c *nodePoolUpgradePublicClient) NodePools(namespace string) platform.NodePoolInterface {
	return &nodePoolUpgradeCreateClient{
		NodePoolInterface: c.V1alpha1PublicInterface.NodePools(namespace),
		restClient:        c.RESTClient(), namespace: namespace, upgrade: c.upgrade,
	}
}

type nodePoolUpgradeCreateClient struct {
	platform.NodePoolInterface
	restClient rest.Interface
	namespace  string
	upgrade    *NodePoolUpgrade
}

// The embedded SDK fields retain the normal create body; the explicit fields
// extend only the rolling-update settings the SDK cannot encode.
type nodePoolCreateRequest struct {
	*v1alpha1.NodePool
	Spec nodePoolCreateSpec `json:"spec"`
}

type nodePoolCreateSpec struct {
	v1alpha1.NodePoolSpec
	MaxSurge       string `json:"maxSurge,omitempty"`
	MaxUnavailable string `json:"maxUnavailable,omitempty"`
}

func (c *nodePoolUpgradeCreateClient) Create(ctx context.Context, obj *v1alpha1.NodePool,
	_ platform.CreateOptions) (*v1alpha1.NodePool, error) {
	if obj == nil {
		return nil, fmt.Errorf("node pool is required")
	}
	request := nodePoolCreateRequest{
		NodePool: obj,
		Spec:     nodePoolCreateSpec{NodePoolSpec: obj.Spec},
	}
	if c.upgrade != nil {
		request.Spec.MaxSurge = c.upgrade.MaxSurge
		request.Spec.MaxUnavailable = c.upgrade.MaxUnavailable
	}
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to encode node pool create: %w", err)
	}
	result := c.restClient.Post().Namespace(c.namespace).Resource("nodepools").Body(data).Do(ctx)
	if err := result.Error(); err != nil {
		return nil, err
	}
	response, err := result.Raw()
	if err != nil {
		return nil, err
	}
	created := &v1alpha1.NodePool{}
	if err := json.Unmarshal(response, created); err != nil {
		return nil, fmt.Errorf("failed to decode node pool create: %w", err)
	}
	return created, nil
}
