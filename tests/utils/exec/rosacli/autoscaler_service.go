package rosacli

import (
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"

	. "github.com/openshift/rosa/tests/utils/log"
)

type AutoScalerService interface {
	ResourcesCleaner

	CreateAutoScaler(clusterID string, flags ...string) (bytes.Buffer, error)
	DeleteAutoScaler(clusterID string) (bytes.Buffer, error)
	DescribeAutoScaler(clusterID string) (bytes.Buffer, error)
	EditAutoScaler(clusterID string, flags ...string) (bytes.Buffer, error)
	ReflectAutoScalerDescription(result bytes.Buffer) (asd *AutoScalerDescription, err error)
	DescribeAutoScalerAndReflect(clusterID string) (*AutoScalerDescription, error)

	RetrieveHelpForDescribe() (output bytes.Buffer, err error)
	RetrieveHelpForDelete() (output bytes.Buffer, err error)
}

type autoscalerService struct {
	ResourcesService
	created map[string]bool
}

type AutoScalerDescription struct {
	BalanceSimilarNodeGroups      string                   `yaml:"Balance Similar Node Groups,omitempty"`
	SkipNodesWithLocalStorage     string                   `yaml:"Skip Nodes With Local Storage,omitempty"`
	LogVerbosity                  string                   `yaml:"Log Verbosity,omitempty"`
	LabelsIgnoredForNodeBalancing string                   `yaml:"Labels Ignored For Node Balancing,omitempty"`
	IgnoreDaemonSetsUtilization   string                   `yaml:"Ignore DaemonSets Utilization,omitempty"`
	MaxNodeProvisionTime          string                   `yaml:"Maximum Node Provision Time,omitempty"`
	MaxPodGracePeriod             string                   `yaml:"Max Pod Grace Period,omitempty"`
	PodPriorityThreshold          string                   `yaml:"Pod Priority Threshold,omitempty"`
	ResourceLimits                []map[string]interface{} `yaml:"Resource Limits,omitempty"`
	ScaleDown                     []map[string]interface{} `yaml:"Scale Down,omitempty"`
}

func NewAutoScalerService(client *Client) AutoScalerService {
	return &autoscalerService{
		ResourcesService: ResourcesService{
			client: client,
		},
		created: make(map[string]bool),
	}
}

// Create AutoScaler
func (a *autoscalerService) CreateAutoScaler(clusterID string, flags ...string) (output bytes.Buffer, err error) {
	output, err = a.client.Runner.
		Cmd("create", "autoscaler").
		CmdFlags(append(flags, "-c", clusterID)...).
		Run()
	if err == nil {
		a.created[clusterID] = true
	}
	return
}

// Edit AutoScaler
func (a *autoscalerService) EditAutoScaler(clusterID string, flags ...string) (output bytes.Buffer, err error) {
	output, err = a.client.Runner.
		Cmd("edit", "autoscaler").
		CmdFlags(append(flags, "-c", clusterID)...).
		Run()
	return
}

// Describe AutoScaler
func (a *autoscalerService) DescribeAutoScaler(clusterID string) (bytes.Buffer, error) {
	describe := a.client.Runner.
		Cmd("describe", "autoscaler").
		CmdFlags("-c", clusterID)

	return describe.Run()
}

func (a *autoscalerService) DescribeAutoScalerAndReflect(clusterID string) (*AutoScalerDescription, error) {
	output, err := a.DescribeAutoScaler(clusterID)
	if err != nil {
		return nil, err
	}
	return a.ReflectAutoScalerDescription(output)
}

func (a *autoscalerService) ReflectAutoScalerDescription(result bytes.Buffer) (asd *AutoScalerDescription, err error) {
	var data []byte
	res := new(AutoScalerDescription)
	theMap, err := a.client.Parser.TextData.Input(result).Parse().TransformOutput(func(str string) (newStr string) {
		// Apply transformation to avoid issue with the list of min or max field parsing
		newStr = strings.ReplaceAll(str, "- Min:", " Min:")
		newStr = strings.ReplaceAll(newStr, "- Max:", " Max:")
		return
	}).YamlToMap()
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

// Delete AutoScaler
func (a *autoscalerService) DeleteAutoScaler(clusterID string) (output bytes.Buffer, err error) {
	if a.created[clusterID] {
		output, err = a.client.Runner.
			Cmd("delete", "autoscaler").
			CmdFlags("-c", clusterID, "-y").
			Run()
	}
	if err == nil {
		a.created[clusterID] = false
	}
	return
}

// Help for Describe AutoSCaler
func (a *autoscalerService) RetrieveHelpForDescribe() (output bytes.Buffer, err error) {
	return a.client.Runner.Cmd("describe", "autoscaler").CmdFlags("-h").Run()
}

// Help for Delete AutoScaler
func (a *autoscalerService) RetrieveHelpForDelete() (output bytes.Buffer, err error) {
	return a.client.Runner.Cmd("delete", "autoscaler").CmdFlags("-h").Run()
}

func (a *autoscalerService) CleanResources(clusterID string) (errors []error) {
	if a.created[clusterID] {
		Logger.Infof("Remove the autoscaler")
		_, err := a.DeleteAutoScaler(clusterID)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return
}
