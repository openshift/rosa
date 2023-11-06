package ocm

import (
	"net/http"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func (c *Client) GetClusterKubeletConfig(clusterID string) (*cmv1.KubeletConfig, error) {
	response, err := c.ocm.ClustersMgmt().V1().Clusters().Cluster(clusterID).KubeletConfig().Get().Send()

	if response.Status() == http.StatusNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, handleErr(response.Error(), err)
	}

	return response.Body(), nil
}
