package aws_client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (awsClient AWSClient) CreateHostedZone(hostedZoneName string, vpcID string, private bool) (*route53.CreateHostedZoneOutput, error) {
	input := &route53.CreateHostedZoneInput{
		Name: &hostedZoneName,
		HostedZoneConfig: &types.HostedZoneConfig{
			PrivateZone: private,
		},
	}
	if vpcID != "" {
		vpc := &types.VPC{
			VPCId: &vpcID,
		}
		input.VPC = vpc
	}
	resp, err := awsClient.Route53Client.CreateHostedZone(context.TODO(), input)
	if err != nil {
		log.LogError("Create hosted zone failed for vpc %s with name %s: %s", vpcID, hostedZoneName, err.Error())
	} else {
		log.LogError("Create hosted zone succeed for vpc %s with name %s", vpcID, hostedZoneName)
	}
	return resp, err
}

func (awsClient AWSClient) GetHostedZone(hostedZoneID string) (*route53.GetHostedZoneOutput, error) {
	input := &route53.GetHostedZoneInput{
		Id: &hostedZoneID,
	}

	return awsClient.Route53Client.GetHostedZone(context.TODO(), input)
}

func (awsClient AWSClient) ListHostedZoneByDNSName(hostedZoneName string) (*route53.ListHostedZonesByNameOutput, error) {
	var maxItems int32 = 1
	input := &route53.ListHostedZonesByNameInput{
		DNSName:  &hostedZoneName,
		MaxItems: &maxItems,
	}

	return awsClient.Route53Client.ListHostedZonesByName(context.TODO(), input)
}
