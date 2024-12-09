package rosacli

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/openshift/rosa/tests/utils/helper"
	. "github.com/openshift/rosa/tests/utils/log"
)

type NetworkResourcesService interface {
	ResourcesCleaner
	CreateNetworkResources(isEnvSet bool, flags ...string) (bytes.Buffer, error)
}

type networkResourcesService struct {
	ResourcesService

	nr map[string]string
}

func NewNetworkResourceService(client *Client) NetworkResourcesService {
	return &networkResourcesService{
		ResourcesService: ResourcesService{
			client: client,
		},
		nr: make(map[string]string),
	}
}

func (nr *networkResourcesService) CreateNetworkResources(isEnvSet bool,
	flags ...string) (bytes.Buffer, error) {
	if isEnvSet {
		if slices.Contains(flags, "--template-dir") {
			// This condition is applied for , even if we set 'OCM_TEMPLATE_DIR' env vraiable, it will be overrideen
			// by --template-dir flag
			cmd := fmt.Sprintf("export OCM_TEMPLATE_DIR=/ss/home/; rosa create network %s", strings.Join(flags, " "))
			return nr.client.Runner.RunCMD([]string{"bash", "-c", cmd})
		} else {
			templateDirPath, err := helper.GetCurrentWorkingDir()
			if err != nil {
				return bytes.Buffer{}, err
			}
			cmd := fmt.Sprintf("export OCM_TEMPLATE_DIR=%s; rosa create network %s", templateDirPath, strings.Join(flags, " "))
			return nr.client.Runner.RunCMD([]string{"bash", "-c", cmd})
		}
	} else {
		createNetworkResources := nr.client.Runner.
			Cmd("create", "network").
			CmdFlags(flags...)

		return createNetworkResources.Run()
	}
}

func (nr *networkResourcesService) CleanResources(clusterID string) (errors []error) {
	Logger.Debugf("Nothing to clean in NetworkResourcesService Service")
	return
}
