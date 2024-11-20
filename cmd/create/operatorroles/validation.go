package operatorroles

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
				hostedZoneRoleArnFlag,
				vpcEndpointRoleArnFlag,
			)
		}

		if route53RoleArn != "" && vpcEndpointRoleArn == "" {
			return false, errors.UserErrorf(
				arguments.MustUseBothFlagsErrorMessage,
				vpcEndpointRoleArnFlag,
				hostedZoneRoleArnFlag,
			)
		}
	} else {
		return false, nil
	}

	return hostedCp && vpcEndpointRoleArn != "" && route53RoleArn != "", nil
}
