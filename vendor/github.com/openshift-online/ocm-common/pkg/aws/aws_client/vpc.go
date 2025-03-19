package aws_client

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	CON "github.com/openshift-online/ocm-common/pkg/aws/consts"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) ListVPCs(filter ...types.Filter) ([]types.Vpc, error) {
	vpcs := []types.Vpc{}
	input := &ec2.DescribeVpcsInput{}
	if len(filter) != 0 {
		input.Filters = filter
	}
	resp, err := client.Ec2Client.DescribeVpcs(context.TODO(), input)
	if err != nil {
		return vpcs, err
	}
	vpcs = resp.Vpcs
	return vpcs, nil
}

func (client *AWSClient) ListVPCByName(vpcName string) ([]types.Vpc, error) {

	filterKey := "tag:Name"
	filter := []types.Filter{
		{
			Name:   &filterKey,
			Values: []string{vpcName},
		},
	}

	return client.ListVPCs(filter...)
}

func (client *AWSClient) CreateVpc(cidr string, name ...string) (*ec2.CreateVpcOutput, error) {
	vpcName := CON.VpcDefaultName
	if len(name) == 1 {
		vpcName = name[0]
	}
	tags := map[string]string{
		"Name":        vpcName,
		CON.QEFlagKey: CON.QEFLAG,
	}
	input := &ec2.CreateVpcInput{
		CidrBlock:         aws.String(cidr),
		DryRun:            nil,
		InstanceTenancy:   "",
		Ipv4IpamPoolId:    nil,
		Ipv4NetmaskLength: nil,
		TagSpecifications: nil,
	}

	resp, err := client.Ec2Client.CreateVpc(context.TODO(), input)
	if err != nil {
		log.LogError("Create vpc error " + err.Error())
		return nil, err
	}
	log.LogInfo("Create vpc success " + *resp.Vpc.VpcId)
	err = client.WaitForResourceExisting(*resp.Vpc.VpcId, 10)
	if err != nil {
		return resp, err
	}

	_, err = client.TagResource(*resp.Vpc.VpcId, tags)
	if err != nil {
		return resp, err
	}

	log.LogInfo("Created vpc with ID " + *resp.Vpc.VpcId)
	return resp, err
}

// ModifyVpcDnsAttribute will modify the vpc attibutes
// dnsAttribute should be the value of "DnsHostnames" and "DnsSupport"
func (client *AWSClient) ModifyVpcDnsAttribute(vpcID string, dnsAttribute string, status bool) (*ec2.ModifyVpcAttributeOutput, error) {
	inputModifyVpc := &ec2.ModifyVpcAttributeInput{}

	if dnsAttribute == CON.VpcDnsHostnamesAttribute {
		inputModifyVpc = &ec2.ModifyVpcAttributeInput{
			VpcId:              aws.String(vpcID),
			EnableDnsHostnames: &types.AttributeBooleanValue{Value: aws.Bool(status)},
		}
	} else if dnsAttribute == CON.VpcDnsSupportAttribute {
		inputModifyVpc = &ec2.ModifyVpcAttributeInput{
			VpcId:            aws.String(vpcID),
			EnableDnsSupport: &types.AttributeBooleanValue{Value: aws.Bool(status)},
		}
	}

	resp, err := client.Ec2Client.ModifyVpcAttribute(context.TODO(), inputModifyVpc)
	if err != nil {
		log.LogError("Modify vpc dns attribute failed " + err.Error())
		return nil, err
	}
	log.LogInfo("Modify vpc dns attribute %s successfully for %s", dnsAttribute, vpcID)
	return resp, err
}

func (client *AWSClient) DeleteVpc(vpcID string) (*ec2.DeleteVpcOutput, error) {
	input := &ec2.DeleteVpcInput{
		VpcId:  aws.String(vpcID),
		DryRun: nil,
	}

	resp, err := client.Ec2Client.DeleteVpc(context.TODO(), input)
	if err != nil {
		log.LogError("Delete vpc %s failed with error %s", vpcID, err.Error())
		return nil, err
	}
	log.LogInfo("Delete vpc %s successfuly ", vpcID)
	return resp, err

}
func (client *AWSClient) DescribeVPC(vpcID string) (types.Vpc, error) {
	var vpc types.Vpc
	input := &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	}

	resp, err := client.Ec2Client.DescribeVpcs(context.TODO(), input)
	if err != nil {
		return vpc, err
	}
	vpc = resp.Vpcs[0]
	return vpc, err
}

func (client *AWSClient) ListEndpointAssociation(vpcID string) ([]types.VpcEndpoint, error) {
	vpcFilterKey := "vpc-id"
	filters := []types.Filter{
		{
			Name:   &vpcFilterKey,
			Values: []string{vpcID},
		},
	}

	input := ec2.DescribeVpcEndpointsInput{
		Filters: filters,
	}
	resp, err := client.Ec2Client.DescribeVpcEndpoints(context.TODO(), &input)
	if err != nil {
		return nil, err
	}
	return resp.VpcEndpoints, err
}

func (client *AWSClient) DeleteVPCEndpoints(vpcID string) error {
	vpcEndpoints, err := client.ListEndpointAssociation(vpcID)
	if err != nil {
		return err
	}
	var endpoints = []string{}
	for _, ve := range vpcEndpoints {
		endpoints = append(endpoints, *ve.VpcEndpointId)
	}
	if len(endpoints) != 0 {
		input := &ec2.DeleteVpcEndpointsInput{
			VpcEndpointIds: endpoints,
		}
		_, err = client.Ec2Client.DeleteVpcEndpoints(context.TODO(), input)
		if err != nil {
			log.LogError("Delete vpc endpoints %s failed: %s", strings.Join(endpoints, ","), err.Error())
		} else {
			log.LogInfo("Delete vpc endpoints %s successfully", strings.Join(endpoints, ","))
		}
	}
	return err
}

func (client *AWSClient) CreateVPCEndpoint(vpcID string, serviceName string, vpcEndpointType types.VpcEndpointType) error {
	input := &ec2.CreateVpcEndpointInput{
		VpcId:           &vpcID,
		ServiceName:     &serviceName,
		VpcEndpointType: vpcEndpointType,
	}
	output, err := client.Ec2Client.CreateVpcEndpoint(context.TODO(), input)
	if err != nil {
		log.LogError("Create vpc endpoints failed: %s", err.Error())
	} else {
		log.LogInfo("Create vpc endpoints %s successfully", *output.VpcEndpoint.VpcEndpointId)
	}
	return err
}
