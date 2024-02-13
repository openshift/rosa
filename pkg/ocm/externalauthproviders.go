package ocm

import cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

// need to change to external auth providers once the api is in
// for now keeps nodepool for templating
func (c *Client) CreateExternalAuthProviders(clusterID string, nodePool *cmv1.NodePool) (*cmv1.NodePool, error) {
	response, err := c.ocm.ClustersMgmt().V1().
		Clusters().Cluster(clusterID).
		NodePools().
		Add().Body(nodePool).
		Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Body(), nil
}

// same for get
func (c *Client) GetExternalAuthProviders(clusterID string, nodePoolID string) (*cmv1.NodePool, bool, error) {
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

// same for update
func (c *Client) UpdateExternalAuthProviders(clusterID string, nodePool *cmv1.NodePool) (*cmv1.NodePool, error) {
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

// same for delete
func (c *Client) DeleteExternalAuthProviders(clusterID string, nodePoolID string) error {
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
