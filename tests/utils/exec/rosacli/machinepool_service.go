package rosacli

import (
	"bytes"

	"gopkg.in/yaml.v3"

	common "github.com/openshift/rosa/tests/utils/common"
	. "github.com/openshift/rosa/tests/utils/log"
)

type MachinePoolService interface {
	ResourcesCleaner

	ListMachinePool(clusterID string) (bytes.Buffer, error)
	DescribeMachinePool(clusterID string, mpID string) (bytes.Buffer, error)
	CreateMachinePool(clusterID string, name string, flags ...string) (bytes.Buffer, error)
	EditMachinePool(clusterID string, machinePoolName string, flags ...string) (bytes.Buffer, error)
	DeleteMachinePool(clusterID string, machinePoolName string) (bytes.Buffer, error)

	ReflectMachinePoolList(result bytes.Buffer) (mpl MachinePoolList, err error)
	ReflectMachinePoolDescription(result bytes.Buffer) (*MachinePoolDescription, error)
	ListAndReflectMachinePools(clusterID string) (mpl MachinePoolList, err error)

	ReflectNodePoolList(result bytes.Buffer) (*NodePoolList, error)
	ListAndReflectNodePools(clusterID string) (*NodePoolList, error)
	ReflectNodePoolDescription(result bytes.Buffer) (npd *NodePoolDescription, err error)
	DescribeAndReflectNodePool(clusterID string, name string) (*NodePoolDescription, error)
	DescribeAndReflectMachinePool(clusterID string, name string) (*MachinePoolDescription, error)

	RetrieveHelpForCreate() (bytes.Buffer, error)
	RetrieveHelpForEdit() (bytes.Buffer, error)
}

type machinepoolService struct {
	ResourcesService

	machinePools map[string][]string
}

func NewMachinePoolService(client *Client) MachinePoolService {
	return &machinepoolService{
		ResourcesService: ResourcesService{
			client: client,
		},
		machinePools: make(map[string][]string),
	}
}

// Struct for the 'rosa list machinepool' output for non-hosted-cp clusters
type MachinePool struct {
	ID               string `json:"ID,omitempty"`
	AutoScaling      string `json:"AUTOSCALING,omitempty"`
	Replicas         string `json:"REPLICAS,omitempty"`
	DiskSize         string `json:"DISK SIZE,omitempty"`
	InstanceType     string `json:"INSTANCE TYPE,omitempty"`
	Labels           string `json:"LABELS,omitempty"`
	Taints           string `json:"TAINTS,omitempty"`
	AvalaiblityZones string `json:"AVAILABILITY ZONES,omitempty"`
	Subnets          string `json:"SUBNET,omitempty"`
	SpotInstances    string `json:"SPOT INSTANCES,omitempty"`
	SecurityGroupIDs string `json:"SG IDs,omitempty"`
}
type MachinePoolList struct {
	MachinePools []*MachinePool `json:"MachinePools,omitempty"`
}

// Struct for the 'rosa list machinepool' output for non-hosted-cp clusters
type MachinePoolDescription struct {
	AvailablityZones string `yaml:"Availability zones,omitempty"`
	AutoScaling      string `yaml:"Autoscaling,omitempty"`
	ClusterID        string `yaml:"Cluster ID,omitempty"`
	DiskSize         string `yaml:"Disk size,omitempty"`
	ID               string `yaml:"ID,omitempty"`
	InstanceType     string `yaml:"Instance type,omitempty"`
	Labels           string `yaml:"Labels,omitempty"`
	Replicas         string `yaml:"Replicas,omitempty"`
	SecurityGroupIDs string `yaml:"Security Group IDs,omitempty"`
	Subnets          string `yaml:"Subnets,omitempty"`
	SpotInstances    string `yaml:"Spot instances,omitempty"`
	Taints           string `yaml:"Taints,omitempty"`
	Tags             string `yaml:"Tags,omitempty"`
}

// Struct for the 'rosa list machinepool' output for hosted-cp clusters
type NodePool struct {
	ID               string `json:"ID,omitempty"`
	AutoScaling      string `json:"AUTOSCALING,omitempty"`
	Replicas         string `json:"REPLICAS,omitempty"`
	InstanceType     string `json:"INSTANCE TYPE,omitempty"`
	Labels           string `json:"LABELS,omitempty"`
	Taints           string `json:"TAINTS,omitempty"`
	AvalaiblityZones string `json:"AVAILABILITY ZONES,omitempty"`
	Subnet           string `json:"SUBNET,omitempty"`
	Version          string `json:"VERSION,omitempty"`
	AutoRepair       string `json:"AUTOREPAIR,omitempty"`
	TuningConfigs    string `json:"TUNING CONFIGS,omitempty"`
	Message          string `json:"MESSAGE,omitempty"`
}

type NodePoolList struct {
	NodePools []*NodePool `json:"NodePools,omitempty"`
}

type NodePoolDescription struct {
	ID                         string              `yaml:"ID,omitempty"`
	ClusterID                  string              `yaml:"Cluster ID,omitempty"`
	AutoScaling                string              `yaml:"Autoscaling,omitempty"`
	DesiredReplicas            string              `yaml:"Desired replicas,omitempty"`
	CurrentReplicas            string              `yaml:"Current replicas,omitempty"`
	InstanceType               string              `yaml:"Instance type,omitempty"`
	KubeletConfigs             string              `yaml:"Kubelet configs,omitempty"`
	Labels                     string              `yaml:"Labels,omitempty"`
	Tags                       string              `yaml:"Tags,omitempty"`
	Taints                     string              `yaml:"Taints,omitempty"`
	AvalaiblityZones           string              `yaml:"Availability zone,omitempty"`
	Subnet                     string              `yaml:"Subnet,omitempty"`
	Version                    string              `yaml:"Version,omitempty"`
	AutoRepair                 string              `yaml:"Autorepair,omitempty"`
	TuningConfigs              string              `yaml:"Tuning configs,omitempty"`
	ManagementUpgrade          []map[string]string `yaml:"Management upgrade,omitempty"`
	Message                    string              `yaml:"Message,omitempty"`
	ScheduledUpgrade           string              `yaml:"Scheduled upgrade,omitempty"`
	AdditionalSecurityGroupIDs string              `yaml:"Additional security group IDs,omitempty"`
	NodeDrainGracePeriod       string              `yaml:"Node drain grace period,omitempty"`
}

// Create MachinePool
func (m *machinepoolService) CreateMachinePool(
	clusterID string, name string, flags ...string) (output bytes.Buffer, err error) {
	output, err = m.client.Runner.
		Cmd("create", "machinepool").
		CmdFlags(append(flags, "-c", clusterID, "--name", name)...).
		Run()
	if err == nil {
		m.machinePools[clusterID] = append(m.machinePools[clusterID], name)
	}
	return
}

// List MachinePool
func (m *machinepoolService) ListMachinePool(clusterID string) (bytes.Buffer, error) {
	listMachinePool := m.client.Runner.
		Cmd("list", "machinepool").
		CmdFlags("-c", clusterID)
	return listMachinePool.Run()
}

// Describe MachinePool
func (m *machinepoolService) DescribeMachinePool(clusterID string, mpID string) (bytes.Buffer, error) {
	describeMp := m.client.Runner.
		Cmd("describe", "machinepool").
		CmdFlags(mpID, "-c", clusterID)
	return describeMp.Run()
}

// DescribeAndReflectMachinePool
func (m *machinepoolService) DescribeAndReflectMachinePool(
	clusterID string, mpID string) (*MachinePoolDescription, error) {
	output, err := m.DescribeMachinePool(clusterID, mpID)
	if err != nil {
		return nil, err
	}
	return m.ReflectMachinePoolDescription(output)
}

// Delete MachinePool
func (m *machinepoolService) DeleteMachinePool(
	clusterID string, machinePoolName string) (output bytes.Buffer, err error) {
	output, err = m.client.Runner.
		Cmd("delete", "machinepool").
		CmdFlags("-c", clusterID, machinePoolName, "-y").
		Run()
	if err == nil {
		m.machinePools[clusterID] = common.RemoveFromStringSlice(m.machinePools[clusterID], machinePoolName)
	}
	return
}

// Edit MachinePool
func (m *machinepoolService) EditMachinePool(
	clusterID string, machinePoolName string, flags ...string) (bytes.Buffer, error) {
	editMachinePool := m.client.Runner.
		Cmd("edit", "machinepool", machinePoolName).
		CmdFlags(append(flags, "-c", clusterID)...)

	return editMachinePool.Run()
}

// Pasrse the result of 'rosa list machinepool' to MachinePoolList struct
func (m *machinepoolService) ReflectMachinePoolList(result bytes.Buffer) (mpl MachinePoolList, err error) {
	mpl = MachinePoolList{}
	theMap := m.client.Parser.TableData.Input(result).Parse().Output()
	for _, machinepoolItem := range theMap {
		mp := &MachinePool{}
		err = MapStructure(machinepoolItem, mp)
		if err != nil {
			return
		}
		mpl.MachinePools = append(mpl.MachinePools, mp)
	}
	return mpl, err
}

// Pasrse the result of 'rosa list machinepool' to MachinePoolList struct
func (m *machinepoolService) ListAndReflectMachinePools(clusterID string) (mpl MachinePoolList, err error) {
	mpl = MachinePoolList{}
	output, err := m.ListMachinePool(clusterID)
	if err != nil {
		return mpl, err
	}

	mpl, err = m.ReflectMachinePoolList(output)
	return mpl, err
}

// Pasrse the result of 'rosa list machinepool' to MachinePoolList struct
func (m *machinepoolService) ReflectMachinePoolDescription(
	result bytes.Buffer) (mp *MachinePoolDescription, err error) {
	mp = new(MachinePoolDescription)
	theMap, _ := m.client.Parser.TextData.Input(result).Parse().YamlToMap()

	data, _ := yaml.Marshal(&theMap)
	err = yaml.Unmarshal(data, mp)
	return mp, err
}

func (m *machinepoolService) CleanResources(clusterID string) (errors []error) {
	var mpsToDel []string
	mpsToDel = append(mpsToDel, m.machinePools[clusterID]...)
	for _, mpID := range mpsToDel {
		Logger.Infof("Remove remaining machinepool '%s'", mpID)
		_, err := m.DeleteMachinePool(clusterID, mpID)
		if err != nil {
			errors = append(errors, err)
		}
	}

	return
}

// Get specified machinepool by machinepool id
func (mpl MachinePoolList) Machinepool(id string) (mp *MachinePool) {
	for _, mpItem := range mpl.MachinePools {
		if mpItem.ID == id {
			mp = mpItem
			return
		}
	}
	return
}

func (m *machinepoolService) ListAndReflectNodePools(clusterID string) (npl *NodePoolList, err error) {
	output, err := m.ListMachinePool(clusterID)
	if err != nil {
		return nil, err
	}
	return m.ReflectNodePoolList(output)
}

func (m *machinepoolService) DescribeAndReflectNodePool(clusterID string, mpID string) (*NodePoolDescription, error) {
	output, err := m.DescribeMachinePool(clusterID, mpID)
	if err != nil {
		return nil, err
	}
	return m.ReflectNodePoolDescription(output)
}

func (m *machinepoolService) ReflectNodePoolList(result bytes.Buffer) (npl *NodePoolList, err error) {
	npl = new(NodePoolList)
	theMap := m.client.Parser.TableData.Input(result).Parse().Output()
	for _, nodepoolItem := range theMap {
		np := &NodePool{}
		err = MapStructure(nodepoolItem, np)
		if err != nil {
			return
		}
		npl.NodePools = append(npl.NodePools, np)
	}
	return npl, err
}

// Create MachinePool
func (m *machinepoolService) RetrieveHelpForCreate() (output bytes.Buffer, err error) {
	return m.client.Runner.Cmd("create", "machinepool").CmdFlags("-h").Run()
}

// Edit Machinepool
func (m *machinepoolService) RetrieveHelpForEdit() (output bytes.Buffer, err error) {
	return m.client.Runner.Cmd("edit", "machinepool").CmdFlags("-h").Run()
}

// Pasrse the result of 'rosa describe cluster' to the RosaClusterDescription struct
func (m *machinepoolService) ReflectNodePoolDescription(result bytes.Buffer) (*NodePoolDescription, error) {
	theMap, err := m.client.Parser.TextData.Input(result).Parse().YamlToMap()
	if err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(&theMap)
	if err != nil {
		return nil, err
	}
	npd := new(NodePoolDescription)
	err = yaml.Unmarshal(data, npd)
	return npd, err
}

// Get specified nodepool by nodepool id
func (npl NodePoolList) Nodepool(id string) (np *NodePool) {
	for _, npItem := range npl.NodePools {
		if npItem.ID == id {
			np = npItem
			return
		}
	}
	return
}
