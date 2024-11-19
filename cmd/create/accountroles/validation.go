package accountroles

import (
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/arguments"
)

func validateSharedVpcInputs(hostedCp bool, vpcEndpointRoleArn string,
	route53RoleArn string) (bool, error) {
	if hostedCp {
		if vpcEndpointRoleArn != "" && route53RoleArn == "" {
			return false, errors.UserErrorf(
				arguments.MustUseBothFlagsErrorMessage,
				route53RoleArnFlag,
				vpcEndpointRoleArnFlag,
			)
		}

		if route53RoleArn != "" && vpcEndpointRoleArn == "" {
			return false, errors.UserErrorf(
				arguments.MustUseBothFlagsErrorMessage,
				vpcEndpointRoleArnFlag,
				route53RoleArnFlag,
			)
		}
	}

	return hostedCp && vpcEndpointRoleArn != "" && route53RoleArn != "", nil
}
