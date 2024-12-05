package cluster

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

const (
	hcpSharedVpcFlagOnlyErrorMsg = "setting the '%s' flag is only supported when creating a Hosted Control Plane " +
		"cluster"

	hcpSharedVpcFlagNotFilledErrorMsg = "must supply '%s' flag when creating a Hosted Control Plane shared VPC cluster"

	isNotValidArnErrorMsg = "ARN supplied with flag '%s' is not a valid ARN format"
)

func validateHcpSharedVpcArgs(route53RoleArn string, vpcEndpointRoleArn string,
	ingressPrivateHostedZoneId string, hcpInternalCommunicationHostedZoneId string) error {

	if route53RoleArn == "" {
		return fmt.Errorf(hcpSharedVpcFlagNotFilledErrorMsg, route53RoleArnFlag)
	} else if !arn.IsARN(route53RoleArn) {
		return fmt.Errorf(isNotValidArnErrorMsg, route53RoleArnFlag)
	}
	if vpcEndpointRoleArn == "" {
		return fmt.Errorf(hcpSharedVpcFlagNotFilledErrorMsg, vpcEndpointRoleArnFlag)
	} else if !arn.IsARN(vpcEndpointRoleArn) {
		return fmt.Errorf(isNotValidArnErrorMsg, vpcEndpointRoleArnFlag)
	}
	if ingressPrivateHostedZoneId == "" {
		return fmt.Errorf(hcpSharedVpcFlagNotFilledErrorMsg, ingressPrivateHostedZoneIdFlag)
	}
	if hcpInternalCommunicationHostedZoneId == "" {
		return fmt.Errorf(hcpSharedVpcFlagNotFilledErrorMsg, hcpInternalCommunicationHostedZoneIdFlag)
	}
	return nil
}
