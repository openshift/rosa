package ocm

import (
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	errors "github.com/zgalor/weberr"
)

func (c *Client) FetchClusterMigrations(clusterId string) (*cmv1.ClusterMigrationsListResponse,
	error) {
	collection := c.ocm.ClustersMgmt().V1().Clusters().Cluster(clusterId)
	resources := collection.Migrations().List()

	clusterMigrations, err := resources.Send()
	if err != nil {
		return nil, errors.UserWrapf(err, "Can't retrieve migrations for cluster '%s'\n", clusterId)
	}

	return clusterMigrations, nil
}
