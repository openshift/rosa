package rosacli

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/tests/utils/log"
)

type AccountRoleService interface {
	Create(flags ...string) (bytes.Buffer, error)
	List(flags ...string) (bytes.Buffer, error)
	Upgrade(flags ...string) (bytes.Buffer, error)
	Delete(flags ...string) (bytes.Buffer, error)
}
type OCMRoleService interface {
	Create(flags ...string) (bytes.Buffer, error)
	List(flags ...string) (bytes.Buffer, error)
	Link(flags ...string) (bytes.Buffer, error)
	UnLink(flags ...string) (bytes.Buffer, error)
	Delete(flags ...string) (bytes.Buffer, error)
}
type UserRoleService interface {
	Create(flags ...string) (bytes.Buffer, error)
	List(flags ...string) (bytes.Buffer, error)
	Delete(flags ...string) (bytes.Buffer, error)
	Link(flags ...string) (bytes.Buffer, error)
	UnLink(flags ...string) (bytes.Buffer, error)
}
type OperatorRoleService interface {
	Create(flags ...string) (bytes.Buffer, error)
	List(flags ...string) (bytes.Buffer, error)
	Upgrade(flags ...string) (bytes.Buffer, error)
	Delete(flags ...string) (bytes.Buffer, error)
}

type accountRoleService struct {
	ResourcesService
}
type userRoleService struct {
	ResourcesService
}
type ocmRoleService struct {
	ResourcesService
}
type operatorRoleService struct {
	ResourcesService
}

func NewAccountRoleService(client *Client) AccountRoleService {
	return &accountRoleService{
		ResourcesService: ResourcesService{
			client: client,
		},
	}
}

func NewOCMRoleService(client *Client) OCMRoleService {
	return &ocmRoleService{
		ResourcesService: ResourcesService{
			client: client,
		},
	}
}

func NewUserRoleService(client *Client) UserRoleService {
	return &userRoleService{
		ResourcesService: ResourcesService{
			client: client,
		},
	}
}

func NewOperatorRoleService(client *Client) OperatorRoleService {
	return &operatorRoleService{
		ResourcesService: ResourcesService{
			client: client,
		},
	}
}

// ************ Account role service ************
// Struct for account role creation
type AccountRoles struct {
	ControPlaneRole string `json:"Control plane: omitempty"`
	WorkRole        string `json:"Worker: omitempty"`
	SupportRole     string `json:"Support: omitempty"`
	InstallerRole   string `json:"Installer: omitempty"`
}

func (accRole *accountRoleService) Create(flags ...string) (bytes.Buffer, error) {
	create := accRole.client.Runner.Cmd(
		"create", "account-roles",
	).
		CmdFlags(flags...)
	return create.Run()
}

// ReflectAccountRoleCreationResult will generate created roles to struct by prefix and cluster type
// output should be output of accountRoleService.List
func GenerateAccountRoles(input bytes.Buffer, rolePrefix string, hcp bool) *AccountRoles {
	if rolePrefix == "" {
		rolePrefix = aws.DefaultPrefix
	}
	roleCreated := &AccountRoles{}
	roleRegexp := regexp.MustCompile(fmt.Sprintf(`%s-((?!HCP))(Installer|Support|Worker|ControlPlane)+-Role`, rolePrefix))
	if hcp {
		roleRegexp = regexp.MustCompile(fmt.Sprintf(`%s-HCP-ROSA-(Installer|Support|Worker|ControlPlane)+-Role`, rolePrefix))
	}
	parser := NewParser()
	parser.JsonData.input = input
	output := parser.JsonData.Parse().output
	if output == nil {
		log.Logger.Warn("Didn't get any account roles listed in the result")
		return roleCreated
	}
	parsedResult := output.([]interface{})
	roleMap := map[string]interface{}{}
	for index, _ := range parsedResult {
		roleName := parser.JsonData.DigString(index, "RoleName")
		if roleRegexp.MatchString(roleName) {
			roleMap[parser.JsonData.DigString(index, "RoleType")] = parser.JsonData.DigString(index, "RoleARN")
		}
	}
	err := MapStructure(roleMap, roleCreated)
	if err != nil {
		log.Logger.Error(err.Error())
	}

	return roleCreated
}
func (accRole *accountRoleService) List(flags ...string) (bytes.Buffer, error) {
	create := accRole.client.Runner.Cmd(
		"list", "account-roles",
	).
		CmdFlags(flags...)
	return create.Run()
}
func (accRole *accountRoleService) Delete(flags ...string) (bytes.Buffer, error) {
	create := accRole.client.Runner.Cmd(
		"delete", "account-roles",
	).
		CmdFlags(flags...)
	return create.Run()
}
func (accRole *accountRoleService) Upgrade(flags ...string) (bytes.Buffer, error) {
	create := accRole.client.Runner.Cmd(
		"upgrade", "account-roles",
	).
		CmdFlags(flags...)
	return create.Run()
}

// ************ OCM role service ************
func (ocmRole *ocmRoleService) Create(flags ...string) (bytes.Buffer, error) {
	create := ocmRole.client.Runner.Cmd(
		"create", "ocm-role",
	).
		CmdFlags(flags...)
	return create.Run()
}
func (ocmRole *ocmRoleService) List(flags ...string) (bytes.Buffer, error) {
	create := ocmRole.client.Runner.Cmd(
		"list", "ocm-role",
	).
		CmdFlags(flags...)
	return create.Run()
}
func (ocmRole *ocmRoleService) Delete(flags ...string) (bytes.Buffer, error) {
	create := ocmRole.client.Runner.Cmd(
		"delete", "ocm-role",
	).
		CmdFlags(flags...)
	return create.Run()
}
func (ocmRole *ocmRoleService) Link(flags ...string) (bytes.Buffer, error) {
	create := ocmRole.client.Runner.Cmd(
		"link", "ocm-role",
	).
		CmdFlags(flags...)
	return create.Run()
}

func (ocmRole *ocmRoleService) UnLink(flags ...string) (bytes.Buffer, error) {
	create := ocmRole.client.Runner.Cmd(
		"unlink", "ocm-role",
	).
		CmdFlags(flags...)
	return create.Run()
}

// *********** User role service *****************
func (userRole *userRoleService) Create(flags ...string) (bytes.Buffer, error) {
	create := userRole.client.Runner.Cmd(
		"create", "user-role",
	).
		CmdFlags(flags...)
	return create.Run()
}
func (userRole *userRoleService) List(flags ...string) (bytes.Buffer, error) {
	create := userRole.client.Runner.Cmd(
		"list", "user-role",
	).
		CmdFlags(flags...)
	return create.Run()
}
func (userRole *userRoleService) Delete(flags ...string) (bytes.Buffer, error) {
	create := userRole.client.Runner.Cmd(
		"delete", "user-role",
	).
		CmdFlags(flags...)
	return create.Run()
}
func (userRole *userRoleService) Link(flags ...string) (bytes.Buffer, error) {
	create := userRole.client.Runner.Cmd(
		"link", "user-role",
	).
		CmdFlags(flags...)
	return create.Run()
}

func (userRole *userRoleService) UnLink(flags ...string) (bytes.Buffer, error) {
	create := userRole.client.Runner.Cmd(
		"unlink", "user-role",
	).
		CmdFlags(flags...)
	return create.Run()
}

// *********** Operator role service ****************
func (operatorRole *operatorRoleService) Create(flags ...string) (bytes.Buffer, error) {
	create := operatorRole.client.Runner.Cmd(
		"create", "operator-roles",
	).
		CmdFlags(flags...)
	return create.Run()
}
func (operatorRole *operatorRoleService) List(flags ...string) (bytes.Buffer, error) {
	create := operatorRole.client.Runner.Cmd(
		"list", "operator-roles",
	).
		CmdFlags(flags...)
	return create.Run()
}
func (operatorRole *operatorRoleService) Delete(flags ...string) (bytes.Buffer, error) {
	create := operatorRole.client.Runner.Cmd(
		"delete", "operator-roles",
	).
		CmdFlags(flags...)
	return create.Run()
}
func (operatorRole *operatorRoleService) Upgrade(flags ...string) (bytes.Buffer, error) {
	create := operatorRole.client.Runner.Cmd(
		"upgrade", "operator-roles",
	).
		CmdFlags(flags...)
	return create.Run()
}
