package rosacli

import (
	"bytes"

	"gopkg.in/yaml.v3"

	"k8s.io/apimachinery/pkg/util/wait"

	. "github.com/openshift/rosa/tests/utils/log"
)

type BreakGlassCredentialService interface {
	ResourcesCleaner

	CreateBreakGlassCredential(clusterID string, flags ...string) (bytes.Buffer, error)
	DeleteBreakGlassCredential(clusterID string) (bytes.Buffer, error)
	ListBreakGlassCredentials(clusterID string) (bytes.Buffer, error)
	ReflectBreakGlassCredentialLists(result bytes.Buffer) (bgcl *BreakGlassCredentialList, err error)
	DescribeBreakGlassCredential(clusterID string, bgcID string) (bytes.Buffer, error)
	ReflectBreakGlassCredentialDescription(result bytes.Buffer) (bgcd *BreakGlassCredentialDescription, err error)
	GetIssuedCredential(clusterID string, bgcID string) (bytes.Buffer, error)
	WaitForBreakGlassCredentialToStatus(clusterID string, status string, userName string) wait.ConditionFunc

	ListBreakGlassCredentialsAndReflect(clusterID string) (*BreakGlassCredentialList, error)
	DescribeBreakGlassCredentialsAndReflect(clusterID string, bgcID string) (*BreakGlassCredentialDescription, error)

	RetrieveHelpForCreate() (bytes.Buffer, error)
	RetrieveHelpForList() (bytes.Buffer, error)
	RetrieveHelpForDescribe() (bytes.Buffer, error)
	RetrieveHelpForDelete() (bytes.Buffer, error)
}

type breakglasscredentialService struct {
	ResourcesService
	created map[string]bool
}

type BreakGlassCredential struct {
	ID       string `json:"ID,omitempty"`
	Username string `json:"Username,omitempty"`
	Status   string `json:"Status,omitempty"`
}

type BreakGlassCredentialList struct {
	BreakGlassCredentials []*BreakGlassCredential `json:"BreakGlassCredentials,omitempty"`
}

type BreakGlassCredentialDescription struct {
	ID       string `yaml:"ID,omitempty"`
	Username string `yaml:"Username,omitempty"`
	ExpireAt string `yaml:"Expire at,omitempty"`
	Status   string `yaml:"Status,omitempty"`
}

func NewBreakGlassCredentialService(client *Client) BreakGlassCredentialService {
	return &breakglasscredentialService{
		ResourcesService: ResourcesService{
			client: client,
		},
		created: make(map[string]bool),
	}
}

// Create BreakGlassCredential
func (b *breakglasscredentialService) CreateBreakGlassCredential(clusterID string,
	flags ...string) (output bytes.Buffer, err error) {
	output, err = b.client.Runner.
		Cmd("create", "break-glass-credential").
		CmdFlags(append(flags, "-c", clusterID)...).
		Run()
	if err == nil {
		b.created[clusterID] = true
	}
	return
}

// List BreakGlassCredentials
func (b *breakglasscredentialService) ListBreakGlassCredentials(clusterID string) (output bytes.Buffer, err error) {
	output, err = b.client.Runner.
		Cmd("list", "break-glass-credential").
		CmdFlags("-c", clusterID).
		Run()
	return
}

// Check the breakGlassCredential with the userName exists in the breakGlassCredentialsList
func (breakglassCredList BreakGlassCredentialList) IsPresent(
	userName string) (existed bool, breakGlassCredential *BreakGlassCredential) {
	existed = false
	for _, breakGlassCred := range breakglassCredList.BreakGlassCredentials {
		if breakGlassCred.Username == userName {
			existed = true
			breakGlassCredential = breakGlassCred
			break
		}
	}
	return
}

func (b *breakglasscredentialService) ReflectBreakGlassCredentialLists(
	result bytes.Buffer) (bgcl *BreakGlassCredentialList, err error) {
	bgcl = &BreakGlassCredentialList{}
	theMap := b.client.Parser.TableData.Input(result).Parse().Output()
	for _, bgcItem := range theMap {
		breakGlassCredential := &BreakGlassCredential{}
		err = MapStructure(bgcItem, breakGlassCredential)
		if err != nil {
			return
		}
		bgcl.BreakGlassCredentials = append(bgcl.BreakGlassCredentials, breakGlassCredential)
	}
	return
}

func (b *breakglasscredentialService) ListBreakGlassCredentialsAndReflect(
	clusterID string) (*BreakGlassCredentialList, error) {
	output, err := b.ListBreakGlassCredentials(clusterID)
	if err != nil {
		return nil, err
	}
	return b.ReflectBreakGlassCredentialLists(output)
}

// Describe BreakGlassCredential
func (b *breakglasscredentialService) DescribeBreakGlassCredential(
	clusterID string, bgcID string) (bytes.Buffer, error) {
	describe := b.client.Runner.
		Cmd("describe", "break-glass-credential", bgcID).
		CmdFlags("-c", clusterID)

	return describe.Run()
}

func (b *breakglasscredentialService) DescribeBreakGlassCredentialsAndReflect(
	clusterID string, bgcID string) (*BreakGlassCredentialDescription, error) {
	output, err := b.DescribeBreakGlassCredential(clusterID, bgcID)
	if err != nil {
		return nil, err
	}
	return b.ReflectBreakGlassCredentialDescription(output)
}

func (b *breakglasscredentialService) ReflectBreakGlassCredentialDescription(
	result bytes.Buffer) (bgcd *BreakGlassCredentialDescription, err error) {
	var data []byte
	res := &BreakGlassCredentialDescription{}
	theMap, err := b.client.Parser.TextData.Input(result).Parse().YamlToMap()
	if err != nil {
		return
	}
	data, err = yaml.Marshal(&theMap)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(data, res)
	return res, err
}

// Delete BreakGlassCredential
func (b *breakglasscredentialService) DeleteBreakGlassCredential(clusterID string) (output bytes.Buffer, err error) {
	output, err = b.client.Runner.
		Cmd("revoke", "break-glass-credential").
		CmdFlags("-c", clusterID, "-y").
		Run()
	return
}

func (b *breakglasscredentialService) GetIssuedCredential(clusterID string, bgcID string) (bytes.Buffer, error) {
	output, err := b.client.Runner.
		Cmd("describe", "break-glass-credential").
		CmdFlags("-c", clusterID, "--id", bgcID, "--kubeconfig").
		Run()
	return output, err
}

func (b *breakglasscredentialService) WaitForBreakGlassCredentialToStatus(
	clusterID string, status string, userName string) wait.ConditionFunc {
	return func() (bool, error) {
		breakGlassCredList, err := b.ListBreakGlassCredentialsAndReflect(clusterID)
		if err != nil {
			return false, err
		}
		_, breakGlassCredential := breakGlassCredList.IsPresent(userName)
		Logger.Infof("The status for break-glass-credential %s is %s\n", breakGlassCredential.ID, breakGlassCredential.Status)
		return breakGlassCredential.Status == status, err
	}
}

// Help for Create BreakGlassCredential
func (b *breakglasscredentialService) RetrieveHelpForCreate() (output bytes.Buffer, err error) {
	return b.client.Runner.Cmd("create", "break-glass-credential").CmdFlags("-h").Run()
}

// Help for List BreakGlassCredential
func (b *breakglasscredentialService) RetrieveHelpForList() (output bytes.Buffer, err error) {
	return b.client.Runner.Cmd("list", "break-glass-credential").CmdFlags("-h").Run()
}

// Help for Describe BreakGlassCredential
func (b *breakglasscredentialService) RetrieveHelpForDescribe() (output bytes.Buffer, err error) {
	return b.client.Runner.Cmd("describe", "break-glass-credential").CmdFlags("-h").Run()
}

// Help for Delete BreakGlassCredential
func (b *breakglasscredentialService) RetrieveHelpForDelete() (output bytes.Buffer, err error) {
	return b.client.Runner.Cmd("revoke", "break-glass-credential").CmdFlags("-h").Run()
}

func (b *breakglasscredentialService) CleanResources(clusterID string) (errors []error) {
	if b.created[clusterID] {
		Logger.Infof("Remove remaining break-glass-credentials")
		_, err := b.DeleteBreakGlassCredential(clusterID)
		if err != nil {
			errors = append(errors, err)
		}
	}

	return
}
