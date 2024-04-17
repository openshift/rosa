package vpc_client

// PrepareInternetGateway will return the existing internet gateway if there is one attached to the vpc
// Otherwise, it will create a new one and attach to the VPC
func (vpc *VPC) PrepareInternetGateway() (igwID string, err error) {
	igws, err := vpc.AWSClient.ListInternetGateWay(vpc.VpcID)
	if err != nil {
		return "", err
	}
	if len(igws) != 0 {
		return *igws[0].InternetGatewayId, nil
	}
	igw, err := vpc.AWSClient.CreateInternetGateway()
	if err != nil {
		return "", err
	}
	_, err = vpc.AWSClient.AttachInternetGateway(*igw.InternetGateway.InternetGatewayId, vpc.VpcID)
	if err != nil {
		return "", err
	}
	return *igw.InternetGateway.InternetGatewayId, nil
}

func (vpc *VPC) DeleteVPCInternetGateWays() error {
	igws, err := vpc.AWSClient.ListInternetGateWay(vpc.VpcID)
	if err != nil {
		return err
	}
	for _, igw := range igws {
		_, err = vpc.AWSClient.DetachInternetGateway(*igw.InternetGatewayId, vpc.VpcID)
		if err != nil {
			return err
		}
		_, err = vpc.AWSClient.DeleteInternetGateway(*igw.InternetGatewayId)
		if err != nil {
			return err
		}
	}
	return nil
}
