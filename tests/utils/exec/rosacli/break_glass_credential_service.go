package rosacli

import (
	"bytes"

	"k8s.io/apimachinery/pkg/util/wait"

	. "github.com/openshift/rosa/tests/utils/log"
)

type BreakGlassCredentialService interface {
	ResourcesCleaner

	Create() *breakglasscredentialService
	List() *breakglasscredentialService
	Describe(bgcID ...string) *breakglasscredentialService
	Revoke() *breakglasscredentialService
	Help() *breakglasscredentialService

	// CreateBreakGlassCredential(clusterID string, flags ...string) (bytes.Buffer, error)
	// DeleteBreakGlassCredential(clusterID string) (bytes.Buffer, error)
	// ListBreakGlassCredentials(clusterID string) (bytes.Buffer, error)
	// ReflectBreakGlassCredentialLists(result bytes.Buffer) (bgcl *BreakGlassCredentialList, err error)
	// DescribeBreakGlassCredential(clusterID string, bgcID string) (bytes.Buffer, error)
	// ReflectBreakGlassCredentialDescription(result bytes.Buffer) (bgcd *BreakGlassCredentialDescription, err error)
	// GetIssuedCredential(clusterID string, bgcID string) (bytes.Buffer, error)
	WaitForStatus(clusterID string, status string, userName string) wait.ConditionFunc

	// ListBreakGlassCredentialsAndReflect(clusterID string) (*BreakGlassCredentialList, error)
	// DescribeBreakGlassCredentialsAndReflect(clusterID string, bgcID string) (*BreakGlassCredentialDescription, error)

	// RetrieveHelpForCreate() (bytes.Buffer, error)
	// RetrieveHelpForList() (bytes.Buffer, error)
	// RetrieveHelpForDescribe() (bytes.Buffer, error)
	// RetrieveHelpForDelete() (bytes.Buffer, error)
}

type breakglasscredentialService struct {
	ResourcesService
	created   map[string]bool
	Action    string
	ClusterID string
}

type BreakGlassCredential struct {
	ID       string `json:"ID,omitempty"`
	Username string `json:"Username,omitempty"`
	Status   string `json:"Status,omitempty"`
}

type BreakGlassCredentialList []*BreakGlassCredential

// type BreakGlassCredentialList struct {
// 	BreakGlassCredentials []BreakGlassCredential `json:"BreakGlassCredentials,omitempty"`
// }

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

func (b *breakglasscredentialService) Create() *breakglasscredentialService {
	b.Action = "create"
	b.client.Runner.Cmd("create", "break-glass-credential")
	return b
}

func (b *breakglasscredentialService) List() *breakglasscredentialService {
	b.Action = "list"
	b.client.Runner.Cmd("list", "break-glass-credential")
	return b
}

func (b *breakglasscredentialService) Revoke() *breakglasscredentialService {
	b.Action = "revoke"
	b.client.Runner.Cmd("revoke", "break-glass-credential")
	return b
}

func (b *breakglasscredentialService) Describe(bgcID ...string) *breakglasscredentialService {
	b.Action = "describe"
	if len(bgcID) > 0 {
		b.client.Runner.Cmd("describe", "break-glass-credential", bgcID[0])
	} else {
		b.client.Runner.Cmd("describe", "break-glass-credential")
	}
	return b
}

func (b *breakglasscredentialService) Parameters(clusterID string, flags ...string) *breakglasscredentialService {
	b.client.Runner.CmdFlags(append(flags, "-c", clusterID)...)
	return b
}

func (b *breakglasscredentialService) Help() *breakglasscredentialService {
	b.client.Runner.CmdFlags("-h")
	return b
}

func (b *breakglasscredentialService) Run() (output bytes.Buffer, err error) {
	return b.client.Runner.Run()
}

func (b *breakglasscredentialService) ToStruct() (interface{}, error) {
	output, err := b.client.Runner.Run()
	if err != nil {
		return nil, err
	}

	switch b.Action {
	case "describe":
		theMap, err := b.client.Parser.TextData.Input(output).Parse().YamlToMap()
		if err != nil {
			return nil, err
		}
		s := &BreakGlassCredentialDescription{}
		err = MapStructure(theMap, s)
		return s, err
	case "list":
		var bgcl BreakGlassCredentialList
		theMap := b.client.Parser.TableData.Input(output).Parse().Output()
		for _, bgcItem := range theMap {
			s := &BreakGlassCredential{}
			err = MapStructure(bgcItem, s)
			if err != nil {
				return nil, err
			}
			bgcl = append(bgcl, s)
		}
		return bgcl, err
	}

	return nil, err
}

// // Create BreakGlassCredential
// func (b *breakglasscredentialService) CreateBreakGlassCredential(clusterID string, flags ...string) (output bytes.Buffer, err error) {
// 	output, err = b.client.Runner.
// 		Cmd("create", "break-glass-credential").
// 		CmdFlags(append(flags, "-c", clusterID)...).
// 		Run()
// 	if err == nil {
// 		b.created[clusterID] = true
// 	}
// 	return
// }

// // List BreakGlassCredentials
// func (b *breakglasscredentialService) ListBreakGlassCredentials(clusterID string) (output bytes.Buffer, err error) {
// 	output, err = b.client.Runner.
// 		Cmd("list", "break-glass-credential").
// 		CmdFlags("-c", clusterID).
// 		Run()
// 	return
// }

// Check the breakGlassCredential with the userName exists in the breakGlassCredentialsList
func (bgcl BreakGlassCredentialList) IsPresent(userName string) (existed bool, breakGlassCredential *BreakGlassCredential) {
	existed = false
	breakGlassCredential = &BreakGlassCredential{}
	for _, breakGlassCred := range bgcl {
		if breakGlassCred.Username == userName {
			existed = true
			breakGlassCredential = breakGlassCred
			break
		}
	}
	return
}

// func (b *breakglasscredentialService) ReflectBreakGlassCredentialLists(result bytes.Buffer) (bgcl *BreakGlassCredentialList, err error) {
// 	bgcl = &BreakGlassCredentialList{}
// 	theMap := b.client.Parser.TableData.Input(result).Parse().Output()
// 	for _, bgcItem := range theMap {
// 		breakGlassCredential := &BreakGlassCredential{}
// 		err = MapStructure(bgcItem, breakGlassCredential)
// 		if err != nil {
// 			return
// 		}
// 		bgcl.BreakGlassCredentials = append(bgcl.BreakGlassCredentials, *breakGlassCredential)
// 	}
// 	return
// }

// func (b *breakglasscredentialService) ListBreakGlassCredentialsAndReflect(clusterID string) (*BreakGlassCredentialList, error) {
// 	output, err := b.ListBreakGlassCredentials(clusterID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return b.ReflectBreakGlassCredentialLists(output)
// }

// Describe BreakGlassCredential
// func (b *breakglasscredentialService) DescribeBreakGlassCredential(clusterID string, bgcID string) (bytes.Buffer, error) {
// 	describe := b.client.Runner.
// 		Cmd("describe", "break-glass-credential", bgcID).
// 		CmdFlags("-c", clusterID)

// 	return describe.Run()
// }

// func (b *breakglasscredentialService) DescribeBreakGlassCredentialsAndReflect(clusterID string, bgcID string) (*BreakGlassCredentialDescription, error) {
// 	output, err := b.DescribeBreakGlassCredential(clusterID, bgcID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return b.ReflectBreakGlassCredentialDescription(output)
// }

// func (b *breakglasscredentialService) ReflectBreakGlassCredentialDescription(result bytes.Buffer) (bgcd *BreakGlassCredentialDescription, err error) {
// 	var data []byte
// 	res := &BreakGlassCredentialDescription{}
// 	theMap, err := b.client.Parser.TextData.Input(result).Parse().YamlToMap()
// 	if err != nil {
// 		return
// 	}
// 	data, err = yaml.Marshal(&theMap)
// 	if err != nil {
// 		return
// 	}
// 	err = yaml.Unmarshal(data, res)
// 	return res, err
// }

// Delete BreakGlassCredential
// func (b *breakglasscredentialService) DeleteBreakGlassCredential(clusterID string) (output bytes.Buffer, err error) {
// 	output, err = b.client.Runner.
// 		Cmd("revoke", "break-glass-credential").
// 		CmdFlags("-c", clusterID, "-y").
// 		Run()
// 	return
// }

// func (b *breakglasscredentialService) GetIssuedCredential(clusterID string, bgcID string) (bytes.Buffer, error) {
// 	output, err := b.client.Runner.
// 		Cmd("describe", "break-glass-credential").
// 		CmdFlags("-c", clusterID, "--id", bgcID, "--kubeconfig").
// 		Run()
// 	return output, err
// }

func (b *breakglasscredentialService) WaitForStatus(clusterID string, status string, userName string) wait.ConditionFunc {
	return func() (bool, error) {
		breakGlassCredList, err := b.List().Parameters(clusterID).ToStruct()
		if err != nil {
			return false, err
		}
		_, breakGlassCredential := breakGlassCredList.(BreakGlassCredentialList).IsPresent(userName)
		Logger.Infof("The status for break-glass-credential %s is %s\n", breakGlassCredential.ID, breakGlassCredential.Status)
		return breakGlassCredential.Status == status, err
	}
}

// // Help for Create BreakGlassCredential
// func (b *breakglasscredentialService) RetrieveHelpForCreate() (output bytes.Buffer, err error) {
// 	return b.client.Runner.Cmd("create", "break-glass-credential").CmdFlags("-h").Run()
// }

// // Help for List BreakGlassCredential
// func (b *breakglasscredentialService) RetrieveHelpForList() (output bytes.Buffer, err error) {
// 	return b.client.Runner.Cmd("list", "break-glass-credential").CmdFlags("-h").Run()
// }

// // Help for Describe BreakGlassCredential
// func (b *breakglasscredentialService) RetrieveHelpForDescribe() (output bytes.Buffer, err error) {
// 	return b.client.Runner.Cmd("describe", "break-glass-credential").CmdFlags("-h").Run()
// }

// // Help for Delete BreakGlassCredential
// func (b *breakglasscredentialService) RetrieveHelpForDelete() (output bytes.Buffer, err error) {
// 	return b.client.Runner.Cmd("revoke", "break-glass-credential").CmdFlags("-h").Run()
// }

func (b *breakglasscredentialService) CleanResources(clusterID string) (errors []error) {
	if b.created[clusterID] {
		Logger.Infof("Remove remaining break-glass-credentials")
		_, err := b.Revoke().Help().Run()
		if err != nil {
			errors = append(errors, err)
		}
	}

	return
}
