package aws_client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func (awsClient AWSClient) CreateHostedZone(hostedZoneName string, callerReference string, vpcID string, region string, private bool) (*route53.CreateHostedZoneOutput, error) {
	// CreateHostedZone is a function used for hosted zone creation on AWS
	// callReference is a required field of CreateHostedZoneInput struct, which used to identifies the request as a unique string.
	// Usually random string or date/time stamp can be used as callReference.
	input := &route53.CreateHostedZoneInput{
		Name:            &hostedZoneName,
		CallerReference: &callerReference,
		HostedZoneConfig: &types.HostedZoneConfig{
			PrivateZone: private,
		},
	}
	if vpcID != "" {
		vpc := &types.VPC{
			VPCId:     &vpcID,
			VPCRegion: types.VPCRegion(region),
		}
		input.VPC = vpc
	}

	resp, err := awsClient.Route53Client.CreateHostedZone(context.TODO(), input)
	if err != nil {
		log.LogError("Create hosted zone failed for vpc %s with name %s: %s", vpcID, hostedZoneName, err.Error())
	} else {
		log.LogInfo("Create hosted zone succeed for vpc %s with name %s", vpcID, hostedZoneName)
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

func (awsClient AWSClient) DeleteHostedZone(hostedZoneID string) error {
	input := &route53.DeleteHostedZoneInput{
		Id: &hostedZoneID,
	}

	_, err := awsClient.Route53Client.DeleteHostedZone(context.TODO(), input)
	return err
}
