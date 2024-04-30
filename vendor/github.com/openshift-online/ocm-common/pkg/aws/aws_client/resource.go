package aws_client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) ResourceExisting(resourceID string) bool {
	splitedResource := strings.SplitN(resourceID, "-", 2) //Just split the first -
	resourceType := splitedResource[0]
	switch resourceType {
	case "sg":
		input := &ec2.DescribeSecurityGroupsInput{
			GroupIds: []string{
				resourceID,
			},
		}
		output, err := client.Ec2Client.DescribeSecurityGroups(context.TODO(), input)
		if err != nil {
			if strings.Contains(err.Error(), "NotFound") {
				return false
			} else {
				log.LogError(err.Error())
				return false
			}
		}
		if len(output.SecurityGroups) != 0 {
			return true
		}
	case "subnet":
		subnetInput := &ec2.DescribeSubnetsInput{
			SubnetIds: []string{resourceID},
		}
		subnetOutput, err := client.Ec2Client.DescribeSubnets(context.TODO(), subnetInput)
		if err != nil {
			if strings.Contains(err.Error(), "NotFound") {
				return false
			} else {
				log.LogError(err.Error())
				return false
			}
		}
		if len(subnetOutput.Subnets) != 0 {
			return true
		}

		vpcInput := &ec2.DescribeVpcsInput{
			VpcIds: []string{resourceID},
		}
		vpcOutput, err := client.Ec2Client.DescribeVpcs(context.TODO(), vpcInput)
		if err != nil {
			if strings.Contains(err.Error(), "NotFound") {
				return false
			} else {
				log.LogError(err.Error())
				return false
			}
		}
		if len(vpcOutput.Vpcs) != 0 {
			return true
		}
	case "rtb":
		rbtInput := &ec2.DescribeRouteTablesInput{
			RouteTableIds: []string{
				resourceID,
			},
		}
		rbtOutput, err := client.Ec2Client.DescribeRouteTables(context.TODO(), rbtInput)
		if err != nil {
			if strings.Contains(err.Error(), "NotFound") {
				return false
			} else {
				log.LogError(err.Error())
				return false
			}
		}
		if len(rbtOutput.RouteTables) != 0 {
			return true
		}
	case "vpc":
		vpcInput := &ec2.DescribeVpcsInput{
			VpcIds: []string{
				resourceID,
			},
		}
		vpcOutput, err := client.Ec2Client.DescribeVpcs(context.TODO(), vpcInput)
		if err != nil {
			if strings.Contains(err.Error(), "NotFound") {
				return false
			} else {
				log.LogError(err.Error())
				return false
			}
		}
		if len(vpcOutput.Vpcs) != 0 {
			return true
		}
	case "eipalloc":
		input := &ec2.DescribeAddressesInput{
			AllocationIds: []string{
				resourceID,
			},
		}
		eipOutput, err := client.Ec2Client.DescribeAddresses(context.TODO(), input)
		if err != nil {
			if strings.Contains(err.Error(), "NotFound") {
				return false
			} else {
				log.LogError(err.Error())
				return false
			}
		}
		if len(eipOutput.Addresses) != 0 {
			return true
		}
	case "igw":
		input := &ec2.DescribeInternetGatewaysInput{
			InternetGatewayIds: []string{
				resourceID,
			},
		}
		output, err := client.Ec2Client.DescribeInternetGateways(context.TODO(), input)
		if err != nil {
			if strings.Contains(err.Error(), "NotFound") {
				return false
			} else {
				log.LogError(err.Error())
				return false
			}
		}
		if len(output.InternetGateways) != 0 {
			return true
		}
	case "nat":
		input := &ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []string{
				resourceID,
			},
		}
		output, err := client.Ec2Client.DescribeNatGateways(context.TODO(), input)
		if err != nil {
			if strings.Contains(err.Error(), "NotFound") {
				return false
			} else {
				log.LogError(err.Error())
				return false
			}
		}
		if len(output.NatGateways) != 0 {
			log.LogDebug("Current NAT gateway %s status %s ", resourceID, output.NatGateways[0].State)
			status := string(output.NatGateways[0].State)
			if status == "available" {
				return true
			}

		}
	// role should use "role-<rolename>" to pass
	case "role":
		role, _ := client.GetRole(splitedResource[1])
		return role != nil
	// policy should use "policy-<policy arn>" as parameter
	case "policy":
		policy, _ := client.GetIAMPolicy(splitedResource[1])
		return policy != nil
	default:
		log.LogError("Unknow resource type: %s of resource %s .Please define it in the method ResourceExisting.", resourceType, resourceID)
	}
	return false
}

func (client *AWSClient) ResourceDeleted(resourceID string) bool {
	var deleted bool = true
	splitedResource := strings.Split(resourceID, "-")
	resourceType := splitedResource[0]
	switch resourceType {
	case "subnet":
		subnetInput := &ec2.DescribeSubnetsInput{
			SubnetIds: []string{resourceID},
		}
		subnetOutput, err := client.Ec2Client.DescribeSubnets(context.TODO(), subnetInput)
		if err != nil {
			if strings.Contains(err.Error(), "NotFound") {
				return true
			} else {
				log.LogError(err.Error())
				return false
			}
		}
		if len(subnetOutput.Subnets) != 0 {
			deleted = false
		}
	case "rtb":
		rbtInput := &ec2.DescribeRouteTablesInput{
			RouteTableIds: []string{
				resourceID,
			},
		}
		rbtOutput, err := client.Ec2Client.DescribeRouteTables(context.TODO(), rbtInput)
		if err != nil {
			if strings.Contains(err.Error(), "NotFound") {
				return true
			} else {
				log.LogError(err.Error())
				return false
			}
		}
		if len(rbtOutput.RouteTables) != 0 {
			deleted = false
		}
	case "vpc":
		vpcInput := &ec2.DescribeVpcsInput{
			VpcIds: []string{
				resourceID,
			},
		}
		vpcOutput, err := client.Ec2Client.DescribeVpcs(context.TODO(), vpcInput)
		if err != nil {
			if strings.Contains(err.Error(), "NotFound") {
				return true
			} else {
				log.LogError(err.Error())
				return false
			}
		}
		if len(vpcOutput.Vpcs) != 0 {
			deleted = false
		}
	case "eipalloc":
		input := &ec2.DescribeAddressesInput{
			AllocationIds: []string{
				resourceID,
			},
		}
		eipOutput, err := client.Ec2Client.DescribeAddresses(context.TODO(), input)
		if err != nil {
			if strings.Contains(err.Error(), "NotFound") {
				return true
			} else {
				log.LogError(err.Error())
				return false
			}
		}
		if len(eipOutput.Addresses) != 0 {
			deleted = false
		}
	case "igw":
		input := &ec2.DescribeInternetGatewaysInput{
			InternetGatewayIds: []string{
				resourceID,
			},
		}
		output, err := client.Ec2Client.DescribeInternetGateways(context.TODO(), input)
		if err != nil {
			if strings.Contains(err.Error(), "NotFound") {
				return true
			} else {
				log.LogError(err.Error())
				return false
			}
		}
		if len(output.InternetGateways) != 0 {
			deleted = false
		}
	case "sg":
		input := &ec2.DescribeSecurityGroupsInput{
			GroupIds: []string{
				resourceID,
			},
		}
		output, err := client.Ec2Client.DescribeSecurityGroups(context.TODO(), input)
		if err != nil {
			if strings.Contains(err.Error(), "NotFound") {
				return true
			} else {
				log.LogError(err.Error())
				return false
			}
		}
		if len(output.SecurityGroups) != 0 {
			deleted = false
		}
	case "nat":
		input := &ec2.DescribeNatGatewaysInput{
			NatGatewayIds: []string{
				resourceID,
			},
		}
		output, err := client.Ec2Client.DescribeNatGateways(context.TODO(), input)
		if err != nil {
			log.LogError(err.Error())
			return false
		}
		if len(output.NatGateways) != 0 {
			log.LogDebug("Current NAT gateway %s status %s", resourceID, output.NatGateways[0].State)
			status := string(output.NatGateways[0].State)
			if status != "deleted" {
				deleted = false
			}

		}
	default:
		log.LogError("Unknow resource type: %s of resource %s .Please define it in the method ResourceExisting.", resourceType, resourceID)
	}
	return deleted
}

// WaitForResourceExisting will wait for the resource created in <timeout> seconds
func (client AWSClient) WaitForResourceExisting(resourceID string, timeout int) error {
	now := time.Now()
	for now.Add(time.Duration(timeout) * time.Second).After(time.Now()) {
		if client.ResourceExisting(resourceID) {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout after %d seconds for waiting resource created: %s", timeout, resourceID)
}

// WaitForResourceExisting will wait for the resource created in <timeout> seconds
func (client AWSClient) WaitForResourceDeleted(resourceID string, timeout int) error {
	now := time.Now()
	for now.Add(time.Duration(timeout) * time.Second).After(time.Now()) {
		if client.ResourceDeleted(resourceID) {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("Timeout after %d seconds for waiting resource deleted: %s", timeout, resourceID)
}
