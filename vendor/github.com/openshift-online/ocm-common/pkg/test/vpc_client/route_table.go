package vpc_client

func (vpc *VPC) DescribeSubnetRTMappings(subnets ...*Subnet) error {
	rts, err := vpc.AWSClient.ListCustomerRouteTables(vpc.VpcID)
	if err != nil {
		return err
	}
	for _, subnet := range subnets {
		subnet.RTable = getSubnetRouteTable(subnet.ID, rts)
	}

	return nil
}

// DeleteVPCRouteTables will delete all of route table resources including associations and routes
func (vpc *VPC) DeleteVPCRouteTables(vpcID string) error {
	rts, err := vpc.AWSClient.ListCustomerRouteTables(vpcID)
	if err != nil {
		return err
	}
	for _, rt := range rts {
		for _, asso := range rt.Associations {
			_, err = vpc.AWSClient.DisassociateRouteTableAssociation(*asso.RouteTableAssociationId)
			if err != nil {
				return err
			}
		}
		err = vpc.AWSClient.DeleteRouteTable(*rt.RouteTableId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (vpc *VPC) DeleteRouteTableChains(routeTables ...string) error {
	for _, routeTable := range routeTables {
		err := vpc.AWSClient.DeleteRouteTableChain(routeTable)
		if err != nil {
			return err
		}
	}
	return nil
}
