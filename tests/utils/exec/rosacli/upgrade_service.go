package rosacli

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/rosa/tests/utils/helper"
	"github.com/openshift/rosa/tests/utils/log"
)

const (
	defaultYStreamPollInterval = time.Second
	defaultYStreamPollTimeout  = time.Second
)

type UpgradeService interface {
	ResourcesCleaner

	ListUpgrades(flags ...string) (bytes.Buffer, error)
	ReflectUpgradeVersionList(result bytes.Buffer) (upgradeVersionList UpgradeVersionList, err error)
	DescribeUpgrade(clusterID string, flags ...string) (bytes.Buffer, error)
	DescribeUpgradeAndReflect(clusterID string) (*UpgradeDescription, error)
	DeleteUpgrade(flags ...string) (bytes.Buffer, error)
	Upgrade(flags ...string) (bytes.Buffer, error)

	WaitForUpgradeToState(clusterID string, state string, timeout int) error
	WaitForAvailableYStreamUpgrade(
		clusterID string,
		preparation *YStreamUpgradePreparation,
		waitInterval time.Duration,
		waitTimeout time.Duration,
	) (string, UpgradeVersionList, error)
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

// Struct of `rosa list upgrade`
type UpgradeVersion struct {
	Version string `json:"VERSION,omitempty"`
	Notes   string `json:"NOTES,omitempty"`
}
type UpgradeVersionList struct {
	UpgradeVersions []UpgradeVersion `json:"UpgradeVersions,omitempty"`
}

func (u UpgradeVersionList) FindNextMinorUpgrade(clusterVersion string) (string, error) {
	_, clusterMinor, _, err := helper.ParseVersion(clusterVersion)
	if err != nil {
		return "", fmt.Errorf("failed to parse cluster version %q: %w", clusterVersion, err)
	}

	var fallback string
	for _, upgradeVersion := range u.UpgradeVersions {
		version := strings.TrimSpace(upgradeVersion.Version)
		if version == "" {
			continue
		}

		_, upgradeMinor, _, err := helper.ParseVersion(version)
		if err != nil {
			return "", fmt.Errorf("failed to parse upgrade version %q: %w", version, err)
		}

		if upgradeMinor != clusterMinor+1 {
			continue
		}

		if strings.Contains(strings.ToLower(upgradeVersion.Notes), "recommended") {
			return version, nil
		}
		if fallback == "" {
			fallback = version
		}
	}

	return fallback, nil
}

func (u *upgradeService) ListUpgrades(flags ...string) (bytes.Buffer, error) {
	describe := u.client.Runner.
		Cmd("list", "upgrade").
		CmdFlags(flags...)
	return describe.Run()
}

func (u *upgradeService) ReflectUpgradeVersionList(
	result bytes.Buffer,
) (upgradeVersionList UpgradeVersionList, err error) {
	upgradeVersionList = UpgradeVersionList{}
	theMap := u.client.Parser.TableData.Input(result).Parse().Output()
	for _, upgradeVersionItem := range theMap {
		upgradeVersion := &UpgradeVersion{}
		err = MapStructure(upgradeVersionItem, upgradeVersion)
		if err != nil {
			return
		}
		upgradeVersionList.UpgradeVersions = append(upgradeVersionList.UpgradeVersions, *upgradeVersion)
	}
	return upgradeVersionList, err
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

func waitForAvailableYStreamUpgrade(
	clusterID string,
	preparation *YStreamUpgradePreparation,
	waitInterval time.Duration,
	waitTimeout time.Duration,
	fetch func() (bytes.Buffer, UpgradeVersionList, error),
) (string, UpgradeVersionList, error) {
	if strings.TrimSpace(clusterID) == "" {
		return "", UpgradeVersionList{}, fmt.Errorf(
			"cluster ID is required when waiting for y-stream upgrade targets")
	}
	if preparation == nil {
		return "", UpgradeVersionList{}, fmt.Errorf(
			"y-stream upgrade preparation is required for cluster %s", clusterID)
	}

	if waitInterval <= 0 {
		waitInterval = defaultYStreamPollInterval
	}
	if waitTimeout <= 0 {
		waitTimeout = defaultYStreamPollTimeout
	}

	var lastOutput bytes.Buffer
	var lastList UpgradeVersionList
	var lastErr error
	var result string

	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	defer cancel()

	pollErr := wait.PollUntilContextCancel(ctx, waitInterval, true,
		func(ctx context.Context) (bool, error) {
			output, upgradeVersionList, err := fetch()
			lastOutput = output
			lastList = upgradeVersionList
			if err != nil {
				lastErr = err
				return false, nil
			}
			upgradingVersion, findErr := upgradeVersionList.FindNextMinorUpgrade(
				preparation.ClusterVersion)
			if findErr != nil {
				lastErr = findErr
				return false, nil
			}
			lastErr = nil
			result = upgradingVersion
			return upgradingVersion != "", nil
		},
	)

	if pollErr == nil {
		return result, lastList, nil
	}

	timeoutErr := fmt.Errorf(
		"timeout after %s waiting for y-stream upgrade target for cluster %s "+
			"(version=%s, current channel=%q, desired channel=%q)",
		waitTimeout,
		clusterID,
		preparation.ClusterVersion,
		preparation.CurrentChannel,
		preparation.DesiredChannel,
	)
	if lastErr != nil {
		return "", lastList, fmt.Errorf(
			"%w: last error: %v; last list output:\n%s",
			timeoutErr,
			lastErr,
			lastOutput.String(),
		)
	}

	return "", lastList, fmt.Errorf(
		"%w: no next-minor upgrade target found; last list output:\n%s",
		timeoutErr,
		lastOutput.String(),
	)
}

func (u *upgradeService) WaitForAvailableYStreamUpgrade(
	clusterID string,
	preparation *YStreamUpgradePreparation,
	waitInterval time.Duration,
	waitTimeout time.Duration,
) (string, UpgradeVersionList, error) {
	return waitForAvailableYStreamUpgrade(
		clusterID,
		preparation,
		waitInterval,
		waitTimeout,
		func() (bytes.Buffer, UpgradeVersionList, error) {
			output, err := u.ListUpgrades("-c", clusterID)
			if err != nil {
				return output, UpgradeVersionList{}, err
			}

			upgradeVersionList, err := u.ReflectUpgradeVersionList(output)
			if err != nil {
				return output, UpgradeVersionList{}, err
			}

			return output, upgradeVersionList, nil
		},
	)
}

func (u *upgradeService) CleanResources(clusterID string) (errors []error) {
	log.Logger.Debugf("Nothing to clean in Version Service")
	return
}

func (u *upgradeService) WaitForUpgradeToState(clusterID string, state string, timeout int) error {
	startTime := time.Now()
	for time.Now().Before(startTime.Add(time.Duration(timeout) * time.Minute)) {
		UD, err := u.DescribeUpgradeAndReflect(clusterID)
		if err != nil {
			return err
		} else {
			if UD.UpgradeState == state {
				return nil
			}
			time.Sleep(1 * time.Minute)
		}
	}
	return fmt.Errorf("ERROR!Timeout after %d minutes to wait for the upgrade into status %s of cluster %s",
		timeout, state, clusterID)
}
