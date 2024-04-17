package vpc_client

func (vpc *VPC) DeleteVPCELBs() error {
	elbs, err := vpc.AWSClient.DescribeLoadBalancers(vpc.VpcID)
	if err != nil {
		return err
	}

	for _, elb := range elbs {
		err = vpc.AWSClient.DeleteELB(elb)
		if err != nil {
			return err
		}
	}
	return nil
}
