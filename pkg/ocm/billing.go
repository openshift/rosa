package ocm

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/openshift/rosa/pkg/helper"
)

var ROSAHypershiftQuota = "cluster|byoc|moa|marketplace"

var awsAccountRegexp = regexp.MustCompile(`^[0-9]{12}$`)

func (c *Client) GetBillingAccounts() ([]string, error) {
	acctResponse, err := c.ocm.AccountsMgmt().V1().CurrentAccount().
		Get().
		Send()
	if err != nil {
		return nil, handleErr(acctResponse.Error(), err)
	}
	organization := acctResponse.Body().Organization().ID()
	search := fmt.Sprintf("quota_id='%s'", ROSAHypershiftQuota)
	quotaCostResponse, err := c.ocm.AccountsMgmt().V1().Organizations().
		Organization(organization).
		QuotaCost().
		List().
		Parameter("fetchCloudAccounts", true).
		Parameter("search", search).
		Page(1).
		Size(-1).
		Send()
	if err != nil {
		return nil, handleErr(quotaCostResponse.Error(), err)
	}

	var billingAccounts []string
	for _, item := range quotaCostResponse.Items().Slice() {
		for _, account := range item.CloudAccounts() {
			if account.CloudProviderID() == "aws" && !helper.Contains(billingAccounts, account.CloudAccountID()) {
				billingAccounts = append(billingAccounts, account.CloudAccountID())
			}
		}
	}

	if len(billingAccounts) == 0 {
		return billingAccounts, errors.New("no billing accounts found")
	}

	return billingAccounts, nil
}

func IsValidAWSAccount(account string) bool {
	return awsAccountRegexp.MatchString(account)
}
