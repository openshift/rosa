package rosacli

import (
	"bytes"

	"gopkg.in/yaml.v3"

	. "github.com/openshift/rosa/tests/utils/log"
)

type ExternalAuthProviderService interface {
	ResourcesCleaner

	CreateExternalAuthProvider(clusterID string, flags ...string) (bytes.Buffer, error)
	DeleteExternalAuthProvider(clusterID string, eapID string) (bytes.Buffer, error)
	ListExternalAuthProvider(clusterID string) (bytes.Buffer, error)
	ReflectExternalAuthProviderLists(result bytes.Buffer) (eapl *ExternalAuthProviderList, err error)
	DescribeExternalAuthProvider(clusterID string, eapID string) (bytes.Buffer, error)
	ReflectExternalAuthProviderDescription(result bytes.Buffer) (eapd *ExternalAuthProviderDescription, err error)

	ListExternalAuthProviderAndReflect(clusterID string) (*ExternalAuthProviderList, error)
	DescribeExternalAuthProviderAndReflect(clusterID string, eapID string) (*ExternalAuthProviderDescription, error)

	RetrieveHelpForList() (output bytes.Buffer, err error)
	RetrieveHelpForDescribe() (output bytes.Buffer, err error)
	RetrieveHelpForDelete() (output bytes.Buffer, err error)
}

type externalauthproviderService struct {
	ResourcesService
	created map[string]bool
}

type ExternalAuthProvider struct {
	Name      string `json:"NAME,omitempty"`
	IssuerUrl string `json:"ISSUER URL,omitempty"`
}

type ExternalAuthProviderList struct {
	ExternalAuthProviders []ExternalAuthProvider `json:"ExternalAuthProviders,omitempty"`
}

type ExternalAuthProviderDescription struct {
	ID                    string   `yaml:"ID,omitempty"`
	ClusterID             string   `yaml:"Cluster ID,omitempty"`
	IssuerAudiences       []string `yaml:"Issuer audiences,omitempty"`
	IssuerUrl             string   `yaml:"Issuer Url,omitempty"`
	ClaimMappingsGroup    string   `yaml:"Claim mappings group,omitempty"`
	ClaimMappingsUserName string   `yaml:"Claim mappings username,omitempty"`
	ClaimValidationRules  []string `yaml:"Claim validation rules,omitempty"`
	ConsoleClientID       string   `yaml:"Console client id,omitempty"`
}

func NewExternalAuthProviderService(client *Client) ExternalAuthProviderService {
	return &externalauthproviderService{
		ResourcesService: ResourcesService{
			client: client,
		},
		created: make(map[string]bool),
	}
}

// Create ExternalAuthProvider
func (e *externalauthproviderService) CreateExternalAuthProvider(
	clusterID string, flags ...string) (output bytes.Buffer, err error) {
	output, err = e.client.Runner.
		Cmd("create", "external-auth-provider").
		CmdFlags(append(flags, "-c", clusterID)...).
		Run()
	if err == nil {
		e.created[clusterID] = true
	}
	return
}

// List ExternalAuthProviders
func (e *externalauthproviderService) ListExternalAuthProvider(clusterID string) (output bytes.Buffer, err error) {
	output, err = e.client.Runner.
		Cmd("list", "external-auth-provider").
		CmdFlags("-c", clusterID).
		Run()
	return
}

func (e *externalauthproviderService) ReflectExternalAuthProviderLists(
	result bytes.Buffer) (eapl *ExternalAuthProviderList, err error) {
	eapl = &ExternalAuthProviderList{}
	theMap := e.client.Parser.TableData.Input(result).Parse().Output()
	for _, eapItem := range theMap {
		externalAuthProvider := &ExternalAuthProvider{}
		err = MapStructure(eapItem, externalAuthProvider)
		if err != nil {
			return
		}
		eapl.ExternalAuthProviders = append(eapl.ExternalAuthProviders, *externalAuthProvider)
	}
	return
}

func (e *externalauthproviderService) ListExternalAuthProviderAndReflect(
	clusterID string) (*ExternalAuthProviderList, error) {
	output, err := e.ListExternalAuthProvider(clusterID)
	if err != nil {
		return nil, err
	}
	return e.ReflectExternalAuthProviderLists(output)
}

// Describe ExternalAuthProvider
func (e *externalauthproviderService) DescribeExternalAuthProvider(
	clusterID string, eapID string) (bytes.Buffer, error) {
	describe := e.client.Runner.
		Cmd("describe", "external-auth-provider", eapID).
		CmdFlags("-c", clusterID)

	return describe.Run()
}

func (e *externalauthproviderService) DescribeExternalAuthProviderAndReflect(
	clusterID string, eapID string) (*ExternalAuthProviderDescription, error) {
	output, err := e.DescribeExternalAuthProvider(clusterID, eapID)
	if err != nil {
		return nil, err
	}
	return e.ReflectExternalAuthProviderDescription(output)
}

func (e *externalauthproviderService) ReflectExternalAuthProviderDescription(
	result bytes.Buffer) (eapd *ExternalAuthProviderDescription, err error) {
	var data []byte
	res := &ExternalAuthProviderDescription{}
	theMap, err := e.client.Parser.TextData.Input(result).Parse().YamlToMap()
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

// Delete ExternalAuthProvider
func (e *externalauthproviderService) DeleteExternalAuthProvider(
	clusterID string, eapID string) (output bytes.Buffer, err error) {
	output, err = e.client.Runner.
		Cmd("delete", "external-auth-provider", eapID).
		CmdFlags("-c", clusterID, "-y").
		Run()
	return
}

func (e *externalauthproviderService) CleanResources(clusterID string) (errors []error) {
	if e.created[clusterID] {
		Logger.Infof("Remove remaining extrenal-auth-providers")
		externalaAuthProviders, err := e.ListExternalAuthProviderAndReflect(clusterID)
		if err != nil {
			return
		}
		for _, externalAuthProvider := range externalaAuthProviders.ExternalAuthProviders {
			_, err := e.DeleteExternalAuthProvider(clusterID, externalAuthProvider.Name)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}
	return
}

// Help for List ExternalAuthProvider
func (e *externalauthproviderService) RetrieveHelpForList() (output bytes.Buffer, err error) {
	return e.client.Runner.Cmd("list", "external-auth-provider").CmdFlags("-h").Run()
}

// Help for Describe ExternalAuthProvider
func (e *externalauthproviderService) RetrieveHelpForDescribe() (output bytes.Buffer, err error) {
	return e.client.Runner.Cmd("describe", "external-auth-provider").CmdFlags("-h").Run()
}

// Help for Delete ExternalAuthProvider
func (e *externalauthproviderService) RetrieveHelpForDelete() (output bytes.Buffer, err error) {
	return e.client.Runner.Cmd("delete", "external-auth-provider").CmdFlags("-h").Run()
}
