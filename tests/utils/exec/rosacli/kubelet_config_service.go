package rosacli

import (
	"bytes"

	"gopkg.in/yaml.v3"

	"github.com/openshift/rosa/tests/utils/log"
)

type KubeletConfigService interface {
	ResourcesCleaner
	ListKubeletConfigs(clusterID string, flags ...string) (bytes.Buffer, error)
	ListKubeletConfigsAndReflect(clusterID string, flags ...string) (kubes *KubeletConfigList, err error)
	ReflectKubeletConfigs(output bytes.Buffer) (kubes *KubeletConfigList, err error)
	DescribeKubeletConfig(clusterID string, flags ...string) (bytes.Buffer, error)
	ReflectKubeletConfig(result bytes.Buffer) *KubeletConfig
	EditKubeletConfig(clusterID string, flags ...string) (bytes.Buffer, error)
	DeleteKubeletConfig(clusterID string, flags ...string) (bytes.Buffer, error)
	CreateKubeletConfig(clusterID string, flags ...string) (bytes.Buffer, error)
}

type kubeletConfigService struct {
	ResourcesService

	created map[string]bool
}

func NewKubeletConfigService(client *Client) KubeletConfigService {
	return &kubeletConfigService{
		ResourcesService: ResourcesService{
			client: client,
		},
		created: make(map[string]bool),
	}
}

// Struct for the 'rosa describe/list kubeletconfig(s)' output
type KubeletConfig struct {
	ID           string `yaml:"ID,omitempty" json:"ID,omitempty"`
	Name         string `yaml:"Name,omitempty" json:"NAME,omitempty"`
	PodPidsLimit string `yaml:"Pod Pids Limit,omitempty" json:"POD PIDS LIMIT,omitempty"`
}

// Struct for the 'rosa list kubeletconfigs'
type KubeletConfigList struct {
	KubeletConfigs []*KubeletConfig
}

func (kl *KubeletConfigList) KubeletConfig(kubeName string) *KubeletConfig {
	for _, kubeletconfig := range kl.KubeletConfigs {
		if kubeletconfig.Name == kubeName {
			return kubeletconfig
		}
	}
	return nil
}

// List kubeletconfigs
func (k *kubeletConfigService) ListKubeletConfigs(clusterID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	list := k.client.Runner.
		Cmd("list", "kubeletconfigs").
		CmdFlags(combflags...)

	return list.Run()
}

// Reflect kubeletconfigs
func (k *kubeletConfigService) ReflectKubeletConfigs(output bytes.Buffer) (kubes *KubeletConfigList, err error) {
	kubes = &KubeletConfigList{}
	theMap := k.client.Parser.TableData.Input(output).Parse().Output()
	for _, kubeletConfigItem := range theMap {
		kube := &KubeletConfig{}
		err = MapStructure(kubeletConfigItem, kube)
		if err != nil {
			return
		}
		kubes.KubeletConfigs = append(kubes.KubeletConfigs, kube)
	}
	return kubes, err
}

// ListKubeletConfigsAndReflect will list the kubeletconfigs
func (k *kubeletConfigService) ListKubeletConfigsAndReflect(
	clusterID string, flags ...string) (kubes *KubeletConfigList, err error) {
	var output bytes.Buffer
	output, err = k.ListKubeletConfigs(clusterID, flags...)
	if err != nil {
		return
	}
	kubes, err = k.ReflectKubeletConfigs(output)
	return
}

// List and reflect kubeletconfigs
func (k *kubeletConfigService) ListAndReflectKubeletConfigs(clusterID string, flags ...string) {

}

// Describe Kubeletconfig
func (k *kubeletConfigService) DescribeKubeletConfig(clusterID string, flags ...string) (bytes.Buffer, error) {
	cmdflags := append([]string{"-c", clusterID}, flags...)
	describe := k.client.Runner.
		Cmd("describe", "kubeletconfig").
		CmdFlags(cmdflags...)

	return describe.Run()
}

// Pasrse the result of 'rosa describe kubeletconfig' to the KubeletConfig struct
func (k *kubeletConfigService) ReflectKubeletConfig(result bytes.Buffer) *KubeletConfig {
	res := new(KubeletConfig)
	theMap, _ := k.client.Parser.TextData.Input(result).Parse().YamlToMap()
	data, _ := yaml.Marshal(&theMap)
	yaml.Unmarshal(data, res)
	return res
}

// Edit the kubeletconfig
func (k *kubeletConfigService) EditKubeletConfig(clusterID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	editCluster := k.client.Runner.
		Cmd("edit", "kubeletconfig").
		CmdFlags(combflags...)
	return editCluster.Run()
}

// Delete the kubeletconfig
func (k *kubeletConfigService) DeleteKubeletConfig(clusterID string, flags ...string) (output bytes.Buffer, err error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	editCluster := k.client.Runner.
		Cmd("delete", "kubeletconfig").
		CmdFlags(combflags...)
	output, err = editCluster.Run()
	if err == nil {
		k.created[clusterID] = false
	}
	return
}

// Create the kubeletconfig
func (k *kubeletConfigService) CreateKubeletConfig(clusterID string, flags ...string) (output bytes.Buffer, err error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	createCluster := k.client.Runner.
		Cmd("create", "kubeletconfig").
		CmdFlags(combflags...)
	output, err = createCluster.Run()
	if err == nil {
		k.created[clusterID] = true
	}
	return
}

func (k *kubeletConfigService) CleanResources(clusterID string) (errors []error) {
	if k.created[clusterID] {
		log.Logger.Infof("Remove remaining kubelet config")
		_, err := k.DeleteKubeletConfig(clusterID,
			"-y",
		)
		if err != nil {
			errors = append(errors, err)
		}
	}

	return
}
