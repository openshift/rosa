// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package seaplane

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	seaplaneclientset "github.com/openshift-online/rosa-seaplane-api/clientset"
	"github.com/openshift-online/rosa-seaplane-api/clientset/platform"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

// WithNodePoolMetadataUpdates preserves explicit empty labels and taints in PUT
// requests, and echoes existing platform-managed fields in PATCH requests for
// API versions that validate omitted fields as zero values.
func WithNodePoolMetadataUpdates(client seaplaneclientset.Interface,
	labelsChanged, taintsChanged bool) seaplaneclientset.Interface {
	return WithNodePoolPatchUpdates(client, labelsChanged, taintsChanged, false)
}

// WithNodePoolPatchUpdates preserves explicit empty labels and taints in PUT
// requests, echoes existing platform-managed fields in PATCH requests, and can
// translate the on-demand Spot max-price sentinel into an explicit JSON reset.
func WithNodePoolPatchUpdates(client seaplaneclientset.Interface,
	labelsChanged, taintsChanged, spotMaxPriceReset bool) seaplaneclientset.Interface {
	return &nodePoolMetadataClient{
		Interface:         client,
		labelsChanged:     labelsChanged,
		taintsChanged:     taintsChanged,
		spotMaxPriceReset: spotMaxPriceReset,
	}
}

type nodePoolMetadataClient struct {
	seaplaneclientset.Interface
	labelsChanged, taintsChanged bool
	spotMaxPriceReset            bool
}

func (c *nodePoolMetadataClient) SeaplaneV1alpha1() platform.V1alpha1PublicInterface {
	return &nodePoolMetadataPublicClient{
		V1alpha1PublicInterface: c.Interface.SeaplaneV1alpha1(),
		labelsChanged:           c.labelsChanged,
		taintsChanged:           c.taintsChanged,
		spotMaxPriceReset:       c.spotMaxPriceReset,
	}
}

type nodePoolMetadataPublicClient struct {
	platform.V1alpha1PublicInterface
	labelsChanged, taintsChanged bool
	spotMaxPriceReset            bool
}

func (c *nodePoolMetadataPublicClient) NodePools() platform.NodePoolInterface {
	return &nodePoolMetadataUpdateClient{
		NodePoolInterface: c.V1alpha1PublicInterface.NodePools(),
		restClient:        c.RESTClient(),
		labelsChanged:     c.labelsChanged,
		taintsChanged:     c.taintsChanged,
		spotMaxPriceReset: c.spotMaxPriceReset,
	}
}

type nodePoolMetadataUpdateClient struct {
	platform.NodePoolInterface
	restClient                   rest.Interface
	labelsChanged, taintsChanged bool
	spotMaxPriceReset            bool
}

// Patch keeps the Platform API's non-standard field-level errors in validation responses.
func (c *nodePoolMetadataUpdateClient) Patch(ctx context.Context, name string, patchType types.PatchType,
	data []byte, opts platform.PatchOptions) (*v1alpha1.NodePool, error) {
	if c.spotMaxPriceReset {
		var err error
		data, err = resetNodePoolSpotMaxPrice(data)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare node pool patch: %w", err)
		}
	}
	if c.restClient == nil {
		return c.NodePoolInterface.Patch(ctx, name, patchType, data, opts)
	}
	current, err := c.Get(ctx, name, platform.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node pool %q before patch: %w", name, err)
	}
	if current == nil {
		return nil, fmt.Errorf("node pool %q was not returned before patch", name)
	}
	data, err = addNodePoolNoOpFields(data, current)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare node pool patch: %w", err)
	}

	result := c.restClient.Patch(patchType).Resource("nodepools").Name(name).Body(data).Do(ctx)
	response, rawErr := result.Raw()
	if err := result.Error(); err != nil {
		return nil, WithAPIErrorBodyDetails(err, response)
	}
	if rawErr != nil {
		return nil, rawErr
	}

	updated := &v1alpha1.NodePool{}
	if err := json.Unmarshal(response, updated); err != nil {
		return nil, fmt.Errorf("failed to decode node pool patch: %w", err)
	}
	return updated, nil
}

// TODO: Remove this workaround after the Platform API raw-spec validation fix is deployed.
// addNodePoolNoOpFields works around Platform API versions whose typed PATCH
// validation sees omitted value-type fields as zero values. Including the
// current values keeps immutable and service-set fields unchanged.
func addNodePoolNoOpFields(data []byte, current *v1alpha1.NodePool) ([]byte, error) {
	var request map[string]any
	if err := json.Unmarshal(data, &request); err != nil {
		return nil, fmt.Errorf("decode patch document: %w", err)
	}
	spec, ok := request["spec"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("patch document must contain a spec object")
	}
	nodePool, ok := spec["nodePool"].(map[string]any)
	if !ok {
		if _, exists := spec["nodePool"]; exists {
			return nil, fmt.Errorf("patch spec.nodePool must be an object")
		}
		nodePool = make(map[string]any)
	}
	platform, ok := nodePool["platform"].(map[string]any)
	if !ok {
		if _, exists := nodePool["platform"]; exists {
			return nil, fmt.Errorf("patch spec.nodePool.platform must be an object")
		}
		platform = make(map[string]any)
	}
	platform["type"] = current.Spec.NodePool.Platform.Type
	nodePool["platform"] = platform
	nodePool["management"] = current.Spec.NodePool.Management
	spec["nodePool"] = nodePool
	return json.Marshal(request)
}

// resetNodePoolSpotMaxPrice converts the non-empty on-demand sentinel used by
// generated update overlays into the explicit empty string required to clear it.
func resetNodePoolSpotMaxPrice(data []byte) ([]byte, error) {
	var request map[string]any
	if err := json.Unmarshal(data, &request); err != nil {
		return nil, fmt.Errorf("decode patch document: %w", err)
	}
	spec, ok := request["spec"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("patch document must contain a spec object")
	}
	nodePool, ok := spec["nodePool"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("patch spec.nodePool must be an object")
	}
	nodePlatform, ok := nodePool["platform"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("patch spec.nodePool.platform must be an object")
	}
	aws, ok := nodePlatform["aws"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("patch spec.nodePool.platform.aws must be an object")
	}
	placement, ok := aws["placement"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("patch spec.nodePool.platform.aws.placement must be an object")
	}
	spot, ok := placement["spot"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("patch spec.nodePool.platform.aws.placement.spot must be an object")
	}
	spot["maxPrice"] = ""
	return json.Marshal(request)
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
	if obj == nil || obj.Name == "" {
		return nil, fmt.Errorf("node pool name is required")
	}
	if obj.UID == "" {
		return nil, fmt.Errorf("node pool UID is required")
	}
	if c.restClient == nil {
		return c.NodePoolInterface.Update(ctx, obj, platform.UpdateOptions{})
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
	result := c.restClient.Put().Resource("nodepools").Name(obj.Name).Body(data).Do(ctx)
	response, rawErr := result.Raw()
	if err := result.Error(); err != nil {
		return nil, WithAPIErrorBodyDetails(err, response)
	}
	if rawErr != nil {
		return nil, rawErr
	}
	updated := &v1alpha1.NodePool{}
	if err := json.Unmarshal(response, updated); err != nil {
		return nil, fmt.Errorf("failed to decode node pool update: %w", err)
	}
	return updated, nil
}
