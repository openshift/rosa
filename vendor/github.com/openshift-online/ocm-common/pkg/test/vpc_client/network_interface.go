package vpc_client

func (vpc *VPC) DeleteVPCNetworkInterfaces() error {
	networkInterfaces, err := vpc.AWSClient.DescribeNetWorkInterface(vpc.VpcID)
	if err != nil {
		return err
	}

	for _, networkInterface := range networkInterfaces {
		err = vpc.AWSClient.DeleteNetworkInterface(networkInterface)
		if err != nil {
			return err
		}
	}
	return nil
}
