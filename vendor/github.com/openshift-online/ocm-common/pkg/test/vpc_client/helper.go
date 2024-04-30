package vpc_client

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func GetSubnetsRouteTables(routeTables []types.RouteTable, subnetIDs ...string) map[string]*types.RouteTable {
	rtMap := map[string]*types.RouteTable{}

	for _, subnetID := range subnetIDs {
		rtMap[subnetID] = getSubnetRouteTable(subnetID, routeTables)
	}

	return rtMap

}

func getSubnetRouteTable(
	subnetID string, routeTables []types.RouteTable) *types.RouteTable {
	var mainTable *types.RouteTable
	for i := range routeTables {
		for _, assoc := range routeTables[i].Associations {
			if aws.ToString(assoc.SubnetId) == subnetID {
				return &routeTables[i]
			}
			if aws.ToBool(assoc.Main) {
				mainTable = &routeTables[i]
			}
		}
	}
	// If there is no explicit association, the subnet will be implicitly
	// associated with the VPC's main routing table.
	return mainTable
}

func getTagName(tags []types.Tag) string {
	name := ""
	for _, tag := range tags {
		if *tag.Key == "Name" {
			name = *tag.Value
		}
	}
	return name
}
