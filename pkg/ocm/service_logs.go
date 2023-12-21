package ocm

import (
	slv1 "github.com/openshift-online/ocm-sdk-go/servicelogs/v1"
)

func (c *Client) ListServiceLogs(clusterId string) (*slv1.ClustersClusterLogsListResponse, error) {
	return c.ocm.ServiceLogs().V1().Clusters().ClusterLogs().List().ClusterID(clusterId).Send()
}
