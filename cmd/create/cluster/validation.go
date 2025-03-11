package cluster

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

const (
	hcpSharedVpcFlagOnlyErrorMsg = "setting the '%s' flag is only supported when creating a Hosted Control Plane " +
		"cluster"

	hcpSharedVpcFlagNotFilledErrorMsg = "must supply '%s' flag when creating a Hosted Control Plane shared VPC cluster"

	isNotValidArnErrorMsg = "ARN supplied with flag '%s' is not a valid ARN format"

	isNotGovcloudFeature = "Hosted Control Plane shared VPC clusters are not supported on Govcloud regions; %s"
	pleaseRemoveFlags    = "Please remove the following flags: %s"
)

func validateHcpSharedVpcArgs(route53RoleArn string, vpcEndpointRoleArn string,
	ingressPrivateHostedZoneId string, hcpInternalCommunicationHostedZoneId string, fedrampEnabled bool) error {

	if fedrampEnabled {
		stringsUsed := []string{}
		for key, val := range map[string]string{
			route53RoleArnFlag:                       route53RoleArn,
			vpcEndpointRoleArnFlag:                   vpcEndpointRoleArn,
			ingressPrivateHostedZoneIdFlag:           ingressPrivateHostedZoneId,
			hcpInternalCommunicationHostedZoneIdFlag: hcpInternalCommunicationHostedZoneId,
		} {
			if val != "" {
				stringsUsed = append(stringsUsed, key)
			}
		}
		if len(stringsUsed) > 0 {
			flags := "'" + strings.Join(stringsUsed, "', '") + "'"
			return fmt.Errorf(isNotGovcloudFeature, fmt.Sprintf(pleaseRemoveFlags, flags))
		}
	}

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
