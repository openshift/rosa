package operatorroles

import errors "github.com/zgalor/weberr"

func validateSharedVpcInputs(hostedCp bool, vpcEndpointRoleArn string,
	route53RoleArn string) (bool, error) {

	if !hostedCp {
		if vpcEndpointRoleArn != "" {
			return false, errors.UserErrorf("Can only use '%s' flag for Hosted Control Plane operator roles",
				vpcEndpointRoleArnFlag)
		}
	} else {
		if vpcEndpointRoleArn != "" && route53RoleArn == "" {
			return false, errors.UserErrorf(
				"Must supply '%s' flag when using the '%s' flag",
				hostedZoneRoleArnFlag,
				vpcEndpointRoleArnFlag,
			)
		}

		if route53RoleArn != "" && vpcEndpointRoleArn == "" {
			return false, errors.UserErrorf(
				"Must supply '%s' flag when using the '%s' flag",
				vpcEndpointRoleArnFlag,
				hostedZoneRoleArnFlag,
			)
		}
	}

	return hostedCp && vpcEndpointRoleArn != "" && route53RoleArn != "", nil
}
