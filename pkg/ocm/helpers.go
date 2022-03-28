/*
Copyright (c) 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ocm

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/arn"
	semver "github.com/hashicorp/go-version"
	errors "github.com/zgalor/weberr"

	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ocmerrors "github.com/openshift-online/ocm-sdk-go/errors"

	"github.com/openshift/rosa/pkg/helper"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	ANY                 = "any"
	HibernateCapability = "capability.organization.hibernate_cluster"
	//Pendo Events
	Success   = "Success"
	Failure   = "Failure"
	Response  = "Response"
	ClusterID = "ClusterID"
	Version   = "Version"
	Username  = "Username"
	URL       = "URL"

	OCMRoleLabel  = "sts_ocm_role"
	USERRoleLabel = "sts_user_role"
)

// Regular expression to used to make sure that the identifier or name given by the user is
// safe and that it there is no risk of SQL injection:
var clusterKeyRE = regexp.MustCompile(`^(\w|-)+$`)

// Cluster names must be valid DNS-1035 labels, so they must consist of lower case alphanumeric
// characters or '-', start with an alphabetic character, and end with an alphanumeric character
var clusterNameRE = regexp.MustCompile(`^[a-z]([-a-z0-9]{0,13}[a-z0-9])?$`)

var badUsernameRE = regexp.MustCompile(`^(~|\.?\.|.*[:\/%].*)$`)

func IsValidClusterKey(clusterKey string) bool {
	return clusterKeyRE.MatchString(clusterKey)
}

func IsValidClusterName(clusterName string) bool {
	return clusterNameRE.MatchString(clusterName)
}

func ClusterNameValidator(name interface{}) error {
	if str, ok := name.(string); ok {
		if !IsValidClusterName(str) {
			return fmt.Errorf("Cluster name must consist of no more than 15 lowercase " +
				"alphanumeric characters or '-', start with a letter, and end with an " +
				"alphanumeric character.")
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", name)
}

func ValidateHTTPProxy(val interface{}) error {
	if httpProxy, ok := val.(string); ok {
		if httpProxy == "" {
			return nil
		}
		url, err := url.ParseRequestURI(httpProxy)
		if err != nil {
			return fmt.Errorf("Invalid http-proxy value '%s'", httpProxy)
		}
		if url.Scheme != "http" {
			return errors.Errorf("%s", "Expected http-proxy to have an http:// scheme")
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", val)
}

func ValidateAdditionalTrustBundle(val interface{}) error {
	if additionalTrustBundleFile, ok := val.(string); ok {
		if additionalTrustBundleFile == "" {
			return nil
		}
		cert, err := ioutil.ReadFile(additionalTrustBundleFile)
		if err != nil {
			return err
		}
		additionalTrustBundle := string(cert)
		if additionalTrustBundle == "" {
			return errors.Errorf("%s", "Trust bundle file is empty")
		}
		additionalTrustBundleBytes := []byte(additionalTrustBundle)
		if !x509.NewCertPool().AppendCertsFromPEM(additionalTrustBundleBytes) {
			return errors.Errorf("%s", "Failed to parse additional trust bundle")
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", val)
}

func IsValidUsername(username string) bool {
	return !badUsernameRE.MatchString(username)
}

func IsEmptyCIDR(cidr net.IPNet) bool {
	return cidr.String() == "<nil>"
}

// Determine whether a resources is compatible with ROSA clusters in general
func isCompatible(relatedResource *amsv1.RelatedResource) bool {
	product := strings.ToLower(relatedResource.Product())
	cloudProvider := strings.ToLower(relatedResource.CloudProvider())
	byoc := strings.ToLower(relatedResource.BYOC())

	// nolint:goconst
	return (product == ANY || product == "rosa" || product == "moa") &&
		(cloudProvider == ANY || cloudProvider == "aws") &&
		(byoc == ANY || byoc == "byoc")
}

func handleErr(res *ocmerrors.Error, err error) error {
	msg := res.Reason()
	if msg == "" {
		msg = err.Error()
	}
	// Hack to always display the correct terms and conditions message
	if res.Code() == "CLUSTERS-MGMT-451" {
		msg = "You must accept the Terms and Conditions in order to continue.\n" +
			"Go to https://www.redhat.com/wapps/tnc/ackrequired?site=ocm&event=register\n" +
			"Once you accept the terms, you will need to retry the action that was blocked."
	}
	errType := errors.ErrorType(res.Status())
	return errType.Set(errors.Errorf("%s", msg))
}

func (c *Client) GetDefaultClusterFlavors(flavour string) (dMachinecidr *net.IPNet, dPodcidr *net.IPNet,
	dServicecidr *net.IPNet, dhostPrefix int) {
	flavourGetResponse, err := c.ocm.ClustersMgmt().V1().Flavours().Flavour(flavour).Get().Send()
	if err != nil {
		flavourGetResponse, _ = c.ocm.ClustersMgmt().V1().Flavours().Flavour("osd-4").Get().Send()
	}
	network, ok := flavourGetResponse.Body().GetNetwork()
	if !ok {
		return nil, nil, nil, 0
	}
	_, dMachinecidr, err = net.ParseCIDR(network.MachineCIDR())
	if err != nil {
		dMachinecidr = nil
	}
	_, dPodcidr, err = net.ParseCIDR(network.PodCIDR())
	if err != nil {
		dPodcidr = nil
	}
	_, dServicecidr, err = net.ParseCIDR(network.ServiceCIDR())
	if err != nil {
		dServicecidr = nil
	}
	dhostPrefix, _ = network.GetHostPrefix()
	return dMachinecidr, dPodcidr, dServicecidr, dhostPrefix
}

func (c *Client) LogEvent(key string, body map[string]string) {
	event, err := cmv1.NewEvent().Key(key).Body(body).Build()
	if err == nil {
		_, _ = c.ocm.ClustersMgmt().V1().
			Events().
			Add().
			Body(event).
			Send()
	}
}

func (c *Client) GetCurrentAccount() (*amsv1.Account, error) {
	response, err := c.ocm.AccountsMgmt().V1().
		CurrentAccount().
		Get().
		Send()
	if err != nil {
		if response.Status() == http.StatusNotFound {
			return nil, nil
		}
		return nil, handleErr(response.Error(), err)
	}
	return response.Body(), nil
}

func (c *Client) GetCurrentOrganization() (id string, externalID string, err error) {
	acctResponse, err := c.GetCurrentAccount()

	if err != nil {
		return
	}
	id = acctResponse.Organization().ID()
	externalID = acctResponse.Organization().ExternalID()

	return
}

func (c *Client) IsHibernateCapabilityEnabled() error {
	organizationID, _, err := c.GetCurrentOrganization()
	if err != nil {
		return err
	}
	isCapabilityEnable, err := c.IsCapabilityEnabled(HibernateCapability, organizationID)

	if err != nil {
		return err
	}
	if !isCapabilityEnable {
		return fmt.Errorf("The '%s' capability is not set for org '%s'", HibernateCapability, organizationID)
	}
	return nil
}

func (c *Client) IsCapabilityEnabled(capabilityName string, orgID string) (bool, error) {
	capabilityResponse, err := c.ocm.AccountsMgmt().V1().Organizations().
		Organization(orgID).Get().Parameter("fetchCapabilities", true).Send()

	if err != nil {
		return false, handleErr(capabilityResponse.Error(), err)
	}
	if len(capabilityResponse.Body().Capabilities()) > 0 {
		for _, capability := range capabilityResponse.Body().Capabilities() {
			if capability.Name() == capabilityName {
				return capability.Value() == "true", nil
			}
		}
	}
	return false, nil
}

// ASCII codes of important characters:
const (
	aCode    = 97
	zCode    = 122
	zeroCode = 48
	nineCode = 57
)

// Number of letters and digits:
const (
	letterCount = zCode - aCode + 1
	digitCount  = nineCode - zeroCode + 1
)

func RandomLabel(size int) string {
	value := rand.Int() // #nosec G404
	chars := make([]byte, size)
	for size > 0 {
		size--
		if size%2 == 0 {
			chars[size] = byte(aCode + value%letterCount)
			value = value / letterCount
		} else {
			chars[size] = byte(zeroCode + value%digitCount)
			value = value / digitCount
		}
	}
	return string(chars)
}

func (c *Client) UnlinkUserRoleFromAccount(accountID string, roleARN string) error {
	linkedRoles, err := c.GetAccountLinkedUserRoles(accountID)
	if err != nil {
		return err
	}

	if helper.Contains(linkedRoles, roleARN) {
		linkedRoles = helper.RemoveStrFromSlice(linkedRoles, roleARN)

		if len(linkedRoles) > 0 {
			newRoleARN := strings.Join(linkedRoles, ",")
			label, err := amsv1.NewLabel().Key(USERRoleLabel).Value(newRoleARN).Build()
			if err != nil {
				return err
			}

			resp, err := c.ocm.AccountsMgmt().V1().Accounts().Account(accountID).Labels().
				Labels(USERRoleLabel).Update().Body(label).Send()
			if err != nil {
				return handleErr(resp.Error(), err)
			}
		} else {
			resp, err := c.ocm.AccountsMgmt().V1().Accounts().Account(accountID).Labels().
				Labels(USERRoleLabel).Delete().Send()
			if err != nil {
				return handleErr(resp.Error(), err)
			}
		}

		return nil
	}

	return errors.UserErrorf("Role ARN '%s' is not linked with the current account '%s'", roleARN, accountID)
}

func (c *Client) LinkAccountRole(accountID string, roleARN string) error {
	resp, err := c.ocm.AccountsMgmt().V1().Accounts().Account(accountID).
		Labels().Labels("sts_user_role").Get().Send()
	if err != nil && resp.Status() != 404 {
		if resp.Status() == 403 {
			return errors.Forbidden.UserErrorf("%v", err)
		}
		return handleErr(resp.Error(), err)
	}
	existingARN := resp.Body().Value()
	exists := false
	if existingARN != "" {
		existingARNArr := strings.Split(existingARN, ",")
		if len(existingARNArr) > 0 {
			for _, value := range existingARNArr {
				if value == roleARN {
					exists = true
					break
				}
			}
		}
	}
	if exists {
		return nil
	}
	if existingARN != "" {
		roleARN = existingARN + "," + roleARN
	}
	labelBuilder, err := amsv1.NewLabel().Key("sts_user_role").Value(roleARN).Build()
	if err != nil {
		return err
	}
	_, err = c.ocm.AccountsMgmt().V1().Accounts().Account(accountID).
		Labels().Add().Body(labelBuilder).Send()
	if err != nil {
		return handleErr(resp.Error(), err)
	}
	return err
}

func (c *Client) UnlinkOCMRoleFromOrg(orgID string, roleARN string) error {
	linkedRoles, err := c.GetOrganizationLinkedOCMRoles(orgID)
	if err != nil {
		return err
	}

	if helper.Contains(linkedRoles, roleARN) {
		linkedRoles = helper.RemoveStrFromSlice(linkedRoles, roleARN)

		if len(linkedRoles) > 0 {
			newRoleARN := strings.Join(linkedRoles, ",")
			label, err := amsv1.NewLabel().Key(OCMRoleLabel).Value(newRoleARN).Build()
			if err != nil {
				return err
			}

			resp, err := c.ocm.AccountsMgmt().V1().Organizations().Organization(orgID).Labels().
				Labels(OCMRoleLabel).Update().Body(label).Send()
			if err != nil {
				return handleErr(resp.Error(), err)
			}
		} else {
			resp, err := c.ocm.AccountsMgmt().V1().Organizations().Organization(orgID).Labels().
				Labels(OCMRoleLabel).Delete().Send()
			if err != nil {
				return handleErr(resp.Error(), err)
			}
		}

		return nil
	}

	return errors.UserErrorf("Role-arn '%s' is not linked with the organization account '%s'", roleARN, orgID)
}

func (c *Client) LinkOrgToRole(orgID string, roleARN string) (bool, error) {
	parsedARN, err := arn.Parse(roleARN)
	if err != nil {
		return false, err
	}
	exists, existingARN, selectedARN, err := c.CheckIfAWSAccountExists(orgID, parsedARN.AccountID)
	if err != nil {
		return false, err
	}
	if exists {
		if selectedARN != roleARN {
			return false, errors.UserErrorf("User organization '%s' has role-arn '%s' associated. "+
				"Only one role can be linked per AWS account per organization", orgID, selectedARN)
		}
		return false, nil
	}
	if existingARN != "" {
		roleARN = existingARN + "," + roleARN
	}
	labelBuilder, err := amsv1.NewLabel().Key(OCMRoleLabel).Value(roleARN).Build()
	if err != nil {
		return false, err
	}

	resp, err := c.ocm.AccountsMgmt().V1().Organizations().Organization(orgID).
		Labels().Add().Body(labelBuilder).Send()
	if err != nil {
		return false, handleErr(resp.Error(), err)
	}
	return true, nil
}

func (c *Client) GetAccountLinkedUserRoles(accountID string) ([]string, error) {
	resp, err := c.ocm.AccountsMgmt().V1().Accounts().Account(accountID).
		Labels().Labels(USERRoleLabel).Get().Send()
	if err != nil && resp.Status() != http.StatusNotFound {
		return nil, handleErr(resp.Error(), err)
	}

	return strings.Split(resp.Body().Value(), ","), nil
}

func (c *Client) GetOrganizationLinkedOCMRoles(orgID string) ([]string, error) {
	resp, err := c.ocm.AccountsMgmt().V1().Organizations().Organization(orgID).
		Labels().Labels(OCMRoleLabel).Get().Send()
	if err != nil && resp.Status() != http.StatusNotFound {
		return nil, err
	}

	return strings.Split(resp.Body().Value(), ","), nil
}

func (c *Client) CheckIfAWSAccountExists(orgID string, awsAccountID string) (bool, string, string, error) {
	resp, err := c.ocm.AccountsMgmt().V1().Organizations().Organization(orgID).
		Labels().Labels(OCMRoleLabel).Get().Send()
	if err != nil && resp.Status() != 404 {
		if resp.Status() == 403 {
			return false, "", "", errors.Forbidden.UserErrorf("%v", err)
		}
		return false, "", "", handleErr(resp.Error(), err)
	}
	existingARN := resp.Body().Value()
	exists := false
	selectedARN := ""
	if existingARN != "" {
		existingARNArr := strings.Split(existingARN, ",")
		if len(existingARNArr) > 0 {
			for _, value := range existingARNArr {
				parsedARN, err := arn.Parse(value)
				if err != nil {
					return false, "", "", err
				}
				if parsedARN.AccountID == awsAccountID {
					exists = true
					selectedARN = value
					break
				}
			}
		}
	}
	return exists, existingARN, selectedARN, nil
}

/**
We should allow only one role per aws account per organization
If the user request same ocm role we should let them proceed to ensure they can add admin role
if not exists or attach policies or link etc
if the user request diff ocm role name we error out
*/
func (c *Client) CheckRoleExists(orgID string, roleName string, awsAccountID string) (bool, string, string, error) {
	exists, _, selectedARN, err := c.CheckIfAWSAccountExists(orgID, awsAccountID)
	if err != nil {
		return false, "", "", err
	}
	if !exists {
		return false, "", "", nil
	}
	existingRole := strings.SplitN(selectedARN, "/", 2)
	if len(existingRole) > 1 && existingRole[1] == roleName {
		return false, "", "", nil
	}
	return true, existingRole[1], selectedARN, nil
}

func GetVersionMinor(ver string) string {
	rawID := strings.Replace(ver, "openshift-v", "", 1)
	version, err := semver.NewVersion(rawID)
	if err != nil {
		segments := strings.Split(rawID, ".")
		return fmt.Sprintf("%s.%s", segments[0], segments[1])
	}
	segments := version.Segments()
	return fmt.Sprintf("%d.%d", segments[0], segments[1])
}

func CheckSupportedVersion(clusterVersion string, operatorVersion string) (bool, error) {
	v1, err := semver.NewVersion(clusterVersion)
	if err != nil {
		return false, err
	}
	v2, err := semver.NewVersion(operatorVersion)
	if err != nil {
		return false, err
	}
	//Cluster version is greater than or equal to operator version
	return v1.GreaterThanOrEqual(v2), nil
}
