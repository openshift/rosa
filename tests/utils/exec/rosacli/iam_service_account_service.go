package rosacli

import (
	"bytes"
)

type IAMServiceAccountService interface {
	CreateIAMServiceAccountRole(flags ...string) (bytes.Buffer, error)
	DeleteIAMServiceAccountRole(flags ...string) (bytes.Buffer, error)
	ListIAMServiceAccountRoles(flags ...string) (bytes.Buffer, error)
	DescribeIAMServiceAccountRole(flags ...string) (bytes.Buffer, error)
	ReflectIamServiceAccountList(result bytes.Buffer) (isasl IamServiceAccountList, err error)
}

type iamServiceAccountService struct {
	ResourcesService

	iamServiceAccountRole map[string][]string
}

// Struct for the 'rosa list machinepool' output for non-hosted-cp clusters
type IamServiceAccountRole struct {
	Name           string `json:"NAME,omitempty"`
	Arn            string `json:"ARN,omitempty"`
	Cluster        string `json:"CLUSTER,omitempty"`
	Namespace      string `json:"NAMESPACE,omitempty"`
	ServiceAccount string `json:"SERVICE ACCOUNT,omitempty"`
	Create         string `json:"CREATED,omitempty"`
}
type IamServiceAccountList struct {
	IamServiceAccountRoles []*IamServiceAccountRole `json:"IamServiceAccountRoles,omitempty"`
}

func (i *iamServiceAccountService) CreateIAMServiceAccountRole(flags ...string) (bytes.Buffer, error) {
	return i.client.Runner.
		Cmd("create", "iamserviceaccount").
		CmdFlags(flags...).
		Run()
}

func (i *iamServiceAccountService) DeleteIAMServiceAccountRole(flags ...string) (bytes.Buffer, error) {
	return i.client.Runner.
		Cmd("delete", "iamserviceaccount").
		CmdFlags(flags...).
		Run()
}

func (i *iamServiceAccountService) ListIAMServiceAccountRoles(flags ...string) (bytes.Buffer, error) {
	return i.client.Runner.
		Cmd("list", "iamserviceaccount").
		CmdFlags(flags...).
		Run()
}

func (i *iamServiceAccountService) DescribeIAMServiceAccountRole(flags ...string) (bytes.Buffer, error) {
	return i.client.Runner.
		Cmd("describe", "iamserviceaccount").
		CmdFlags(flags...).
		Run()
}

func NewIAMServiceAccountService(client *Client) IAMServiceAccountService {
	return &iamServiceAccountService{
		ResourcesService: ResourcesService{
			client: client,
		},
		iamServiceAccountRole: make(map[string][]string),
	}
}

// Pasrse the result of 'rosa list iamserviceaccount' to IamServiceAccountList struct
func (i *iamServiceAccountService) ListAndReflectIamServiceAccountRoles(clusterID string) (isasl IamServiceAccountList, err error) {
	isasl = IamServiceAccountList{}
	output, err := i.ListIAMServiceAccountRoles(clusterID)
	if err != nil {
		return isasl, err
	}

	isasl, err = i.ReflectIamServiceAccountList(output)
	return isasl, err
}

// Pasrse the result of 'rosa list machinepool' to MachinePoolList struct
func (i *iamServiceAccountService) ReflectIamServiceAccountList(result bytes.Buffer) (isasl IamServiceAccountList, err error) {
	isasl = IamServiceAccountList{}
	theMap := i.client.Parser.TableData.Input(result).Parse().Output()
	for _, osarItem := range theMap {
		isar := &IamServiceAccountRole{}
		err = MapStructure(osarItem, isar)
		if err != nil {
			return
		}
		isasl.IamServiceAccountRoles = append(isasl.IamServiceAccountRoles, isar)
	}
	return isasl, err
}

func (isal IamServiceAccountList) GetIAMServiceAccountRoleByName(roleName string) (isar IamServiceAccountRole) {
	for _, v := range isal.IamServiceAccountRoles {
		if v.Name == roleName {
			return *v
		}
	}
	return
}
