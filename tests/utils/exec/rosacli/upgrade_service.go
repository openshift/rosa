package rosacli

import (
	"bytes"

	"gopkg.in/yaml.v3"

	"github.com/openshift/rosa/tests/utils/log"
)

type UpgradeService interface {
	ResourcesCleaner

	ListUpgrades(clusterID string, flags ...string) (bytes.Buffer, error)
	DescribeUpgrade(clusterID string, flags ...string) (bytes.Buffer, error)
	DescribeUpgradeAndReflect(clusterID string) (*UpgradeDescription, error)
	DeleteUpgrade(flags ...string) (bytes.Buffer, error)
	Upgrade(flags ...string) (bytes.Buffer, error)
}

type upgradeService struct {
	ResourcesService
}

func NewUpgradeService(client *Client) UpgradeService {
	return &upgradeService{
		ResourcesService: ResourcesService{
			client: client,
		},
	}
}

// Struct for the 'rosa describe upgrade' output
type UpgradeDescription struct {
	ID                         string `yaml:"ID,omitempty"`
	ClusterID                  string `yaml:"Cluster ID,omitempty"`
	NextRun                    string `yaml:"Next Run,omitempty"`
	Version                    string `yaml:"Version,omitempty"`
	UpgradeState               string `yaml:"Upgrade State,omitempty"`
	StateMesage                string `yaml:"State Message,omitempty"`
	ScheduleType               string `yaml:"Schedule Type,omitempty"`
	ScheduleAt                 string `yaml:"Schedule At,omitempty"`
	EnableMinorVersionUpgrades string `yaml:"Enable minor version upgrades,omitempty"`
}

func (u *upgradeService) ListUpgrades(clusterID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"--cluster", clusterID}, flags...)
	describe := u.client.Runner.
		Cmd("list", "upgrade").
		CmdFlags(combflags...)
	return describe.Run()
}

func (u *upgradeService) DescribeUpgrade(clusterID string, flags ...string) (bytes.Buffer, error) {
	combflags := append([]string{"-c", clusterID}, flags...)
	describe := u.client.Runner.
		Cmd("describe", "upgrade").
		CmdFlags(combflags...)
	return describe.Run()
}

func (u *upgradeService) DescribeUpgradeAndReflect(clusterID string) (res *UpgradeDescription, err error) {
	output, err := u.DescribeUpgrade(clusterID)
	if err != nil {
		return nil, err
	}
	return u.ReflectUpgradeDescription(output)
}

func (u *upgradeService) ReflectUpgradeDescription(result bytes.Buffer) (res *UpgradeDescription, err error) {
	var data []byte
	res = new(UpgradeDescription)
	theMap, err := u.client.
		Parser.
		TextData.
		Input(result).
		Parse().
		YamlToMap()
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

func (u *upgradeService) DeleteUpgrade(flags ...string) (bytes.Buffer, error) {
	DeleteUpgrade := u.client.Runner.
		Cmd("delete", "upgrade").
		CmdFlags(flags...)
	return DeleteUpgrade.Run()
}

func (u *upgradeService) Upgrade(flags ...string) (bytes.Buffer, error) {
	upgrade := u.client.Runner.
		Cmd("upgrade", "cluster").
		CmdFlags(flags...)
	return upgrade.Run()
}

func (u *upgradeService) CleanResources(clusterID string) (errors []error) {
	log.Logger.Debugf("Nothing to clean in Version Service")
	return
}
