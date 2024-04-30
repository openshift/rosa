package aws_client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/openshift-online/ocm-common/pkg/log"
)

func (client *AWSClient) ListNetWorkAcls(vpcID string) ([]types.NetworkAcl, error) {
	vpcFilter := "vpc-id"
	customizedAcls := []types.NetworkAcl{}
	filter := []types.Filter{
		types.Filter{
			Name: &vpcFilter,
			Values: []string{
				vpcID,
			},
		},
	}
	describeACLInput := &ec2.DescribeNetworkAclsInput{
		Filters: filter,
	}
	output, err := client.Ec2Client.DescribeNetworkAcls(context.TODO(), describeACLInput)
	if err != nil {
		return nil, err
	}
	customizedAcls = append(customizedAcls, output.NetworkAcls...)
	return customizedAcls, nil
}

// RuleAction : deny/allow
// Protocol: TCP --> 6
func (client *AWSClient) AddNetworkAclEntry(networkAclId string, egress bool, protocol string, ruleAction string, ruleNumber int32, fromPort int32, toPort int32, cidrBlock string) (*ec2.CreateNetworkAclEntryOutput, error) {
	input := &ec2.CreateNetworkAclEntryInput{
		Egress:       aws.Bool(egress),
		NetworkAclId: aws.String(networkAclId),
		Protocol:     aws.String(protocol),
		RuleAction:   types.RuleAction(ruleAction),
		RuleNumber:   aws.Int32(ruleNumber),
		CidrBlock:    aws.String(cidrBlock),
		PortRange: &types.PortRange{
			From: aws.Int32(fromPort),
			To:   aws.Int32(toPort),
		},
	}
	resp, err := client.Ec2Client.CreateNetworkAclEntry(context.TODO(), input)
	if err != nil {
		log.LogError("Create NetworkAcl rule failed " + err.Error())
		return nil, err
	}
	log.LogInfo("Create NetworkAcl rule success " + networkAclId)
	return resp, err
}

func (client *AWSClient) DeleteNetworkAclEntry(networkAclId string, egress bool, ruleNumber int32) (*ec2.DeleteNetworkAclEntryOutput, error) {
	input := &ec2.DeleteNetworkAclEntryInput{
		Egress:       aws.Bool(egress),
		NetworkAclId: aws.String(networkAclId),
		RuleNumber:   aws.Int32(ruleNumber),
	}
	resp, err := client.Ec2Client.DeleteNetworkAclEntry(context.TODO(), input)
	if err != nil {
		log.LogError("Delete NetworkAcl rule failed " + err.Error())
		return nil, err
	}
	log.LogInfo("Delete NetworkAcl rule success " + networkAclId)
	return resp, err

}
