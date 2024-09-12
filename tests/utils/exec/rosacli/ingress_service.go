package rosacli

import (
	"bytes"

	"gopkg.in/yaml.v3"

	common "github.com/openshift/rosa/tests/utils/common"
	. "github.com/openshift/rosa/tests/utils/log"
)

type IngressService interface {
	ResourcesCleaner

	EditIngress(clusterID string, ingressID string, flags ...string) (bytes.Buffer, error)
	ListIngress(clusterID string, flags ...string) (bytes.Buffer, error)
	DeleteIngress(clusterID string, ingressID string) (bytes.Buffer, error)
	ReflectIngressList(result bytes.Buffer) (res *IngressList, err error)
	DescribeIngress(clusterID string, ingressID string) (bytes.Buffer, error)
	DescribeIngressAndReflect(clusterID string, ingressID string) (res *Ingress, err error)
}

type ingressService struct {
	ResourcesService

	ingress map[string][]string
}

func NewIngressService(client *Client) IngressService {
	return &ingressService{
		ResourcesService: ResourcesService{
			client: client,
		},
		ingress: make(map[string][]string),
	}
}

func (i *ingressService) CleanResources(clusterID string) (errors []error) {
	var igsToDel []string
	igsToDel = append(igsToDel, i.ingress[clusterID]...)
	for _, igID := range igsToDel {
		Logger.Infof("Remove remaining ingress '%s'", igID)
		_, err := i.DeleteIngress(clusterID, igID)
		if err != nil {
			errors = append(errors, err)
		}
	}

	return
}

// Struct for the 'rosa describe ingress' output
type IngressList struct {
	Ingresses []*Ingress `json:"Ingresses,omitempty"`
}
type Ingress struct {
	ClusterID                string `yaml:"Cluster ID,omitempty"`
	ID                       string `yaml:"ID,omitempty" json:"ID,omitempty"`
	ApplicationRouter        string `yaml:"APPLICATION ROUTER,omitempty" json:"APPLICATION ROUTER,omitempty"`
	Private                  string `yaml:"Private,omitempty" json:"PRIVATE,omitempty"`
	Default                  string `yaml:"Default,omitempty" json:"DEFAULT,omitempty"`
	RouteSelectors           string `yaml:"Route Selectors,omitempty" json:"ROUTE SELECTORS,omitempty"`
	LBType                   string `yaml:"LB-Type,omitempty" json:"LB-TYPE,omitempty"`
	ExcludeNamespace         string `yaml:"Exclude Namespce,omitempty" json:"EXCLUDED NAMESPACE,omitempty"`
	WildcardPolicy           string `yaml:"Wildcard Policy,omitempty" json:"WILDCARD POLICY,omitempty"`
	NamespaceOwnershipPolicy string `yaml:"Namespace Ownership Policy,omitempty" json:"NAMESPACE OWNERSHIP,omitempty"`
}

// Get specified ingress by ingress id
func (inl IngressList) Ingress(id string) (in *Ingress) {
	for _, inItem := range inl.Ingresses {
		if inItem.ID == id {
			in = inItem
			return
		}
	}
	return
}

// Edit the cluster ingress
func (i *ingressService) EditIngress(clusterID string, ingressID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	editIngress := i.client.Runner.
		Cmd("edit", "ingress", ingressID).
		CmdFlags(combflags...)
	return editIngress.Run()
}

// List the cluster ingress
func (i *ingressService) DescribeIngress(clusterID string, ingressID string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, ingressID)
	describeIngress := i.client.Runner.
		Cmd("describe", "ingress").
		CmdFlags(combflags...)
	return describeIngress.Run()
}

// Parse the result of 'rosa list ingress' to Ingress struct
func (i *ingressService) DescribeIngressAndReflect(clusterID string, ingressID string) (res *Ingress, err error) {
	output, err := i.DescribeIngress(clusterID, ingressID)
	if err != nil {
		return
	}
	res = &Ingress{}
	theMap, _ := i.client.Parser.TextData.Input(output).Parse().YamlToMap()
	data, _ := yaml.Marshal(&theMap)
	yaml.Unmarshal(data, res)
	return res, err
}

// List the cluster ingress
func (i *ingressService) ListIngress(clusterID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	listIngress := i.client.Runner.
		Cmd("list", "ingress").
		CmdFlags(combflags...)
	return listIngress.Run()
}

// Parse the result of 'rosa list ingress' to Ingress struct
func (i *ingressService) ReflectIngressList(result bytes.Buffer) (res *IngressList, err error) {
	res = &IngressList{}
	theMap := i.client.Parser.TableData.Input(result).Parse().Output()
	for _, ingressItem := range theMap {
		in := &Ingress{}
		err = MapStructure(ingressItem, in)
		if err != nil {
			return
		}
		res.Ingresses = append(res.Ingresses, in)
	}
	return res, err
}

// Delete the ingress
func (i *ingressService) DeleteIngress(clusterID string, ingressID string) (output bytes.Buffer, err error) {
	var flags []string

	if len(clusterID) > 0 {
		flags = append(flags, "-c", clusterID)
	}
	if len(ingressID) > 0 {
		flags = append(flags, ingressID)
	}
	output, err = i.client.Runner.
		Cmd("delete", "ingress").
		CmdFlags(append(flags, "-y")...).
		Run()
	if err == nil {
		i.ingress[clusterID] = common.RemoveFromStringSlice(i.ingress[clusterID], ingressID)
	}
	return
}
