package cluster

import (
	"fmt"

	"github.com/spf13/cobra"
	errors "github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/rosa"
)

// nolint:lll
const sharedVpcDocsLink = "https://docs.openshift.com/rosa/rosa_install_access_delete_clusters/rosa-shared-vpc-config.html"

func isSubnetBelongToSharedVpc(r *rosa.Runtime, accountID string, subnetIDs []string,
	mapSubnetIDToSubnet map[string]aws.Subnet) bool {
	for _, subnetID := range subnetIDs {
		ownerID := mapSubnetIDToSubnet[subnetID].OwnerID
		if ownerID != "" && ownerID != accountID {
			r.Reporter.Infof(fmt.Sprintf("Subnet with ID '%s' is shared by AWS account '%s', "+
				"the cluster will be installed into a shared VPC. For more details %s.", subnetID, ownerID, sharedVpcDocsLink))
			return true
		}
	}

	return false
}

func getPrivateHostedZoneID(cmd *cobra.Command, privateHostedZoneID string) (string, error) {
	res, err := interactive.GetString(interactive.Input{
		Question: "Ingress private hosted zone ID",
		Help:     cmd.Flags().Lookup("private-hosted-zone-id").Usage,
		Default:  privateHostedZoneID,
		Required: true,
	})
	// TODO: Update error when we deprecate the old flags
	if err != nil {
		return "", errors.Errorf("Expected a valid value for 'private-hosted-zone-id': %s", err)
	}

	return res, nil
}

func getHcpInternalCommunicationHostedZoneId(cmd *cobra.Command, hcpInternalHostedZoneId string) (string, error) {
	res, err := interactive.GetString(interactive.Input{
		Question: "Hosted Control Plane internal communication hosted zone ID",
		Help:     cmd.Flags().Lookup(hcpInternalCommunicationHostedZoneIdFlag).Usage,
		Default:  hcpInternalHostedZoneId,
		Required: true,
	})
	if err != nil {
		return "", errors.Errorf("Expected a valid value for '%s': %s", hcpInternalCommunicationHostedZoneIdFlag,
			err)
	}

	return res, nil
}

func getSharedVpcRoleArn(cmd *cobra.Command, sharedVpcRoleArn string) (string, error) {
	res, err := interactive.GetString(interactive.Input{
		Question: "Shared VPC role ARN (Route53 role ARN)", // TODO: Change once we deprecate the old flags
		Help:     cmd.Flags().Lookup("shared-vpc-role-arn").Usage,
		Default:  sharedVpcRoleArn,
		Required: true,
		Validators: []interactive.Validator{
			aws.ARNValidator,
		},
	})
	// TODO: Update error when we deprecate the old flags
	if err != nil {
		return "", errors.Errorf("Expected a valid value for 'shared-vpc-role-arn': %s", err)
	}

	return res, nil
}

func getVpcEndpointRoleArn(cmd *cobra.Command, vpcEndpointRoleArn string) (string, error) {
	res, err := interactive.GetString(interactive.Input{
		Question: "VPC endpoint role ARN",
		Help:     cmd.Flags().Lookup(vpcEndpointRoleArnFlag).Usage,
		Default:  vpcEndpointRoleArn,
		Required: true,
		Validators: []interactive.Validator{
			aws.ARNValidator,
		},
	})
	if err != nil {
		return "", errors.Errorf("Expected a valid value for '%s': %s", vpcEndpointRoleArnFlag, err)
	}

	return res, nil
}

func getBaseDomain(r *rosa.Runtime, cmd *cobra.Command, baseDomain string) (string, error) {
	dnsDomains, err := getAvailableBaseDomains(r)
	if err != nil {
		return "", err
	}

	res, err := interactive.GetOption(interactive.Input{
		Question: "Base Domain",
		Help:     cmd.Flags().Lookup("base-domain").Usage,
		Default:  baseDomain,
		Required: true,
		Options:  dnsDomains,
	})
	if err != nil {
		return "", errors.Errorf("Expected a valid value for 'base-domain': %s", err)
	}

	return res, nil
}

func getAvailableBaseDomains(r *rosa.Runtime) ([]string, error) {
	organizationID, _, err := r.OCMClient.GetCurrentOrganization()
	if err != nil {
		return nil, errors.Errorf("Failed to get current OCM organization ID: %s", err)
	}

	dnsDomains, err := r.OCMClient.ListDNSDomains(
		fmt.Sprintf("user_defined='true' and cluster.id='' and organization.id='%s'", organizationID))
	if err != nil {
		return nil, errors.Errorf("Failed to list DNS domains: %s", err)
	}

	var dnsDomainsIDs []string
	for _, dnsDomain := range dnsDomains {
		dnsDomainsIDs = append(dnsDomainsIDs, dnsDomain.ID())
	}

	return dnsDomainsIDs, nil
}
