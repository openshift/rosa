package ocm

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/openshift/rosa/pkg/helper"
)

var ROSAHypershiftQuota = "cluster|byoc|moa|marketplace"

var awsAccountRegexp = regexp.MustCompile(`^[0-9]{12}$`)

func (c *Client) GetBillingAccounts() ([]*v1.CloudAccount, error) {
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

	var billingAccounts []*v1.CloudAccount
	for _, item := range quotaCostResponse.Items().Slice() {
		billingAccounts = append(billingAccounts, item.CloudAccounts()...)
	}

	if len(billingAccounts) == 0 {
		return billingAccounts, errors.New("No billing account associated. " +
			"Go to https://docs.openshift.com/rosa/rosa_architecture/rosa-understanding.html" +
			" to learn how to associate your billing account." +
			" You must associate a billing account to continue.")
	}

	return billingAccounts, nil
}

func GenerateBillingAccountsList(cloudAccounts []*v1.CloudAccount) []string {
	var billingAccounts []string
	for _, account := range cloudAccounts {
		if account.CloudProviderID() == "aws" && !helper.ContainsPrefix(billingAccounts, account.CloudAccountID()) {
			var contractString string
			if HasValidContracts(account) {
				contractString = " [Contract enabled]"
			}
			contractEnabledBillingAccount := account.CloudAccountID() + contractString
			billingAccounts = append(billingAccounts, contractEnabledBillingAccount)
		}
	}
	return billingAccounts

}

func GetNumsOfVCPUsAndClusters(dimensions []*v1.ContractDimension) (int, int) {
	numOfVCPUs := 0
	numOfClusters := 0
	for _, dimension := range dimensions {
		switch dimension.Name() {
		case "four_vcpu_hour":
			numOfVCPUs, _ = strconv.Atoi(dimension.Value())
		case "control_plane":
			numOfClusters, _ = strconv.Atoi(dimension.Value())
		}
	}
	return numOfVCPUs, numOfClusters
}

func HasValidContracts(cloudAccount *v1.CloudAccount) bool {
	//currently, an AWS account will have only one ROSA HCP active contract at a time
	contract := cloudAccount.Contracts()[0]
	numberOfVCPUs, numberOfClusters := GetNumsOfVCPUsAndClusters(contract.Dimensions())
	return numberOfVCPUs > 0 ||
		numberOfClusters > 0
}

func IsValidAWSAccount(account string) bool {
	return awsAccountRegexp.MatchString(account)
}
