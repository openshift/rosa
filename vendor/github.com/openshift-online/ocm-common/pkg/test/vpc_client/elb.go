package vpc_client

import "time"

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
	if len(elbs) != 0 {
		time.Sleep(time.Minute) // sleep 1 minute to wait for the LBs totally deleted
	}
	return nil
}
