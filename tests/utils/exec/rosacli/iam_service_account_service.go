package rosacli

import (
	"bytes"
)

type IAMServiceAccountService interface {
	CreateIAMServiceAccountRole(flags ...string) (bytes.Buffer, error)
	DeleteIAMServiceAccountRole(flags ...string) (bytes.Buffer, error)
	ListIAMServiceAccountRoles(flags ...string) (bytes.Buffer, error)
	DescribeIAMServiceAccountRole(flags ...string) (bytes.Buffer, error)
}

type iamServiceAccountService struct {
	ResourcesService
}

func (i *iamServiceAccountService) CreateIAMServiceAccountRole(flags ...string) (bytes.Buffer, error) {
	createIAMServiceAccount := append([]string{"create", "iamserviceaccount"}, flags...)
	return i.client.Runner.RunCMD(createIAMServiceAccount)
}

func (i *iamServiceAccountService) DeleteIAMServiceAccountRole(flags ...string) (bytes.Buffer, error) {
	deleteIAMServiceAccount := append([]string{"delete", "iamserviceaccount"}, flags...)
	return i.client.Runner.RunCMD(deleteIAMServiceAccount)
}

func (i *iamServiceAccountService) ListIAMServiceAccountRoles(flags ...string) (bytes.Buffer, error) {
	listIAMServiceAccounts := append([]string{"list", "iamserviceaccounts"}, flags...)
	return i.client.Runner.RunCMD(listIAMServiceAccounts)
}

func (i *iamServiceAccountService) DescribeIAMServiceAccountRole(flags ...string) (bytes.Buffer, error) {
	describeIAMServiceAccount := append([]string{"describe", "iamserviceaccount"}, flags...)
	return i.client.Runner.RunCMD(describeIAMServiceAccount)
}

func NewIAMServiceAccountService(client *Client) IAMServiceAccountService {
	return &iamServiceAccountService{
		ResourcesService{
			client: client,
		},
	}
}
