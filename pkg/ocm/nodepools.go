package ocm

import (
	"net/http"
	"slices"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// CreateNodePoolResult bundles the created node pool with any server-side warnings.
type CreateNodePoolResult struct {
	NodePool *cmv1.NodePool
	Warnings []string
}

func (c *Client) CreateNodePool(clusterID string, nodePool *cmv1.NodePool) (*cmv1.NodePool, error) {
	result, err := c.CreateNodePoolWithWarnings(clusterID, nodePool)
	if err != nil {
		return nil, err
	}
	return result.NodePool, nil
}

// CreateNodePoolWithWarnings creates a node pool and returns any Warning headers from the response.
func (c *Client) CreateNodePoolWithWarnings(clusterID string, nodePool *cmv1.NodePool) (*CreateNodePoolResult, error) {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		NodePools().
		Add().Body(nodePool).
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return &CreateNodePoolResult{
		NodePool: response.Body(),
		Warnings: extractWarningHeaders(response.Header()),
	}, nil
}

func extractWarningHeaders(headers http.Header) []string {
	if headers == nil {
		return nil
	}

	rawWarnings := headers.Values("Warning")
	warnings := make([]string, 0, len(rawWarnings))
	for _, raw := range rawWarnings {
		warning := strings.TrimSpace(raw)
		if warning == "" {
			continue
		}

		firstQuote := strings.Index(warning, `"`)
		lastQuote := strings.LastIndex(warning, `"`)
		if firstQuote >= 0 && lastQuote > firstQuote {
			warning = warning[firstQuote+1 : lastQuote]
		} else if parts := strings.SplitN(warning, " - ", 2); len(parts) == 2 {
			warning = strings.TrimSpace(parts[1])
		}

		if warning != "" {
			warnings = append(warnings, warning)
		}
	}

	return warnings
}

func (c *Client) FindNodePoolsUsingKubeletConfig(
	clusterId string,
	kubeletName string) ([]*cmv1.NodePool, error) {

	nodePools, err := c.GetNodePools(clusterId)
	if err != nil {
		return []*cmv1.NodePool{}, err
	}

	var found []*cmv1.NodePool

	for _, n := range nodePools {
		if len(n.KubeletConfigs()) != 0 {
			if slices.Contains(n.KubeletConfigs(), kubeletName) {
				found = append(found, n)
			}
		}
	}

	return found, nil
}

func (c *Client) GetNodePools(clusterID string) ([]*cmv1.NodePool, error) {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		NodePools().
		List().Page(1).Size(-1).
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Items().Slice(), nil
}

func (c *Client) GetNodePool(clusterID string, nodePoolID string) (*cmv1.NodePool, bool, error) {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		NodePools().
		NodePool(nodePoolID).
		Get().
		Send()
	if response.Status() == 404 {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, handleErr(response.Error(), err)
	}
	return response.Body(), true, nil
}

func (c *Client) UpdateNodePool(clusterID string, nodePool *cmv1.NodePool) (*cmv1.NodePool, error) {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		NodePools().NodePool(nodePool.ID()).
		Update().Body(nodePool).
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Body(), nil
}

func (c *Client) DeleteNodePool(clusterID string, nodePoolID string) error {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		NodePools().NodePool(nodePoolID).
		Delete().
		Send()
	if err != nil {
		return handleErr(response.Error(), err)
	}
	return nil
}
