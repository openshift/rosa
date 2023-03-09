package ocm

import (
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/zgalor/weberr"
)

func (c *Client) CreateRedHatHostedOidcConfig(secretsArn string, installerRoleArn string) (*cmv1.HostedOidcConfig, error) {
	oidcConfig, err := cmv1.NewHostedOidcConfig().
		InstallerRoleArn("arn:aws:iam::.../ManagedOpenShift-Installer-Role").
		OidcPrivateKeySecretArn("arn:aws:secretsmanager:xxx").Build()
	if err != nil {
		return nil, weberr.Errorf("Failed to create Hosted Oidc Config: %v", clusterKey, err)
	}
	response, err := c.ocm.ClustersMgmt().V1().HostedOidcConfigs().
		Add().Body(oidcConfig).Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	return response.Body(), nil
}

// At the moment there is only one per organization, so no reason to check others in list
func (c *Client) GetRedHatHostedOidcConfig() (*cmv1.HostedOidcConfig, error) {
	response, err := c.ocm.ClustersMgmt().V1().HostedOidcConfigs().List().Send()
	if err != nil {
		return nil, handleErr(response.Error(), err)
	}
	if response.Total() < 1 {
		return nil, weberr.Errorf("No Red Hat Hosted OIDC Honfigurations for your organization are present")
	}
	return response.Items().Get(0), nil
}

func (c *Client) DeleteRedHatHostedOidcConfig() error {
	response, err := c.GetRedHatHostedOidcConfig()
	if err != nil {
		return err
	}
	_, err := c.ocm.ClustersMgmt().V1().
		HostedOidcConfig(response.ID()).Delete().Send()
	if err != nil {
		return err
	}
	return nil
}
