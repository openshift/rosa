package rosacli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openshift/rosa/tests/utils/common"
	. "github.com/openshift/rosa/tests/utils/log"
)

var RoleTypeSuffixMap = map[string]string{
	"Installer":     "Installer-Role",
	"Support":       "Support-Role",
	"Control plane": "ControlPlane-Role",
	"Worker":        "Worker-Role",
}

type AccountRolesUnit struct {
	InstallerRole    string `json:"Installer,omitempty"`
	SupportRole      string `json:"Support,omitempty"`
	WorkerRole       string `json:"Worker,omitempty"`
	ControlPlaneRole string `json:"Control plane,omitempty"`
}

type OCMResourceService interface {
	ResourcesCleaner

	List() *ocmResourceService
	Create() *ocmResourceService
	Delete() *ocmResourceService
	Link() *ocmResourceService
	Unlink() *ocmResourceService

	Whoami() (bytes.Buffer, error)
	ReflectAccountsInfo(result bytes.Buffer) *AccountsInfo
	UserInfo() (res *AccountsInfo, err error)

	CreateAccountRole(flags ...string) (bytes.Buffer, error)
	ReflectAccountRoleList(result bytes.Buffer) (arl AccountRoleList, err error)
	DeleteAccountRole(flags ...string) (bytes.Buffer, error)
	ListAccountRole() (AccountRoleList, bytes.Buffer, error)
	UpgradeAccountRole(flags ...string) (bytes.Buffer, error)

	ListOCMRole() (OCMRoleList, bytes.Buffer, error)
	DeleteOCMRole(flags ...string) (bytes.Buffer, error)
	LinkOCMRole(flags ...string) (bytes.Buffer, error)
	UnlinkOCMRole(flags ...string) (bytes.Buffer, error)
	CreateOCMRole(flags ...string) (bytes.Buffer, error)
	ReflectOCMRoleList(result bytes.Buffer) (orl OCMRoleList, err error)

	ListOIDCConfig() (OIDCConfigList, bytes.Buffer, error)
	ListInstanceTypes() (InstanceTypesList, bytes.Buffer, error)
	DeleteOIDCConfig(flags ...string) (bytes.Buffer, error)
	CreateOIDCConfig(flags ...string) (bytes.Buffer, error)
	ReflectOIDCConfigList(result bytes.Buffer) (oidclist OIDCConfigList, err error)
	GetOIDCIdFromList(providerURL string) (string, error)
	GetOIDCConfigFromList(oidcConfigID string) (OIDCConfig, error)
	ListOperatorRoles(flags ...string) (bytes.Buffer, error)
	DeleteOperatorRoles(flags ...string) (bytes.Buffer, error)
	CreateOperatorRoles(flags ...string) (bytes.Buffer, error)
	ReflectOperatorRoleList(result bytes.Buffer) (opl OperatorRoleList, err error)

	CreateOIDCProvider(flags ...string) (bytes.Buffer, error)
	DeleteOIDCProvider(flags ...string) (bytes.Buffer, error)

	Token(flags ...string) (bytes.Buffer, error)
}

type ocmResourceService struct {
	ResourcesService
	ResourceName string
}

func NewOCMResourceService(client *Client) OCMResourceService {
	return &ocmResourceService{
		ResourcesService: ResourcesService{
			client: client,
		},
	}
}

// Struct for the 'rosa list instance-types' output
type InstanceTypes struct {
	ID        string `json:"ID,omitempty"`
	CATEGORY  string `json:"CATEGORY,omitempty"`
	CPU_CORES string `json:"CPU_CORES,omitempty"`
	MEMORY    string `json:"MEMORY,omitempty"`
}
type InstanceTypesList struct {
	InstanceTypesList []InstanceTypes `json:"InstanceTypesList,omitempty"`
}

// Struct for the 'rosa list region' output
type CloudRegion struct {
	ID                  string `json:"ID,omitempty"`
	Name                string `json:"NAME,omitempty"`
	MultiAZSupported    string `json:"MULTI-AZ SUPPORT,omitempty"`
	HypershiftSupported string `json:"HOSTED-CP SUPPORT,omitempty"`
}

// Struct for the 'rosa list user-role' output
type UserRole struct {
	RoleName string `json:"ROLE NAME,omitempty"`
	RoleArn  string `json:"ROLE ARN,omitempty"`
	Linded   string `json:"LINKED,omitempty"`
}

type UserRoleList []*UserRole

// Struct for the 'rosa list ocm-role' output
type OCMRole struct {
	RoleName   string `json:"ROLE NAME,omitempty"`
	RoleArn    string `json:"ROLE ARN,omitempty"`
	Linded     string `json:"LINKED,omitempty"`
	Admin      string `json:"ADMIN,omitempty"`
	AwsManaged string `json:"AWS MANAGED,omitempty"`
}

type OCMRoleList struct {
	OCMRoleList []OCMRole `json:"OCMRoleList,omitempty"`
}
type AccountsInfo struct {
	AWSArn                    string `json:"AWS ARN,omitempty"`
	AWSAccountID              string `json:"AWS Account ID,omitempty"`
	AWSDefaultRegion          string `json:"AWS Default Region,omitempty"`
	OCMApi                    string `json:"OCM API,omitempty"`
	OCMAccountEmail           string `json:"OCM Account Email,omitempty"`
	OCMAccountID              string `json:"OCM Account ID,omitempty"`
	OCMAccountName            string `json:"OCM Account Name,omitempty"`
	OCMAccountUsername        string `json:"OCM Account Username,omitempty"`
	OCMOrganizationExternalID string `json:"OCM Organization External ID,omitempty"`
	OCMOrganizationID         string `json:"OCM Organization ID,omitempty"`
	OCMOrganizationName       string `json:"OCM Organization Name,omitempty"`
}

type AccountRole struct {
	RoleName         string `json:"ROLE NAME,omitempty"`
	RoleType         string `json:"ROLE TYPE,omitempty"`
	RoleArn          string `json:"ROLE ARN,omitempty"`
	OpenshiftVersion string `json:"OPENSHIFT VERSION,omitempty"`
	AWSManaged       string `json:"AWS Managed,omitempty"`
}
type AccountRoleList struct {
	AccountRoleList []*AccountRole `json:"AccountRoleList,omitempty"`
}
type OperatorRole struct {
	AWSManaged        string `json:"AWS Managed,omitempty"`
	ClusterID         string `json:"CLUSTER ID,omitempty"`
	InUse             string `json:"IN USE,omitempty"`
	Operatorname      string `json:"OPERATOR NAME,omitempty"`
	OperatorNamespace string `json:"OPERATOR NAMESPACE,omitempty"`
	Policies          string `json:"POLICIES,omitempty"`
	RoleName          string `json:"ROLE NAME,omitempty"`
	RoleArn           string `json:"ROLE ARN,omitempty"`
	Version           string `json:"VERSION,omitempty"`
}

type OperatorRoleList struct {
	OperatorRoleList []*OperatorRole `json:"OperatorRoleList,omitempty"`
}
type OIDCConfig struct {
	ID        string `json:"ID,omitempty"`
	Managed   string `json:"MANAGED,omitempty"`
	IssuerUrl string `json:"ISSUER URL,omitempty"`
	SecretArn string `json:"SECRET ARN,omitempty"`
}
type OIDCConfigList struct {
	OIDCConfigList []OIDCConfig `json:"OIDCConfigList,omitempty"`
}

var textResources = []string{
	"whoami",
}

func (ors *ocmResourceService) List() *ocmResourceService {
	ors.client.Runner.Cmd("list")
	return ors
}

func (ors *ocmResourceService) Create() *ocmResourceService {
	ors.client.Runner.Cmd("create")
	return ors
}

func (ors *ocmResourceService) Delete() *ocmResourceService {
	ors.client.Runner.Cmd("delete")
	return ors
}

func (ors *ocmResourceService) Link() *ocmResourceService {
	ors.client.Runner.Cmd("link")
	return ors
}

func (ors *ocmResourceService) Unlink() *ocmResourceService {
	ors.client.Runner.Cmd("unlink")
	return ors
}

func (ors *ocmResourceService) Upgrade() *ocmResourceService {
	ors.client.Runner.Cmd("upgrade")
	return ors
}

func (ors *ocmResourceService) Parameters(flags ...string) *ocmResourceService {
	ors.client.Runner.CmdFlags(flags...)
	return ors
}

func (ors *ocmResourceService) Help() (output bytes.Buffer, err error) {
	return ors.client.Runner.CmdFlags("-h").Run()
}

func (ors *ocmResourceService) Run() (output bytes.Buffer, err error) {
	return ors.client.Runner.Run()
}

func (ors *ocmResourceService) Regions() *ocmResourceService {
	ors.ResourceName = "regions"
	ors.client.Runner.AppendCmd("regions")
	return ors
}

func (ors *ocmResourceService) InstanceTypes() *ocmResourceService {
	ors.ResourceName = "instance-types"
	ors.client.Runner.AppendCmd("instance-types")
	return ors
}

func (ors *ocmResourceService) UserRole() *ocmResourceService {
	ors.ResourceName = "user-role"
	ors.client.Runner.AppendCmd("user-role")
	return ors
}

func (ors *ocmResourceService) OCMRole() *ocmResourceService {
	ors.ResourceName = "ocm-role"
	ors.client.Runner.AppendCmd("ocm-role")
	return ors
}

func (ors *ocmResourceService) AccountRoles() *ocmResourceService {
	ors.ResourceName = "account-roles"
	ors.client.Runner.AppendCmd("account-roles")
	return ors
}

func (ors *ocmResourceService) OIDCConfig() *ocmResourceService {
	ors.ResourceName = "oidc-config"
	ors.client.Runner.AppendCmd("oidc-config")
	return ors
}

func (ors *ocmResourceService) OperatorRole() *ocmResourceService {
	ors.ResourceName = "operator-roles"
	ors.client.Runner.AppendCmd("operator-roles")
	return ors
}

func (ors *ocmResourceService) OIDCProvider() *ocmResourceService {
	ors.ResourceName = "oidc-provider"
	ors.client.Runner.AppendCmd("oidc-provider")
	return ors
}

func (ors *ocmResourceService) ToStruct() (interface{}, error) {
	resourceName := ors.ResourceName
	if common.SliceContains(textResources, resourceName) {
		ors.client.Runner.JsonFormat()
		defer ors.client.Runner.UnsetFormat()
	}

	result, err := ors.client.Runner.Run()
	if err != nil {
		return nil, err
	}

	if common.SliceContains(textResources, resourceName) {
		jsonData := ors.client.Parser.JsonData.Input(result).Parse()
		switch resourceName {
		case "whoami":
			s := new(AccountsInfo)
			data, _ := json.Marshal(jsonData.Output())
			err = json.Unmarshal(data, s)
			return s, err
		}
	} else {
		theMap := ors.client.Parser.TableData.Input(result).Parse().Output()
		switch resourceName {
		case "regions":
			var regionList []*CloudRegion
			for _, regionItem := range theMap {
				s := &CloudRegion{}
				err = MapStructure(regionItem, s)
				if err != nil {
					return nil, err
				}
				regionList = append(regionList, s)
			}
			return regionList, nil
		case "instance-types":
			var instanceTypeList []*InstanceTypes
			for _, instanceTypeItem := range theMap {
				s := &InstanceTypes{}
				err = MapStructure(instanceTypeItem, s)
				if err != nil {
					return nil, err
				}
				instanceTypeList = append(instanceTypeList, s)
			}
			return instanceTypeList, nil
		case "ocm-role":
			var ocmRoleList []*OCMRole
			for _, ocmRoleItem := range theMap {
				s := &OCMRole{}
				err = MapStructure(ocmRoleItem, s)
				if err != nil {
					return nil, err
				}
				ocmRoleList = append(ocmRoleList, s)
			}
			return ocmRoleList, nil
		case "user-role":
			var userRoleList UserRoleList
			for _, userroleItem := range theMap {
				s := &UserRole{}
				err = MapStructure(userroleItem, s)
				if err != nil {
					return nil, err
				}
				userRoleList = append(userRoleList, s)
			}
			return userRoleList, nil
		case "account-roles":
			var accountRoleList []*AccountRole
			for _, accountRoleItem := range theMap {
				s := &AccountRole{}
				err = MapStructure(accountRoleItem, s)
				if err != nil {
					return nil, err
				}
				accountRoleList = append(accountRoleList, s)
			}
			return accountRoleList, nil
		case "oidc-config":
			var oidcConfigList []*OIDCConfig
			for _, oidcConfigItem := range theMap {
				s := &OIDCConfig{}
				err = MapStructure(oidcConfigItem, s)
				if err != nil {
					return nil, err
				}
				oidcConfigList = append(oidcConfigList, s)
			}
			return oidcConfigList, nil
		case "operator-roles":
			var operatorRoleList []*OperatorRole
			for _, operatorRoleItem := range theMap {
				s := &OperatorRole{}
				err = MapStructure(operatorRoleItem, s)
				if err != nil {
					return nil, err
				}
				operatorRoleList = append(operatorRoleList, s)
			}
			return operatorRoleList, nil
		}
	}

	return nil, err
}

// Pasrse the result of 'rosa list instance-types' to InstanceTypes struct
func (ors *ocmResourceService) ReflectInstanceTypesList(result bytes.Buffer) (url InstanceTypesList, err error) {
	url = InstanceTypesList{}
	theMap := ors.client.Parser.TableData.Input(result).Parse().Output()
	for _, instanceTypeItem := range theMap {
		ur := &InstanceTypes{}
		err = MapStructure(instanceTypeItem, ur)
		if err != nil {
			return
		}
		url.InstanceTypesList = append(url.InstanceTypesList, *ur)
	}
	return
}

// ListInstanceTypes implements OCMResourceService.
func (ors *ocmResourceService) ListInstanceTypes() (InstanceTypesList, bytes.Buffer, error) {
	ors.client.Runner.cmdArgs = []string{}
	listInstanceTypes := ors.client.Runner.
		Cmd("list", "instance-types")
	output, err := listInstanceTypes.Run()
	if err != nil {
		return InstanceTypesList{}, output, err
	}
	instanceList, err := ors.ReflectInstanceTypesList(output)
	return instanceList, output, err
}

// List region
func (ors *ocmResourceService) ListRegion(flags ...string) ([]*CloudRegion, bytes.Buffer, error) {
	listRegion := ors.client.Runner
	listRegion = listRegion.Cmd("list", "regions").CmdFlags(flags...)
	output, err := listRegion.Run()
	if err != nil {
		return []*CloudRegion{}, output, err
	}
	rList, err := ors.ReflectRegionList(output)
	return rList, output, err
}

// Pasrse the result of 'rosa regions' to the RegionInfo struct
func (ors *ocmResourceService) ReflectRegionList(result bytes.Buffer) (regions []*CloudRegion, err error) {
	theMap := ors.client.Parser.TableData.Input(result).Parse().Output()
	for _, regionItem := range theMap {
		region := &CloudRegion{}
		err = MapStructure(regionItem, region)
		if err != nil {
			return
		}
		regions = append(regions, region)
	}
	return
}

// Pasrse the result of 'rosa whoami' to the AccountsInfo struct
func (ors *ocmResourceService) ReflectAccountsInfo(result bytes.Buffer) *AccountsInfo {
	res := new(AccountsInfo)
	theMap, _ := ors.client.Parser.TextData.Input(result).Parse().JsonToMap()
	data, _ := json.Marshal(&theMap)
	json.Unmarshal(data, res)
	return res
}

func (ors *ocmResourceService) UserInfo() (res *AccountsInfo, err error) {
	output, err := ors.Whoami()
	if err != nil {
		return
	}
	return ors.ReflectAccountsInfo(output), err
}

// run `rosa whoami` command
func (ors *ocmResourceService) Whoami() (bytes.Buffer, error) {
	ors.client.Runner.cmdArgs = []string{}
	whoami := ors.client.Runner.Cmd("whoami")
	return whoami.Run()
}

// Get specified user-role by user-role prefix and ocmAccountUsername
func (url UserRoleList) UserRole(prefix string, ocmAccountUsername string) (userRoles *UserRole) {
	userRoleName := fmt.Sprintf("%s-User-%s-Role", prefix, ocmAccountUsername)
	for _, roleItme := range url {
		if roleItme.RoleName == userRoleName {
			Logger.Infof("Find the userRole %s ~", userRoleName)
			return roleItme
		}
	}
	return
}

// run `rosa create account-roles` command
func (ors *ocmResourceService) CreateAccountRole(flags ...string) (bytes.Buffer, error) {
	createAccountRole := ors.client.Runner
	createAccountRole = createAccountRole.Cmd("create", "account-roles").CmdFlags(flags...)
	return createAccountRole.Run()
}

// Pasrse the result of 'rosa list account-roles' to AccountRoleList struct
func (ors *ocmResourceService) ReflectAccountRoleList(result bytes.Buffer) (arl AccountRoleList, err error) {
	arl = AccountRoleList{}
	theMap := ors.client.Parser.TableData.Input(result).Parse().Output()
	for _, accountRoleItem := range theMap {
		ar := &AccountRole{}
		err = MapStructure(accountRoleItem, ar)
		if err != nil {
			return
		}
		arl.AccountRoleList = append(arl.AccountRoleList, ar)
	}
	return
}

// run `rosa delete account-roles` command
func (ors *ocmResourceService) DeleteAccountRole(flags ...string) (bytes.Buffer, error) {
	deleteAccountRole := ors.client.Runner
	deleteAccountRole = deleteAccountRole.Cmd("delete", "account-roles").CmdFlags(flags...)
	return deleteAccountRole.Run()
}

// run `rosa list account-roles` command
func (ors *ocmResourceService) ListAccountRole() (AccountRoleList, bytes.Buffer, error) {
	ors.client.Runner.cmdArgs = []string{}
	listAccountRole := ors.client.Runner.
		Cmd("list", "account-roles")
	output, err := listAccountRole.Run()
	if err != nil {
		return AccountRoleList{}, output, err
	}
	arl, err := ors.ReflectAccountRoleList(output)
	return arl, output, err

}

// Get specified account roles by prefix
func (arl AccountRoleList) AccountRoles(prefix string) (accountRoles []*AccountRole) {
	for _, roleItme := range arl.AccountRoleList {
		if strings.Contains(roleItme.RoleName, prefix) {
			accountRoles = append(accountRoles, roleItme)
		}
	}
	return
}

// Get all specified account roles by prefix and classic
func (arl AccountRoleList) DigAccountRoles(prefix string, hcp bool) *AccountRolesUnit {
	var accRoles *AccountRolesUnit = new(AccountRolesUnit)
	roleMap := map[string]interface{}{}
	matchedAccountRoles := arl.AccountRoles(prefix)
	for _, role := range matchedAccountRoles {
		if hcp && strings.Contains(role.RoleName, "HCP") {
			roleMap[role.RoleType] = role.RoleArn
			continue
		}

		if !strings.Contains(role.RoleName, "HCP") {
			roleMap[role.RoleType] = role.RoleArn
			continue
		}

	}
	MapStructure(roleMap, accRoles)
	return accRoles
}

// Get specified account role by the arn
func (arl AccountRoleList) AccountRole(arn string) (accountRole *AccountRole) {
	for _, roleItem := range arl.AccountRoleList {
		if roleItem.RoleArn == arn {
			return roleItem
		}
	}
	return
}

// run `rosa upgrade account-roles` command
func (ors *ocmResourceService) UpgradeAccountRole(flags ...string) (bytes.Buffer, error) {
	upgradeAccountRole := ors.client.Runner
	upgradeAccountRole = upgradeAccountRole.Cmd("upgrade", "account-roles").CmdFlags(flags...)
	return upgradeAccountRole.Run()
}

func (arl AccountRoleList) InstallerRole(prefix string, hostedcp bool) (accountRole *AccountRole) {
	roleType := RoleTypeSuffixMap["Installer"]
	if hostedcp {
		roleType = "HCP-ROSA-" + roleType
	}
	for _, roleItem := range arl.AccountRoleList {
		// if hostedcp && strings.Contains(lines[i], "-HCP-ROSA-Installer-Role") {
		// 	return lines[i], nil
		// }
		// if !hostedcp && !strings.Contains(lines[i], "-ROSA-Installer-Role") && strings.Contains(lines[i], "-Installer-Role") {
		// 	return lines[i], nil
		// }
		if hostedcp && strings.Contains(roleItem.RoleName, prefix) && strings.Contains(roleItem.RoleName, roleType) {
			return roleItem
		}
		if !hostedcp && strings.Contains(roleItem.RoleName, prefix) && strings.Contains(roleItem.RoleName, roleType) && !strings.Contains(roleItem.RoleName, "HCP-ROSA-") {
			return roleItem
		}
	}
	return
}

// run `rosa create ocm-role` command
func (ors *ocmResourceService) CreateOCMRole(flags ...string) (bytes.Buffer, error) {
	createOCMRole := ors.client.Runner
	createOCMRole = createOCMRole.Cmd("create", "ocm-role").CmdFlags(flags...)
	return createOCMRole.Run()
}

// run `rosa list ocm-role` command
func (ors *ocmResourceService) ListOCMRole() (OCMRoleList, bytes.Buffer, error) {
	ors.client.Runner.cmdArgs = []string{}
	listOCMRole := ors.client.Runner.
		Cmd("list", "ocm-role")
	output, err := listOCMRole.Run()
	if err != nil {
		return OCMRoleList{}, output, err
	}
	orl, err := ors.ReflectOCMRoleList(output)
	return orl, output, err
}

// run `rosa delete ocm-role` command
func (ors *ocmResourceService) DeleteOCMRole(flags ...string) (bytes.Buffer, error) {
	deleteOCMRole := ors.client.Runner
	deleteOCMRole = deleteOCMRole.Cmd("delete", "ocm-role").CmdFlags(flags...)
	return deleteOCMRole.Run()
}

// run `rosa link ocm-role` command
func (ors *ocmResourceService) LinkOCMRole(flags ...string) (bytes.Buffer, error) {
	linkOCMRole := ors.client.Runner
	linkOCMRole = linkOCMRole.Cmd("link", "ocm-role").CmdFlags(flags...)
	return linkOCMRole.Run()
}

// run `rosa unlink ocm-role` command
func (ors *ocmResourceService) UnlinkOCMRole(flags ...string) (bytes.Buffer, error) {
	unlinkOCMRole := ors.client.Runner
	unlinkOCMRole = unlinkOCMRole.Cmd("unlink", "ocm-role").CmdFlags(flags...)
	return unlinkOCMRole.Run()
}

// Pasrse the result of 'rosa list user-ocm' to NodePoolList struct
func (ors *ocmResourceService) ReflectOCMRoleList(result bytes.Buffer) (orl OCMRoleList, err error) {
	orl = OCMRoleList{}
	theMap := ors.client.Parser.TableData.Input(result).Parse().Output()
	for _, ocmRoleItem := range theMap {
		or := &OCMRole{}
		err = MapStructure(ocmRoleItem, or)
		if err != nil {
			return
		}
		orl.OCMRoleList = append(orl.OCMRoleList, *or)
	}
	return
}

// Get specified ocm-role by ocm-role prefix and ocmOUsername
func (url OCMRoleList) OCMRole(prefix string, ocmOrganizationExternalID string) (userRoles OCMRole) {
	ocmRoleName := fmt.Sprintf("%s-OCM-Role-%s", prefix, ocmOrganizationExternalID)
	for _, roleItme := range url.OCMRoleList {
		if roleItme.RoleName == ocmRoleName {
			Logger.Infof("Find the ocm Role %s ~", ocmRoleName)
			return roleItme
		}
	}
	return
}

// Get the ocm-role which is linked to org
func (url OCMRoleList) FindLinkedOCMRole() (userRoles OCMRole) {
	for _, roleItme := range url.OCMRoleList {
		if roleItme.Linded == "Yes" {
			Logger.Infof("Find one linked ocm Role %s ~", roleItme.RoleName)
			return roleItme
		}
	}
	return
}

// run `rosa create oidc-config` command
func (ors *ocmResourceService) CreateOIDCConfig(flags ...string) (bytes.Buffer, error) {
	createOIDCConfig := ors.client.Runner
	createOIDCConfig = createOIDCConfig.Cmd("create", "oidc-config").CmdFlags(flags...)
	return createOIDCConfig.Run()
}

// run `rosa list oidc-config` command
func (ors *ocmResourceService) ListOIDCConfig() (OIDCConfigList, bytes.Buffer, error) {
	ors.client.Runner.cmdArgs = []string{}
	listOIDCConfig := ors.client.Runner.
		Cmd("list", "oidc-config")
	output, err := listOIDCConfig.Run()
	if err != nil {
		return OIDCConfigList{}, output, err
	}
	oidcl, err := ors.ReflectOIDCConfigList(output)
	return oidcl, output, err

}

// run `rosa delete oidc-config` command
func (ors *ocmResourceService) DeleteOIDCConfig(flags ...string) (bytes.Buffer, error) {
	deleteOIDCConfig := ors.client.Runner
	deleteOIDCConfig = deleteOIDCConfig.Cmd("delete", "oidc-config").CmdFlags(flags...)
	return deleteOIDCConfig.Run()
}

// Pasrse the result of 'rosa list oidc-config' to OIDCConfigList struct
func (ors *ocmResourceService) ReflectOIDCConfigList(result bytes.Buffer) (oidcl OIDCConfigList, err error) {
	oidcl = OIDCConfigList{}
	theMap := ors.client.Parser.TableData.Input(result).Parse().Output()
	for _, oidcConfigItem := range theMap {
		oidc := &OIDCConfig{}
		err = MapStructure(oidcConfigItem, oidc)
		if err != nil {
			return
		}
		oidcl.OIDCConfigList = append(oidcl.OIDCConfigList, *oidc)
	}
	return
}

// Get the oidc id by the provider url
func (ors *ocmResourceService) GetOIDCIdFromList(providerURL string) (string, error) {
	oidcConfigList, _, err := ors.ListOIDCConfig()
	if err != nil {
		return "", err
	}
	for _, item := range oidcConfigList.OIDCConfigList {
		if strings.Contains(item.IssuerUrl, providerURL) {
			return item.ID, nil
		}
	}
	Logger.Warnf("No oidc with the url %s is found.", providerURL)
	return "", nil
}
func (ors *ocmResourceService) GetOIDCConfigFromList(oidcConfigID string) (OIDCConfig, error) {
	var expectedOIDC OIDCConfig
	oidcConfigList, _, err := ors.ListOIDCConfig()
	if err != nil {
		return expectedOIDC, err
	}
	for _, item := range oidcConfigList.OIDCConfigList {
		if strings.EqualFold(item.ID, oidcConfigID) {
			expectedOIDC = item
			break
		}
	}
	return expectedOIDC, nil
}

// Get specified oidc-config by oidc-config-id
func (oidcl OIDCConfigList) OIDCConfig(id string) (oidc OIDCConfig) {
	for _, item := range oidcl.OIDCConfigList {
		if item.ID == id {
			return item
		}
	}
	return
}

// run `rosa create operator-roles` command
func (ors *ocmResourceService) CreateOperatorRoles(flags ...string) (bytes.Buffer, error) {
	createOperatorRoles := ors.client.Runner
	createOperatorRoles = createOperatorRoles.Cmd("create", "operator-roles").CmdFlags(flags...)
	return createOperatorRoles.Run()
}

// run `rosa delete operator-roles` command
func (ors *ocmResourceService) DeleteOperatorRoles(flags ...string) (bytes.Buffer, error) {
	deleteOperatorRoles := ors.client.Runner
	deleteOperatorRoles = deleteOperatorRoles.Cmd("delete", "operator-roles").CmdFlags(flags...)
	return deleteOperatorRoles.Run()
}

// run `rosa list operator-roles`
func (ors *ocmResourceService) ListOperatorRoles(flags ...string) (bytes.Buffer, error) {
	listOperatorRoles := ors.client.Runner
	listOperatorRoles = listOperatorRoles.Cmd("list", "operator-roles").CmdFlags(flags...)
	return listOperatorRoles.Run()
}

// Pasrse the result of 'rosa list operator-roles' to OperatorRoleList struct
func (ors *ocmResourceService) ReflectOperatorRoleList(result bytes.Buffer) (opl OperatorRoleList, err error) {
	opl = OperatorRoleList{}
	theMap := ors.client.Parser.TableData.Input(result).Parse().Output()
	for _, accountRoleItem := range theMap {
		opr := &OperatorRole{}
		err = MapStructure(accountRoleItem, opr)
		if err != nil {
			return
		}
		opl.OperatorRoleList = append(opl.OperatorRoleList, opr)
	}
	return
}

// run `rosa create oidc-provider` command
func (ors *ocmResourceService) CreateOIDCProvider(flags ...string) (bytes.Buffer, error) {
	createOIDCProvider := ors.client.Runner
	createOIDCProvider = createOIDCProvider.Cmd("create", "oidc-provider").CmdFlags(flags...)
	return createOIDCProvider.Run()
}

// run `rosa delete oidc-provider` command
func (ors *ocmResourceService) DeleteOIDCProvider(flags ...string) (bytes.Buffer, error) {
	deleteOIDCProvider := ors.client.Runner
	deleteOIDCProvider = deleteOIDCProvider.Cmd("delete", "oidc-provider").CmdFlags(flags...)
	return deleteOIDCProvider.Run()
}

func (ors *ocmResourceService) CleanResources(clusterID string) (errors []error) {
	Logger.Debugf("Nothing releated to cluster was done there")
	return
}

func (ors *ocmResourceService) Token(flags ...string) (bytes.Buffer, error) {
	token := ors.client.Runner
	token = token.Cmd("token").CmdFlags(flags...)
	return token.Run()
}
